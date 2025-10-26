package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rpattn/engql/internal/db"
	"github.com/rpattn/engql/internal/domain"
)

// entityRepository implements EntityRepository interface
type entityRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	// cache reference field lookups keyed by schema ID to avoid repeated
	// schema fetches when normalising entity references.
	referenceFieldCache sync.Map
}

type referenceFieldCacheEntry struct {
	fieldName string
	found     bool
}

type skipEntityValidationContextKey struct{}

// WithSkipEntityValidation marks the context so batch inserts can skip
// PostgreSQL trigger validation. Only set this when upstream validation has
// already guaranteed data quality.
func WithSkipEntityValidation(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipEntityValidationContextKey{}, true)
}

func shouldSkipEntityValidation(ctx context.Context) bool {
	flag, ok := ctx.Value(skipEntityValidationContextKey{}).(bool)
	return ok && flag
}

type flushBatchMeta struct {
	BatchID        uuid.UUID
	OrganizationID uuid.UUID
	SchemaID       uuid.UUID
	EntityType     string
	SourceFile     string
	ExpectedRows   int
	SkipValidation bool
}

// NewEntityRepository creates a new entity repository
func NewEntityRepository(queries *db.Queries, pool *pgxpool.Pool) EntityRepository {
	return &entityRepository{
		queries: queries,
		pool:    pool,
	}
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// Create creates a new entity
func (r *entityRepository) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	if err := r.ensureReferenceNormalization(ctx, entity.SchemaID, entity.Properties, true); err != nil {
		return domain.Entity{}, err
	}

	propertiesJSON, err := entity.GetPropertiesAsJSONB()
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to marshal properties: %w", err)
	}

	row, err := r.queries.CreateEntity(ctx, db.CreateEntityParams{
		OrganizationID: entity.OrganizationID,
		SchemaID:       entity.SchemaID,
		EntityType:     entity.EntityType,
		Path:           entity.Path,
		Properties:     propertiesJSON,
	})
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to create entity: %w", err)
	}

	return r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
}

// CreateBatch stages entity rows for asynchronous flushing.
func (r *entityRepository) CreateBatch(ctx context.Context, items []EntityBatchItem, opts EntityBatchOptions) (EntityBatchResult, error) {
	result := EntityBatchResult{}
	if r.pool == nil {
		return result, errors.New("entity repository not initialized")
	}
	if len(items) == 0 {
		return result, nil
	}

	batchID := uuid.New()
	result.BatchID = batchID

	rows := make([][]any, 0, len(items))
	for _, item := range items {
		pathValue := pgtype.Text{}
		if item.Path != "" {
			pathValue = pgtype.Text{String: item.Path, Valid: true}
		}

		var properties map[string]any
		if len(item.PropertiesJSON) > 0 {
			if err := json.Unmarshal(item.PropertiesJSON, &properties); err != nil {
				return EntityBatchResult{}, fmt.Errorf("failed to decode batch properties: %w", err)
			}
		}
		if properties == nil {
			properties = make(map[string]any)
		}

		if err := r.ensureReferenceNormalization(ctx, item.SchemaID, properties, true); err != nil {
			return EntityBatchResult{}, err
		}

		normalizedJSON, err := json.Marshal(properties)
		if err != nil {
			return EntityBatchResult{}, fmt.Errorf("failed to encode batch properties: %w", err)
		}

		rows = append(rows, []any{
			batchID,
			item.OrganizationID,
			item.SchemaID,
			item.EntityType,
			pathValue,
			json.RawMessage(normalizedJSON),
		})
	}

	stagedCount, err := r.stageBatch(ctx, batchID, rows)
	if err != nil {
		return EntityBatchResult{}, err
	}
	result.RowsStaged = int(stagedCount)

	first := items[0]
	fileName := pgtype.Text{}
	sourceFile := strings.TrimSpace(opts.SourceFile)
	if sourceFile != "" {
		fileName = pgtype.Text{String: sourceFile, Valid: true}
	}

	skipValidation := shouldSkipEntityValidation(ctx)
	insertErr := r.queries.InsertEntityIngestBatch(ctx, db.InsertEntityIngestBatchParams{
		ID:             batchID,
		OrganizationID: first.OrganizationID,
		SchemaID:       first.SchemaID,
		EntityType:     first.EntityType,
		FileName:       fileName,
		RowsStaged:     int32(stagedCount),
		SkipValidation: skipValidation,
	})
	if insertErr != nil {
		_ = r.purgeStagedBatch(ctx, batchID)
		return EntityBatchResult{}, fmt.Errorf("failed to record batch metadata: %w", insertErr)
	}

	r.scheduleFlush(flushBatchMeta{
		BatchID:        batchID,
		OrganizationID: first.OrganizationID,
		SchemaID:       first.SchemaID,
		EntityType:     first.EntityType,
		SourceFile:     sourceFile,
		ExpectedRows:   int(stagedCount),
		SkipValidation: skipValidation,
	})

	return result, nil
}

func (r *entityRepository) stageBatch(ctx context.Context, batchID uuid.UUID, rows [][]any) (int64, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to begin staging transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	count, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"entities_ingest"},
		[]string{"batch_id", "organization_id", "schema_id", "entity_type", "path", "properties"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to stage entity batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit staging transaction: %w", err)
	}

	log.Printf("[entityRepository] staged batch %s (rows=%d)", batchID, count)

	return count, nil
}

func (r *entityRepository) purgeStagedBatch(ctx context.Context, batchID uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM entities_ingest WHERE batch_id = $1", batchID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *entityRepository) scheduleFlush(meta flushBatchMeta) {
	flushCtx := context.Background()
	if meta.SkipValidation {
		flushCtx = WithSkipEntityValidation(flushCtx)
	}

	flushCtx, cancel := context.WithTimeout(flushCtx, 15*time.Minute)
	go func() {
		defer cancel()
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[entityRepository] panic while flushing batch %s: %v", meta.BatchID, rec)
			}
		}()

		if err := r.queries.MarkEntityIngestBatchFlushing(flushCtx, meta.BatchID); err != nil {
			log.Printf("[entityRepository] failed to mark batch %s as flushing: %v", meta.BatchID, err)
		}

		log.Printf("[entityRepository] flushing batch %s (expected=%d skipValidation=%t)", meta.BatchID, meta.ExpectedRows, meta.SkipValidation)

		inserted, err := r.flushStagedBatch(flushCtx, meta.BatchID)
		if err != nil {
			log.Printf("[entityRepository] failed to flush batch %s: %v", meta.BatchID, err)
			if markErr := r.queries.MarkEntityIngestBatchFailed(flushCtx, db.MarkEntityIngestBatchFailedParams{
				ID:           meta.BatchID,
				ErrorMessage: pgtype.Text{String: truncateError(err), Valid: true},
			}); markErr != nil {
				log.Printf("[entityRepository] failed to mark batch %s as failed: %v", meta.BatchID, markErr)
			}
			return
		}

		if err := r.queries.MarkEntityIngestBatchCompleted(flushCtx, db.MarkEntityIngestBatchCompletedParams{
			RowsFlushed: int32(inserted),
			ID:          meta.BatchID,
		}); err != nil {
			log.Printf("[entityRepository] flushed batch %s but failed to mark completion: %v", meta.BatchID, err)
			return
		}

		log.Printf("[entityRepository] flushed batch %s into entities (expected=%d inserted=%d)", meta.BatchID, meta.ExpectedRows, inserted)
	}()
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	const maxLen = 512
	msg := err.Error()
	if len(msg) > maxLen {
		return msg[:maxLen]
	}
	return msg
}

func (r *entityRepository) flushStagedBatch(ctx context.Context, batchID uuid.UUID) (int, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to begin flush transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SET LOCAL synchronous_commit = 'off'"); err != nil {
		return 0, fmt.Errorf("failed to relax synchronous commit: %w", err)
	}

	if shouldSkipEntityValidation(ctx) {
		if _, err := tx.Exec(ctx, "SET LOCAL app.skip_entity_property_validation = 'on'"); err != nil {
			return 0, fmt.Errorf("failed to configure batch transaction: %w", err)
		}
	}

	res, err := tx.Exec(ctx, `
        INSERT INTO entities (organization_id, schema_id, entity_type, path, properties)
        SELECT organization_id, schema_id, entity_type, path, properties
        FROM entities_ingest
        WHERE batch_id = $1
        ORDER BY organization_id, entity_type, path
    `, batchID)
	if err != nil {
		return 0, fmt.Errorf("failed to flush staged entities: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM entities_ingest WHERE batch_id = $1", batchID); err != nil {
		return 0, fmt.Errorf("failed to clean staging rows: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit flush transaction: %w", err)
	}

	return int(res.RowsAffected()), nil
}

func (r *entityRepository) ListIngestBatches(ctx context.Context, organizationID *uuid.UUID, statuses []string, limit int, offset int) ([]IngestBatchRecord, error) {
	if len(statuses) == 0 {
		return []IngestBatchRecord{}, nil
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	orgParam := toPGUUID(organizationID)

	rows, err := r.queries.ListEntityIngestBatchesByStatus(ctx, db.ListEntityIngestBatchesByStatusParams{
		Statuses:       statuses,
		OrganizationID: orgParam,
		PageOffset:     int32(offset),
		PageLimit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingest batches: %w", err)
	}

	records := make([]IngestBatchRecord, 0, len(rows))
	for _, row := range rows {
		record := IngestBatchRecord{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			SchemaID:       row.SchemaID,
			EntityType:     row.EntityType,
			RowsStaged:     int(row.RowsStaged),
			RowsFlushed:    int(row.RowsFlushed),
			SkipValidation: row.SkipValidation,
			Status:         row.Status,
			EnqueuedAt:     safeTimestamptz(row.EnqueuedAt),
			StartedAt:      timestamptzPtr(row.StartedAt),
			CompletedAt:    timestamptzPtr(row.CompletedAt),
			UpdatedAt:      row.UpdatedAt,
		}

		if row.FileName.Valid {
			val := row.FileName.String
			record.FileName = &val
		}
		if row.ErrorMessage.Valid {
			msg := row.ErrorMessage.String
			record.ErrorMessage = &msg
		}

		records = append(records, record)
	}

	return records, nil
}

func (r *entityRepository) GetIngestBatchStats(ctx context.Context, organizationID *uuid.UUID) (IngestBatchStats, error) {
	orgParam := toPGUUID(organizationID)

	row, err := r.queries.EntityIngestBatchStats(ctx, orgParam)
	if err != nil {
		return IngestBatchStats{}, fmt.Errorf("failed to fetch ingest batch stats: %w", err)
	}

	return IngestBatchStats{
		TotalBatches:      row.TotalBatches,
		InProgressBatches: row.InProgressBatches,
		CompletedBatches:  row.CompletedBatches,
		FailedBatches:     row.FailedBatches,
		TotalRowsStaged:   row.TotalRowsStaged,
		TotalRowsFlushed:  row.TotalRowsFlushed,
	}, nil
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	s := value.String
	return &s
}

func safeTimestamptz(value pgtype.Timestamptz) time.Time {
	if value.Valid {
		return value.Time
	}
	return time.Time{}
}

func toPGUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	var buf [16]byte
	copy(buf[:], id[:])
	return pgtype.UUID{Bytes: buf, Valid: true}
}

// GetByID retrieves an entity by ID
func (r *entityRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error) {
	row, err := r.queries.GetEntity(ctx, id)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to get entity: %w", err)
	}

	return r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
}

// GetByIDs retrieves multiple entities by their IDs.
func (r *entityRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Entity, error) {
	if len(ids) == 0 {
		return []domain.Entity{}, nil
	}

	rows, err := r.queries.GetEntitiesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities by IDs: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// GetHistoryByVersion retrieves a historical entity snapshot by version.
func (r *entityRepository) GetHistoryByVersion(ctx context.Context, entityID uuid.UUID, version int64) (domain.EntityHistory, error) {
	row, err := r.queries.GetEntityHistoryByVersion(ctx, db.GetEntityHistoryByVersionParams{
		EntityID: entityID,
		Version:  version,
	})
	if err != nil {
		return domain.EntityHistory{}, fmt.Errorf("failed to get entity history: %w", err)
	}

	return buildEntityHistory(row)
}

// ListHistory retrieves all historical versions for an entity.
func (r *entityRepository) ListHistory(ctx context.Context, entityID uuid.UUID) ([]domain.EntityHistory, error) {
	rows, err := r.queries.ListEntityHistory(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity history: %w", err)
	}

	history := make([]domain.EntityHistory, len(rows))
	for i, row := range rows {
		snapshot, err := buildEntityHistory(row)
		if err != nil {
			return nil, err
		}
		history[i] = snapshot
	}

	return history, nil
}

// List retrieves entities for an organization applying optional filters.
func (r *entityRepository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	filter *domain.EntityFilter,
	sort *domain.EntitySort,
	limit int,
	offset int,
) ([]domain.Entity, int, error) {
	params := db.ListEntitiesParams{
		OrganizationID: organizationID,
		EntityType:     "",
		PropertyKeys:   nil,
		PropertyValues: nil,
		TextSearch:     "",
		PageLimit:      int32(limit),
		PageOffset:     int32(offset),
		SortField:      string(domain.EntitySortFieldCreatedAt),
		SortDirection:  string(domain.SortDirectionDesc),
		SortProperty:   sql.NullString{},
	}

	if filter != nil {
		if filter.EntityType != "" {
			params.EntityType = filter.EntityType
		}

		for _, propertyFilter := range filter.PropertyFilters {
			if propertyFilter.Key == "" {
				continue
			}
			params.PropertyKeys = append(params.PropertyKeys, propertyFilter.Key)
			params.PropertyValues = append(params.PropertyValues, "%"+propertyFilter.Value+"%")
		}

		if trimmed := strings.TrimSpace(filter.TextSearch); trimmed != "" {
			params.TextSearch = "%" + trimmed + "%"
		}
	}

	if sort != nil {
		switch sort.Field {
		case domain.EntitySortFieldCreatedAt,
			domain.EntitySortFieldUpdatedAt,
			domain.EntitySortFieldEntityType,
			domain.EntitySortFieldPath,
			domain.EntitySortFieldVersion:
			params.SortField = string(sort.Field)
		case domain.EntitySortFieldProperty:
			if sort.PropertyKey != "" {
				params.SortField = string(sort.Field)
				params.SortProperty = sql.NullString{String: sort.PropertyKey, Valid: true}
			}
		}

		switch sort.Direction {
		case domain.SortDirectionAsc:
			params.SortDirection = string(domain.SortDirectionAsc)
		case domain.SortDirectionDesc:
			params.SortDirection = string(domain.SortDirectionDesc)
		}
	}

	rows, err := r.queries.ListEntities(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list entities: %w", err)
	}

	if len(rows) == 0 {
		return nil, 0, nil
	}

	entities := make([]domain.Entity, len(rows))
	var totalCount int

	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		entities[i] = entity

		if i == 0 {
			totalCount = int(row.TotalCount)
		}
	}

	return entities, totalCount, nil
}

// ListByType retrieves all entities of a specific type for an organization
func (r *entityRepository) ListByType(ctx context.Context, organizationID uuid.UUID, entityType string) ([]domain.Entity, error) {
	rows, err := r.queries.ListEntitiesByType(ctx, db.ListEntitiesByTypeParams{
		OrganizationID: organizationID,
		EntityType:     entityType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list entities by type: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// GetByReference resolves an entity by its canonical reference value.
func (r *entityRepository) GetByReference(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValue string) (domain.Entity, error) {
	fieldName, found, err := r.referenceFieldForType(ctx, organizationID, entityType)
	if err != nil {
		return domain.Entity{}, err
	}
	if !found {
		return domain.Entity{}, fmt.Errorf("entity type %s does not declare a reference field", entityType)
	}

	normalized := strings.TrimSpace(referenceValue)
	if normalized == "" {
		return domain.Entity{}, fmt.Errorf("reference value cannot be empty")
	}

	row, err := r.queries.GetEntityByReference(ctx, db.GetEntityByReferenceParams{
		OrganizationID: organizationID,
		EntityType:     entityType,
		FieldName:      fieldName,
		ReferenceValue: normalized,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Entity{}, fmt.Errorf("no %s entity found for reference %q: %w", entityType, normalized, err)
		}
		return domain.Entity{}, fmt.Errorf("failed to lookup entity by reference: %w", err)
	}

	return r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
}

// ListByReferences resolves every entity whose reference matches one of the provided values.
func (r *entityRepository) ListByReferences(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValues []string) ([]domain.Entity, error) {
	if len(referenceValues) == 0 {
		return []domain.Entity{}, nil
	}

	fieldName, found, err := r.referenceFieldForType(ctx, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("entity type %s does not declare a reference field", entityType)
	}

	normalized := make([]string, 0, len(referenceValues))
	seen := make(map[string]struct{}, len(referenceValues))
	for _, value := range referenceValues {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return []domain.Entity{}, nil
	}

	rows, err := r.queries.ListEntitiesByReferences(ctx, db.ListEntitiesByReferencesParams{
		OrganizationID:  organizationID,
		EntityType:      entityType,
		FieldName:       fieldName,
		ReferenceValues: normalized,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list entities by reference: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// Update updates an entity
func (r *entityRepository) Update(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	if err := r.ensureReferenceNormalization(ctx, entity.SchemaID, entity.Properties, true); err != nil {
		return domain.Entity{}, err
	}

	propertiesJSON, err := entity.GetPropertiesAsJSONB()
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to marshal properties: %w", err)
	}

	row, err := r.queries.UpdateEntity(ctx, db.UpdateEntityParams{
		ID:         entity.ID,
		SchemaID:   entity.SchemaID,
		EntityType: entity.EntityType,
		Path:       entity.Path,
		Properties: propertiesJSON,
	})
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to update entity: %w", err)
	}

	return r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
}

// Delete deletes an entity
func (r *entityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteEntity(ctx, id); err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	return nil
}

// RollbackEntity restores a previous entity version as a new version
func (r *entityRepository) RollbackEntity(ctx context.Context, id string, toVersion int64, reason string) error {
	entityID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid entity id: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to open transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	history, err := qtx.GetEntityHistoryByVersion(ctx, db.GetEntityHistoryByVersionParams{
		EntityID: entityID,
		Version:  toVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to load entity history: %w", err)
	}

	rollbackReason := strings.TrimSpace(reason)
	if rollbackReason == "" {
		rollbackReason = "ROLLBACK"
	} else {
		rollbackReason = "ROLLBACK: " + rollbackReason
	}

	setReasonSQL := fmt.Sprintf("SET LOCAL app.reason = %s", quoteLiteral(rollbackReason))
	if _, err := tx.Exec(ctx, setReasonSQL); err != nil {
		return fmt.Errorf("failed to set rollback reason: %w", err)
	}

	_, currentErr := qtx.GetEntity(ctx, entityID)
	if currentErr == nil {
		if _, err := qtx.UpdateEntity(ctx, db.UpdateEntityParams{
			ID:         entityID,
			SchemaID:   history.SchemaID,
			EntityType: history.EntityType,
			Path:       history.Path,
			Properties: history.Properties,
		}); err != nil {
			return fmt.Errorf("failed to apply rollback update: %w", err)
		}
	} else {
		if !errors.Is(currentErr, pgx.ErrNoRows) {
			return fmt.Errorf("failed to fetch entity for rollback: %w", currentErr)
		}

		maxVersion, err := qtx.GetMaxEntityHistoryVersion(ctx, entityID)
		if err != nil {
			return fmt.Errorf("failed to compute next entity version: %w", err)
		}
		nextVersion := maxVersion + 1

		if err := qtx.UpsertEntityFromHistory(ctx, db.UpsertEntityFromHistoryParams{
			ID:             entityID,
			OrganizationID: history.OrganizationID,
			SchemaID:       history.SchemaID,
			EntityType:     history.EntityType,
			Path:           history.Path,
			Properties:     history.Properties,
			Version:        nextVersion,
			CreatedAt:      history.CreatedAt,
		}); err != nil {
			return fmt.Errorf("failed to restore deleted entity: %w", err)
		}

		if err := qtx.InsertEntityHistoryRecord(ctx, db.InsertEntityHistoryRecordParams{
			EntityID:       entityID,
			OrganizationID: history.OrganizationID,
			SchemaID:       history.SchemaID,
			EntityType:     history.EntityType,
			Path:           history.Path,
			Properties:     history.Properties,
			CreatedAt:      history.CreatedAt,
			UpdatedAt:      time.Now(),
			Version:        nextVersion,
			ChangeType:     "ROLLBACK",
			Reason:         pgtype.Text{String: rollbackReason, Valid: true},
		}); err != nil {
			return fmt.Errorf("failed to record rollback history: %w", err)
		}

		// Ensure triggers capture the restored state for future updates
		if _, err := tx.Exec(ctx, "SET LOCAL app.reason = NULL"); err != nil {
			return fmt.Errorf("failed to clear rollback reason: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	return nil
}

// GetAncestors retrieves ancestor entities
func (r *entityRepository) GetAncestors(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	rows, err := r.queries.GetEntityAncestors(ctx, db.GetEntityAncestorsParams{
		OrganizationID: organizationID,
		Column2:        path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entity ancestors: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// GetDescendants retrieves descendant entities
func (r *entityRepository) GetDescendants(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	rows, err := r.queries.GetEntityDescendants(ctx, db.GetEntityDescendantsParams{
		OrganizationID: organizationID,
		Column2:        pgtype.Text{String: path, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entity descendants: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// GetChildren retrieves direct child entities
func (r *entityRepository) GetChildren(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	rows, err := r.queries.GetEntityChildren(ctx, db.GetEntityChildrenParams{
		OrganizationID: organizationID,
		Column2:        pgtype.Text{String: path, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entity children: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// GetSiblings retrieves sibling entities
func (r *entityRepository) GetSiblings(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	rows, err := r.queries.GetEntitySiblings(ctx, db.GetEntitySiblingsParams{
		OrganizationID: organizationID,
		Column2:        pgtype.Text{String: path, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entity siblings: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// FilterByProperty filters entities by JSONB property match
func (r *entityRepository) FilterByProperty(ctx context.Context, organizationID uuid.UUID, filter map[string]any) ([]domain.Entity, error) {
	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filter: %w", err)
	}

	rows, err := r.queries.FilterEntitiesByProperty(ctx, db.FilterEntitiesByPropertyParams{
		OrganizationID: organizationID,
		Properties:     filterJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to filter entities by property: %w", err)
	}

	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := r.buildEntity(ctx, row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// Count returns the total count of entities for an organization
func (r *entityRepository) Count(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	count, err := r.queries.GetEntityCount(ctx, organizationID)
	if err != nil {
		return 0, fmt.Errorf("failed to get entity count: %w", err)
	}
	return count, nil
}

// CountByType returns the count of entities of a specific type for an organization
func (r *entityRepository) CountByType(ctx context.Context, organizationID uuid.UUID, entityType string) (int64, error) {
	count, err := r.queries.GetEntityCountByType(ctx, db.GetEntityCountByTypeParams{
		OrganizationID: organizationID,
		EntityType:     entityType,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get entity count by type: %w", err)
	}
	return count, nil
}

func (r *entityRepository) ensureReferenceNormalization(ctx context.Context, schemaID uuid.UUID, properties map[string]any, strict bool) error {
	if properties == nil {
		return nil
	}

	fieldName, found, err := r.referenceFieldForSchema(ctx, schemaID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	raw, exists := properties[fieldName]
	if !exists {
		return nil
	}

	str, ok := raw.(string)
	if !ok {
		if strict {
			return fmt.Errorf("reference field %s must be a string", fieldName)
		}
		return nil
	}

	normalized := strings.TrimSpace(str)
	if normalized == "" {
		if strict {
			return fmt.Errorf("reference field %s cannot be empty", fieldName)
		}
		properties[fieldName] = normalized
		return nil
	}

	properties[fieldName] = normalized
	return nil
}

func (r *entityRepository) referenceFieldForSchema(ctx context.Context, schemaID uuid.UUID) (string, bool, error) {
	if cached, ok := r.referenceFieldCache.Load(schemaID); ok {
		entry := cached.(referenceFieldCacheEntry)
		return entry.fieldName, entry.found, nil
	}

	row, err := r.queries.GetEntitySchema(ctx, schemaID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.referenceFieldCache.Store(schemaID, referenceFieldCacheEntry{found: false})
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to load entity schema %s: %w", schemaID, err)
	}

	fieldName, found, err := extractReferenceField(row.Fields)
	if err != nil {
		return "", false, err
	}

	r.referenceFieldCache.Store(schemaID, referenceFieldCacheEntry{fieldName: fieldName, found: found})
	return fieldName, found, nil
}

func (r *entityRepository) referenceFieldForType(ctx context.Context, organizationID uuid.UUID, entityType string) (string, bool, error) {
	row, err := r.queries.GetEntitySchemaByName(ctx, db.GetEntitySchemaByNameParams{
		OrganizationID: organizationID,
		Name:           entityType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to load schema for entity type %s: %w", entityType, err)
	}

	return extractReferenceField(row.Fields)
}

func extractReferenceField(fieldsJSON []byte) (string, bool, error) {
	fields, err := domain.FromJSONBFields(fieldsJSON)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse schema fields: %w", err)
	}

	for _, field := range fields {
		if strings.EqualFold(string(field.Type), string(domain.FieldTypeReference)) {
			if field.Name == "" {
				return "", false, fmt.Errorf("reference field must declare a name")
			}
			return field.Name, true, nil
		}
	}

	return "", false, nil
}

func (r *entityRepository) buildEntity(
	ctx context.Context,
	id uuid.UUID,
	orgID uuid.UUID,
	schemaID uuid.UUID,
	entityType string,
	path string,
	propertiesJSON json.RawMessage,
	version int64,
	createdAt time.Time,
	updatedAt time.Time,
) (domain.Entity, error) {
	properties, err := domain.FromJSONBProperties(propertiesJSON)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to decode properties for entity %s: %w", id, err)
	}

	if err := r.ensureReferenceNormalization(ctx, schemaID, properties, false); err != nil {
		return domain.Entity{}, err
	}

	return domain.Entity{
		ID:             id,
		OrganizationID: orgID,
		SchemaID:       schemaID,
		EntityType:     entityType,
		Path:           path,
		Properties:     properties,
		Version:        version,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func buildEntityHistory(row db.EntitiesHistory) (domain.EntityHistory, error) {
	properties, err := domain.FromJSONBProperties(row.Properties)
	if err != nil {
		return domain.EntityHistory{}, fmt.Errorf("failed to decode properties for entity history %s: %w", row.ID, err)
	}

	return domain.EntityHistory{
		ID:             row.ID,
		EntityID:       row.EntityID,
		OrganizationID: row.OrganizationID,
		SchemaID:       row.SchemaID,
		EntityType:     row.EntityType,
		Path:           row.Path,
		Properties:     properties,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Version:        row.Version,
		ChangeType:     row.ChangeType,
		ChangedAt:      timestamptzPtr(row.ChangedAt),
		Reason:         textPtr(row.Reason),
	}, nil
}

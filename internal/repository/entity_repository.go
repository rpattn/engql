package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
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

	return buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
}

// CreateBatch inserts multiple entities using PostgreSQL COPY for efficiency.
func (r *entityRepository) CreateBatch(ctx context.Context, items []EntityBatchItem) (int, error) {
	if r.pool == nil {
		return 0, errors.New("entity repository not initialized")
	}
	if len(items) == 0 {
		return 0, nil
	}

	batchID := uuid.New()
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		pathValue := pgtype.Text{}
		if item.Path != "" {
			pathValue = pgtype.Text{
				String: item.Path,
				Valid:  true,
			}
		}

		rows = append(rows, []any{
			batchID,
			item.OrganizationID,
			item.SchemaID,
			item.EntityType,
			pathValue,
			json.RawMessage(item.PropertiesJSON),
		})
	}

	stagedCount, err := r.stageBatch(ctx, batchID, rows)
	if err != nil {
		return 0, err
	}

	skipValidation := shouldSkipEntityValidation(ctx)
	r.scheduleFlush(batchID, int(stagedCount), skipValidation)

	return int(stagedCount), nil
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

func (r *entityRepository) scheduleFlush(batchID uuid.UUID, expected int, skipValidation bool) {
	flushCtx := context.Background()
	if skipValidation {
		flushCtx = WithSkipEntityValidation(flushCtx)
	}

	flushCtx, cancel := context.WithTimeout(flushCtx, 15*time.Minute)
	go func() {
		defer cancel()
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[entityRepository] panic while flushing batch %s: %v", batchID, rec)
			}
		}()

		log.Printf("[entityRepository] flushing batch %s (expected=%d skipValidation=%t)", batchID, expected, skipValidation)

		inserted, err := r.flushStagedBatch(flushCtx, batchID)
		if err != nil {
			log.Printf("[entityRepository] failed to flush batch %s: %v", batchID, err)
			return
		}

		log.Printf("[entityRepository] flushed batch %s into entities (expected=%d inserted=%d)", batchID, expected, inserted)
	}()
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

// GetByID retrieves an entity by ID
func (r *entityRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error) {
	row, err := r.queries.GetEntity(ctx, id)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to get entity: %w", err)
	}

	return buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
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
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// List retrieves entities for an organization applying optional filters.
func (r *entityRepository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	filter *domain.EntityFilter,
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
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
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
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}

	return entities, nil
}

// Update updates an entity
func (r *entityRepository) Update(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
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

	return buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
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
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
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
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
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
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
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
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
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

	return convertFilterRows(rows)
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

func convertEntityListRows(rows []db.ListEntitiesRow) ([]domain.Entity, error) {
	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}
	return entities, nil
}

func convertFilterRows(rows []db.FilterEntitiesByPropertyRow) ([]domain.Entity, error) {
	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		entity, err := buildEntity(row.ID, row.OrganizationID, row.SchemaID, row.EntityType, row.Path, row.Properties, row.Version, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		entities[i] = entity
	}
	return entities, nil
}

func buildEntity(
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

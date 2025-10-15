package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// List retrieves all entities for an organization
func (r *entityRepository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
	offset int,
) ([]domain.Entity, int, error) {
	params := db.ListEntitiesParams{
		OrganizationID: organizationID,
		Limit:          int32(limit),
		Offset:         int32(offset),
	}
	rows, err := r.queries.ListEntities(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list entities: %w", err)
	}

	entities, err := convertEntityListRows(rows)
	if err != nil {
		return nil, 0, err
	}

	totalCount := 0
	if len(rows) > 0 {
		totalCount = int(rows[0].TotalCount)
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

package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"graphql-engineering-api/internal/db"
	"graphql-engineering-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// entityRepository implements EntityRepository interface
type entityRepository struct {
	queries *db.Queries
}

// NewEntityRepository creates a new entity repository
func NewEntityRepository(queries *db.Queries) EntityRepository {
	return &entityRepository{
		queries: queries,
	}
}

// Create creates a new entity
func (r *entityRepository) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	propertiesJSON, err := entity.GetPropertiesAsJSONB()
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to marshal properties: %w", err)
	}

	row, err := r.queries.CreateEntity(ctx, db.CreateEntityParams{
		OrganizationID: entity.OrganizationID,
		EntityType:     entity.EntityType,
		Path:           entity.Path,
		Properties:     propertiesJSON,
	})
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to create entity: %w", err)
	}

	properties, err := domain.FromJSONBProperties(row.Properties)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to unmarshal properties: %w", err)
	}

	return domain.Entity{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		EntityType:     row.EntityType,
		Path:           row.Path,
		Properties:     properties,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// GetByID retrieves an entity by ID
func (r *entityRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error) {
	row, err := r.queries.GetEntity(ctx, id)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to get entity: %w", err)
	}

	properties, err := domain.FromJSONBProperties(row.Properties)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to unmarshal properties: %w", err)
	}

	return domain.Entity{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		EntityType:     row.EntityType,
		Path:           row.Path,
		Properties:     properties,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
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
		props, err := domain.FromJSONBProperties(row.Properties)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal properties for entity %s: %w", row.ID, err)
		}

		entities[i] = domain.Entity{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			EntityType:     row.EntityType,
			Path:           row.Path,
			Properties:     props,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
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
	// Fetch paginated rows
	params := db.ListEntitiesParams{
		OrganizationID: organizationID,
		Limit:          int32(limit),
		Offset:         int32(offset),
	}
	rows, err := r.queries.ListEntities(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list entities: %w", err)
	}

	entities, err := r.dbRowsToEntities(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert rows to entities: %w", err)
	}

	// Fetch total count
	totalCount := 0
	if len(rows) > 0 {
		totalCount = int(rows[0].TotalCount)
	}

	return entities, int(totalCount), nil
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

	return r.rowsToEntities(rows)
}

// Update updates an entity
func (r *entityRepository) Update(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	propertiesJSON, err := entity.GetPropertiesAsJSONB()
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to marshal properties: %w", err)
	}

	row, err := r.queries.UpdateEntity(ctx, db.UpdateEntityParams{
		ID:         entity.ID,
		EntityType: entity.EntityType,
		Path:       entity.Path,
		Properties: propertiesJSON,
	})
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to update entity: %w", err)
	}

	properties, err := domain.FromJSONBProperties(row.Properties)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("failed to unmarshal properties: %w", err)
	}

	return domain.Entity{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		EntityType:     row.EntityType,
		Path:           row.Path,
		Properties:     properties,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// Delete deletes an entity
func (r *entityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteEntity(ctx, id); err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
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

	return r.rowsToEntities(rows)
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

	return r.rowsToEntities(rows)
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

	return r.rowsToEntities(rows)
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

	return r.rowsToEntities(rows)
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

	return r.rowsToEntities(rows)
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

// rowsToEntities converts database rows to domain entities
func (r *entityRepository) rowsToEntities(rows []db.Entity) ([]domain.Entity, error) {
	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		properties, err := domain.FromJSONBProperties(row.Properties)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal properties for entity %s: %w", row.ID, err)
		}

		entities[i] = domain.Entity{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			EntityType:     row.EntityType,
			Path:           row.Path,
			Properties:     properties,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}

	return entities, nil
}

// rowsToEntities converts database rows to domain entities
func (r *entityRepository) dbRowsToEntities(rows []db.ListEntitiesRow) ([]domain.Entity, error) {
	entities := make([]domain.Entity, len(rows))
	for i, row := range rows {
		properties, err := domain.FromJSONBProperties(row.Properties)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal properties for entity %s: %w", row.ID, err)
		}

		entities[i] = domain.Entity{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			EntityType:     row.EntityType,
			Path:           row.Path,
			Properties:     properties,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}

	return entities, nil
}

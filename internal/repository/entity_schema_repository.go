package repository

import (
	"context"
	"fmt"

	"graphql-engineering-api/internal/db"
	"graphql-engineering-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// entitySchemaRepository implements EntitySchemaRepository interface
type entitySchemaRepository struct {
	queries *db.Queries
}

// NewEntitySchemaRepository creates a new entity schema repository
func NewEntitySchemaRepository(queries *db.Queries) EntitySchemaRepository {
	return &entitySchemaRepository{
		queries: queries,
	}
}

// Create creates a new entity schema
func (r *entitySchemaRepository) Create(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	fieldsJSON, err := schema.GetFieldsAsJSONB()
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to marshal fields: %w", err)
	}

	row, err := r.queries.CreateEntitySchema(ctx, db.CreateEntitySchemaParams{
		OrganizationID: schema.OrganizationID,
		Name:           schema.Name,
		Description:    pgtype.Text{String: schema.Description, Valid: true},
		Fields:         fieldsJSON,
	})
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to create entity schema: %w", err)
	}

	fields, err := domain.FromJSONBFields(row.Fields)
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	return domain.EntitySchema{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Description:    row.Description.String,
		Fields:         fields,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// GetByID retrieves an entity schema by ID
func (r *entitySchemaRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.EntitySchema, error) {
	row, err := r.queries.GetEntitySchema(ctx, id)
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to get entity schema: %w", err)
	}

	fields, err := domain.FromJSONBFields(row.Fields)
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	return domain.EntitySchema{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Description:    row.Description.String,
		Fields:         fields,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// GetByName retrieves an entity schema by organization ID and name
func (r *entitySchemaRepository) GetByName(ctx context.Context, organizationID uuid.UUID, name string) (domain.EntitySchema, error) {
	row, err := r.queries.GetEntitySchemaByName(ctx, db.GetEntitySchemaByNameParams{
		OrganizationID: organizationID,
		Name:           name,
	})
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to get entity schema by name: %w", err)
	}

	fields, err := domain.FromJSONBFields(row.Fields)
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	return domain.EntitySchema{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Description:    row.Description.String,
		Fields:         fields,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// List retrieves all entity schemas for an organization
func (r *entitySchemaRepository) List(ctx context.Context, organizationID uuid.UUID) ([]domain.EntitySchema, error) {
	rows, err := r.queries.ListEntitySchemas(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity schemas: %w", err)
	}

	schemas := make([]domain.EntitySchema, len(rows))
	for i, row := range rows {
		fields, err := domain.FromJSONBFields(row.Fields)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal fields for schema %s: %w", row.Name, err)
		}

		schemas[i] = domain.EntitySchema{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			Name:           row.Name,
			Description:    row.Description.String,
			Fields:         fields,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}

	return schemas, nil
}

// Update updates an entity schema
func (r *entitySchemaRepository) Update(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	fieldsJSON, err := schema.GetFieldsAsJSONB()
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to marshal fields: %w", err)
	}

	row, err := r.queries.UpdateEntitySchema(ctx, db.UpdateEntitySchemaParams{
		ID:          schema.ID,
		Name:        schema.Name,
		Description: pgtype.Text{String: schema.Description, Valid: true},
		Fields:      fieldsJSON,
	})
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to update entity schema: %w", err)
	}

	fields, err := domain.FromJSONBFields(row.Fields)
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	return domain.EntitySchema{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Description:    row.Description.String,
		Fields:         fields,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// Delete deletes an entity schema
func (r *entitySchemaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteEntitySchema(ctx, id); err != nil {
		return fmt.Errorf("failed to delete entity schema: %w", err)
	}
	return nil
}

// Exists checks if an entity schema exists for the given organization and name
func (r *entitySchemaRepository) Exists(ctx context.Context, organizationID uuid.UUID, name string) (bool, error) {
	exists, err := r.queries.SchemaExists(ctx, db.SchemaExistsParams{
		OrganizationID: organizationID,
		Name:           name,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check schema existence: %w", err)
	}
	return exists, nil
}

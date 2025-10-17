package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rpattn/engql/internal/db"
	"github.com/rpattn/engql/internal/domain"
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

func (r *entitySchemaRepository) Create(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	return r.insertSchema(ctx, schema)
}

func (r *entitySchemaRepository) CreateVersion(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	return r.insertSchema(ctx, schema)
}

func (r *entitySchemaRepository) ArchiveSchema(ctx context.Context, schemaID uuid.UUID) error {
	err := r.queries.MarkEntitySchemaInactive(ctx, schemaID)
	if err != nil {
		return fmt.Errorf("failed to archive entity schema: %w", err)
	}
	return nil
}

// GetByID retrieves an entity schema by ID
func (r *entitySchemaRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.EntitySchema, error) {
	row, err := r.queries.GetEntitySchema(ctx, id)
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to get entity schema: %w", err)
	}

	return mapSchemaRow(row.ID, row.OrganizationID, row.Name, row.Description, row.Fields, row.Version, row.PreviousVersionID, row.Status, row.CreatedAt, row.UpdatedAt)
}

// GetByName retrieves the latest entity schema by organization ID and name
func (r *entitySchemaRepository) GetByName(ctx context.Context, organizationID uuid.UUID, name string) (domain.EntitySchema, error) {
	row, err := r.queries.GetEntitySchemaByName(ctx, db.GetEntitySchemaByNameParams{
		OrganizationID: organizationID,
		Name:           name,
	})
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to get entity schema by name: %w", err)
	}

	return mapSchemaRow(row.ID, row.OrganizationID, row.Name, row.Description, row.Fields, row.Version, row.PreviousVersionID, row.Status, row.CreatedAt, row.UpdatedAt)
}

// List retrieves the latest version of all schemas for an organization
func (r *entitySchemaRepository) List(ctx context.Context, organizationID uuid.UUID) ([]domain.EntitySchema, error) {
	rows, err := r.queries.ListEntitySchemas(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity schemas: %w", err)
	}

	result := make([]domain.EntitySchema, len(rows))
	for i, row := range rows {
		mapped, mapErr := mapSchemaRow(row.ID, row.OrganizationID, row.Name, row.Description, row.Fields, row.Version, row.PreviousVersionID, row.Status, row.CreatedAt, row.UpdatedAt)
		if mapErr != nil {
			return nil, mapErr
		}
		result[i] = mapped
	}
	return result, nil
}

// ListVersions returns every version for a given schema name
func (r *entitySchemaRepository) ListVersions(ctx context.Context, organizationID uuid.UUID, name string) ([]domain.EntitySchema, error) {
	rows, err := r.queries.ListEntitySchemaVersions(ctx, db.ListEntitySchemaVersionsParams{
		OrganizationID: organizationID,
		Name:           name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list schema versions: %w", err)
	}

	result := make([]domain.EntitySchema, len(rows))
	for i, row := range rows {
		mapped, mapErr := mapSchemaRow(row.ID, row.OrganizationID, row.Name, row.Description, row.Fields, row.Version, row.PreviousVersionID, row.Status, row.CreatedAt, row.UpdatedAt)
		if mapErr != nil {
			return nil, mapErr
		}
		result[i] = mapped
	}
	return result, nil
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

// insertSchema persists a schema version row.
func (r *entitySchemaRepository) insertSchema(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	fieldsJSON, err := schema.GetFieldsAsJSONB()
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to marshal fields: %w", err)
	}

	var previous pgtype.UUID
	if schema.PreviousVersionID != nil {
		previous = pgtype.UUID{Valid: true}
		prevVal := *schema.PreviousVersionID
		copy(previous.Bytes[:], prevVal[:])
	}

	row, err := r.queries.CreateEntitySchemaAndArchivePrevious(ctx, db.CreateEntitySchemaAndArchivePreviousParams{
		OrganizationID:    schema.OrganizationID,
		Name:              schema.Name,
		Description:       pgtype.Text{String: schema.Description, Valid: schema.Description != ""},
		Fields:            fieldsJSON,
		Version:           schema.Version,
		PreviousVersionID: previous,
		Status:            string(schema.Status),
	})
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to insert entity schema: %w", err)
	}

	return mapSchemaRow(row.ID, row.OrganizationID, row.Name, row.Description, row.Fields, row.Version, row.PreviousVersionID, row.Status, row.CreatedAt, row.UpdatedAt)
}

func mapSchemaRow(id uuid.UUID, orgID uuid.UUID, name string, description pgtype.Text, fieldsJSON []byte, version string, previous pgtype.UUID, status string, createdAt, updatedAt time.Time) (domain.EntitySchema, error) {
	fields, err := domain.FromJSONBFields(fieldsJSON)
	if err != nil {
		return domain.EntitySchema{}, fmt.Errorf("failed to unmarshal fields for schema %s: %w", name, err)
	}

	var previousID *uuid.UUID
	if previous.Valid {
		prev, convErr := uuid.FromBytes(previous.Bytes[:])
		if convErr != nil {
			return domain.EntitySchema{}, fmt.Errorf("invalid previous version identifier: %w", convErr)
		}
		previousID = &prev
	}

	return domain.EntitySchema{
		ID:                id,
		OrganizationID:    orgID,
		Name:              name,
		Description:       description.String,
		Fields:            fields,
		Version:           version,
		PreviousVersionID: previousID,
		Status:            domain.SchemaStatus(status),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}

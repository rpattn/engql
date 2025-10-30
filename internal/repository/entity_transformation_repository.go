package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rpattn/engql/internal/db"
	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type entityTransformationRepository struct {
	queries *db.Queries
}

// NewEntityTransformationRepository returns a repository for DAG definitions.
func NewEntityTransformationRepository(queries *db.Queries, _ db.DBTX) EntityTransformationRepository {
	return &entityTransformationRepository{
		queries: queries,
	}
}

func (r *entityTransformationRepository) Create(ctx context.Context, transformation domain.EntityTransformation) (domain.EntityTransformation, error) {
	if transformation.ID == uuid.Nil {
		transformation.ID = uuid.New()
	}
	nodesJSON, err := domain.EntityTransformationNodesToJSON(transformation.Nodes)
	if err != nil {
		return domain.EntityTransformation{}, fmt.Errorf("marshal nodes: %w", err)
	}
	row, err := r.queries.CreateEntityTransformation(ctx, db.CreateEntityTransformationParams{
		ID:             transformation.ID,
		OrganizationID: transformation.OrganizationID,
		Name:           transformation.Name,
		Description:    pgtype.Text{String: transformation.Description, Valid: transformation.Description != ""},
		Nodes:          nodesJSON,
	})
	if err != nil {
		return domain.EntityTransformation{}, fmt.Errorf("create entity transformation: %w", err)
	}
	return mapTransformationRow(convertEntityTransformationRow(row))
}

func (r *entityTransformationRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.EntityTransformation, error) {
	row, err := r.queries.GetEntityTransformation(ctx, id)
	if err != nil {
		return domain.EntityTransformation{}, fmt.Errorf("get entity transformation: %w", err)
	}
	return mapTransformationRow(convertEntityTransformationRow(row))
}

func (r *entityTransformationRepository) ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]domain.EntityTransformation, error) {
	rows, err := r.queries.ListEntityTransformationsByOrganization(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list entity transformations: %w", err)
	}
	result := make([]domain.EntityTransformation, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapTransformationRow(convertEntityTransformationRow(row))
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (r *entityTransformationRepository) Update(ctx context.Context, transformation domain.EntityTransformation) (domain.EntityTransformation, error) {
	nodesJSON, err := domain.EntityTransformationNodesToJSON(transformation.Nodes)
	if err != nil {
		return domain.EntityTransformation{}, fmt.Errorf("marshal nodes: %w", err)
	}
	name := pgtype.Text{Valid: false}
	if transformation.Name != "" {
		name = pgtype.Text{String: transformation.Name, Valid: true}
	}
	desc := pgtype.Text{Valid: false}
	if transformation.Description != "" {
		desc = pgtype.Text{String: transformation.Description, Valid: true}
	}
	row, err := r.queries.UpdateEntityTransformation(ctx, db.UpdateEntityTransformationParams{
		Name:        name,
		Description: desc,
		Nodes:       nodesJSON,
		ID:          transformation.ID,
	})
	if err != nil {
		return domain.EntityTransformation{}, fmt.Errorf("update entity transformation: %w", err)
	}
	return mapTransformationRow(convertEntityTransformationRow(row))
}

func (r *entityTransformationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteEntityTransformation(ctx, id); err != nil {
		return fmt.Errorf("delete entity transformation: %w", err)
	}
	return nil
}

type transformationRow struct {
	id             uuid.UUID
	organizationID uuid.UUID
	name           string
	description    pgtype.Text
	nodes          []byte
	createdAt      time.Time
	updatedAt      time.Time
}

func convertEntityTransformationRow(row db.EntityTransformation) transformationRow {
	return transformationRow{
		id:             row.ID,
		organizationID: row.OrganizationID,
		name:           row.Name,
		description:    row.Description,
		nodes:          row.Nodes,
		createdAt:      row.CreatedAt,
		updatedAt:      row.UpdatedAt,
	}
}

func mapTransformationRow(row transformationRow) (domain.EntityTransformation, error) {
	nodes, err := domain.EntityTransformationNodesFromJSON(row.nodes)
	if err != nil {
		return domain.EntityTransformation{}, fmt.Errorf("unmarshal nodes: %w", err)
	}
	description := ""
	if row.description.Valid {
		description = row.description.String
	}
	return domain.EntityTransformation{
		ID:             row.id,
		OrganizationID: row.organizationID,
		Name:           row.name,
		Description:    description,
		Nodes:          nodes,
		CreatedAt:      row.createdAt,
		UpdatedAt:      row.updatedAt,
	}, nil
}

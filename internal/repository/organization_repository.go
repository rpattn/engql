package repository

import (
	"context"
	"fmt"

	"github.com/rpattn/engql/internal/db"
	"github.com/rpattn/engql/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// organizationRepository implements OrganizationRepository interface
type organizationRepository struct {
	queries *db.Queries
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(queries *db.Queries) OrganizationRepository {
	return &organizationRepository{
		queries: queries,
	}
}

// Create creates a new organization
func (r *organizationRepository) Create(ctx context.Context, org domain.Organization) (domain.Organization, error) {
	row, err := r.queries.CreateOrganization(ctx, db.CreateOrganizationParams{
		Name:        org.Name,
		Description: pgtype.Text{String: org.Description, Valid: true},
	})
	if err != nil {
		return domain.Organization{}, fmt.Errorf("failed to create organization: %w", err)
	}

	return domain.Organization{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// GetByID retrieves an organization by ID
func (r *organizationRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	row, err := r.queries.GetOrganization(ctx, id)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("failed to get organization: %w", err)
	}

	return domain.Organization{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// GetByName retrieves an organization by name
func (r *organizationRepository) GetByName(ctx context.Context, name string) (domain.Organization, error) {
	row, err := r.queries.GetOrganizationByName(ctx, name)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("failed to get organization by name: %w", err)
	}

	return domain.Organization{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// List retrieves all organizations
func (r *organizationRepository) List(ctx context.Context) ([]domain.Organization, error) {
	rows, err := r.queries.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	organizations := make([]domain.Organization, len(rows))
	for i, row := range rows {
		organizations[i] = domain.Organization{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description.String,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}

	return organizations, nil
}

// Update updates an organization
func (r *organizationRepository) Update(ctx context.Context, org domain.Organization) (domain.Organization, error) {
	row, err := r.queries.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		ID:          org.ID,
		Name:        org.Name,
		Description: pgtype.Text{String: org.Description, Valid: true},
	})
	if err != nil {
		return domain.Organization{}, fmt.Errorf("failed to update organization: %w", err)
	}

	return domain.Organization{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// Delete deletes an organization
func (r *organizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteOrganization(ctx, id); err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	return nil
}

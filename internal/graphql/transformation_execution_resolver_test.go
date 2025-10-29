package graphql

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"
	"github.com/rpattn/engql/internal/transformations"
)

type trackingTransformationRepository struct {
	transformation domain.EntityTransformation
}

var _ repository.EntityTransformationRepository = (*trackingTransformationRepository)(nil)

func (t *trackingTransformationRepository) Create(ctx context.Context, transformation domain.EntityTransformation) (domain.EntityTransformation, error) {
	return domain.EntityTransformation{}, fmt.Errorf("not implemented")
}

func (t *trackingTransformationRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.EntityTransformation, error) {
	if t.transformation.ID != uuid.Nil && t.transformation.ID != id {
		return domain.EntityTransformation{}, fmt.Errorf("unexpected transformation id: %s", id)
	}
	return t.transformation, nil
}

func (t *trackingTransformationRepository) ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]domain.EntityTransformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (t *trackingTransformationRepository) Update(ctx context.Context, transformation domain.EntityTransformation) (domain.EntityTransformation, error) {
	return domain.EntityTransformation{}, fmt.Errorf("not implemented")
}

func (t *trackingTransformationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

type trackingEntityRepo struct {
	records    []domain.Entity
	lastLimit  int
	lastOffset int
	calls      int
}

func (t *trackingEntityRepo) List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, sort *domain.EntitySort, limit int, offset int) ([]domain.Entity, int, error) {
	t.lastLimit = limit
	t.lastOffset = offset
	t.calls++
	return append([]domain.Entity(nil), t.records...), len(t.records), nil
}

type stubSchemaProvider struct{}

func (stubSchemaProvider) GetByName(ctx context.Context, organizationID uuid.UUID, entityType string) (domain.EntitySchema, error) {
	return domain.EntitySchema{}, nil
}

func TestTransformationExecutionSortsBeforePaginating(t *testing.T) {
	orgID := uuid.New()
	loadID := uuid.New()
	materializeID := uuid.New()

	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadID,
				Name: "load",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "User",
				},
			},
			{
				ID:     materializeID,
				Name:   "materialize",
				Type:   domain.TransformationNodeMaterialize,
				Inputs: []uuid.UUID{loadID},
				Materialize: &domain.EntityTransformationMaterializeConfig{
					Outputs: []domain.EntityTransformationMaterializeOutput{
						{
							Alias: "table",
							Fields: []domain.EntityTransformationMaterializeFieldMapping{
								{SourceAlias: "users", SourceField: "name", OutputField: "name"},
							},
						},
					},
				},
			},
		},
	}

	repo := &trackingTransformationRepository{transformation: transformation}

	entityRecords := []domain.Entity{
		{ID: uuid.New(), OrganizationID: orgID, EntityType: "User", Properties: map[string]any{"name": "Alice"}},
		{ID: uuid.New(), OrganizationID: orgID, EntityType: "User", Properties: map[string]any{"name": "Bob"}},
		{ID: uuid.New(), OrganizationID: orgID, EntityType: "User", Properties: map[string]any{"name": "Charlie"}},
	}
	entityRepo := &trackingEntityRepo{records: entityRecords}
	executor := transformations.NewExecutor(entityRepo, stubSchemaProvider{})

	resolver := &Resolver{
		entityTransformationRepo: repo,
		transformationExecutor:   executor,
	}

	limit := 1
	offset := 0
	direction := graph.SortDirectionDesc
	conn, err := resolver.TransformationExecution(
		context.Background(),
		transformation.ID.String(),
		nil,
		&graph.TransformationExecutionSortInput{Alias: "table", Field: "name", Direction: &direction},
		&graph.PaginationInput{Limit: &limit, Offset: &offset},
	)
	if err != nil {
		t.Fatalf("resolver error: %v", err)
	}

	if entityRepo.calls != 1 {
		t.Fatalf("expected single repo call, got %d", entityRepo.calls)
	}
	if entityRepo.lastOffset != 0 {
		t.Fatalf("expected repo offset 0, got %d", entityRepo.lastOffset)
	}
	if entityRepo.lastLimit < len(entityRecords) {
		t.Fatalf("expected repo limit to cover all records, got %d", entityRepo.lastLimit)
	}

	if conn == nil {
		t.Fatalf("expected non-nil connection result")
	}
	if len(conn.Rows) != 1 {
		t.Fatalf("expected 1 row from resolver, got %d", len(conn.Rows))
	}
	if len(conn.Rows[0].Values) == 0 {
		t.Fatalf("expected row values")
	}
	value := conn.Rows[0].Values[0].Value
	if value == nil || *value != "Charlie" {
		t.Fatalf("expected row value Charlie, got %v", value)
	}

	if conn.PageInfo == nil {
		t.Fatalf("expected page info")
	}
	if conn.PageInfo.TotalCount != len(entityRecords) {
		t.Fatalf("expected total count %d, got %d", len(entityRecords), conn.PageInfo.TotalCount)
	}
	if !conn.PageInfo.HasNextPage {
		t.Fatalf("expected next page to be available")
	}
}

func TestTransformationExecutionAppliesFiltersBeforePagination(t *testing.T) {
	orgID := uuid.New()
	loadID := uuid.New()
	materializeID := uuid.New()

	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadID,
				Name: "load",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "User",
				},
			},
			{
				ID:     materializeID,
				Name:   "materialize",
				Type:   domain.TransformationNodeMaterialize,
				Inputs: []uuid.UUID{loadID},
				Materialize: &domain.EntityTransformationMaterializeConfig{
					Outputs: []domain.EntityTransformationMaterializeOutput{
						{
							Alias: "table",
							Fields: []domain.EntityTransformationMaterializeFieldMapping{
								{SourceAlias: "users", SourceField: "name", OutputField: "name"},
							},
						},
					},
				},
			},
		},
	}

	repo := &trackingTransformationRepository{transformation: transformation}

	entityRecords := []domain.Entity{
		{ID: uuid.New(), OrganizationID: orgID, EntityType: "User", Properties: map[string]any{"name": "Alice"}},
		{ID: uuid.New(), OrganizationID: orgID, EntityType: "User", Properties: map[string]any{"name": "Bob"}},
		{ID: uuid.New(), OrganizationID: orgID, EntityType: "User", Properties: map[string]any{"name": "Charlie"}},
	}
	entityRepo := &trackingEntityRepo{records: entityRecords}
	executor := transformations.NewExecutor(entityRepo, stubSchemaProvider{})

	resolver := &Resolver{
		entityTransformationRepo: repo,
		transformationExecutor:   executor,
	}

	limit := 1
	offset := 0
	charlie := "Charlie"
	filters := []*graph.TransformationExecutionFilterInput{
		{
			Alias: "table",
			Field: "name",
			Value: &charlie,
		},
	}

	conn, err := resolver.TransformationExecution(
		context.Background(),
		transformation.ID.String(),
		filters,
		nil,
		&graph.PaginationInput{Limit: &limit, Offset: &offset},
	)
	if err != nil {
		t.Fatalf("resolver error: %v", err)
	}

	if entityRepo.lastLimit < len(entityRecords) {
		t.Fatalf("expected repository limit to cover all records, got %d", entityRepo.lastLimit)
	}

	if conn == nil {
		t.Fatalf("expected non-nil connection result")
	}
	if len(conn.Rows) != 1 {
		t.Fatalf("expected 1 row from resolver, got %d", len(conn.Rows))
	}
	value := conn.Rows[0].Values[0].Value
	if value == nil || *value != "Charlie" {
		t.Fatalf("expected row value Charlie, got %v", value)
	}

	if conn.PageInfo == nil {
		t.Fatalf("expected page info")
	}
	if conn.PageInfo.TotalCount != 1 {
		t.Fatalf("expected total count 1, got %d", conn.PageInfo.TotalCount)
	}
	if conn.PageInfo.HasNextPage {
		t.Fatalf("expected no next page")
	}
	if conn.PageInfo.HasPreviousPage {
		t.Fatalf("expected no previous page")
	}
}

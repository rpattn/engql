package transformations

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rpattn/engql/internal/domain"
)

type mockEntityRepository struct {
	entities []domain.Entity
}

func (m *mockEntityRepository) List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, sort *domain.EntitySort, limit int, offset int) ([]domain.Entity, int, error) {
	var result []domain.Entity
	for _, entity := range m.entities {
		if entity.OrganizationID != organizationID {
			continue
		}
		if filter != nil {
			if filter.EntityType != "" && entity.EntityType != filter.EntityType {
				continue
			}
			if len(filter.PropertyFilters) > 0 {
				matched := true
				for _, pf := range filter.PropertyFilters {
					value := entity.Properties[pf.Key]
					if pf.Value != "" && value != pf.Value {
						matched = false
						break
					}
					if pf.Exists != nil {
						if *pf.Exists && value == nil {
							matched = false
							break
						}
					}
				}
				if !matched {
					continue
				}
			}
		}
		result = append(result, entity)
	}
	return result, len(result), nil
}

func TestExecutor_LoadAndFilter(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"status": "active",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"status": "inactive",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadNodeID := uuid.New()
	filterNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "test",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadNodeID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
				},
			},
			{
				ID:     filterNodeID,
				Name:   "active-only",
				Type:   domain.TransformationNodeFilter,
				Inputs: []uuid.UUID{loadNodeID},
				Filter: &domain.EntityTransformationFilterConfig{
					Alias: "users",
					Filters: []domain.PropertyFilter{
						{Key: "status", Value: "active"},
					},
				},
			},
		},
	}
	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("expected total count 1, got %d", result.TotalCount)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	record := result.Records[0]
	entity := record.Entities["users"]
	if entity == nil {
		t.Fatalf("expected entity for alias users")
	}
	if entity.Properties["status"] != "active" {
		t.Fatalf("unexpected status %v", entity.Properties["status"])
	}
}

func TestExecutor_FilterFallbackAlias(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"status": "active",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadNodeID := uuid.New()
	filterNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "filter-fallback",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadNodeID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "source",
					EntityType: "user",
				},
			},
			{
				ID:     filterNodeID,
				Name:   "filter",
				Type:   domain.TransformationNodeFilter,
				Inputs: []uuid.UUID{loadNodeID},
				Filter: &domain.EntityTransformationFilterConfig{
					Alias: "filtered",
					Filters: []domain.PropertyFilter{
						{Key: "status", Value: "active"},
					},
				},
			},
		},
	}
	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	record := result.Records[0]
	entity := record.Entities["source"]
	if entity == nil {
		t.Fatalf("expected entity for fallback alias")
	}
	if entity.Properties["status"] != "active" {
		t.Fatalf("unexpected status %v", entity.Properties["status"])
	}
}

func TestExecutor_FilterAmbiguousAliasError(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"id":     "1",
					"status": "active",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "order",
				Properties: map[string]any{
					"id": "1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadUsersID := uuid.New()
	loadOrdersID := uuid.New()
	joinNodeID := uuid.New()
	filterNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "filter-ambiguous",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadUsersID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
				},
			},
			{
				ID:   loadOrdersID,
				Name: "load-orders",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "orders",
					EntityType: "order",
				},
			},
			{
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadUsersID, loadOrdersID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "users",
					RightAlias: "orders",
					OnField:    "id",
				},
			},
			{
				ID:     filterNodeID,
				Name:   "filter",
				Type:   domain.TransformationNodeFilter,
				Inputs: []uuid.UUID{joinNodeID},
				Filter: &domain.EntityTransformationFilterConfig{
					Alias:   "",
					Filters: []domain.PropertyFilter{{Key: "status", Value: "active"}},
				},
			},
		},
	}
	_, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err == nil {
		t.Fatalf("expected error when alias is ambiguous")
	}
}

func TestExecutor_FilterAliasNotFoundError(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"id":     "1",
					"status": "active",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "order",
				Properties: map[string]any{
					"id": "1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadUsersID := uuid.New()
	loadOrdersID := uuid.New()
	joinNodeID := uuid.New()
	filterNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "filter-missing-alias",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadUsersID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
				},
			},
			{
				ID:   loadOrdersID,
				Name: "load-orders",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "orders",
					EntityType: "order",
				},
			},
			{
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadUsersID, loadOrdersID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "users",
					RightAlias: "orders",
					OnField:    "id",
				},
			},
			{
				ID:     filterNodeID,
				Name:   "filter",
				Type:   domain.TransformationNodeFilter,
				Inputs: []uuid.UUID{joinNodeID},
				Filter: &domain.EntityTransformationFilterConfig{
					Alias:   "missing",
					Filters: []domain.PropertyFilter{{Key: "status", Value: "active"}},
				},
			},
		},
	}
	_, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err == nil {
		t.Fatalf("expected error when alias cannot be resolved")
	}
}

func TestExecutor_Project(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"id":    "1",
					"name":  "Alice",
					"email": "alice@example.com",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadNodeID := uuid.New()
	projectNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "project-test",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadNodeID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
				},
			},
			{
				ID:     projectNodeID,
				Name:   "project",
				Type:   domain.TransformationNodeProject,
				Inputs: []uuid.UUID{loadNodeID},
				Project: &domain.EntityTransformationProjectConfig{
					Alias:  "users",
					Fields: []string{"id", "email"},
				},
			},
		},
	}
	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("expected total count 1, got %d", result.TotalCount)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	record := result.Records[0]
	entity := record.Entities["users"]
	if entity == nil {
		t.Fatalf("expected entity for alias users")
	}
	if _, ok := entity.Properties["name"]; ok {
		t.Fatalf("expected name to be projected out, got %v", entity.Properties)
	}
	if entity.Properties["email"] != "alice@example.com" {
		t.Fatalf("unexpected email %v", entity.Properties["email"])
	}
}

func TestExecutor_ProjectFallbackAlias(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"id":   "1",
					"name": "Alice",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadNodeID := uuid.New()
	projectNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "project-fallback",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadNodeID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "source",
					EntityType: "user",
				},
			},
			{
				ID:     projectNodeID,
				Name:   "project",
				Type:   domain.TransformationNodeProject,
				Inputs: []uuid.UUID{loadNodeID},
				Project: &domain.EntityTransformationProjectConfig{
					Alias:  "projection",
					Fields: []string{"id"},
				},
			},
		},
	}
	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	record := result.Records[0]
	if _, ok := record.Entities["source"]; ok {
		t.Fatalf("expected source alias to be replaced, got %v", record.Entities)
	}
	entity := record.Entities["projection"]
	if entity == nil {
		t.Fatalf("expected projection alias to exist")
	}
	if entity.Properties["id"] != "1" {
		t.Fatalf("unexpected projected value %v", entity.Properties["id"])
	}
	if _, ok := entity.Properties["name"]; ok {
		t.Fatalf("expected name to be removed, got %v", entity.Properties)
	}
}

func TestExecutor_ProjectAliasMissingError(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"id": "1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "order",
				Properties: map[string]any{
					"id": "1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadUsersID := uuid.New()
	loadOrdersID := uuid.New()
	joinNodeID := uuid.New()
	projectNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "project-alias-missing",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadUsersID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
				},
			},
			{
				ID:   loadOrdersID,
				Name: "load-orders",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "orders",
					EntityType: "order",
				},
			},
			{
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadUsersID, loadOrdersID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "users",
					RightAlias: "orders",
					OnField:    "id",
				},
			},
			{
				ID:     projectNodeID,
				Name:   "project",
				Type:   domain.TransformationNodeProject,
				Inputs: []uuid.UUID{joinNodeID},
				Project: &domain.EntityTransformationProjectConfig{
					Alias:  "missing",
					Fields: []string{"id"},
				},
			},
		},
	}
	_, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err == nil {
		t.Fatalf("expected error when alias missing")
	}
}

func TestExecutor_Sort(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"name": "Charlie",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"name": "Bob",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadNodeID := uuid.New()
	sortNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "sort-test",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadNodeID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
				},
			},
			{
				ID:     sortNodeID,
				Name:   "sort",
				Type:   domain.TransformationNodeSort,
				Inputs: []uuid.UUID{loadNodeID},
				Sort: &domain.EntityTransformationSortConfig{
					Alias:     "users",
					Field:     "name",
					Direction: domain.JoinSortAsc,
				},
			},
		},
	}
	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 2 {
		t.Fatalf("expected total count 2, got %d", result.TotalCount)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result.Records))
	}
	if result.Records[0].Entities["users"].Properties["name"] != "Bob" {
		t.Fatalf("expected Bob first, got %v", result.Records[0].Entities["users"].Properties["name"])
	}
	if result.Records[1].Entities["users"].Properties["name"] != "Charlie" {
		t.Fatalf("expected Charlie second, got %v", result.Records[1].Entities["users"].Properties["name"])
	}
}

func TestExecutor_SortFallbackAlias(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"name": "Charlie",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"name": "Bob",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadNodeID := uuid.New()
	sortNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "sort-fallback",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadNodeID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "source",
					EntityType: "user",
				},
			},
			{
				ID:     sortNodeID,
				Name:   "sort",
				Type:   domain.TransformationNodeSort,
				Inputs: []uuid.UUID{loadNodeID},
				Sort: &domain.EntityTransformationSortConfig{
					Alias:     "sorted",
					Field:     "name",
					Direction: domain.JoinSortAsc,
				},
			},
		},
	}
	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result.Records))
	}
	first := result.Records[0].Entities["source"]
	if first == nil || first.Properties["name"] != "Bob" {
		t.Fatalf("expected Bob first, got %v", first)
	}
	second := result.Records[1].Entities["source"]
	if second == nil || second.Properties["name"] != "Charlie" {
		t.Fatalf("expected Charlie second, got %v", second)
	}
}

func TestExecutor_SortAliasMissingError(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"id": "1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "order",
				Properties: map[string]any{
					"id": "1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo)
	loadUsersID := uuid.New()
	loadOrdersID := uuid.New()
	joinNodeID := uuid.New()
	sortNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "sort-alias-missing",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadUsersID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
				},
			},
			{
				ID:   loadOrdersID,
				Name: "load-orders",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "orders",
					EntityType: "order",
				},
			},
			{
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadUsersID, loadOrdersID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "users",
					RightAlias: "orders",
					OnField:    "id",
				},
			},
			{
				ID:     sortNodeID,
				Name:   "sort",
				Type:   domain.TransformationNodeSort,
				Inputs: []uuid.UUID{joinNodeID},
				Sort: &domain.EntityTransformationSortConfig{
					Alias:     "missing",
					Field:     "id",
					Direction: domain.JoinSortAsc,
				},
			},
		},
	}
	_, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err == nil {
		t.Fatalf("expected error when sort alias missing")
	}
}

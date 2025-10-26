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

package transformations

import (
	"context"
	"fmt"
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
					value, ok := entity.Properties[pf.Key]
					if pf.Exists != nil {
						if *pf.Exists {
							if !ok {
								matched = false
								break
							}
						} else {
							if ok {
								if pf.Value == "" && len(pf.InArray) == 0 {
									if str, okStr := value.(string); okStr {
										if str != "" {
											matched = false
											break
										}
									} else if value != nil {
										matched = false
										break
									}
								} else {
									matched = false
									break
								}
							}
						}
					}
					if pf.Value != "" {
						if !ok {
							matched = false
							break
						}
						if fmt.Sprintf("%v", value) != pf.Value {
							matched = false
							break
						}
					}
					if len(pf.InArray) > 0 {
						if !ok {
							matched = false
							break
						}
						valueStr := fmt.Sprintf("%v", value)
						found := false
						for _, candidate := range pf.InArray {
							if valueStr == candidate {
								found = true
								break
							}
						}
						if !found {
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

type mockSchemaProvider struct {
	schemas map[string]domain.EntitySchema
}

func (m *mockSchemaProvider) GetByName(ctx context.Context, organizationID uuid.UUID, entityType string) (domain.EntitySchema, error) {
	if m == nil {
		return domain.EntitySchema{}, fmt.Errorf("schema provider not configured")
	}
	if schema, ok := m.schemas[entityType]; ok {
		return schema, nil
	}
	return domain.EntitySchema{}, fmt.Errorf("schema %s not found", entityType)
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
	executor := NewExecutor(repo, nil)
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
	executor := NewExecutor(repo, nil)
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

func TestExecutor_Materialize(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             userID,
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"firstName": "Alice",
					"status":    "active",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo, nil)
	loadNodeID := uuid.New()
	materializeNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "materialize",
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
				ID:     materializeNodeID,
				Name:   "materialize-users",
				Type:   domain.TransformationNodeMaterialize,
				Inputs: []uuid.UUID{loadNodeID},
				Materialize: &domain.EntityTransformationMaterializeConfig{
					Outputs: []domain.EntityTransformationMaterializeOutput{
						{
							Alias: "materialized",
							Fields: []domain.EntityTransformationMaterializeFieldMapping{
								{SourceAlias: "users", SourceField: "firstName", OutputField: "firstName"},
								{SourceAlias: "users", SourceField: "status", OutputField: "status"},
								{SourceAlias: "users", SourceField: "id", OutputField: "id"},
							},
						},
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
	if len(record.Entities) != 1 {
		t.Fatalf("expected exactly one alias after materialize, got %d", len(record.Entities))
	}
	entity := record.Entities["materialized"]
	if entity == nil {
		t.Fatalf("expected entity for alias materialized")
	}
	if entity.ID != userID {
		t.Fatalf("expected materialized entity to retain ID %s, got %s", userID, entity.ID)
	}
	if entity.Properties["firstName"] != "Alice" {
		t.Fatalf("unexpected firstName %v", entity.Properties["firstName"])
	}
	if entity.Properties["status"] != "active" {
		t.Fatalf("unexpected status %v", entity.Properties["status"])
	}
	idProp, ok := entity.Properties["id"].(string)
	if !ok {
		t.Fatalf("expected id property to be string, got %T", entity.Properties["id"])
	}
	if idProp != userID.String() {
		t.Fatalf("expected id property %s, got %s", userID, idProp)
	}
}

func TestExecutor_MaterializeUnionAnyAlias(t *testing.T) {
	orgID := uuid.New()
	firstID := uuid.New()
	secondID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             firstID,
				OrganizationID: orgID,
				EntityType:     "first",
				Properties: map[string]any{
					"name": "First",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             secondID,
				OrganizationID: orgID,
				EntityType:     "second",
				Properties: map[string]any{
					"name": "Second",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo, nil)

	loadFirstID := uuid.New()
	loadSecondID := uuid.New()
	unionNodeID := uuid.New()
	materializeNodeID := uuid.New()

	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "union-materialize-any-alias",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadFirstID,
				Name: "load-first",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "first",
					EntityType: "first",
				},
			},
			{
				ID:   loadSecondID,
				Name: "load-second",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "second",
					EntityType: "second",
				},
			},
			{
				ID:     unionNodeID,
				Name:   "union",
				Type:   domain.TransformationNodeUnion,
				Inputs: []uuid.UUID{loadFirstID, loadSecondID},
			},
			{
				ID:     materializeNodeID,
				Name:   "materialize",
				Type:   domain.TransformationNodeMaterialize,
				Inputs: []uuid.UUID{unionNodeID},
				Materialize: &domain.EntityTransformationMaterializeConfig{
					Outputs: []domain.EntityTransformationMaterializeOutput{
						{
							Alias: "result",
							Fields: []domain.EntityTransformationMaterializeFieldMapping{
								{SourceAlias: anyAliasSentinel, SourceField: "id", OutputField: "id"},
								{SourceAlias: anyAliasSentinel, SourceField: "name", OutputField: "name"},
							},
						},
					},
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

	seen := make(map[string]bool)
	for _, record := range result.Records {
		entity := record.Entities["result"]
		if entity == nil {
			t.Fatalf("expected materialized entity for alias result")
		}
		idProp, ok := entity.Properties["id"].(string)
		if !ok {
			t.Fatalf("expected id property to be string, got %T", entity.Properties["id"])
		}
		switch idProp {
		case firstID.String():
			seen["first"] = true
			if entity.Properties["name"] != "First" {
				t.Fatalf("unexpected name %v for first entity", entity.Properties["name"])
			}
			if entity.ID != firstID {
				t.Fatalf("expected entity ID %s, got %s", firstID, entity.ID)
			}
		case secondID.String():
			seen["second"] = true
			if entity.Properties["name"] != "Second" {
				t.Fatalf("unexpected name %v for second entity", entity.Properties["name"])
			}
			if entity.ID != secondID {
				t.Fatalf("expected entity ID %s, got %s", secondID, entity.ID)
			}
		default:
			t.Fatalf("unexpected id property %s", idProp)
		}
	}

	if !seen["first"] || !seen["second"] {
		t.Fatalf("expected materialized records for both union inputs")
	}
}

func TestExecutor_MaterializeThenFilter(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"firstName": "Alice",
					"status":    "active",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"firstName": "Bob",
					"status":    "inactive",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	executor := NewExecutor(repo, nil)
	loadNodeID := uuid.New()
	materializeNodeID := uuid.New()
	filterNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "materialize-filter",
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
				ID:     materializeNodeID,
				Name:   "materialize-users",
				Type:   domain.TransformationNodeMaterialize,
				Inputs: []uuid.UUID{loadNodeID},
				Materialize: &domain.EntityTransformationMaterializeConfig{
					Outputs: []domain.EntityTransformationMaterializeOutput{
						{
							Alias: "flattened",
							Fields: []domain.EntityTransformationMaterializeFieldMapping{
								{SourceAlias: "users", SourceField: "status", OutputField: "status"},
							},
						},
					},
				},
			},
			{
				ID:     filterNodeID,
				Name:   "filter-active",
				Type:   domain.TransformationNodeFilter,
				Inputs: []uuid.UUID{materializeNodeID},
				Filter: &domain.EntityTransformationFilterConfig{
					Filters: []domain.PropertyFilter{{Key: "status", Value: "active"}},
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("expected 1 result, got %d", result.TotalCount)
	}
	record := result.Records[0]
	if _, ok := record.Entities["flattened"]; !ok {
		t.Fatalf("expected flattened alias after materialize")
	}
	if record.Entities["flattened"].Properties["status"] != "active" {
		t.Fatalf("unexpected status %v", record.Entities["flattened"].Properties["status"])
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
	executor := NewExecutor(repo, nil)
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

func TestExecutor_LoadFilterExistsFalse(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"status": "",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties:     map[string]any{},
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
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
	executor := NewExecutor(repo, nil)
	loadNodeID := uuid.New()
	existsFalse := false
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "load-exists-false",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadNodeID,
				Name: "load-users",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "users",
					EntityType: "user",
					Filters: []domain.PropertyFilter{
						{Key: "status", Exists: &existsFalse},
					},
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
	for _, record := range result.Records {
		entity := record.Entities["users"]
		if entity == nil {
			t.Fatalf("expected entity for alias users")
		}
		status, ok := entity.Properties["status"]
		if ok {
			if str, _ := status.(string); str != "" {
				t.Fatalf("expected empty status, got %v", status)
			}
		}
	}
}

func TestExecutor_FilterExistsFalse(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"status": "",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "user",
				Properties:     map[string]any{},
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
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
	executor := NewExecutor(repo, nil)
	loadNodeID := uuid.New()
	filterNodeID := uuid.New()
	existsFalse := false
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "filter-exists-false",
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
				Name:   "filter-users",
				Type:   domain.TransformationNodeFilter,
				Inputs: []uuid.UUID{loadNodeID},
				Filter: &domain.EntityTransformationFilterConfig{
					Alias: "users",
					Filters: []domain.PropertyFilter{
						{Key: "status", Exists: &existsFalse},
					},
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
	for _, record := range result.Records {
		entity := record.Entities["users"]
		if entity == nil {
			t.Fatalf("expected entity for alias users")
		}
		status, ok := entity.Properties["status"]
		if ok {
			if str, _ := status.(string); str != "" {
				t.Fatalf("expected empty status, got %v", status)
			}
		}
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
	executor := NewExecutor(repo, nil)
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
	executor := NewExecutor(repo, nil)
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
	executor := NewExecutor(repo, nil)
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
	executor := NewExecutor(repo, nil)
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
	executor := NewExecutor(repo, nil)
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
	executor := NewExecutor(repo, nil)
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
	executor := NewExecutor(repo, nil)
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

func TestExecutor_JoinRespectsExecutionWindow(t *testing.T) {
	orgID := uuid.New()
	leftCount := 30
	rightCount := 25
	repo := &mockEntityRepository{}
	for i := 0; i < leftCount; i++ {
		repo.entities = append(repo.entities, domain.Entity{
			ID:             uuid.New(),
			OrganizationID: orgID,
			EntityType:     "left",
			Properties: map[string]any{
				"join": "shared",
				"idx":  i,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}
	for j := 0; j < rightCount; j++ {
		repo.entities = append(repo.entities, domain.Entity{
			ID:             uuid.New(),
			OrganizationID: orgID,
			EntityType:     "right",
			Properties: map[string]any{
				"join": "shared",
				"idx":  j,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}
	executor := NewExecutor(repo, nil)

	loadLeftID := uuid.New()
	loadRightID := uuid.New()
	joinNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "paged-join",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadLeftID,
				Name: "load-left",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "left",
					EntityType: "left",
				},
			},
			{
				ID:   loadRightID,
				Name: "load-right",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "right",
					EntityType: "right",
				},
			},
			{
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadLeftID, loadRightID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "left",
					RightAlias: "right",
					OnField:    "join",
				},
			},
		},
	}

	opts := domain.EntityTransformationExecutionOptions{Limit: 10, Offset: 50}
	totalCombos := leftCount * rightCount
	if totalCombos <= opts.Offset+opts.Limit {
		t.Fatalf("expected more combinations than requested page window")
	}

	result, err := executor.Execute(context.Background(), transformation, opts)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.Records) != opts.Limit {
		t.Fatalf("expected %d records, got %d", opts.Limit, len(result.Records))
	}
	if result.TotalCount != opts.Limit {
		t.Fatalf("expected total count %d, got %d", opts.Limit, result.TotalCount)
	}

	expectedFirstLeft := opts.Offset / rightCount
	expectedFirstRight := opts.Offset % rightCount
	first := result.Records[0]
	leftEntity := first.Entities["left"]
	rightEntity := first.Entities["right"]
	if leftEntity == nil || rightEntity == nil {
		t.Fatalf("expected joined entities in first record")
	}
	if got, ok := leftEntity.Properties["idx"].(int); !ok || got != expectedFirstLeft {
		t.Fatalf("expected first left idx %d, got %v", expectedFirstLeft, leftEntity.Properties["idx"])
	}
	if got, ok := rightEntity.Properties["idx"].(int); !ok || got != expectedFirstRight {
		t.Fatalf("expected first right idx %d, got %v", expectedFirstRight, rightEntity.Properties["idx"])
	}

	lastIndex := opts.Offset + opts.Limit - 1
	expectedLastLeft := lastIndex / rightCount
	expectedLastRight := lastIndex % rightCount
	last := result.Records[len(result.Records)-1]
	leftEntity = last.Entities["left"]
	rightEntity = last.Entities["right"]
	if leftEntity == nil || rightEntity == nil {
		t.Fatalf("expected joined entities in last record")
	}
	if got, ok := leftEntity.Properties["idx"].(int); !ok || got != expectedLastLeft {
		t.Fatalf("expected last left idx %d, got %v", expectedLastLeft, leftEntity.Properties["idx"])
	}
	if got, ok := rightEntity.Properties["idx"].(int); !ok || got != expectedLastRight {
		t.Fatalf("expected last right idx %d, got %v", expectedLastRight, rightEntity.Properties["idx"])
	}
}

func TestExecutor_JoinEntityReferenceMatchesIDs(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "device",
				Properties: map[string]any{
					"owner": userID.String(),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             userID,
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"username": "primary",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	schemaProvider := &mockSchemaProvider{
		schemas: map[string]domain.EntitySchema{
			"device": {
				OrganizationID: orgID,
				Name:           "device",
				Fields: []domain.FieldDefinition{
					{
						Name:                "owner",
						Type:                domain.FieldTypeEntityReference,
						ReferenceEntityType: "user",
					},
				},
			},
		},
	}
	executor := NewExecutor(repo, schemaProvider)

	loadDevicesID := uuid.New()
	loadUsersID := uuid.New()
	joinNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "join-entity-reference",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadDevicesID,
				Name: "load-devices",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "devices",
					EntityType: "device",
				},
			},
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
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadDevicesID, loadUsersID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "devices",
					RightAlias: "users",
					OnField:    "owner",
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("expected 1 record, got %d", result.TotalCount)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	record := result.Records[0]
	right := record.Entities["users"]
	if right == nil {
		t.Fatalf("expected right entity to be joined")
	}
	if right.ID != userID {
		t.Fatalf("expected user %s, got %s", userID, right.ID)
	}
}

func TestExecutor_JoinEntityReferenceArrayFanout(t *testing.T) {
	orgID := uuid.New()
	firstUser := uuid.New()
	secondUser := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "group",
				Properties: map[string]any{
					"members": []string{firstUser.String(), secondUser.String(), firstUser.String()},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             firstUser,
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"username": "first",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             secondUser,
				OrganizationID: orgID,
				EntityType:     "user",
				Properties: map[string]any{
					"username": "second",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	schemaProvider := &mockSchemaProvider{
		schemas: map[string]domain.EntitySchema{
			"group": {
				OrganizationID: orgID,
				Name:           "group",
				Fields: []domain.FieldDefinition{
					{
						Name:                "members",
						Type:                domain.FieldTypeEntityReferenceArray,
						ReferenceEntityType: "user",
					},
				},
			},
		},
	}
	executor := NewExecutor(repo, schemaProvider)

	loadGroupsID := uuid.New()
	loadUsersID := uuid.New()
	joinNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "join-entity-reference-array",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadGroupsID,
				Name: "load-groups",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "groups",
					EntityType: "group",
				},
			},
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
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadGroupsID, loadUsersID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "groups",
					RightAlias: "users",
					OnField:    "members",
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 2 {
		t.Fatalf("expected 2 records, got %d", result.TotalCount)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result.Records))
	}
	seen := make(map[uuid.UUID]struct{})
	for _, record := range result.Records {
		right := record.Entities["users"]
		if right == nil {
			t.Fatalf("expected joined user entity")
		}
		seen[right.ID] = struct{}{}
	}
	if len(seen) != 2 {
		t.Fatalf("expected two distinct user matches, got %d", len(seen))
	}
	if _, ok := seen[firstUser]; !ok {
		t.Fatalf("expected matches to include first user")
	}
	if _, ok := seen[secondUser]; !ok {
		t.Fatalf("expected matches to include second user")
	}
}

func TestExecutor_JoinReferenceMatchesCanonicalValue(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "ticket",
				Properties: map[string]any{
					"accountRef": "acct-001",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "account",
				Properties: map[string]any{
					"slug": "acct-001",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "account",
				Properties: map[string]any{
					"slug": "acct-002",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	schemaProvider := &mockSchemaProvider{
		schemas: map[string]domain.EntitySchema{
			"ticket": {
				OrganizationID: orgID,
				Name:           "ticket",
				Fields: []domain.FieldDefinition{
					{
						Name:                "accountRef",
						Type:                domain.FieldTypeReference,
						ReferenceEntityType: "account",
					},
				},
			},
			"account": {
				OrganizationID: orgID,
				Name:           "account",
				Fields: []domain.FieldDefinition{
					{Name: "slug", Type: domain.FieldTypeReference},
					{Name: "alternate", Type: domain.FieldTypeReference},
				},
			},
		},
	}
	executor := NewExecutor(repo, schemaProvider)

	loadTicketsID := uuid.New()
	loadAccountsID := uuid.New()
	joinNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "join-reference-canonical",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadTicketsID,
				Name: "load-tickets",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "tickets",
					EntityType: "ticket",
				},
			},
			{
				ID:   loadAccountsID,
				Name: "load-accounts",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "accounts",
					EntityType: "account",
				},
			},
			{
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadTicketsID, loadAccountsID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "tickets",
					RightAlias: "accounts",
					OnField:    "accountRef",
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("expected single joined record, got %d", result.TotalCount)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	record := result.Records[0]
	right := record.Entities["accounts"]
	if right == nil {
		t.Fatalf("expected account entity to be joined")
	}
	if slug, _ := right.Properties["slug"].(string); slug != "acct-001" {
		t.Fatalf("expected canonical slug acct-001, got %v", right.Properties["slug"])
	}
}

func TestExecutor_JoinReferenceRespectsReferenceEntityType(t *testing.T) {
	orgID := uuid.New()
	repo := &mockEntityRepository{
		entities: []domain.Entity{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "ticket",
				Properties: map[string]any{
					"accountRef": "acct-001",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "account",
				Properties: map[string]any{
					"slug": "acct-001",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				EntityType:     "contact",
				Properties: map[string]any{
					"slug": "acct-001",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	schemaProvider := &mockSchemaProvider{
		schemas: map[string]domain.EntitySchema{
			"ticket": {
				OrganizationID: orgID,
				Name:           "ticket",
				Fields: []domain.FieldDefinition{
					{
						Name:                "accountRef",
						Type:                domain.FieldTypeReference,
						ReferenceEntityType: "account",
					},
				},
			},
			"account": {
				OrganizationID: orgID,
				Name:           "account",
				Fields: []domain.FieldDefinition{
					{Name: "slug", Type: domain.FieldTypeReference},
				},
			},
			"contact": {
				OrganizationID: orgID,
				Name:           "contact",
				Fields: []domain.FieldDefinition{
					{Name: "slug", Type: domain.FieldTypeReference},
				},
			},
		},
	}
	executor := NewExecutor(repo, schemaProvider)

	loadTicketsID := uuid.New()
	loadAccountsID := uuid.New()
	loadContactsID := uuid.New()
	projectContactsID := uuid.New()
	unionID := uuid.New()
	joinNodeID := uuid.New()
	transformation := domain.EntityTransformation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "join-reference-entity-type",
		Nodes: []domain.EntityTransformationNode{
			{
				ID:   loadTicketsID,
				Name: "load-tickets",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "tickets",
					EntityType: "ticket",
				},
			},
			{
				ID:   loadAccountsID,
				Name: "load-accounts",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "accounts",
					EntityType: "account",
				},
			},
			{
				ID:   loadContactsID,
				Name: "load-contacts",
				Type: domain.TransformationNodeLoad,
				Load: &domain.EntityTransformationLoadConfig{
					Alias:      "contacts",
					EntityType: "contact",
				},
			},
			{
				ID:     projectContactsID,
				Name:   "rename-contacts",
				Type:   domain.TransformationNodeProject,
				Inputs: []uuid.UUID{loadContactsID},
				Project: &domain.EntityTransformationProjectConfig{
					Alias: "accounts",
				},
			},
			{
				ID:     unionID,
				Name:   "union-refs",
				Type:   domain.TransformationNodeUnion,
				Inputs: []uuid.UUID{loadAccountsID, projectContactsID},
			},
			{
				ID:     joinNodeID,
				Name:   "join",
				Type:   domain.TransformationNodeJoin,
				Inputs: []uuid.UUID{loadTicketsID, unionID},
				Join: &domain.EntityTransformationJoinConfig{
					LeftAlias:  "tickets",
					RightAlias: "accounts",
					OnField:    "accountRef",
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), transformation, domain.EntityTransformationExecutionOptions{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalCount != 1 {
		t.Fatalf("expected single joined record, got %d", result.TotalCount)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	record := result.Records[0]
	right := record.Entities["accounts"]
	if right == nil {
		t.Fatalf("expected account entity to be joined")
	}
	if right.EntityType != "account" {
		t.Fatalf("expected joined entity type account, got %s", right.EntityType)
	}
}

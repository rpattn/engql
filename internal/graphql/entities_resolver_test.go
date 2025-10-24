package graphql

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/middleware"
	"github.com/rpattn/engql/internal/repository"
)

func TestEntitiesReturnsResultsWithoutReferenceField(t *testing.T) {
	orgID := uuid.New()
	parentSchemaID := uuid.New()
	childSchemaID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	schemaRepo := &stubSchemaRepoForEntities{
		schemas: map[string]domain.EntitySchema{
			schemaKeyForEntities(orgID, "Parent"): {
				ID:             parentSchemaID,
				OrganizationID: orgID,
				Name:           "Parent",
				Fields: []domain.FieldDefinition{
					{
						Name:                "linked_ids",
						Type:                domain.FieldTypeEntityReferenceArray,
						ReferenceEntityType: "Child",
					},
				},
			},
			schemaKeyForEntities(orgID, "Child"): {
				ID:             childSchemaID,
				OrganizationID: orgID,
				Name:           "Child",
				Fields:         []domain.FieldDefinition{},
			},
		},
	}

	now := time.Now()
	parentEntity := domain.Entity{
		ID:             parentID,
		OrganizationID: orgID,
		SchemaID:       parentSchemaID,
		EntityType:     "Parent",
		Properties: map[string]any{
			"linked_ids": []any{childID.String()},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	childEntity := domain.Entity{
		ID:             childID,
		OrganizationID: orgID,
		SchemaID:       childSchemaID,
		EntityType:     "Child",
		Properties: map[string]any{
			"name": "child",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	entityRepo := &stubEntityRepoForEntities{
		list: []domain.Entity{parentEntity},
		entities: map[uuid.UUID]domain.Entity{
			parentID: parentEntity,
			childID:  childEntity,
		},
	}

	resolver := &Resolver{
		entityRepo:       entityRepo,
		entitySchemaRepo: schemaRepo,
	}

	loaderMiddleware := middleware.DataLoaderMiddleware(entityRepo)
	ctxCh := make(chan context.Context, 1)
	handler := loaderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxCh <- r.Context()
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var ctx context.Context
	select {
	case ctx = <-ctxCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for dataloader context")
	}

	entityType := "Parent"
	conn, err := resolver.Entities(ctx, orgID.String(), &graph.EntityFilter{EntityType: &entityType}, nil)
	if err != nil {
		t.Fatalf("expected entities query to succeed, got error: %v", err)
	}

	if conn == nil || conn.Entities == nil {
		t.Fatalf("expected entity connection with results, got %#v", conn)
	}

	if len(conn.Entities) != 1 {
		t.Fatalf("expected one entity, got %d", len(conn.Entities))
	}

	linked, err := resolver.LinkedEntities(ctx, conn.Entities[0])
	if err != nil {
		t.Fatalf("expected linked entities resolver to succeed, got error: %v", err)
	}

	if len(linked) != 1 {
		t.Fatalf("expected one linked entity, got %d", len(linked))
	}

	if linked[0].ID != childID.String() {
		t.Fatalf("expected linked entity %s, got %s", childID, linked[0].ID)
	}
}

type stubSchemaRepoForEntities struct {
	schemas map[string]domain.EntitySchema
}

func (s *stubSchemaRepoForEntities) Create(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubSchemaRepoForEntities) GetByID(ctx context.Context, id uuid.UUID) (domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubSchemaRepoForEntities) GetByName(ctx context.Context, organizationID uuid.UUID, name string) (domain.EntitySchema, error) {
	if schema, ok := s.schemas[schemaKeyForEntities(organizationID, name)]; ok {
		return schema, nil
	}
	return domain.EntitySchema{}, fmt.Errorf("schema %s not found", name)
}

func (s *stubSchemaRepoForEntities) List(ctx context.Context, organizationID uuid.UUID) ([]domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubSchemaRepoForEntities) ListVersions(ctx context.Context, organizationID uuid.UUID, name string) ([]domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubSchemaRepoForEntities) CreateVersion(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubSchemaRepoForEntities) Exists(ctx context.Context, organizationID uuid.UUID, name string) (bool, error) {
	panic("not implemented")
}

func (s *stubSchemaRepoForEntities) ArchiveSchema(ctx context.Context, schemaID uuid.UUID) error {
	panic("not implemented")
}

type stubEntityRepoForEntities struct {
	list     []domain.Entity
	entities map[uuid.UUID]domain.Entity
}

func (s *stubEntityRepoForEntities) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) CreateBatch(ctx context.Context, items []repository.EntityBatchItem, opts repository.EntityBatchOptions) (repository.EntityBatchResult, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error) {
	if entity, ok := s.entities[id]; ok {
		return entity, nil
	}
	return domain.Entity{}, fmt.Errorf("entity %s not found", id)
}

func (s *stubEntityRepoForEntities) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Entity, error) {
	result := make([]domain.Entity, 0, len(ids))
	for _, id := range ids {
		if entity, ok := s.entities[id]; ok {
			result = append(result, entity)
		}
	}
	return result, nil
}

func (s *stubEntityRepoForEntities) GetHistoryByVersion(ctx context.Context, entityID uuid.UUID, version int64) (domain.EntityHistory, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) ListHistory(ctx context.Context, entityID uuid.UUID) ([]domain.EntityHistory, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, limit int, offset int) ([]domain.Entity, int, error) {
	copied := make([]domain.Entity, len(s.list))
	copy(copied, s.list)
	return copied, len(copied), nil
}

func (s *stubEntityRepoForEntities) ListByType(ctx context.Context, organizationID uuid.UUID, entityType string) ([]domain.Entity, error) {
	copied := make([]domain.Entity, len(s.list))
	copy(copied, s.list)
	return copied, nil
}

func (s *stubEntityRepoForEntities) GetByReference(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValue string) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) ListByReferences(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValues []string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) Update(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) Delete(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) RollbackEntity(ctx context.Context, id string, toVersion int64, reason string) error {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) GetAncestors(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) GetDescendants(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) GetChildren(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) GetSiblings(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) FilterByProperty(ctx context.Context, organizationID uuid.UUID, filter map[string]any) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) Count(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) CountByType(ctx context.Context, organizationID uuid.UUID, entityType string) (int64, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) ListIngestBatches(ctx context.Context, organizationID *uuid.UUID, statuses []string, limit int, offset int) ([]repository.IngestBatchRecord, error) {
	panic("not implemented")
}

func (s *stubEntityRepoForEntities) GetIngestBatchStats(ctx context.Context, organizationID *uuid.UUID) (repository.IngestBatchStats, error) {
	panic("not implemented")
}

func schemaKeyForEntities(orgID uuid.UUID, name string) string {
	return orgID.String() + ":" + strings.ToLower(name)
}

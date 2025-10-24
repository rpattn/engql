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

func TestConvertReferenceValuesToIDsHandlesValidUUIDs(t *testing.T) {
	parent := &graph.Entity{ID: "parent"}
	refID := uuid.New().String()

	refMap := map[string][]*graph.Entity{
		refID: {parent},
	}
	idParents := make(map[string][]*graph.Entity)
	missingIDs := make(map[string]struct{})

	handled, invalid := convertReferenceValuesToIDs(refMap, idParents, missingIDs)

	if !handled {
		t.Fatalf("expected references to be converted to IDs")
	}
	if len(invalid) != 0 {
		t.Fatalf("expected no invalid references, got %v", invalid)
	}

	if _, ok := missingIDs[refID]; !ok {
		t.Fatalf("expected %s to be recorded as missing id", refID)
	}

	parents, ok := idParents[refID]
	if !ok {
		t.Fatalf("expected parent slice for id %s", refID)
	}
	if len(parents) != 1 || parents[0] != parent {
		t.Fatalf("unexpected parents slice %#v", parents)
	}
}

func TestConvertReferenceValuesToIDsReportsInvalidValues(t *testing.T) {
	invalidValue := "not-a-uuid"
	refMap := map[string][]*graph.Entity{
		invalidValue: {nil},
	}

	idParents := make(map[string][]*graph.Entity)
	missingIDs := make(map[string]struct{})

	handled, invalid := convertReferenceValuesToIDs(refMap, idParents, missingIDs)

	if handled {
		t.Fatalf("expected conversion to fail for invalid values")
	}
	if len(invalid) != 1 || invalid[0] != invalidValue {
		t.Fatalf("unexpected invalid references %v", invalid)
	}

	if len(idParents) != 0 {
		t.Fatalf("expected no id parents, got %#v", idParents)
	}
	if len(missingIDs) != 0 {
		t.Fatalf("expected no missing ids, got %#v", missingIDs)
	}
}

func TestHydrateLinkedEntitiesFallsBackToIDLookupWhenReferenceFieldMissing(t *testing.T) {
	orgID := uuid.New()
	parentSchemaID := uuid.New()
	childSchemaID := uuid.New()
	childID := uuid.New()

	schemaRepo := &stubLinkedSchemaRepo{
		schemas: map[string]domain.EntitySchema{
			schemaKey(orgID, "Parent"): {
				ID:             parentSchemaID,
				OrganizationID: orgID,
				Name:           "Parent",
				Fields: []domain.FieldDefinition{
					{
						Name:                "linkedEntities",
						Type:                domain.FieldTypeEntityReferenceArray,
						ReferenceEntityType: "Child",
					},
				},
			},
			schemaKey(orgID, "Child"): {
				ID:             childSchemaID,
				OrganizationID: orgID,
				Name:           "Child",
				Fields:         []domain.FieldDefinition{},
			},
		},
	}

	childEntity := domain.Entity{
		ID:             childID,
		OrganizationID: orgID,
		SchemaID:       childSchemaID,
		EntityType:     "Child",
		Properties: map[string]any{
			"name": "child",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	entityRepo := &stubLinkedEntityRepo{
		entities: map[uuid.UUID]domain.Entity{
			childID: childEntity,
		},
	}

	resolver := &Resolver{
		entityRepo:       entityRepo,
		entitySchemaRepo: schemaRepo,
	}

	parent := &graph.Entity{
		ID:             uuid.New().String(),
		OrganizationID: orgID.String(),
		EntityType:     "Parent",
		Properties:     fmt.Sprintf("{\"linkedEntities\":[\"%s\"]}", childID.String()),
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

	if err := resolver.hydrateLinkedEntities(ctx, []*graph.Entity{parent}); err != nil {
		t.Fatalf("unexpected hydration error: %v", err)
	}

	if len(parent.LinkedEntities) != 1 {
		t.Fatalf("expected 1 linked entity, got %d", len(parent.LinkedEntities))
	}
	if parent.LinkedEntities[0].ID != childID.String() {
		t.Fatalf("expected linked entity %s, got %s", childID.String(), parent.LinkedEntities[0].ID)
	}
}

func TestHydrateLinkedEntitiesHandlesMissingReferenceEntityType(t *testing.T) {
	orgID := uuid.New()
	parentSchemaID := uuid.New()
	childSchemaID := uuid.New()
	childID := uuid.New()

	schemaRepo := &stubLinkedSchemaRepo{
		schemas: map[string]domain.EntitySchema{
			schemaKey(orgID, "Parent"): {
				ID:             parentSchemaID,
				OrganizationID: orgID,
				Name:           "Parent",
				Fields: []domain.FieldDefinition{
					{
						Name:                "linkedEntities",
						Type:                domain.FieldTypeEntityReferenceArray,
						ReferenceEntityType: "",
					},
				},
			},
			schemaKey(orgID, "Child"): {
				ID:             childSchemaID,
				OrganizationID: orgID,
				Name:           "Child",
				Fields:         []domain.FieldDefinition{},
			},
		},
	}

	childEntity := domain.Entity{
		ID:             childID,
		OrganizationID: orgID,
		SchemaID:       childSchemaID,
		EntityType:     "Child",
		Properties: map[string]any{
			"name": "child",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	entityRepo := &stubLinkedEntityRepo{
		entities: map[uuid.UUID]domain.Entity{
			childID: childEntity,
		},
	}

	resolver := &Resolver{
		entityRepo:       entityRepo,
		entitySchemaRepo: schemaRepo,
	}

	parent := &graph.Entity{
		ID:             uuid.New().String(),
		OrganizationID: orgID.String(),
		EntityType:     "Parent",
		Properties:     fmt.Sprintf("{\"linkedEntities\":[\"%s\"]}", childID.String()),
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

	if err := resolver.hydrateLinkedEntities(ctx, []*graph.Entity{parent}); err != nil {
		t.Fatalf("unexpected hydration error: %v", err)
	}

	if len(parent.LinkedEntities) != 1 {
		t.Fatalf("expected 1 linked entity, got %d", len(parent.LinkedEntities))
	}
	if parent.LinkedEntities[0].ID != childID.String() {
		t.Fatalf("expected linked entity %s, got %s", childID.String(), parent.LinkedEntities[0].ID)
	}
}

type stubLinkedSchemaRepo struct {
	schemas map[string]domain.EntitySchema
}

func (s *stubLinkedSchemaRepo) Create(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubLinkedSchemaRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubLinkedSchemaRepo) GetByName(ctx context.Context, organizationID uuid.UUID, name string) (domain.EntitySchema, error) {
	if schema, ok := s.schemas[schemaKey(organizationID, name)]; ok {
		return schema, nil
	}
	return domain.EntitySchema{}, fmt.Errorf("schema %s not found", name)
}

func (s *stubLinkedSchemaRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubLinkedSchemaRepo) ListVersions(ctx context.Context, organizationID uuid.UUID, name string) ([]domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubLinkedSchemaRepo) CreateVersion(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	panic("not implemented")
}

func (s *stubLinkedSchemaRepo) Exists(ctx context.Context, organizationID uuid.UUID, name string) (bool, error) {
	panic("not implemented")
}

func (s *stubLinkedSchemaRepo) ArchiveSchema(ctx context.Context, schemaID uuid.UUID) error {
	panic("not implemented")
}

type stubLinkedEntityRepo struct {
	entities map[uuid.UUID]domain.Entity
}

var _ repository.EntityRepository = (*stubLinkedEntityRepo)(nil)

func (s *stubLinkedEntityRepo) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) CreateBatch(ctx context.Context, items []repository.EntityBatchItem, opts repository.EntityBatchOptions) (repository.EntityBatchResult, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error) {
	if entity, ok := s.entities[id]; ok {
		return entity, nil
	}
	return domain.Entity{}, fmt.Errorf("entity %s not found", id)
}

func (s *stubLinkedEntityRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Entity, error) {
	result := make([]domain.Entity, 0, len(ids))
	for _, id := range ids {
		if entity, ok := s.entities[id]; ok {
			result = append(result, entity)
		}
	}
	return result, nil
}

func (s *stubLinkedEntityRepo) GetHistoryByVersion(ctx context.Context, entityID uuid.UUID, version int64) (domain.EntityHistory, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) ListHistory(ctx context.Context, entityID uuid.UUID) ([]domain.EntityHistory, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, limit int, offset int) ([]domain.Entity, int, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) ListByType(ctx context.Context, organizationID uuid.UUID, entityType string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) GetByReference(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValue string) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) ListByReferences(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValues []string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) Update(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) Delete(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) RollbackEntity(ctx context.Context, id string, toVersion int64, reason string) error {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) GetAncestors(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) GetDescendants(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) GetChildren(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) GetSiblings(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) FilterByProperty(ctx context.Context, organizationID uuid.UUID, filter map[string]any) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) Count(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) CountByType(ctx context.Context, organizationID uuid.UUID, entityType string) (int64, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) ListIngestBatches(ctx context.Context, organizationID *uuid.UUID, statuses []string, limit int, offset int) ([]repository.IngestBatchRecord, error) {
	panic("not implemented")
}

func (s *stubLinkedEntityRepo) GetIngestBatchStats(ctx context.Context, organizationID *uuid.UUID) (repository.IngestBatchStats, error) {
	panic("not implemented")
}

func schemaKey(orgID uuid.UUID, name string) string {
	return orgID.String() + ":" + strings.ToLower(name)
}

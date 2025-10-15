package ingestion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"

	"github.com/google/uuid"
)

func TestServiceIngestCreatesSchemaAndEntities(t *testing.T) {
	orgID := uuid.New()
	schemaRepo := &stubSchemaRepo{}
	entityRepo := &stubEntityRepo{}
	logRepo := &stubLogRepo{}

	service := NewService(schemaRepo, entityRepo, logRepo)

	data := `name,age,active
Alice,30,true
Bob,25,false
`
	req := Request{
		OrganizationID: orgID,
		SchemaName:     "Person",
		FileName:       "people.csv",
		Data:           strings.NewReader(data),
	}

	summary, err := service.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}

	if !summary.SchemaCreated {
		t.Fatalf("expected schema to be created")
	}
	if summary.TotalRows != 2 || summary.ValidRows != 2 || summary.InvalidRows != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	if len(schemaRepo.current.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(schemaRepo.current.Fields))
	}

	fieldTypes := map[string]domain.FieldType{}
	for _, field := range schemaRepo.current.Fields {
		fieldTypes[field.Name] = field.Type
	}
	if fieldTypes["name"] != domain.FieldTypeString {
		t.Fatalf("expected name field type string, got %s", fieldTypes["name"])
	}
	if fieldTypes["age"] != domain.FieldTypeInteger {
		t.Fatalf("expected age field type integer, got %s", fieldTypes["age"])
	}
	if fieldTypes["active"] != domain.FieldTypeBoolean {
		t.Fatalf("expected active field type boolean, got %s", fieldTypes["active"])
	}

	if len(entityRepo.created) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(entityRepo.created))
	}
}

func TestServiceIngestAppendsFields(t *testing.T) {
	orgID := uuid.New()
	initialSchema := domain.EntitySchema{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Metrics",
		Fields: []domain.FieldDefinition{
			{
				Name:     "name",
				Type:     domain.FieldTypeString,
				Required: true,
			},
		},
	}

	schemaRepo := &stubSchemaRepo{
		exists:  true,
		current: initialSchema,
	}
	entityRepo := &stubEntityRepo{}
	logRepo := &stubLogRepo{}

	service := NewService(schemaRepo, entityRepo, logRepo)

	data := `name,score
Alpha,42
Beta,100
`
	req := Request{
		OrganizationID: orgID,
		SchemaName:     "Metrics",
		FileName:       "metrics.csv",
		Data:           strings.NewReader(data),
	}

	summary, err := service.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}

	if summary.SchemaCreated {
		t.Fatalf("did not expect schema to be created")
	}
	if len(summary.NewFieldsDetected) != 1 || summary.NewFieldsDetected[0] != "score" {
		t.Fatalf("expected score to be detected as new field, summary: %+v", summary)
	}
	if summary.ValidRows != 2 || summary.InvalidRows != 0 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if len(entityRepo.created) != 2 {
		t.Fatalf("expected 2 entities inserted, got %d", len(entityRepo.created))
	}
	if len(logRepo.entries) != 0 {
		t.Fatalf("did not expect ingestion errors, found %d", len(logRepo.entries))
	}
}

func TestServiceIngestDetectsTypeConflicts(t *testing.T) {
	orgID := uuid.New()
	initialSchema := domain.EntitySchema{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Counters",
		Fields: []domain.FieldDefinition{
			{
				Name:     "count",
				Type:     domain.FieldTypeInteger,
				Required: true,
			},
		},
	}

	schemaRepo := &stubSchemaRepo{
		exists:  true,
		current: initialSchema,
	}
	entityRepo := &stubEntityRepo{}
	logRepo := &stubLogRepo{}

	service := NewService(schemaRepo, entityRepo, logRepo)

	data := `count
3.5
`

	req := Request{
		OrganizationID: orgID,
		SchemaName:     "Counters",
		FileName:       "counters.csv",
		Data:           strings.NewReader(data),
	}

	summary, err := service.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}

	if summary.ValidRows != 0 || summary.InvalidRows != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(summary.SchemaChanges) == 0 {
		t.Fatalf("expected schema change due to type conflict")
	}
	if len(logRepo.entries) == 0 {
		t.Fatalf("expected conflict to be logged")
	}
	if len(entityRepo.created) != 0 {
		t.Fatalf("expected no entities inserted, got %d", len(entityRepo.created))
	}
}

type stubSchemaRepo struct {
	exists   bool
	current  domain.EntitySchema
	versions []domain.EntitySchema
}

func (s *stubSchemaRepo) Create(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	s.exists = true
	s.current = schema
	s.appendVersion(schema)
	return schema, nil
}

func (s *stubSchemaRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.EntitySchema, error) {
	return domain.EntitySchema{}, errors.New("not implemented")
}

func (s *stubSchemaRepo) GetByName(ctx context.Context, organizationID uuid.UUID, name string) (domain.EntitySchema, error) {
	return s.current, nil
}

func (s *stubSchemaRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.EntitySchema, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSchemaRepo) Exists(ctx context.Context, organizationID uuid.UUID, name string) (bool, error) {
	return s.exists, nil
}

func (s *stubSchemaRepo) ListVersions(ctx context.Context, organizationID uuid.UUID, name string) ([]domain.EntitySchema, error) {
	if len(s.versions) == 0 {
		return []domain.EntitySchema{s.current}, nil
	}
	return append([]domain.EntitySchema(nil), s.versions...), nil
}

func (s *stubSchemaRepo) CreateVersion(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	s.exists = true
	s.current = schema
	s.appendVersion(schema)
	return schema, nil
}

func (s *stubSchemaRepo) appendVersion(schema domain.EntitySchema) {
	s.versions = append([]domain.EntitySchema{{ // ensure latest first
		ID:                schema.ID,
		OrganizationID:    schema.OrganizationID,
		Name:              schema.Name,
		Description:       schema.Description,
		Fields:            schema.Fields,
		Version:           schema.Version,
		PreviousVersionID: schema.PreviousVersionID,
		Status:            schema.Status,
		CreatedAt:         schema.CreatedAt,
		UpdatedAt:         schema.UpdatedAt,
	}}, s.versions...)
}

func (s *stubSchemaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (s *stubSchemaRepo) Update(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error) {
	s.current = schema
	return schema, nil
}

type stubEntityRepo struct {
	created []domain.Entity
}

func (s *stubEntityRepo) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	s.created = append(s.created, entity)
	return entity, nil
}

func (s *stubEntityRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error) {
	return domain.Entity{}, errors.New("not implemented")
}

func (s *stubEntityRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Entity, error) {
	return nil, errors.New("not implemented")
}

func (s *stubEntityRepo) List(ctx context.Context, organizationID uuid.UUID, limit int, offset int) ([]domain.Entity, int, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *stubEntityRepo) ListByType(ctx context.Context, organizationID uuid.UUID, entityType string) ([]domain.Entity, error) {
	return nil, errors.New("not implemented")
}

func (s *stubEntityRepo) Update(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	return domain.Entity{}, errors.New("not implemented")
}

func (s *stubEntityRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (s *stubEntityRepo) GetAncestors(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	return nil, errors.New("not implemented")
}

func (s *stubEntityRepo) GetDescendants(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	return nil, errors.New("not implemented")
}

func (s *stubEntityRepo) GetChildren(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	return nil, errors.New("not implemented")
}

func (s *stubEntityRepo) GetSiblings(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	return nil, errors.New("not implemented")
}

func (s *stubEntityRepo) FilterByProperty(ctx context.Context, organizationID uuid.UUID, filter map[string]any) ([]domain.Entity, error) {
	return nil, errors.New("not implemented")
}

func (s *stubEntityRepo) Count(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	return 0, errors.New("not implemented")
}

func (s *stubEntityRepo) CountByType(ctx context.Context, organizationID uuid.UUID, entityType string) (int64, error) {
	return 0, errors.New("not implemented")
}

func (s *stubEntityRepo) RollbackEntity(ctx context.Context, id string, toVersion int64, reason string) error {
	return nil
}

type stubLogRepo struct {
	entries []domain.IngestionLogEntry
}

func (s *stubLogRepo) Record(ctx context.Context, entry domain.IngestionLogEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

var _ repository.EntitySchemaRepository = (*stubSchemaRepo)(nil)
var _ repository.EntityRepository = (*stubEntityRepo)(nil)
var _ repository.IngestionLogRepository = (*stubLogRepo)(nil)

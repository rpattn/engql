package graphql

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"
)

type stubEntityRepository struct {
	current *domain.Entity
	history map[int64]domain.EntityHistory
}

var _ repository.EntityRepository = (*stubEntityRepository)(nil)

func (s *stubEntityRepository) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) CreateBatch(ctx context.Context, items []repository.EntityBatchItem, opts repository.EntityBatchOptions) (repository.EntityBatchResult, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error) {
	if s.current == nil {
		return domain.Entity{}, fmt.Errorf("failed to get entity: %w", pgx.ErrNoRows)
	}
	if s.current.ID != id {
		return domain.Entity{}, fmt.Errorf("failed to get entity: %w", pgx.ErrNoRows)
	}
	return *s.current, nil
}

func (s *stubEntityRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) GetHistoryByVersion(ctx context.Context, entityID uuid.UUID, version int64) (domain.EntityHistory, error) {
	if snapshot, ok := s.history[version]; ok {
		return snapshot, nil
	}
	return domain.EntityHistory{}, fmt.Errorf("failed to get entity history: %w", pgx.ErrNoRows)
}

func (s *stubEntityRepository) ListHistory(ctx context.Context, entityID uuid.UUID) ([]domain.EntityHistory, error) {
	result := make([]domain.EntityHistory, 0, len(s.history))
	for _, snapshot := range s.history {
		result = append(result, snapshot)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version > result[j].Version
	})
	return result, nil
}

func (s *stubEntityRepository) List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, limit int, offset int) ([]domain.Entity, int, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) ListByType(ctx context.Context, organizationID uuid.UUID, entityType string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) Update(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func (s *stubEntityRepository) RollbackEntity(ctx context.Context, id string, toVersion int64, reason string) error {
	panic("not implemented")
}

func (s *stubEntityRepository) GetAncestors(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) GetDescendants(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) GetChildren(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) GetSiblings(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) FilterByProperty(ctx context.Context, organizationID uuid.UUID, filter map[string]any) ([]domain.Entity, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) Count(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) CountByType(ctx context.Context, organizationID uuid.UUID, entityType string) (int64, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) ListIngestBatches(ctx context.Context, organizationID *uuid.UUID, statuses []string, limit int, offset int) ([]repository.IngestBatchRecord, error) {
	panic("not implemented")
}

func (s *stubEntityRepository) GetIngestBatchStats(ctx context.Context, organizationID *uuid.UUID) (repository.IngestBatchStats, error) {
	panic("not implemented")
}

func TestResolverEntityDiff(t *testing.T) {
	entityID := uuid.New()
	schemaID := uuid.New()
	now := time.Now()

	current := domain.Entity{
		ID:             entityID,
		OrganizationID: uuid.New(),
		SchemaID:       schemaID,
		EntityType:     "Example",
		Path:           "root.node",
		Properties: map[string]any{
			"name":  "Target",
			"count": float64(2),
		},
		Version:   3,
		CreatedAt: now,
		UpdatedAt: now,
	}

	history := map[int64]domain.EntityHistory{
		1: {
			ID:             uuid.New(),
			EntityID:       entityID,
			OrganizationID: current.OrganizationID,
			SchemaID:       schemaID,
			EntityType:     "Example",
			Path:           "root.node",
			Properties: map[string]any{
				"name": "Base",
			},
			CreatedAt:  now.Add(-2 * time.Hour),
			UpdatedAt:  now.Add(-2 * time.Hour),
			Version:    1,
			ChangeType: "CREATE",
		},
		2: {
			ID:             uuid.New(),
			EntityID:       entityID,
			OrganizationID: current.OrganizationID,
			SchemaID:       schemaID,
			EntityType:     "Example",
			Path:           "root.node",
			Properties: map[string]any{
				"name": "Mid",
			},
			CreatedAt:  now.Add(-time.Hour),
			UpdatedAt:  now.Add(-time.Hour),
			Version:    2,
			ChangeType: "UPDATE",
		},
	}

	repo := &stubEntityRepository{current: &current, history: history}
	resolver := &Resolver{entityRepo: repo}

	result, err := resolver.EntityDiff(context.Background(), entityID.String(), 1, 3)
	if err != nil {
		t.Fatalf("unexpected resolver error: %v", err)
	}

	if result.Base == nil {
		t.Fatalf("expected base snapshot, got nil")
	}
	if result.Target == nil {
		t.Fatalf("expected target snapshot, got nil")
	}
	if result.Base.Version != 1 {
		t.Errorf("expected base version 1, got %d", result.Base.Version)
	}
	if result.Target.Version != 3 {
		t.Errorf("expected target version 3, got %d", result.Target.Version)
	}
	if result.UnifiedDiff == nil {
		t.Fatalf("expected diff string, got nil")
	}
	if !strings.Contains(*result.UnifiedDiff, "-  name: \"Base\"") {
		t.Errorf("diff missing base change: %s", *result.UnifiedDiff)
	}
	if !strings.Contains(*result.UnifiedDiff, "+  count: 2") {
		t.Errorf("diff missing target addition: %s", *result.UnifiedDiff)
	}
	if len(result.Base.CanonicalText) == 0 || len(result.Target.CanonicalText) == 0 {
		t.Fatalf("expected canonical text for both snapshots")
	}
}

func TestResolverEntityHistory(t *testing.T) {
	entityID := uuid.New()
	schemaID := uuid.New()
	now := time.Now()

	current := domain.Entity{
		ID:             entityID,
		OrganizationID: uuid.New(),
		SchemaID:       schemaID,
		EntityType:     "Example",
		Path:           "root.node",
		Properties: map[string]any{
			"name": "Latest",
		},
		Version:   3,
		CreatedAt: now,
		UpdatedAt: now,
	}

	history := map[int64]domain.EntityHistory{
		1: {
			ID:             uuid.New(),
			EntityID:       entityID,
			OrganizationID: current.OrganizationID,
			SchemaID:       schemaID,
			EntityType:     "Example",
			Path:           "root.node",
			Properties: map[string]any{
				"name": "Initial",
			},
			CreatedAt:  now.Add(-3 * time.Hour),
			UpdatedAt:  now.Add(-3 * time.Hour),
			Version:    1,
			ChangeType: "CREATE",
		},
		2: {
			ID:             uuid.New(),
			EntityID:       entityID,
			OrganizationID: current.OrganizationID,
			SchemaID:       schemaID,
			EntityType:     "Example",
			Path:           "root.node",
			Properties: map[string]any{
				"name": "Mid",
			},
			CreatedAt:  now.Add(-2 * time.Hour),
			UpdatedAt:  now.Add(-2 * time.Hour),
			Version:    2,
			ChangeType: "UPDATE",
		},
	}

	repo := &stubEntityRepository{current: &current, history: history}
	resolver := &Resolver{entityRepo: repo}

	snapshots, err := resolver.EntityHistory(context.Background(), entityID.String())
	if err != nil {
		t.Fatalf("unexpected resolver error: %v", err)
	}

	if len(snapshots) != 3 {
		t.Fatalf("expected three snapshots (current + history), got %d", len(snapshots))
	}

	if snapshots[0].Version != int(current.Version) {
		t.Fatalf("expected current version first, got %d", snapshots[0].Version)
	}

	if snapshots[1].Version != 2 || snapshots[2].Version != 1 {
		t.Fatalf("expected history versions in descending order, got [%d, %d]", snapshots[1].Version, snapshots[2].Version)
	}

	for i, snapshot := range snapshots {
		if len(snapshot.CanonicalText) == 0 {
			t.Fatalf("snapshot %d missing canonical text", i)
		}
	}
}

func TestResolverEntityDiffMissingVersion(t *testing.T) {
	entityID := uuid.New()
	schemaID := uuid.New()
	now := time.Now()

	current := domain.Entity{
		ID:             entityID,
		OrganizationID: uuid.New(),
		SchemaID:       schemaID,
		EntityType:     "Example",
		Path:           "root.node",
		Properties: map[string]any{
			"name": "Target",
		},
		Version:   4,
		CreatedAt: now,
		UpdatedAt: now,
	}

	repo := &stubEntityRepository{current: &current, history: map[int64]domain.EntityHistory{}}
	resolver := &Resolver{entityRepo: repo}

	result, err := resolver.EntityDiff(context.Background(), entityID.String(), 2, 4)
	if err != nil {
		t.Fatalf("unexpected resolver error: %v", err)
	}

	if result.Base != nil {
		t.Fatalf("expected missing base snapshot to return nil")
	}
	if result.Target == nil {
		t.Fatalf("expected target snapshot from current entity")
	}
	if result.UnifiedDiff != nil {
		t.Fatalf("expected nil diff when one snapshot missing, got %v", *result.UnifiedDiff)
	}
}

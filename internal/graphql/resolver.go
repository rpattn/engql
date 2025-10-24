package graphql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Resolver handles GraphQL queries and mutations
type Resolver struct {
	orgRepo             repository.OrganizationRepository
	entitySchemaRepo    repository.EntitySchemaRepository
	entityRepo          repository.EntityRepository
	entityJoinRepo      repository.EntityJoinRepository
	referenceFieldCache sync.Map
}

// NewResolver creates a new GraphQL resolver
func NewResolver(
	orgRepo repository.OrganizationRepository,
	entitySchemaRepo repository.EntitySchemaRepository,
	entityRepo repository.EntityRepository,
	entityJoinRepo repository.EntityJoinRepository,
) *Resolver {
	return &Resolver{
		orgRepo:          orgRepo,
		entitySchemaRepo: entitySchemaRepo,
		entityRepo:       entityRepo,
		entityJoinRepo:   entityJoinRepo,
	}
}

// Query resolvers

// Organizations returns all organizations
func (r *Resolver) Organizations(ctx context.Context) ([]*graph.Organization, error) {
	orgs, err := r.orgRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	result := make([]*graph.Organization, len(orgs))
	for i, org := range orgs {
		result[i] = &graph.Organization{
			ID:          org.ID.String(),
			Name:        org.Name,
			Description: &org.Description,
			CreatedAt:   org.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   org.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// Organization returns a specific organization by ID
func (r *Resolver) Organization(ctx context.Context, id string) (*graph.Organization, error) {
	orgID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	org, err := r.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &graph.Organization{
		ID:          org.ID.String(),
		Name:        org.Name,
		Description: &org.Description,
		CreatedAt:   org.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   org.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// OrganizationByName returns a specific organization by name
func (r *Resolver) OrganizationByName(ctx context.Context, name string) (*graph.Organization, error) {
	org, err := r.orgRepo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization by name: %w", err)
	}

	return &graph.Organization{
		ID:          org.ID.String(),
		Name:        org.Name,
		Description: &org.Description,
		CreatedAt:   org.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   org.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// EntitySchemas returns all entity schemas for an organization
func (r *Resolver) EntitySchemas(ctx context.Context, organizationID string) ([]*graph.EntitySchema, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	schemas, err := r.entitySchemaRepo.List(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity schemas: %w", err)
	}

	result := make([]*graph.EntitySchema, len(schemas))
	for i, schema := range schemas {
		result[i] = toGraphEntitySchema(schema)
	}

	return result, nil
}

// EntitySchema returns a specific entity schema by ID
func (r *Resolver) EntitySchema(ctx context.Context, id string) (*graph.EntitySchema, error) {
	schemaID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID: %w", err)
	}

	schema, err := r.entitySchemaRepo.GetByID(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity schema: %w", err)
	}

	return toGraphEntitySchema(schema), nil
}

// EntitySchemaByName returns a specific entity schema by organization ID and name
func (r *Resolver) EntitySchemaByName(ctx context.Context, organizationID, name string) (*graph.EntitySchema, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	schema, err := r.entitySchemaRepo.GetByName(ctx, orgID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity schema by name: %w", err)
	}

	return toGraphEntitySchema(schema), nil
}

// EntitySchemaVersions lists all schema versions for a given name
func (r *Resolver) EntitySchemaVersions(ctx context.Context, organizationID, name string) ([]*graph.EntitySchema, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	versions, err := r.entitySchemaRepo.ListVersions(ctx, orgID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list schema versions: %w", err)
	}

	result := make([]*graph.EntitySchema, len(versions))
	for i, schema := range versions {
		result[i] = toGraphEntitySchema(schema)
	}
	return result, nil
}

// Entities returns entities with filtering and pagination
func (r *Resolver) Entities(ctx context.Context, organizationID string, filter *graph.EntityFilter, pagination *graph.PaginationInput) (*graph.EntityConnection, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Default pagination
	limit := 10
	offset := 0
	if pagination != nil {
		if pagination.Limit != nil {
			limit = *pagination.Limit
		}
		if pagination.Offset != nil {
			offset = *pagination.Offset
		}
	}

	// Fetch only the requested page from the repository
	domainFilter := convertEntityFilter(filter)

	entities, totalCount, err := r.entityRepo.List(ctx, orgID, domainFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	ctxWithCache, cache := ensureEntityCache(ctx)

	result := make([]*graph.Entity, 0, len(entities))
	var errs []error

	for _, entity := range entities {
		mapped, err := r.mapDomainEntity(ctxWithCache, entity)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result = append(result, mapped)
		if mapped.ID != "" {
			cache[mapped.ID] = mapped
		}
	}

	if err := r.hydrateLinkedEntities(ctxWithCache, result); err != nil {
		errs = append(errs, err)
	}

	if err := combineErrors(errs); err != nil {
		return nil, err
	}

	hasNextPage := offset+limit < totalCount
	hasPreviousPage := offset > 0

	return &graph.EntityConnection{
		Entities: result,
		PageInfo: &graph.PageInfo{
			HasNextPage:     hasNextPage,
			HasPreviousPage: hasPreviousPage,
			TotalCount:      totalCount,
		},
	}, nil
}

func convertEntityFilter(filter *graph.EntityFilter) *domain.EntityFilter {
	if filter == nil {
		return nil
	}

	result := &domain.EntityFilter{}

	if filter.EntityType != nil {
		result.EntityType = strings.TrimSpace(*filter.EntityType)
	}

	if len(filter.PropertyFilters) > 0 {
		for _, pf := range filter.PropertyFilters {
			if pf == nil {
				continue
			}
			key := strings.TrimSpace(pf.Key)
			if key == "" {
				continue
			}
			value := ""
			if pf.Value != nil {
				value = strings.TrimSpace(*pf.Value)
			}
			result.PropertyFilters = append(result.PropertyFilters, domain.PropertyFilter{
				Key:     key,
				Value:   value,
				Exists:  pf.Exists,
				InArray: pf.InArray,
			})
		}
	}

	if filter.TextSearch != nil {
		result.TextSearch = strings.TrimSpace(*filter.TextSearch)
	}

	if result.EntityType == "" && len(result.PropertyFilters) == 0 && strings.TrimSpace(result.TextSearch) == "" {
		return nil
	}

	return result
}

// GetEntity returns a specific entity by ID
func (r *Resolver) GetEntity(ctx context.Context, id string) (*graph.Entity, error) {
	entityID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	entity, err := r.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	propertiesJSON, err := entity.GetPropertiesAsJSONB()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &graph.Entity{
		ID:             entity.ID.String(),
		OrganizationID: entity.OrganizationID.String(),
		EntityType:     entity.EntityType,
		Path:           entity.Path,
		Properties:     string(propertiesJSON),
		CreatedAt:      entity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      entity.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// EntityDiff compares two versions of an entity and returns a structured diff response.
func (r *Resolver) EntityDiff(ctx context.Context, id string, baseVersion int, targetVersion int) (*graph.EntityDiffResult, error) {
	if r.entityRepo == nil {
		return nil, fmt.Errorf("entity repository not configured")
	}

	entityID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	var current *domain.Entity
	entity, err := r.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to load entity: %w", err)
		}
	} else {
		current = &entity
	}

	baseSnapshot, err := r.loadEntitySnapshot(ctx, entityID, int64(baseVersion), current)
	if err != nil {
		return nil, err
	}

	targetSnapshot, err := r.loadEntitySnapshot(ctx, entityID, int64(targetVersion), current)
	if err != nil {
		return nil, err
	}

	result := &graph.EntityDiffResult{}

	baseView, err := snapshotToGraph(baseSnapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare base snapshot: %w", err)
	}
	result.Base = baseView

	targetView, err := snapshotToGraph(targetSnapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare target snapshot: %w", err)
	}
	result.Target = targetView

	if baseSnapshot != nil && targetSnapshot != nil {
		diff, err := domain.DiffEntitySnapshots(
			fmt.Sprintf("version-%d", baseSnapshot.Version),
			baseSnapshot,
			fmt.Sprintf("version-%d", targetSnapshot.Version),
			targetSnapshot,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to compute entity diff: %w", err)
		}
		result.UnifiedDiff = &diff
	}

	return result, nil
}

// EntityHistory returns the available snapshots for an entity, including the current state when present.
func (r *Resolver) EntityHistory(ctx context.Context, id string) ([]*graph.EntitySnapshotView, error) {
	if r.entityRepo == nil {
		return nil, fmt.Errorf("entity repository not configured")
	}

	entityID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	var current *domain.Entity
	entity, err := r.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to load entity: %w", err)
		}
	} else {
		current = &entity
	}

	history, err := r.entityRepo.ListHistory(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity history: %w", err)
	}

	snapshots := make([]*graph.EntitySnapshotView, 0, len(history)+1)

	if current != nil {
		snapshot := domain.NewEntitySnapshotFromEntity(*current)
		view, err := snapshotToGraph(&snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare current snapshot: %w", err)
		}
		snapshots = append(snapshots, view)
	}

	for _, record := range history {
		if current != nil && record.Version == current.Version {
			continue
		}

		snapshot := domain.NewEntitySnapshotFromHistory(record)
		view, err := snapshotToGraph(&snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare history snapshot: %w", err)
		}
		snapshots = append(snapshots, view)
	}

	return snapshots, nil
}

// EntitiesByType returns entities of a specific type for an organization
func (r *Resolver) EntitiesByType(ctx context.Context, organizationID, entityType string) ([]*graph.Entity, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	entities, err := r.entityRepo.ListByType(ctx, orgID, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities by type: %w", err)
	}

	ctxWithCache, cache := ensureEntityCache(ctx)

	result := make([]*graph.Entity, 0, len(entities))
	var errs []error

	for _, entity := range entities {
		gqlEntity, err := r.mapDomainEntity(ctxWithCache, entity)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result = append(result, gqlEntity)
		if gqlEntity.ID != "" {
			cache[gqlEntity.ID] = gqlEntity
		}
	}

	if err := r.hydrateLinkedEntities(ctxWithCache, result); err != nil {
		errs = append(errs, err)
	}

	if err := combineErrors(errs); err != nil {
		return result, err
	}

	return result, nil
}

func (r *Resolver) loadEntitySnapshot(ctx context.Context, entityID uuid.UUID, version int64, current *domain.Entity) (*domain.EntitySnapshot, error) {
	if current != nil && current.Version == version {
		snapshot := domain.NewEntitySnapshotFromEntity(*current)
		return &snapshot, nil
	}

	history, err := r.entityRepo.GetHistoryByVersion(ctx, entityID, version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load entity history version %d: %w", version, err)
	}

	snapshot := domain.NewEntitySnapshotFromHistory(history)
	return &snapshot, nil
}

func snapshotToGraph(snapshot *domain.EntitySnapshot) (*graph.EntitySnapshotView, error) {
	if snapshot == nil {
		return nil, nil
	}

	lines, err := snapshot.CanonicalText()
	if err != nil {
		return nil, err
	}

	canonical := make([]string, len(lines))
	copy(canonical, lines)

	return &graph.EntitySnapshotView{
		Version:       int(snapshot.Version),
		Path:          snapshot.Path,
		SchemaID:      snapshot.SchemaID.String(),
		EntityType:    snapshot.EntityType,
		CanonicalText: canonical,
	}, nil
}

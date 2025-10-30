package graphql

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"
	"github.com/rpattn/engql/internal/transformations"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Resolver handles GraphQL queries and mutations
type Resolver struct {
	orgRepo                  repository.OrganizationRepository
	entitySchemaRepo         repository.EntitySchemaRepository
	entityRepo               repository.EntityRepository
	entityJoinRepo           repository.EntityJoinRepository
	entityTransformationRepo repository.EntityTransformationRepository
	transformationExecutor   *transformations.Executor
	referenceFieldCache      sync.Map
}

// NewResolver creates a new GraphQL resolver
func NewResolver(
	orgRepo repository.OrganizationRepository,
	entitySchemaRepo repository.EntitySchemaRepository,
	entityRepo repository.EntityRepository,
	entityJoinRepo repository.EntityJoinRepository,
	entityTransformationRepo repository.EntityTransformationRepository,
	transformationExecutor *transformations.Executor,
) *Resolver {
	return &Resolver{
		orgRepo:                  orgRepo,
		entitySchemaRepo:         entitySchemaRepo,
		entityRepo:               entityRepo,
		entityJoinRepo:           entityJoinRepo,
		entityTransformationRepo: entityTransformationRepo,
		transformationExecutor:   transformationExecutor,
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
func (r *Resolver) Entities(ctx context.Context, organizationID string, filter *graph.EntityFilter, pagination *graph.PaginationInput, sort *graph.EntitySortInput) (*graph.EntityConnection, error) {
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

	domainSort := convertEntitySort(sort)

	entities, totalCount, err := r.entityRepo.List(ctx, orgID, domainFilter, domainSort, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Convert to GraphQL type
	result := make([]*graph.Entity, len(entities))
	for i, entity := range entities {
		mapped, err := r.mapDomainEntity(ctx, entity)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
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

func convertEntitySort(sort *graph.EntitySortInput) *domain.EntitySort {
	if sort == nil {
		return nil
	}

	direction := graph.SortDirectionAsc
	if sort.Direction != nil {
		direction = *sort.Direction
	}

	result := &domain.EntitySort{
		Direction: domain.SortDirection(strings.ToLower(string(direction))),
	}

	switch sort.Field {
	case graph.EntitySortFieldCreatedAt:
		result.Field = domain.EntitySortFieldCreatedAt
	case graph.EntitySortFieldUpdatedAt:
		result.Field = domain.EntitySortFieldUpdatedAt
	case graph.EntitySortFieldEntityType:
		result.Field = domain.EntitySortFieldEntityType
	case graph.EntitySortFieldPath:
		result.Field = domain.EntitySortFieldPath
	case graph.EntitySortFieldVersion:
		result.Field = domain.EntitySortFieldVersion
	case graph.EntitySortFieldProperty:
		result.Field = domain.EntitySortFieldProperty
		if sort.PropertyKey != nil {
			result.PropertyKey = strings.TrimSpace(*sort.PropertyKey)
		}
	default:
		return nil
	}

	if result.Direction != domain.SortDirectionAsc && result.Direction != domain.SortDirectionDesc {
		result.Direction = domain.SortDirectionDesc
	}

	if result.Field == domain.EntitySortFieldProperty && strings.TrimSpace(result.PropertyKey) == "" {
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

	gqlEntity, err := r.mapDomainEntity(ctx, entity)
	if err != nil {
		return nil, err
	}

	return gqlEntity, nil
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

// TransformationExecution resolves flattened transformation results.
func (r *Resolver) TransformationExecution(
	ctx context.Context,
	transformationID string,
	filters []*graph.TransformationExecutionFilterInput,
	sortInput *graph.TransformationExecutionSortInput,
	pagination *graph.PaginationInput,
) (*graph.TransformationExecutionConnection, error) {
	id, err := uuid.Parse(transformationID)
	if err != nil {
		return nil, fmt.Errorf("invalid transformation ID: %w", err)
	}

	transformation, err := r.entityTransformationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load transformation: %w", err)
	}

	materializeConfig, err := findMaterializeConfig(transformation)
	if err != nil {
		return nil, err
	}

	columns := buildExecutionColumns(materializeConfig)

	limit := 0
	offset := 0
	if pagination != nil {
		if pagination.Limit != nil {
			limit = *pagination.Limit
		}
		if pagination.Offset != nil {
			offset = *pagination.Offset
		}
	}

	aliasFilters := filtersByAlias(filters)

	runtimeTransformation := cloneTransformation(transformation)

	finalNodeID, err := finalNodeID(runtimeTransformation)
	if err != nil {
		return nil, err
	}

	currentOutput := finalNodeID

	if len(aliasFilters) > 0 {
		aliases := make([]string, 0, len(aliasFilters))
		for alias := range aliasFilters {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			filters := aliasFilters[alias]
			if len(filters) == 0 {
				continue
			}
			filterNodeID := uuid.New()
			filterConfig := &domain.EntityTransformationFilterConfig{
				Alias:   alias,
				Filters: clonePropertyFilters(filters),
			}
			runtimeTransformation.Nodes = append(runtimeTransformation.Nodes, domain.EntityTransformationNode{
				ID:     filterNodeID,
				Name:   fmt.Sprintf("runtime-filter-%s", alias),
				Type:   domain.TransformationNodeFilter,
				Inputs: []uuid.UUID{currentOutput},
				Filter: filterConfig,
			})
			currentOutput = filterNodeID
		}
	}

	if sortInput != nil && strings.TrimSpace(sortInput.Alias) != "" && strings.TrimSpace(sortInput.Field) != "" {
		direction := domain.JoinSortAsc
		if sortInput.Direction != nil && *sortInput.Direction == graph.SortDirectionDesc {
			direction = domain.JoinSortDesc
		}
		sortNodeID := uuid.New()
		runtimeTransformation.Nodes = append(runtimeTransformation.Nodes, domain.EntityTransformationNode{
			ID:     sortNodeID,
			Name:   fmt.Sprintf("runtime-sort-%s-%s", sortInput.Alias, sortInput.Field),
			Type:   domain.TransformationNodeSort,
			Inputs: []uuid.UUID{currentOutput},
			Sort: &domain.EntityTransformationSortConfig{
				Alias:     sortInput.Alias,
				Field:     sortInput.Field,
				Direction: direction,
			},
		})
		currentOutput = sortNodeID
	}

	if limit > 0 || offset > 0 {
		paginateNodeID := uuid.New()
		runtimeTransformation.Nodes = append(runtimeTransformation.Nodes, domain.EntityTransformationNode{
			ID:     paginateNodeID,
			Name:   "runtime-paginate",
			Type:   domain.TransformationNodePaginate,
			Inputs: []uuid.UUID{currentOutput},
			Paginate: &domain.EntityTransformationPaginateConfig{
				Limit:  optionalIntPointer(limit),
				Offset: optionalIntPointer(offset),
			},
		})
		currentOutput = paginateNodeID
	}

	options := domain.EntityTransformationExecutionOptions{
		Limit:  limit,
		Offset: offset,
	}

	execResult, err := r.transformationExecutor.Execute(ctx, runtimeTransformation, options)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transformation: %w", err)
	}

	rows := buildExecutionRows(execResult.Records, columns)

	totalCount := execResult.TotalCount

	hasPrev := offset > 0 && totalCount > 0
	hasNext := limit > 0 && offset+limit < totalCount

	pageInfo := &graph.PageInfo{
		TotalCount:      totalCount,
		HasPreviousPage: hasPrev,
		HasNextPage:     hasNext,
	}

	return &graph.TransformationExecutionConnection{
		Columns:  columns,
		Rows:     rows,
		PageInfo: pageInfo,
	}, nil
}

func findMaterializeConfig(transformation domain.EntityTransformation) (*domain.EntityTransformationMaterializeConfig, error) {
	var config *domain.EntityTransformationMaterializeConfig
	for i := range transformation.Nodes {
		node := transformation.Nodes[i]
		if node.Type != domain.TransformationNodeMaterialize || node.Materialize == nil {
			continue
		}
		copyConfig := *node.Materialize
		config = &copyConfig
	}
	if config == nil {
		return nil, fmt.Errorf("transformation %s missing materialize node", transformation.ID)
	}
	return config, nil
}

func buildExecutionColumns(config *domain.EntityTransformationMaterializeConfig) []*graph.TransformationExecutionColumn {
	if config == nil {
		return []*graph.TransformationExecutionColumn{}
	}
	columns := make([]*graph.TransformationExecutionColumn, 0)
	for _, output := range config.Outputs {
		for _, field := range output.Fields {
			key := columnKey(output.Alias, field.OutputField)
			columns = append(columns, &graph.TransformationExecutionColumn{
				Key:         key,
				Alias:       output.Alias,
				Field:       field.OutputField,
				Label:       field.OutputField,
				SourceAlias: field.SourceAlias,
				SourceField: field.SourceField,
			})
		}
	}
	return columns
}

func filtersByAlias(inputs []*graph.TransformationExecutionFilterInput) map[string][]domain.PropertyFilter {
	result := make(map[string][]domain.PropertyFilter)
	for _, input := range inputs {
		if input == nil {
			continue
		}
		alias := strings.TrimSpace(input.Alias)
		field := strings.TrimSpace(input.Field)
		if alias == "" || field == "" {
			continue
		}
		filter := domain.PropertyFilter{Key: field, Exists: input.Exists}
		if input.Value != nil {
			filter.Value = *input.Value
		}
		if len(input.InArray) > 0 {
			filter.InArray = append([]string(nil), input.InArray...)
		}
		result[alias] = append(result[alias], filter)
	}
	return result
}

func cloneTransformation(t domain.EntityTransformation) domain.EntityTransformation {
	clone := t
	if len(t.Nodes) == 0 {
		clone.Nodes = []domain.EntityTransformationNode{}
		return clone
	}
	clone.Nodes = make([]domain.EntityTransformationNode, len(t.Nodes))
	for i, node := range t.Nodes {
		clone.Nodes[i] = cloneTransformationNode(node)
	}
	return clone
}

func cloneTransformationNode(node domain.EntityTransformationNode) domain.EntityTransformationNode {
	clone := node
	clone.Inputs = append([]uuid.UUID(nil), node.Inputs...)
	if node.Load != nil {
		copyLoad := *node.Load
		copyLoad.Filters = clonePropertyFilters(node.Load.Filters)
		clone.Load = &copyLoad
	}
	if node.Filter != nil {
		copyFilter := *node.Filter
		copyFilter.Filters = clonePropertyFilters(node.Filter.Filters)
		clone.Filter = &copyFilter
	}
	if node.Project != nil {
		copyProject := *node.Project
		copyProject.Fields = append([]string(nil), node.Project.Fields...)
		clone.Project = &copyProject
	}
	if node.Join != nil {
		copyJoin := *node.Join
		clone.Join = &copyJoin
	}
	if node.Materialize != nil {
		copyMaterialize := *node.Materialize
		copyMaterialize.Outputs = make([]domain.EntityTransformationMaterializeOutput, len(node.Materialize.Outputs))
		for i, output := range node.Materialize.Outputs {
			copyOutput := output
			copyOutput.Fields = make([]domain.EntityTransformationMaterializeFieldMapping, len(output.Fields))
			copy(copyOutput.Fields, output.Fields)
			copyMaterialize.Outputs[i] = copyOutput
		}
		clone.Materialize = &copyMaterialize
	}
	if node.Sort != nil {
		copySort := *node.Sort
		clone.Sort = &copySort
	}
	if node.Paginate != nil {
		copyPaginate := *node.Paginate
		if node.Paginate.Limit != nil {
			limitCopy := *node.Paginate.Limit
			copyPaginate.Limit = &limitCopy
		}
		if node.Paginate.Offset != nil {
			offsetCopy := *node.Paginate.Offset
			copyPaginate.Offset = &offsetCopy
		}
		clone.Paginate = &copyPaginate
	}
	return clone
}

func clonePropertyFilters(filters []domain.PropertyFilter) []domain.PropertyFilter {
	if len(filters) == 0 {
		return []domain.PropertyFilter{}
	}
	cloned := make([]domain.PropertyFilter, len(filters))
	for i, filter := range filters {
		cloned[i] = filter
		if len(filter.InArray) > 0 {
			cloned[i].InArray = append([]string(nil), filter.InArray...)
		}
		if filter.Exists != nil {
			existsCopy := *filter.Exists
			cloned[i].Exists = &existsCopy
		}
	}
	return cloned
}

func optionalIntPointer(value int) *int {
	if value <= 0 {
		return nil
	}
	copy := value
	return &copy
}

func finalNodeID(transformation domain.EntityTransformation) (uuid.UUID, error) {
	sorted, err := transformation.TopologicallySortedNodes()
	if err != nil {
		return uuid.Nil, err
	}
	if len(sorted) == 0 {
		return uuid.Nil, fmt.Errorf("transformation %s has no nodes", transformation.ID)
	}
	return sorted[len(sorted)-1].ID, nil
}

func buildExecutionRows(records []domain.EntityTransformationRecord, columns []*graph.TransformationExecutionColumn) []*graph.TransformationExecutionRow {
	rows := make([]*graph.TransformationExecutionRow, 0, len(records))
	for _, record := range records {
		values := make([]*graph.TransformationExecutionValue, 0, len(columns))
		for _, column := range columns {
			var valuePtr *string
			if entity := record.Entities[column.Alias]; entity != nil {
				if raw, ok := entity.Properties[column.Field]; ok {
					str := fmt.Sprintf("%v", raw)
					valuePtr = &str
				}
			}
			values = append(values, &graph.TransformationExecutionValue{
				ColumnKey: column.Key,
				Value:     valuePtr,
			})
		}
		rows = append(rows, &graph.TransformationExecutionRow{Values: values})
	}
	return rows
}

func columnKey(alias, field string) string {
	if alias == "" {
		return field
	}
	if field == "" {
		return alias
	}
	return fmt.Sprintf("%s.%s", alias, field)
}

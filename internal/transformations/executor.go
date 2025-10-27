package transformations

import (
	"context"
	"fmt"
	"strings"

	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
)

// EntityRepository defines the subset of entity storage used by the executor.
type EntityRepository interface {
	List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, sort *domain.EntitySort, limit int, offset int) ([]domain.Entity, int, error)
}

// SchemaProvider exposes schema lookup capabilities used by the executor.
type SchemaProvider interface {
	GetByName(ctx context.Context, organizationID uuid.UUID, entityType string) (domain.EntitySchema, error)
}

// Executor walks a transformation DAG and produces execution results.
type Executor struct {
	entityRepo     EntityRepository
	schemaProvider SchemaProvider
}

// NewExecutor constructs a transformation executor.
func NewExecutor(entityRepo EntityRepository, schemaProvider SchemaProvider) *Executor {
	return &Executor{entityRepo: entityRepo, schemaProvider: schemaProvider}
}

// Execute runs the transformation graph and returns paginated results.
func (e *Executor) Execute(ctx context.Context, transformation domain.EntityTransformation, opts domain.EntityTransformationExecutionOptions) (domain.EntityTransformationExecutionResult, error) {
	sorted, err := transformation.TopologicallySortedNodes()
	if err != nil {
		return domain.EntityTransformationExecutionResult{}, err
	}

	results := make(map[uuid.UUID][]domain.EntityTransformationRecord)
	schemaCache := make(map[string]schemaCacheEntry)

	for _, node := range sorted {
		nodeResults, err := e.executeNode(ctx, transformation, node, results, schemaCache)
		if err != nil {
			return domain.EntityTransformationExecutionResult{}, fmt.Errorf("execute node %s: %w", node.ID, err)
		}
		results[node.ID] = nodeResults
	}

	if len(sorted) == 0 {
		return domain.EntityTransformationExecutionResult{Records: []domain.EntityTransformationRecord{}, TotalCount: 0}, nil
	}

	finalNode := sorted[len(sorted)-1]
	finalRecords := append([]domain.EntityTransformationRecord(nil), results[finalNode.ID]...)
	totalCount := len(finalRecords)

	if opts.Offset > 0 || opts.Limit > 0 {
		finalRecords = domain.PaginateRecords(finalRecords, opts.Limit, opts.Offset)
	}

	return domain.EntityTransformationExecutionResult{Records: finalRecords, TotalCount: totalCount}, nil
}

func (e *Executor) executeNode(
	ctx context.Context,
	transformation domain.EntityTransformation,
	node domain.EntityTransformationNode,
	cache map[uuid.UUID][]domain.EntityTransformationRecord,
	schemaCache map[string]schemaCacheEntry,
) ([]domain.EntityTransformationRecord, error) {
	switch node.Type {
	case domain.TransformationNodeLoad:
		return e.executeLoad(ctx, transformation, node)
	case domain.TransformationNodeFilter:
		return e.executeFilter(node, cache)
	case domain.TransformationNodeProject:
		return e.executeProject(node, cache)
	case domain.TransformationNodeJoin, domain.TransformationNodeLeftJoin, domain.TransformationNodeAntiJoin:
		return e.executeJoin(ctx, transformation.OrganizationID, node, cache, schemaCache)
	case domain.TransformationNodeUnion:
		return e.executeUnion(node, cache)
	case domain.TransformationNodeSort:
		return e.executeSort(node, cache)
	case domain.TransformationNodePaginate:
		return e.executePaginate(node, cache)
	default:
		return nil, fmt.Errorf("unsupported node type %s", node.Type)
	}
}

func (e *Executor) executeLoad(ctx context.Context, transformation domain.EntityTransformation, node domain.EntityTransformationNode) ([]domain.EntityTransformationRecord, error) {
	if node.Load == nil {
		return nil, fmt.Errorf("load node missing configuration")
	}
	limit := 1000
	filter := &domain.EntityFilter{EntityType: node.Load.EntityType, PropertyFilters: node.Load.Filters}
	entities, _, err := e.entityRepo.List(ctx, transformation.OrganizationID, filter, nil, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("load entities: %w", err)
	}
	records := make([]domain.EntityTransformationRecord, 0, len(entities))
	for i := range entities {
		entity := entities[i]
		if !domain.ApplyPropertyFilters(&entity, node.Load.Filters) {
			continue
		}
		entityCopy := entity
		record := domain.EntityTransformationRecord{Entities: map[string]*domain.Entity{node.Load.Alias: &entityCopy}}
		records = append(records, record)
	}
	return records, nil
}

func (e *Executor) executeFilter(node domain.EntityTransformationNode, cache map[uuid.UUID][]domain.EntityTransformationRecord) ([]domain.EntityTransformationRecord, error) {
	if len(node.Inputs) != 1 {
		return nil, fmt.Errorf("filter node requires exactly one input")
	}
	if node.Filter == nil {
		return nil, fmt.Errorf("filter node missing configuration")
	}
	inputRecords, ok := cache[node.Inputs[0]]
	if !ok {
		return nil, fmt.Errorf("filter input not found")
	}
	filterAlias, err := resolveFilterAlias(inputRecords, node.Filter.Alias)
	if err != nil {
		return nil, err
	}
	var filtered []domain.EntityTransformationRecord
	for _, record := range inputRecords {
		entity := record.Entities[filterAlias]
		if domain.ApplyPropertyFilters(entity, node.Filter.Filters) {
			filtered = append(filtered, record.Clone())
		}
	}
	return filtered, nil
}

func (e *Executor) executeProject(node domain.EntityTransformationNode, cache map[uuid.UUID][]domain.EntityTransformationRecord) ([]domain.EntityTransformationRecord, error) {
	if len(node.Inputs) != 1 {
		return nil, fmt.Errorf("project node requires exactly one input")
	}
	if node.Project == nil {
		return nil, fmt.Errorf("project node missing configuration")
	}
	inputRecords, ok := cache[node.Inputs[0]]
	if !ok {
		return nil, fmt.Errorf("project input not found")
	}
	projected := make([]domain.EntityTransformationRecord, 0, len(inputRecords))
	for _, record := range inputRecords {
		clone := record.Clone()
		if len(clone.Entities) == 0 {
			projected = append(projected, clone)
			continue
		}

		targetAlias, sourceAlias, err := resolveProjectAliases(clone.Entities, node.Project.Alias)
		if err != nil {
			return nil, err
		}

		projectedEntity := domain.ProjectEntity(clone.Entities[sourceAlias], node.Project.Fields)
		if sourceAlias != targetAlias {
			delete(clone.Entities, sourceAlias)
		}
		clone.Entities[targetAlias] = projectedEntity
		projected = append(projected, clone)
	}
	return projected, nil
}

func (e *Executor) executeJoin(
	ctx context.Context,
	organizationID uuid.UUID,
	node domain.EntityTransformationNode,
	cache map[uuid.UUID][]domain.EntityTransformationRecord,
	schemaCache map[string]schemaCacheEntry,
) ([]domain.EntityTransformationRecord, error) {
	if len(node.Inputs) != 2 {
		return nil, fmt.Errorf("join node requires two inputs")
	}
	if node.Join == nil {
		return nil, fmt.Errorf("join node missing configuration")
	}
	leftRecords, ok := cache[node.Inputs[0]]
	if !ok {
		return nil, fmt.Errorf("join left input missing")
	}
	rightRecords, ok := cache[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("join right input missing")
	}

	literalRightIndex := make(map[string][]int)
	idRightIndex := make(map[string][]int)
	for idx, record := range rightRecords {
		entity := record.Entities[node.Join.RightAlias]
		if entity == nil {
			continue
		}
		key := fmt.Sprintf("%v", entity.Properties[node.Join.OnField])
		literalRightIndex[key] = append(literalRightIndex[key], idx)
		idRightIndex[entity.ID.String()] = append(idRightIndex[entity.ID.String()], idx)
	}

	var referenceRightIndex map[string][]int
	var referenceIndexAvailable bool
	var referenceIndexBuilt bool
	leftFieldCache := make(map[string]*domain.FieldDefinition)

	var results []domain.EntityTransformationRecord
	for _, leftRecord := range leftRecords {
		leftEntity := leftRecord.Entities[node.Join.LeftAlias]
		if leftEntity == nil {
			continue
		}
		matches := []int{}
		useSchemaStrategy := false

		fieldDef, fieldFound := leftFieldCache[leftEntity.EntityType]
		if !fieldFound {
			schema, schemaErr := e.getSchema(ctx, organizationID, leftEntity.EntityType, schemaCache)
			if schemaErr == nil && schema != nil {
				if field := schemaFieldByName(schema, node.Join.OnField); field != nil {
					copyField := *field
					fieldDef = &copyField
				}
			}
			leftFieldCache[leftEntity.EntityType] = fieldDef
		}

		if fieldDef != nil {
			switch fieldDef.Type {
			case domain.FieldTypeEntityReference, domain.FieldTypeEntityReferenceArray:
				useSchemaStrategy = true
				identifiers := normalizeUUIDStringSlice(leftEntity.Properties[node.Join.OnField])
				if len(identifiers) > 0 {
					for _, value := range identifiers {
						matches = append(matches, idRightIndex[value]...)
					}
				}
			case domain.FieldTypeReference:
				values := normalizeStringSlice(leftEntity.Properties[node.Join.OnField])
				if len(values) == 0 {
					useSchemaStrategy = true
				} else {
					if !referenceIndexBuilt {
						referenceRightIndex, referenceIndexAvailable = e.buildReferenceIndex(ctx, organizationID, node.Join.RightAlias, rightRecords, schemaCache)
						referenceIndexBuilt = true
					}
					if referenceIndexAvailable {
						useSchemaStrategy = true
						referenceEntityType := fieldDef.ReferenceEntityType
						for _, value := range values {
							indices := referenceRightIndex[value]
							if referenceEntityType == "" {
								matches = append(matches, indices...)
								continue
							}
							for _, idx := range indices {
								entity := rightRecords[idx].Entities[node.Join.RightAlias]
								if entity != nil && entity.EntityType == referenceEntityType {
									matches = append(matches, idx)
								}
							}
						}
					}
				}
			}
		}

		if !useSchemaStrategy {
			key := fmt.Sprintf("%v", leftEntity.Properties[node.Join.OnField])
			matches = append(matches, literalRightIndex[key]...)
		}

		deduped := make([]int, 0, len(matches))
		seen := make(map[int]struct{}, len(matches))
		for _, idx := range matches {
			if _, ok := seen[idx]; ok {
				continue
			}
			seen[idx] = struct{}{}
			deduped = append(deduped, idx)
		}

		switch node.Type {
		case domain.TransformationNodeJoin:
			for _, idx := range deduped {
				combined := mergeRecords(leftRecord, rightRecords[idx])
				results = append(results, combined)
			}
		case domain.TransformationNodeLeftJoin:
			if len(deduped) == 0 {
				combined := leftRecord.Clone()
				combined.Entities[node.Join.RightAlias] = nil
				results = append(results, combined)
				continue
			}
			for _, idx := range deduped {
				combined := mergeRecords(leftRecord, rightRecords[idx])
				results = append(results, combined)
			}
		case domain.TransformationNodeAntiJoin:
			if len(deduped) == 0 {
				results = append(results, leftRecord.Clone())
			}
		}
	}
	return results, nil
}

type schemaCacheEntry struct {
	schema *domain.EntitySchema
	err    error
}

func (e *Executor) getSchema(ctx context.Context, organizationID uuid.UUID, entityType string, cache map[string]schemaCacheEntry) (*domain.EntitySchema, error) {
	if entityType == "" {
		return nil, nil
	}
	if entry, ok := cache[entityType]; ok {
		return entry.schema, entry.err
	}
	if e.schemaProvider == nil {
		cache[entityType] = schemaCacheEntry{}
		return nil, nil
	}
	schema, err := e.schemaProvider.GetByName(ctx, organizationID, entityType)
	if err != nil {
		cache[entityType] = schemaCacheEntry{err: err}
		return nil, err
	}
	schemaCopy := schema
	cache[entityType] = schemaCacheEntry{schema: &schemaCopy}
	return &schemaCopy, nil
}

func schemaFieldByName(schema *domain.EntitySchema, fieldName string) *domain.FieldDefinition {
	if schema == nil {
		return nil
	}
	for _, field := range schema.Fields {
		if field.Name == fieldName {
			copyField := field
			return &copyField
		}
	}
	return nil
}

func (e *Executor) buildReferenceIndex(
	ctx context.Context,
	organizationID uuid.UUID,
	alias string,
	rightRecords []domain.EntityTransformationRecord,
	cache map[string]schemaCacheEntry,
) (map[string][]int, bool) {
	if e.schemaProvider == nil {
		return nil, false
	}

	index := make(map[string][]int)
	canonicalFieldCache := make(map[string]string)
	foundCanonical := false

	for idx, record := range rightRecords {
		entity := record.Entities[alias]
		if entity == nil {
			continue
		}

		canonicalField, cached := canonicalFieldCache[entity.EntityType]
		if !cached {
			schema, err := e.getSchema(ctx, organizationID, entity.EntityType, cache)
			if err != nil || schema == nil {
				canonicalFieldCache[entity.EntityType] = ""
				continue
			}
			name, ok := domain.NewReferenceFieldSet(schema.Fields).CanonicalName()
			if !ok {
				canonicalFieldCache[entity.EntityType] = ""
				continue
			}
			canonicalField = name
			canonicalFieldCache[entity.EntityType] = canonicalField
		}

		if canonicalField == "" {
			continue
		}
		foundCanonical = true

		values := normalizeStringSlice(entity.Properties[canonicalField])
		for _, value := range values {
			index[value] = append(index[value], idx)
		}
	}

	if !foundCanonical {
		return nil, false
	}
	return index, true
}

func (e *Executor) executeUnion(node domain.EntityTransformationNode, cache map[uuid.UUID][]domain.EntityTransformationRecord) ([]domain.EntityTransformationRecord, error) {
	if len(node.Inputs) == 0 {
		return nil, fmt.Errorf("union node requires at least one input")
	}
	var results []domain.EntityTransformationRecord
	for _, input := range node.Inputs {
		inputRecords, ok := cache[input]
		if !ok {
			return nil, fmt.Errorf("union input missing")
		}
		for _, record := range inputRecords {
			cloned := record.Clone()
			if err := applyUnionAlias(&cloned, node.Union); err != nil {
				return nil, err
			}
			results = append(results, cloned)
		}
	}
	return results, nil
}

func applyUnionAlias(record *domain.EntityTransformationRecord, config *domain.EntityTransformationUnionConfig) error {
	if record == nil || config == nil {
		return nil
	}
	alias := strings.TrimSpace(config.Alias)
	if alias == "" {
		return nil
	}
	if record.Entities == nil {
		record.Entities = map[string]*domain.Entity{}
	}
	if _, exists := record.Entities[alias]; exists {
		return nil
	}
	if len(record.Entities) == 0 {
		record.Entities[alias] = nil
		return nil
	}
	existingAlias, ok := singleAliasFromEntities(record.Entities)
	if !ok {
		return fmt.Errorf("union alias %q requires records with a single entity alias", alias)
	}
	entity := record.Entities[existingAlias]
	delete(record.Entities, existingAlias)
	record.Entities[alias] = entity
	return nil
}

func (e *Executor) executeSort(node domain.EntityTransformationNode, cache map[uuid.UUID][]domain.EntityTransformationRecord) ([]domain.EntityTransformationRecord, error) {
	if len(node.Inputs) != 1 {
		return nil, fmt.Errorf("sort node requires one input")
	}
	if node.Sort == nil {
		return nil, fmt.Errorf("sort node missing configuration")
	}
	inputRecords, ok := cache[node.Inputs[0]]
	if !ok {
		return nil, fmt.Errorf("sort input missing")
	}
	cloned := cloneRecords(inputRecords)
	if len(cloned) == 0 {
		return cloned, nil
	}
	sortAlias, err := resolveSortAlias(cloned, node.Sort.Alias)
	if err != nil {
		return nil, err
	}
	if sortAlias == "" {
		return cloned, nil
	}
	domain.SortRecords(cloned, sortAlias, node.Sort.Field, node.Sort.Direction)
	return cloned, nil
}

func (e *Executor) executePaginate(node domain.EntityTransformationNode, cache map[uuid.UUID][]domain.EntityTransformationRecord) ([]domain.EntityTransformationRecord, error) {
	if len(node.Inputs) != 1 {
		return nil, fmt.Errorf("paginate node requires one input")
	}
	if node.Paginate == nil {
		return nil, fmt.Errorf("paginate node missing configuration")
	}
	inputRecords, ok := cache[node.Inputs[0]]
	if !ok {
		return nil, fmt.Errorf("paginate input missing")
	}
	limit := 0
	offset := 0
	if node.Paginate.Limit != nil {
		limit = *node.Paginate.Limit
	}
	if node.Paginate.Offset != nil {
		offset = *node.Paginate.Offset
	}
	cloned := cloneRecords(inputRecords)
	return domain.PaginateRecords(cloned, limit, offset), nil
}

func mergeRecords(left domain.EntityTransformationRecord, right domain.EntityTransformationRecord) domain.EntityTransformationRecord {
	merged := left.Clone()
	if merged.Entities == nil {
		merged.Entities = map[string]*domain.Entity{}
	}
	for alias, entity := range right.Entities {
		if entity == nil {
			merged.Entities[alias] = nil
			continue
		}
		copyEntity := entity.WithProperties(entity.Properties)
		merged.Entities[alias] = &copyEntity
	}
	return merged
}

func cloneRecords(records []domain.EntityTransformationRecord) []domain.EntityTransformationRecord {
	cloned := make([]domain.EntityTransformationRecord, 0, len(records))
	for _, record := range records {
		cloned = append(cloned, record.Clone())
	}
	return cloned
}

func resolveProjectAliases(entities map[string]*domain.Entity, desiredAlias string) (targetAlias string, sourceAlias string, err error) {
	if desiredAlias != "" {
		if _, ok := entities[desiredAlias]; ok {
			return desiredAlias, desiredAlias, nil
		}
	}

	fallbackAlias, ok := singleAliasFromEntities(entities)
	if !ok {
		if desiredAlias == "" {
			return "", "", fmt.Errorf("project node requires an alias when multiple entities are present")
		}
		return "", "", fmt.Errorf("project alias %q not found in record", desiredAlias)
	}

	sourceAlias = fallbackAlias
	targetAlias = desiredAlias
	if targetAlias == "" {
		targetAlias = fallbackAlias
	}
	return targetAlias, sourceAlias, nil
}

func resolveSortAlias(records []domain.EntityTransformationRecord, desiredAlias string) (string, error) {
	if desiredAlias != "" {
		for _, record := range records {
			if record.Entities == nil {
				continue
			}
			if _, ok := record.Entities[desiredAlias]; ok {
				return desiredAlias, nil
			}
		}
	}

	fallbackAlias, ok := singleAliasAcrossRecords(records)
	if !ok {
		if desiredAlias == "" {
			if len(records) == 0 {
				return "", nil
			}
			return "", fmt.Errorf("sort node requires an alias when multiple entities are present")
		}
		return "", fmt.Errorf("sort alias %q not found in records", desiredAlias)
	}

	if desiredAlias != "" {
		return fallbackAlias, nil
	}
	return fallbackAlias, nil
}

func resolveFilterAlias(records []domain.EntityTransformationRecord, desiredAlias string) (string, error) {
	if desiredAlias != "" {
		for _, record := range records {
			if record.Entities == nil {
				continue
			}
			if _, ok := record.Entities[desiredAlias]; ok {
				return desiredAlias, nil
			}
		}
	}

	fallbackAlias, ok := singleAliasAcrossRecords(records)
	if !ok {
		if desiredAlias == "" {
			if len(records) == 0 {
				return "", nil
			}
			return "", fmt.Errorf("filter node requires an alias when multiple entities are present")
		}
		return "", fmt.Errorf("filter alias %q not found in records", desiredAlias)
	}

	if desiredAlias != "" {
		return fallbackAlias, nil
	}
	return fallbackAlias, nil
}

func singleAliasFromEntities(entities map[string]*domain.Entity) (string, bool) {
	alias := ""
	count := 0
	for key := range entities {
		alias = key
		count++
		if count > 1 {
			return "", false
		}
	}
	if count == 1 {
		return alias, true
	}
	return "", false
}

func singleAliasAcrossRecords(records []domain.EntityTransformationRecord) (string, bool) {
	alias := ""
	for _, record := range records {
		if len(record.Entities) == 0 {
			continue
		}
		candidate, ok := singleAliasFromEntities(record.Entities)
		if !ok {
			return "", false
		}
		if alias == "" {
			alias = candidate
			continue
		}
		if alias != candidate {
			return "", false
		}
	}
	if alias == "" {
		return "", false
	}
	return alias, true
}

func normalizeStringSlice(value any) []string {
	seen := make(map[string]struct{})
	var result []string

	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		result = append(result, candidate)
	}

	switch v := value.(type) {
	case nil:
	case string:
		add(v)
	case *string:
		if v != nil {
			add(*v)
		}
	case []string:
		for _, item := range v {
			add(item)
		}
	case []*string:
		for _, item := range v {
			if item != nil {
				add(*item)
			}
		}
	case []any:
		for _, item := range v {
			for _, normalized := range normalizeStringSlice(item) {
				add(normalized)
			}
		}
	case fmt.Stringer:
		add(v.String())
	default:
		add(fmt.Sprintf("%v", value))
	}

	return result
}

func normalizeUUIDStringSlice(value any) []string {
	seen := make(map[string]struct{})
	var result []string

	addParsed := func(candidate string) {
		if candidate == "" {
			return
		}
		id, err := uuid.Parse(candidate)
		if err != nil {
			return
		}
		normalized := id.String()
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	appendNormalized := func(values []string) {
		for _, value := range values {
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}

	switch v := value.(type) {
	case nil:
	case uuid.UUID:
		if v != uuid.Nil {
			addParsed(v.String())
		}
	case *uuid.UUID:
		if v != nil && *v != uuid.Nil {
			addParsed(v.String())
		}
	case []uuid.UUID:
		for _, item := range v {
			if item != uuid.Nil {
				addParsed(item.String())
			}
		}
	case []*uuid.UUID:
		for _, item := range v {
			if item != nil && *item != uuid.Nil {
				addParsed(item.String())
			}
		}
	case string:
		addParsed(v)
	case *string:
		if v != nil {
			addParsed(*v)
		}
	case []string:
		for _, item := range v {
			addParsed(item)
		}
	case []any:
		for _, item := range v {
			appendNormalized(normalizeUUIDStringSlice(item))
		}
	default:
		addParsed(fmt.Sprintf("%v", value))
	}

	return result
}

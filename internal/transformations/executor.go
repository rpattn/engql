package transformations

import (
	"context"
	"fmt"
	"sort"

	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
)

const (
	anyAliasSentinel     = "__ANY_ALIAS__"
	defaultLoadBatchSize = 1000
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

type pageRequest struct {
	limit  int
	offset int
}

type pageLimiter struct {
	limit  int
	offset int
	total  int
	max    int
}

func newPageLimiter(req pageRequest) pageLimiter {
	limiter := pageLimiter{limit: req.limit, offset: req.offset}
	if limiter.limit < 0 {
		limiter.limit = 0
	}
	if limiter.offset < 0 {
		limiter.offset = 0
	}
	if limiter.limit > 0 {
		limiter.max = limiter.offset + limiter.limit
	}
	return limiter
}

func (p *pageLimiter) Include() bool {
	p.total++
	if p.limit == 0 {
		return true
	}
	return p.total <= p.max
}

func (p *pageLimiter) Total() int {
	return p.total
}

type nodeResult struct {
	records []domain.EntityTransformationRecord
	total   int
}

func appendPageRequest(existing pageRequest, count int, incoming pageRequest) (pageRequest, int) {
	if count == 0 {
		return incoming, 1
	}
	if existing.limit == 0 || incoming.limit == 0 {
		return pageRequest{}, count + 1
	}
	existingTotal := existing.offset + existing.limit
	incomingTotal := incoming.offset + incoming.limit
	if existingTotal < incomingTotal {
		existingTotal = incomingTotal
	}
	return pageRequest{limit: existingTotal}, count + 1
}

func requestTotal(req pageRequest) int {
	if req.limit == 0 {
		return 0
	}
	return req.offset + req.limit
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

	results := make(map[uuid.UUID]nodeResult)
	schemaCache := make(map[string]schemaCacheEntry)

	nodeRequests := make(map[uuid.UUID]pageRequest)
	requestCounts := make(map[uuid.UUID]int)
	if len(sorted) > 0 {
		finalNode := sorted[len(sorted)-1]
		nodeRequests[finalNode.ID] = pageRequest{limit: opts.Limit, offset: opts.Offset}
		requestCounts[finalNode.ID] = 1
	}

	for i := len(sorted) - 1; i >= 0; i-- {
		node := sorted[i]
		req := nodeRequests[node.ID]

		if node.Type == domain.TransformationNodePaginate && node.Paginate != nil {
			nodeLimit := 0
			nodeOffset := 0
			if node.Paginate.Limit != nil {
				nodeLimit = *node.Paginate.Limit
			}
			if node.Paginate.Offset != nil {
				nodeOffset = *node.Paginate.Offset
			}

			totalNeeded := requestTotal(req)
			if nodeLimit > 0 && (totalNeeded == 0 || totalNeeded > nodeLimit) {
				totalNeeded = nodeLimit
			}

			inputReq := pageRequest{}
			if totalNeeded > 0 {
				inputReq.limit = totalNeeded + nodeOffset
			} else if nodeLimit > 0 {
				inputReq.limit = nodeLimit + nodeOffset
			}

			for _, input := range node.Inputs {
				existing := nodeRequests[input]
				count := requestCounts[input]
				combined, combinedCount := appendPageRequest(existing, count, inputReq)
				nodeRequests[input] = combined
				requestCounts[input] = combinedCount
			}
			continue
		}

		for _, input := range node.Inputs {
			incoming := pageRequest{}
			totalNeeded := requestTotal(req)
			if totalNeeded > 0 {
				incoming.limit = totalNeeded
			}
			existing := nodeRequests[input]
			count := requestCounts[input]
			combined, combinedCount := appendPageRequest(existing, count, incoming)
			nodeRequests[input] = combined
			requestCounts[input] = combinedCount
		}
	}

	for _, node := range sorted {
		req := nodeRequests[node.ID]
		nodeResults, err := e.executeNode(ctx, transformation, node, req, results, schemaCache)
		if err != nil {
			return domain.EntityTransformationExecutionResult{}, fmt.Errorf("execute node %s: %w", node.ID, err)
		}
		results[node.ID] = nodeResults
	}

	if len(sorted) == 0 {
		return domain.EntityTransformationExecutionResult{Records: []domain.EntityTransformationRecord{}, TotalCount: 0}, nil
	}

	finalNode := sorted[len(sorted)-1]
	finalResult := results[finalNode.ID]
	finalRecords := finalResult.records
	if (opts.Limit > 0 || opts.Offset > 0) && finalNode.Type != domain.TransformationNodePaginate {
		finalRecords = domain.PaginateRecords(finalRecords, opts.Limit, opts.Offset)
	}
	finalRecords = append([]domain.EntityTransformationRecord(nil), finalRecords...)

	return domain.EntityTransformationExecutionResult{Records: finalRecords, TotalCount: finalResult.total}, nil
}

func (e *Executor) executeNode(
	ctx context.Context,
	transformation domain.EntityTransformation,
	node domain.EntityTransformationNode,
	req pageRequest,
	cache map[uuid.UUID]nodeResult,
	schemaCache map[string]schemaCacheEntry,
) (nodeResult, error) {
	switch node.Type {
	case domain.TransformationNodeLoad:
		return e.executeLoad(ctx, transformation, node, req)
	case domain.TransformationNodeFilter:
		return e.executeFilter(node, cache, req)
	case domain.TransformationNodeProject:
		return e.executeProject(node, cache, req)
	case domain.TransformationNodeJoin, domain.TransformationNodeLeftJoin, domain.TransformationNodeAntiJoin:
		return e.executeJoin(ctx, transformation.OrganizationID, node, cache, schemaCache, req)
	case domain.TransformationNodeUnion:
		return e.executeUnion(node, cache, req)
	case domain.TransformationNodeMaterialize:
		return e.executeMaterialize(node, cache, req)
	case domain.TransformationNodeSort:
		return e.executeSort(node, cache, req)
	case domain.TransformationNodePaginate:
		return e.executePaginate(node, cache, req)
	default:
		return nodeResult{}, fmt.Errorf("unsupported node type %s", node.Type)
	}
}

func (e *Executor) executeLoad(ctx context.Context, transformation domain.EntityTransformation, node domain.EntityTransformationNode, req pageRequest) (nodeResult, error) {
	if node.Load == nil {
		return nodeResult{}, fmt.Errorf("load node missing configuration")
	}
	filter := &domain.EntityFilter{EntityType: node.Load.EntityType, PropertyFilters: node.Load.Filters}
	limiter := newPageLimiter(req)
	capacity := limiter.max
	if capacity == 0 {
		capacity = defaultLoadBatchSize
	}
	records := make([]domain.EntityTransformationRecord, 0, capacity)
	totalCount := 0
	repoOffset := 0

	for {
		if limiter.limit > 0 && limiter.Total() >= limiter.max {
			break
		}

		batchLimit := defaultLoadBatchSize
		if limiter.limit > 0 {
			remaining := limiter.max - limiter.Total()
			if remaining <= 0 {
				break
			}
			if remaining < batchLimit {
				batchLimit = remaining
			}
		}
		if batchLimit <= 0 {
			batchLimit = defaultLoadBatchSize
		}

		entities, batchTotal, err := e.entityRepo.List(ctx, transformation.OrganizationID, filter, nil, batchLimit, repoOffset)
		if err != nil {
			return nodeResult{}, fmt.Errorf("load entities: %w", err)
		}
		if totalCount == 0 && batchTotal > 0 {
			totalCount = batchTotal
		}
		if len(entities) == 0 {
			break
		}
		repoOffset += len(entities)

		for i := range entities {
			entity := entities[i]
			if !domain.ApplyPropertyFilters(&entity, node.Load.Filters) {
				continue
			}
			if limiter.Include() {
				entityCopy := entity
				record := domain.EntityTransformationRecord{Entities: map[string]*domain.Entity{node.Load.Alias: &entityCopy}}
				records = append(records, record)
			}
			if limiter.limit > 0 && limiter.Total() >= limiter.max {
				break
			}
		}

		if limiter.limit > 0 && limiter.Total() >= limiter.max {
			break
		}
		if len(entities) < batchLimit {
			break
		}
	}

	if totalCount == 0 {
		totalCount = limiter.Total()
	}
	return nodeResult{records: records, total: totalCount}, nil
}

func (e *Executor) executeFilter(node domain.EntityTransformationNode, cache map[uuid.UUID]nodeResult, req pageRequest) (nodeResult, error) {
	if len(node.Inputs) != 1 {
		return nodeResult{}, fmt.Errorf("filter node requires exactly one input")
	}
	if node.Filter == nil {
		return nodeResult{}, fmt.Errorf("filter node missing configuration")
	}
	inputResult, ok := cache[node.Inputs[0]]
	if !ok {
		return nodeResult{}, fmt.Errorf("filter input not found")
	}
	inputRecords := inputResult.records
	filterAlias, err := resolveFilterAlias(inputRecords, node.Filter.Alias)
	if err != nil {
		return nodeResult{}, err
	}
	limiter := newPageLimiter(req)
	capacity := len(inputRecords)
	if limiter.limit > 0 && limiter.max > 0 && limiter.max < capacity {
		capacity = limiter.max
	}
	filtered := make([]domain.EntityTransformationRecord, 0, capacity)
	for _, record := range inputRecords {
		entity := record.Entities[filterAlias]
		if domain.ApplyPropertyFilters(entity, node.Filter.Filters) {
			if limiter.Include() {
				filtered = append(filtered, record.Clone())
			}
		}
	}
	return nodeResult{records: filtered, total: limiter.Total()}, nil
}

func (e *Executor) executeProject(node domain.EntityTransformationNode, cache map[uuid.UUID]nodeResult, req pageRequest) (nodeResult, error) {
	if len(node.Inputs) != 1 {
		return nodeResult{}, fmt.Errorf("project node requires exactly one input")
	}
	if node.Project == nil {
		return nodeResult{}, fmt.Errorf("project node missing configuration")
	}
	inputResult, ok := cache[node.Inputs[0]]
	if !ok {
		return nodeResult{}, fmt.Errorf("project input not found")
	}
	inputRecords := inputResult.records
	limiter := newPageLimiter(req)
	capacity := len(inputRecords)
	if limiter.limit > 0 && limiter.max > 0 && limiter.max < capacity {
		capacity = limiter.max
	}
	projected := make([]domain.EntityTransformationRecord, 0, capacity)
	for _, record := range inputRecords {
		clone := record.Clone()
		if len(clone.Entities) != 0 {
			targetAlias, sourceAlias, err := resolveProjectAliases(clone.Entities, node.Project.Alias)
			if err != nil {
				return nodeResult{}, err
			}

			projectedEntity := domain.ProjectEntity(clone.Entities[sourceAlias], node.Project.Fields)
			if sourceAlias != targetAlias {
				delete(clone.Entities, sourceAlias)
			}
			clone.Entities[targetAlias] = projectedEntity
		}

		if limiter.Include() {
			projected = append(projected, clone)
		}
	}

	total := inputResult.total
	if total == 0 {
		total = limiter.Total()
	}
	return nodeResult{records: projected, total: total}, nil
}

func (e *Executor) executeMaterialize(node domain.EntityTransformationNode, cache map[uuid.UUID]nodeResult, req pageRequest) (nodeResult, error) {
	if len(node.Inputs) != 1 {
		return nodeResult{}, fmt.Errorf("materialize node requires exactly one input")
	}
	if node.Materialize == nil {
		return nodeResult{}, fmt.Errorf("materialize node missing configuration")
	}
	if len(node.Materialize.Outputs) == 0 {
		return nodeResult{}, fmt.Errorf("materialize node requires at least one output")
	}
	inputResult, ok := cache[node.Inputs[0]]
	if !ok {
		return nodeResult{}, fmt.Errorf("materialize input not found")
	}
	inputRecords := inputResult.records

	limiter := newPageLimiter(req)
	capacity := len(inputRecords)
	if limiter.limit > 0 && limiter.max > 0 && limiter.max < capacity {
		capacity = limiter.max
	}
	results := make([]domain.EntityTransformationRecord, 0, capacity)
	for _, record := range inputRecords {
		clone := record.Clone()
		materializedEntities := make(map[string]*domain.Entity, len(node.Materialize.Outputs))
		aliasOrder := sortedEntityAliases(record.Entities)

		for _, output := range node.Materialize.Outputs {
			if output.Alias == "" {
				return nodeResult{}, fmt.Errorf("materialize output alias is required")
			}

			entity, seededFromSource := seedMaterializedEntity(record, output, aliasOrder)
			for _, field := range output.Fields {
				if field.OutputField == "" {
					continue
				}

				aliases := resolveMaterializeAliases(field.SourceAlias, aliasOrder)
				for _, alias := range aliases {
					source := record.Entities[alias]
					value, ok := extractMaterializeValue(source, field.SourceField)
					if !ok {
						continue
					}
					if source != nil && !seededFromSource {
						adoptMaterializeMetadata(entity, source)
						seededFromSource = true
					}
					if entity.Properties == nil {
						entity.Properties = make(map[string]any)
					}
					entity.Properties[field.OutputField] = value
					break
				}
			}

			materializedEntities[output.Alias] = entity
		}

		clone.Entities = materializedEntities
		if limiter.Include() {
			results = append(results, clone)
		}
	}

	total := inputResult.total
	if total == 0 {
		total = limiter.Total()
	}
	return nodeResult{records: results, total: total}, nil
}

func seedMaterializedEntity(record domain.EntityTransformationRecord, output domain.EntityTransformationMaterializeOutput, aliasOrder []string) (*domain.Entity, bool) {
	for _, field := range output.Fields {
		switch field.SourceAlias {
		case "":
			continue
		case anyAliasSentinel:
			for _, alias := range aliasOrder {
				source := record.Entities[alias]
				if source == nil {
					continue
				}
				copy := *source
				copy.Properties = make(map[string]any, len(output.Fields))
				return &copy, true
			}
		default:
			source := record.Entities[field.SourceAlias]
			if source == nil {
				continue
			}
			copy := *source
			copy.Properties = make(map[string]any, len(output.Fields))
			return &copy, true
		}
	}

	return &domain.Entity{ID: uuid.New(), Properties: make(map[string]any, len(output.Fields))}, false
}

func resolveMaterializeAliases(sourceAlias string, aliasOrder []string) []string {
	switch sourceAlias {
	case "", anyAliasSentinel:
		return aliasOrder
	default:
		return []string{sourceAlias}
	}
}

func sortedEntityAliases(entities map[string]*domain.Entity) []string {
	if len(entities) == 0 {
		return nil
	}
	aliases := make([]string, 0, len(entities))
	for alias := range entities {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func adoptMaterializeMetadata(target *domain.Entity, source *domain.Entity) {
	if target == nil || source == nil {
		return
	}
	target.ID = source.ID
	target.OrganizationID = source.OrganizationID
	target.SchemaID = source.SchemaID
	target.EntityType = source.EntityType
	target.Path = source.Path
	target.Version = source.Version
	target.CreatedAt = source.CreatedAt
	target.UpdatedAt = source.UpdatedAt
}

func extractMaterializeValue(source *domain.Entity, field string) (any, bool) {
	if source == nil {
		return nil, false
	}

	switch field {
	case "id", "ID":
		if source.ID == uuid.Nil {
			return nil, false
		}
		return source.ID.String(), true
	case "organizationId", "organization_id":
		if source.OrganizationID == uuid.Nil {
			return nil, false
		}
		return source.OrganizationID.String(), true
	case "schemaId", "schema_id":
		if source.SchemaID == uuid.Nil {
			return nil, false
		}
		return source.SchemaID.String(), true
	case "entityType", "entity_type":
		if source.EntityType == "" {
			return nil, false
		}
		return source.EntityType, true
	case "path":
		if source.Path == "" {
			return nil, false
		}
		return source.Path, true
	case "version":
		if source.Version == 0 {
			return nil, false
		}
		return source.Version, true
	case "createdAt", "created_at":
		if source.CreatedAt.IsZero() {
			return nil, false
		}
		return source.CreatedAt, true
	case "updatedAt", "updated_at":
		if source.UpdatedAt.IsZero() {
			return nil, false
		}
		return source.UpdatedAt, true
	default:
		if source.Properties == nil {
			return nil, false
		}
		value, ok := source.Properties[field]
		return value, ok
	}
}

func (e *Executor) executeJoin(
	ctx context.Context,
	organizationID uuid.UUID,
	node domain.EntityTransformationNode,
	cache map[uuid.UUID]nodeResult,
	schemaCache map[string]schemaCacheEntry,
	req pageRequest,
) (nodeResult, error) {
	if len(node.Inputs) != 2 {
		return nodeResult{}, fmt.Errorf("join node requires two inputs")
	}
	if node.Join == nil {
		return nodeResult{}, fmt.Errorf("join node missing configuration")
	}
	leftResult, ok := cache[node.Inputs[0]]
	if !ok {
		return nodeResult{}, fmt.Errorf("join left input missing")
	}
	rightResult, ok := cache[node.Inputs[1]]
	if !ok {
		return nodeResult{}, fmt.Errorf("join right input missing")
	}
	leftRecords := leftResult.records
	rightRecords := rightResult.records

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

	limiter := newPageLimiter(req)
	capacity := len(leftRecords)
	if limiter.limit > 0 && limiter.max > 0 && limiter.max < capacity {
		capacity = limiter.max
	}
	results := make([]domain.EntityTransformationRecord, 0, capacity)
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
				if limiter.Include() {
					results = append(results, combined)
				}
			}
		case domain.TransformationNodeLeftJoin:
			if len(deduped) == 0 {
				combined := leftRecord.Clone()
				combined.Entities[node.Join.RightAlias] = nil
				if limiter.Include() {
					results = append(results, combined)
				}
				continue
			}
			for _, idx := range deduped {
				combined := mergeRecords(leftRecord, rightRecords[idx])
				if limiter.Include() {
					results = append(results, combined)
				}
			}
		case domain.TransformationNodeAntiJoin:
			if len(deduped) == 0 {
				if limiter.Include() {
					results = append(results, leftRecord.Clone())
				}
			}
		}
	}
	return nodeResult{records: results, total: limiter.Total()}, nil
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

func (e *Executor) executeUnion(node domain.EntityTransformationNode, cache map[uuid.UUID]nodeResult, req pageRequest) (nodeResult, error) {
	if len(node.Inputs) == 0 {
		return nodeResult{}, fmt.Errorf("union node requires at least one input")
	}
	limiter := newPageLimiter(req)
	capacity := 0
	if limiter.limit > 0 && limiter.max > 0 {
		capacity = limiter.max
	} else {
		for _, input := range node.Inputs {
			if inputResult, ok := cache[input]; ok {
				capacity += len(inputResult.records)
			}
		}
	}
	results := make([]domain.EntityTransformationRecord, 0, capacity)
	total := 0
	knownTotal := true
	for _, input := range node.Inputs {
		inputResult, ok := cache[input]
		if !ok {
			return nodeResult{}, fmt.Errorf("union input missing")
		}
		if inputResult.total == 0 {
			knownTotal = false
		} else if knownTotal {
			total += inputResult.total
		}
		for _, record := range inputResult.records {
			if limiter.Include() {
				results = append(results, record.Clone())
			}
		}
	}
	if !knownTotal {
		total = limiter.Total()
	}
	return nodeResult{records: results, total: total}, nil
}

func (e *Executor) executeSort(node domain.EntityTransformationNode, cache map[uuid.UUID]nodeResult, req pageRequest) (nodeResult, error) {
	if len(node.Inputs) != 1 {
		return nodeResult{}, fmt.Errorf("sort node requires one input")
	}
	if node.Sort == nil {
		return nodeResult{}, fmt.Errorf("sort node missing configuration")
	}
	inputResult, ok := cache[node.Inputs[0]]
	if !ok {
		return nodeResult{}, fmt.Errorf("sort input missing")
	}
	cloned := cloneRecords(inputResult.records)
	if len(cloned) == 0 {
		return nodeResult{records: cloned, total: inputResult.total}, nil
	}
	sortAlias, err := resolveSortAlias(cloned, node.Sort.Alias)
	if err != nil {
		return nodeResult{}, err
	}
	if sortAlias == "" {
		total := inputResult.total
		if total == 0 {
			total = len(inputResult.records)
		}
		trimmed := trimToWindow(cloned, req)
		return nodeResult{records: trimmed, total: total}, nil
	}
	domain.SortRecords(cloned, sortAlias, node.Sort.Field, node.Sort.Direction)
	trimmed := trimToWindow(cloned, req)
	total := inputResult.total
	if total == 0 {
		total = len(inputResult.records)
	}
	return nodeResult{records: trimmed, total: total}, nil
}

func (e *Executor) executePaginate(node domain.EntityTransformationNode, cache map[uuid.UUID]nodeResult, req pageRequest) (nodeResult, error) {
	if len(node.Inputs) != 1 {
		return nodeResult{}, fmt.Errorf("paginate node requires one input")
	}
	if node.Paginate == nil {
		return nodeResult{}, fmt.Errorf("paginate node missing configuration")
	}
	inputResult, ok := cache[node.Inputs[0]]
	if !ok {
		return nodeResult{}, fmt.Errorf("paginate input missing")
	}
	limit := 0
	offset := 0
	if node.Paginate.Limit != nil {
		limit = *node.Paginate.Limit
	}
	if node.Paginate.Offset != nil {
		offset = *node.Paginate.Offset
	}
	cloned := cloneRecords(inputResult.records)
	paginated := domain.PaginateRecords(cloned, limit, offset)
	paginated = trimToWindow(paginated, req)

	total := inputResult.total
	if total == 0 {
		total = len(inputResult.records)
	}
	return nodeResult{records: paginated, total: total}, nil
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

func trimToWindow(records []domain.EntityTransformationRecord, req pageRequest) []domain.EntityTransformationRecord {
	if req.limit <= 0 {
		return records
	}
	needed := req.offset + req.limit
	if needed <= 0 {
		return records
	}
	if needed < len(records) {
		return records[:needed]
	}
	return records
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

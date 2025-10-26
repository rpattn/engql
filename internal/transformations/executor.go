package transformations

import (
	"context"
	"fmt"

	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
)

// EntityRepository defines the subset of entity storage used by the executor.
type EntityRepository interface {
	List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, sort *domain.EntitySort, limit int, offset int) ([]domain.Entity, int, error)
}

// Executor walks a transformation DAG and produces execution results.
type Executor struct {
	entityRepo EntityRepository
}

// NewExecutor constructs a transformation executor.
func NewExecutor(entityRepo EntityRepository) *Executor {
	return &Executor{entityRepo: entityRepo}
}

// Execute runs the transformation graph and returns paginated results.
func (e *Executor) Execute(ctx context.Context, transformation domain.EntityTransformation, opts domain.EntityTransformationExecutionOptions) (domain.EntityTransformationExecutionResult, error) {
	sorted, err := transformation.TopologicallySortedNodes()
	if err != nil {
		return domain.EntityTransformationExecutionResult{}, err
	}

	results := make(map[uuid.UUID][]domain.EntityTransformationRecord)

	for _, node := range sorted {
		nodeResults, err := e.executeNode(ctx, transformation, node, results)
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

func (e *Executor) executeNode(ctx context.Context, transformation domain.EntityTransformation, node domain.EntityTransformationNode, cache map[uuid.UUID][]domain.EntityTransformationRecord) ([]domain.EntityTransformationRecord, error) {
	switch node.Type {
	case domain.TransformationNodeLoad:
		return e.executeLoad(ctx, transformation, node)
	case domain.TransformationNodeFilter:
		return e.executeFilter(node, cache)
	case domain.TransformationNodeProject:
		return e.executeProject(node, cache)
	case domain.TransformationNodeJoin, domain.TransformationNodeLeftJoin, domain.TransformationNodeAntiJoin:
		return e.executeJoin(node, cache)
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
	for _, entity := range entities {
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
	var filtered []domain.EntityTransformationRecord
	for _, record := range inputRecords {
		entity := record.Entities[node.Filter.Alias]
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
		clone.Entities[node.Project.Alias] = domain.ProjectEntity(clone.Entities[node.Project.Alias], node.Project.Fields)
		projected = append(projected, clone)
	}
	return projected, nil
}

func (e *Executor) executeJoin(node domain.EntityTransformationNode, cache map[uuid.UUID][]domain.EntityTransformationRecord) ([]domain.EntityTransformationRecord, error) {
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

	rightIndex := make(map[string][]domain.EntityTransformationRecord)
	for _, record := range rightRecords {
		entity := record.Entities[node.Join.RightAlias]
		if entity == nil {
			continue
		}
		key := fmt.Sprintf("%v", entity.Properties[node.Join.OnField])
		rightIndex[key] = append(rightIndex[key], record)
	}

	var results []domain.EntityTransformationRecord
	for _, leftRecord := range leftRecords {
		leftEntity := leftRecord.Entities[node.Join.LeftAlias]
		if leftEntity == nil {
			continue
		}
		key := fmt.Sprintf("%v", leftEntity.Properties[node.Join.OnField])
		matches := rightIndex[key]

		switch node.Type {
		case domain.TransformationNodeJoin:
			for _, rightRecord := range matches {
				combined := mergeRecords(leftRecord, rightRecord)
				results = append(results, combined)
			}
		case domain.TransformationNodeLeftJoin:
			if len(matches) == 0 {
				combined := leftRecord.Clone()
				combined.Entities[node.Join.RightAlias] = nil
				results = append(results, combined)
				continue
			}
			for _, rightRecord := range matches {
				combined := mergeRecords(leftRecord, rightRecord)
				results = append(results, combined)
			}
		case domain.TransformationNodeAntiJoin:
			if len(matches) == 0 {
				results = append(results, leftRecord.Clone())
			}
		}
	}
	return results, nil
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
			results = append(results, record.Clone())
		}
	}
	return results, nil
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
	domain.SortRecords(cloned, node.Sort.Alias, node.Sort.Field, node.Sort.Direction)
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

package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
)

func (r *Resolver) CreateEntityTransformation(ctx context.Context, input graph.CreateEntityTransformationInput) (*graph.EntityTransformation, error) {
	orgID, err := uuid.Parse(input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}
	nodes, err := r.graphNodesToDomain(input.Nodes)
	if err != nil {
		return nil, err
	}
	transformation := domain.EntityTransformation{
		OrganizationID: orgID,
		Name:           input.Name,
		Description:    stringOrEmpty(input.Description),
		Nodes:          nodes,
	}
	created, err := r.entityTransformationRepo.Create(ctx, transformation)
	if err != nil {
		return nil, err
	}
	return mapTransformationToGraph(created), nil
}

func (r *Resolver) UpdateEntityTransformation(ctx context.Context, input graph.UpdateEntityTransformationInput) (*graph.EntityTransformation, error) {
	id, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid transformation ID: %w", err)
	}
	existing, err := r.entityTransformationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.Nodes != nil {
		nodes, err := r.graphNodesToDomain(input.Nodes)
		if err != nil {
			return nil, err
		}
		existing.Nodes = nodes
	}
	updated, err := r.entityTransformationRepo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	return mapTransformationToGraph(updated), nil
}

func (r *Resolver) DeleteEntityTransformation(ctx context.Context, id string) (*bool, error) {
	transformationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid transformation ID: %w", err)
	}
	if err := r.entityTransformationRepo.Delete(ctx, transformationID); err != nil {
		return nil, err
	}
	result := true
	return &result, nil
}

func (r *Resolver) EntityTransformation(ctx context.Context, id string) (*graph.EntityTransformation, error) {
	transformationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid transformation ID: %w", err)
	}
	transformation, err := r.entityTransformationRepo.GetByID(ctx, transformationID)
	if err != nil {
		return nil, err
	}
	return mapTransformationToGraph(transformation), nil
}

func (r *Resolver) EntityTransformations(ctx context.Context, organizationID string) ([]*graph.EntityTransformation, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}
	transformations, err := r.entityTransformationRepo.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	result := make([]*graph.EntityTransformation, 0, len(transformations))
	for _, t := range transformations {
		result = append(result, mapTransformationToGraph(t))
	}
	return result, nil
}

func (r *Resolver) ExecuteEntityTransformation(ctx context.Context, input graph.ExecuteEntityTransformationInput) (*graph.EntityTransformationConnection, error) {
	transformationID, err := uuid.Parse(input.TransformationID)
	if err != nil {
		return nil, fmt.Errorf("invalid transformation ID: %w", err)
	}
	transformation, err := r.entityTransformationRepo.GetByID(ctx, transformationID)
	if err != nil {
		return nil, err
	}
	options := domain.EntityTransformationExecutionOptions{}
	if input.Pagination != nil {
		if input.Pagination.Limit != nil {
			options.Limit = *input.Pagination.Limit
		}
		if input.Pagination.Offset != nil {
			options.Offset = *input.Pagination.Offset
		}
	}
	result, err := r.transformationExecutor.Execute(ctx, transformation, options)
	if err != nil {
		return nil, err
	}
	edges := make([]*graph.EntityTransformationRecordEdge, 0, len(result.Records))
	for _, record := range result.Records {
		edge := &graph.EntityTransformationRecordEdge{}
		entities := make([]*graph.EntityTransformationRecordEntity, 0, len(record.Entities))
		for alias, entity := range record.Entities {
			var gqlEntity *graph.Entity
			if entity != nil {
				gqlEntity = convertEntityToGraph(entity)
			}
			entities = append(entities, &graph.EntityTransformationRecordEntity{
				Alias:  alias,
				Entity: gqlEntity,
			})
		}
		edge.Entities = entities
		edges = append(edges, edge)
	}
	hasNextPage := false
	if options.Limit > 0 && options.Offset+options.Limit < result.TotalCount {
		hasNextPage = true
	}
	pageInfo := &graph.PageInfo{
		HasNextPage:     hasNextPage,
		HasPreviousPage: options.Offset > 0,
		TotalCount:      result.TotalCount,
	}
	return &graph.EntityTransformationConnection{Edges: edges, PageInfo: pageInfo}, nil
}

func (r *Resolver) graphNodesToDomain(inputs []*graph.EntityTransformationNodeInput) ([]domain.EntityTransformationNode, error) {
	result := make([]domain.EntityTransformationNode, 0, len(inputs))
	for _, input := range inputs {
		node, err := graphNodeToDomain(input)
		if err != nil {
			return nil, err
		}
		result = append(result, node)
	}
	return result, nil
}

func graphNodeToDomain(input *graph.EntityTransformationNodeInput) (domain.EntityTransformationNode, error) {
	var id uuid.UUID
	var err error
	if input.ID != nil {
		id, err = uuid.Parse(*input.ID)
		if err != nil {
			return domain.EntityTransformationNode{}, fmt.Errorf("invalid node ID: %w", err)
		}
	} else {
		id = uuid.New()
	}
	node := domain.EntityTransformationNode{
		ID:     id,
		Name:   input.Name,
		Type:   domain.EntityTransformationNodeType(input.Type),
		Inputs: []uuid.UUID{},
	}
	if len(input.Inputs) > 0 {
		node.Inputs = make([]uuid.UUID, len(input.Inputs))
		for i, raw := range input.Inputs {
			parsed, err := uuid.Parse(raw)
			if err != nil {
				return domain.EntityTransformationNode{}, fmt.Errorf("invalid input node ID: %w", err)
			}
			node.Inputs[i] = parsed
		}
	}
	switch node.Type {
	case domain.TransformationNodeLoad:
		if input.Load == nil {
			return domain.EntityTransformationNode{}, fmt.Errorf("load node requires configuration")
		}
		node.Load = &domain.EntityTransformationLoadConfig{
			Alias:      input.Load.Alias,
			EntityType: input.Load.EntityType,
			Filters:    graphFiltersToDomain(input.Load.Filters),
		}
	case domain.TransformationNodeFilter:
		if input.Filter == nil {
			return domain.EntityTransformationNode{}, fmt.Errorf("filter node requires configuration")
		}
		node.Filter = &domain.EntityTransformationFilterConfig{
			Alias:   input.Filter.Alias,
			Filters: graphFiltersToDomain(input.Filter.Filters),
		}
	case domain.TransformationNodeProject:
		if input.Project == nil {
			return domain.EntityTransformationNode{}, fmt.Errorf("project node requires configuration")
		}
		node.Project = &domain.EntityTransformationProjectConfig{
			Alias:  input.Project.Alias,
			Fields: append([]string(nil), input.Project.Fields...),
		}
	case domain.TransformationNodeJoin, domain.TransformationNodeLeftJoin, domain.TransformationNodeAntiJoin:
		if input.Join == nil {
			return domain.EntityTransformationNode{}, fmt.Errorf("join node requires configuration")
		}
		node.Join = &domain.EntityTransformationJoinConfig{
			LeftAlias:  input.Join.LeftAlias,
			RightAlias: input.Join.RightAlias,
			OnField:    input.Join.OnField,
		}
	case domain.TransformationNodeSort:
		if input.Sort == nil {
			return domain.EntityTransformationNode{}, fmt.Errorf("sort node requires configuration")
		}
		node.Sort = &domain.EntityTransformationSortConfig{
			Alias:     input.Sort.Alias,
			Field:     input.Sort.Field,
			Direction: domain.JoinSortDirection(input.Sort.Direction),
		}
	case domain.TransformationNodePaginate:
		if input.Paginate == nil {
			return domain.EntityTransformationNode{}, fmt.Errorf("paginate node requires configuration")
		}
		node.Paginate = &domain.EntityTransformationPaginateConfig{
			Limit:  input.Paginate.Limit,
			Offset: input.Paginate.Offset,
		}
	case domain.TransformationNodeUnion:
		if input.Union != nil {
			alias := ""
			if input.Union.Alias != nil {
				alias = *input.Union.Alias
			}
			node.Union = &domain.EntityTransformationUnionConfig{Alias: alias}
		}
	default:
		return domain.EntityTransformationNode{}, fmt.Errorf("unsupported node type: %s", node.Type)
	}
	return node, nil
}

func graphFiltersToDomain(filters []*graph.PropertyFilter) []domain.PropertyFilter {
	if len(filters) == 0 {
		return []domain.PropertyFilter{}
	}
	result := make([]domain.PropertyFilter, 0, len(filters))
	for _, f := range filters {
		if f == nil {
			continue
		}
		filter := domain.PropertyFilter{Key: f.Key}
		if f.Value != nil {
			filter.Value = *f.Value
		}
		filter.Exists = f.Exists
		if len(f.InArray) > 0 {
			filter.InArray = append([]string(nil), f.InArray...)
		}
		result = append(result, filter)
	}
	return result
}

func mapTransformationToGraph(transformation domain.EntityTransformation) *graph.EntityTransformation {
	description := transformation.Description
	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}
	nodes := make([]*graph.EntityTransformationNode, 0, len(transformation.Nodes))
	for _, node := range transformation.Nodes {
		nodes = append(nodes, mapNodeToGraph(node))
	}
	return &graph.EntityTransformation{
		ID:             transformation.ID.String(),
		OrganizationID: transformation.OrganizationID.String(),
		Name:           transformation.Name,
		Description:    descriptionPtr,
		Nodes:          nodes,
		CreatedAt:      transformation.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      transformation.UpdatedAt.Format(time.RFC3339),
	}
}

func mapNodeToGraph(node domain.EntityTransformationNode) *graph.EntityTransformationNode {
	inputs := make([]string, len(node.Inputs))
	for i, input := range node.Inputs {
		inputs[i] = input.String()
	}
	gqlNode := &graph.EntityTransformationNode{
		ID:     node.ID.String(),
		Name:   node.Name,
		Type:   graph.EntityTransformationNodeType(node.Type),
		Inputs: inputs,
	}
	if node.Load != nil {
		gqlNode.Load = &graph.EntityTransformationLoadConfig{
			Alias:      node.Load.Alias,
			EntityType: node.Load.EntityType,
			Filters:    domainFiltersToGraph(node.Load.Filters),
		}
	}
	if node.Filter != nil {
		gqlNode.Filter = &graph.EntityTransformationFilterConfig{
			Alias:   node.Filter.Alias,
			Filters: domainFiltersToGraph(node.Filter.Filters),
		}
	}
	if node.Project != nil {
		gqlNode.Project = &graph.EntityTransformationProjectConfig{
			Alias:  node.Project.Alias,
			Fields: append([]string(nil), node.Project.Fields...),
		}
	}
	if node.Join != nil {
		gqlNode.Join = &graph.EntityTransformationJoinConfig{
			LeftAlias:  node.Join.LeftAlias,
			RightAlias: node.Join.RightAlias,
			OnField:    node.Join.OnField,
		}
	}
	if node.Union != nil {
		var aliasPtr *string
		if node.Union.Alias != "" {
			alias := node.Union.Alias
			aliasPtr = &alias
		}
		gqlNode.Union = &graph.EntityTransformationUnionConfig{
			Alias: aliasPtr,
		}
	}
	if node.Sort != nil {
		gqlNode.Sort = &graph.EntityTransformationSortConfig{
			Alias:     node.Sort.Alias,
			Field:     node.Sort.Field,
			Direction: graph.JoinSortDirection(node.Sort.Direction),
		}
	}
	if node.Paginate != nil {
		gqlNode.Paginate = &graph.EntityTransformationPaginateConfig{
			Limit:  node.Paginate.Limit,
			Offset: node.Paginate.Offset,
		}
	}
	return gqlNode
}

func domainFiltersToGraph(filters []domain.PropertyFilter) []*graph.PropertyFilterConfig {
	if len(filters) == 0 {
		return []*graph.PropertyFilterConfig{}
	}
	result := make([]*graph.PropertyFilterConfig, 0, len(filters))
	for _, filter := range filters {
		f := &graph.PropertyFilterConfig{Key: filter.Key}
		if filter.Value != "" {
			value := filter.Value
			f.Value = &value
		}
		if filter.Exists != nil {
			f.Exists = filter.Exists
		}
		if len(filter.InArray) > 0 {
			f.InArray = append([]string(nil), filter.InArray...)
		}
		result = append(result, f)
	}
	return result
}

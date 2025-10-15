package graphql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"graphql-engineering-api/graph"
	"graphql-engineering-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Resolver) CreateEntityJoinDefinition(ctx context.Context, input graph.CreateEntityJoinDefinitionInput) (*graph.EntityJoinDefinition, error) {
	orgID, err := uuid.Parse(input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("join definition name is required")
	}

	leftType := strings.TrimSpace(input.LeftEntityType)
	rightType := strings.TrimSpace(input.RightEntityType)
	if leftType == "" || rightType == "" {
		return nil, fmt.Errorf("leftEntityType and rightEntityType must be provided")
	}

	joinTypeInput := graph.JoinTypeReference
	if input.JoinType != nil {
		joinTypeInput = *input.JoinType
	}

	joinType := graphJoinTypeToDomain(joinTypeInput)

	var joinFieldPtr *string
	var joinFieldTypePtr *domain.FieldType

	switch joinType {
	case domain.JoinTypeReference:
		if input.JoinField == nil {
			return nil, fmt.Errorf("joinField must be provided for REFERENCE joins")
		}
		joinFieldValue := strings.TrimSpace(*input.JoinField)
		if joinFieldValue == "" {
			return nil, fmt.Errorf("joinField must be provided for REFERENCE joins")
		}

		canonicalField, fieldType, err := r.resolveJoinField(ctx, orgID, leftType, joinFieldValue, rightType)
		if err != nil {
			return nil, err
		}
		joinFieldPtr = stringPtr(canonicalField)
		joinFieldTypePtr = fieldTypePtr(fieldType)
	case domain.JoinTypeCross:
		if input.JoinField != nil && strings.TrimSpace(*input.JoinField) != "" {
			return nil, fmt.Errorf("joinField must be omitted for CROSS joins")
		}
		if err := r.ensureSchemaExists(ctx, orgID, leftType); err != nil {
			return nil, err
		}
		if err := r.ensureSchemaExists(ctx, orgID, rightType); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported join type %s", joinType)
	}

	leftFilters := convertGraphFiltersToDomain(input.LeftFilters)
	rightFilters := convertGraphFiltersToDomain(input.RightFilters)
	sortCriteria := convertGraphSortsToDomain(input.SortCriteria)

	description := ""
	if input.Description != nil {
		description = strings.TrimSpace(*input.Description)
	}

	created, err := r.entityJoinRepo.Create(ctx, domain.EntityJoinDefinition{
		OrganizationID:  orgID,
		Name:            name,
		Description:     description,
		LeftEntityType:  leftType,
		RightEntityType: rightType,
		JoinType:        joinType,
		JoinField:       joinFieldPtr,
		JoinFieldType:   joinFieldTypePtr,
		LeftFilters:     leftFilters,
		RightFilters:    rightFilters,
		SortCriteria:    sortCriteria,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create join definition: %w", err)
	}

	return mapJoinDefinitionToGraph(created), nil
}

func (r *Resolver) UpdateEntityJoinDefinition(ctx context.Context, input graph.UpdateEntityJoinDefinitionInput) (*graph.EntityJoinDefinition, error) {
	joinID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid join definition ID: %w", err)
	}

	existing, err := r.entityJoinRepo.GetByID(ctx, joinID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load join definition: %w", err)
	}

	if input.Name != nil {
		if trimmed := strings.TrimSpace(*input.Name); trimmed != "" {
			existing.Name = trimmed
		}
	}

	if input.Description != nil {
		existing.Description = strings.TrimSpace(*input.Description)
	}

	leftType := existing.LeftEntityType
	rightType := existing.RightEntityType
	newJoinType := sanitizeJoinType(existing.JoinType)

	if input.LeftEntityType != nil {
		if candidate := strings.TrimSpace(*input.LeftEntityType); candidate != "" {
			leftType = candidate
		}
	}
	if input.RightEntityType != nil {
		if candidate := strings.TrimSpace(*input.RightEntityType); candidate != "" {
			rightType = candidate
		}
	}
	if input.JoinType != nil {
		newJoinType = graphJoinTypeToDomain(*input.JoinType)
	}

	var joinFieldOverride *string
	if input.JoinField != nil {
		trimmed := strings.TrimSpace(*input.JoinField)
		if trimmed != "" {
			joinFieldOverride = stringPtr(trimmed)
		} else {
			joinFieldOverride = nil
		}
	} else {
		joinFieldOverride = existing.JoinField
	}

	switch newJoinType {
	case domain.JoinTypeReference:
		if joinFieldOverride == nil || *joinFieldOverride == "" {
			return nil, fmt.Errorf("joinField must be provided for REFERENCE joins")
		}
		canonicalField, fieldType, err := r.resolveJoinField(ctx, existing.OrganizationID, leftType, *joinFieldOverride, rightType)
		if err != nil {
			return nil, err
		}
		existing.JoinField = stringPtr(canonicalField)
		existing.JoinFieldType = fieldTypePtr(fieldType)
	case domain.JoinTypeCross:
		if err := r.ensureSchemaExists(ctx, existing.OrganizationID, leftType); err != nil {
			return nil, err
		}
		if err := r.ensureSchemaExists(ctx, existing.OrganizationID, rightType); err != nil {
			return nil, err
		}
		existing.JoinField = nil
		existing.JoinFieldType = nil
	default:
		return nil, fmt.Errorf("unsupported join type %s", newJoinType)
	}

	existing.JoinType = newJoinType
	existing.LeftEntityType = leftType
	existing.RightEntityType = rightType

	if input.LeftFilters != nil {
		existing.LeftFilters = convertGraphFiltersToDomain(input.LeftFilters)
	}
	if input.RightFilters != nil {
		existing.RightFilters = convertGraphFiltersToDomain(input.RightFilters)
	}
	if input.SortCriteria != nil {
		existing.SortCriteria = convertGraphSortsToDomain(input.SortCriteria)
	}

	updated, err := r.entityJoinRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update join definition: %w", err)
	}

	return mapJoinDefinitionToGraph(updated), nil
}

func (r *Resolver) DeleteEntityJoinDefinition(ctx context.Context, id string) (*bool, error) {
	joinID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid join definition ID: %w", err)
	}

	if err := r.entityJoinRepo.Delete(ctx, joinID); err != nil {
		return nil, fmt.Errorf("failed to delete join definition: %w", err)
	}

	success := true
	return &success, nil
}

func (r *Resolver) EntityJoinDefinition(ctx context.Context, id string) (*graph.EntityJoinDefinition, error) {
	joinID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid join definition ID: %w", err)
	}

	definition, err := r.entityJoinRepo.GetByID(ctx, joinID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load join definition: %w", err)
	}

	return mapJoinDefinitionToGraph(definition), nil
}

func (r *Resolver) EntityJoinDefinitions(ctx context.Context, organizationID string) ([]*graph.EntityJoinDefinition, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	definitions, err := r.entityJoinRepo.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list join definitions: %w", err)
	}

	result := make([]*graph.EntityJoinDefinition, 0, len(definitions))
	for _, def := range definitions {
		result = append(result, mapJoinDefinitionToGraph(def))
	}
	return result, nil
}

func (r *Resolver) ExecuteEntityJoin(ctx context.Context, input graph.ExecuteEntityJoinInput) (*graph.EntityJoinConnection, error) {
	joinID, err := uuid.Parse(input.JoinID)
	if err != nil {
		return nil, fmt.Errorf("invalid join definition ID: %w", err)
	}

	definition, err := r.entityJoinRepo.GetByID(ctx, joinID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load join definition: %w", err)
	}

	limit, offset := resolvePagination(input.Pagination)

	options := domain.JoinExecutionOptions{
		LeftFilters:  convertGraphFiltersToDomain(input.LeftFilters),
		RightFilters: convertGraphFiltersToDomain(input.RightFilters),
		SortCriteria: convertGraphSortsToDomain(input.SortCriteria),
		Limit:        limit,
		Offset:       offset,
	}

	edges, total, err := r.entityJoinRepo.ExecuteJoin(ctx, definition, options)
	if err != nil {
		return nil, fmt.Errorf("failed to execute join: %w", err)
	}

	graphEdges, err := convertJoinEdgesToGraph(edges)
	if err != nil {
		return nil, err
	}

	hasNext := offset+len(graphEdges) < int(total)
	if options.Limit > 0 && offset+options.Limit < int(total) {
		hasNext = true
	}

	connection := &graph.EntityJoinConnection{
		Edges: graphEdges,
		PageInfo: &graph.PageInfo{
			HasNextPage:     hasNext,
			HasPreviousPage: offset > 0,
			TotalCount:      int(total),
		},
	}

	return connection, nil
}

func (r *Resolver) resolveJoinField(ctx context.Context, organizationID uuid.UUID, leftEntityType, joinField, rightEntityType string) (string, domain.FieldType, error) {
	schema, err := r.entitySchemaRepo.GetByName(ctx, organizationID, leftEntityType)
	if err != nil {
		return "", "", fmt.Errorf("failed to load schema for %s: %w", leftEntityType, err)
	}

	for _, field := range schema.Fields {
		if strings.EqualFold(field.Name, joinField) {
			switch field.Type {
			case domain.FieldTypeEntityReference, domain.FieldTypeEntityReferenceArray, domain.FieldTypeEntityID:
				if field.ReferenceEntityType != "" && !strings.EqualFold(field.ReferenceEntityType, rightEntityType) {
					return "", "", fmt.Errorf("field %s references entity type %s, expected %s", field.Name, field.ReferenceEntityType, rightEntityType)
				}
				return field.Name, field.Type, nil
			default:
				return "", "", fmt.Errorf("field %s is not an entity reference field", field.Name)
			}
		}
	}

	return "", "", fmt.Errorf("field %s not found in schema %s", joinField, leftEntityType)
}

func convertGraphFiltersToDomain(filters []*graph.PropertyFilter) []domain.JoinPropertyFilter {
	if len(filters) == 0 {
		return []domain.JoinPropertyFilter{}
	}

	result := make([]domain.JoinPropertyFilter, 0, len(filters))
	for _, filter := range filters {
		if filter == nil || strings.TrimSpace(filter.Key) == "" {
			continue
		}
		domainFilter := domain.JoinPropertyFilter{
			Key:     strings.TrimSpace(filter.Key),
			InArray: append([]string{}, filter.InArray...),
		}
		if filter.Value != nil {
			val := strings.TrimSpace(*filter.Value)
			domainFilter.Value = &val
		}
		if filter.Exists != nil {
			exists := *filter.Exists
			domainFilter.Exists = &exists
		}
		result = append(result, domainFilter)
	}
	return result
}

func convertGraphSortsToDomain(sorts []*graph.JoinSortInput) []domain.JoinSortCriterion {
	if len(sorts) == 0 {
		return []domain.JoinSortCriterion{}
	}

	result := make([]domain.JoinSortCriterion, 0, len(sorts))
	for _, sort := range sorts {
		if sort == nil || strings.TrimSpace(sort.Field) == "" {
			continue
		}
		dir := domain.JoinSortAsc
		if sort.Direction != nil && *sort.Direction == graph.JoinSortDirectionDesc {
			dir = domain.JoinSortDesc
		}
		side := domain.JoinSideLeft
		if sort.Side == graph.JoinSideRight {
			side = domain.JoinSideRight
		}
		result = append(result, domain.JoinSortCriterion{
			Side:      side,
			Field:     strings.TrimSpace(sort.Field),
			Direction: dir,
		})
	}
	return result
}

func convertJoinEdgesToGraph(edges []domain.EntityJoinEdge) ([]*graph.EntityJoinEdge, error) {
	result := make([]*graph.EntityJoinEdge, 0, len(edges))
	for _, edge := range edges {
		left, err := mapDomainEntity(edge.Left)
		if err != nil {
			return nil, err
		}
		right, err := mapDomainEntity(edge.Right)
		if err != nil {
			return nil, err
		}
		result = append(result, &graph.EntityJoinEdge{
			Left:  left,
			Right: right,
		})
	}
	return result, nil
}

func mapJoinDefinitionToGraph(def domain.EntityJoinDefinition) *graph.EntityJoinDefinition {
	desc := strings.TrimSpace(def.Description)
	var description *string
	if desc != "" {
		description = &desc
	}

	gqlJoinType := graph.JoinType(strings.ToUpper(string(def.JoinType)))
	if gqlJoinType != graph.JoinTypeCross && gqlJoinType != graph.JoinTypeReference {
		gqlJoinType = graph.JoinTypeReference
	}

	var joinFieldType *graph.FieldType
	if def.JoinFieldType != nil {
		ft := graph.FieldType(strings.ToUpper(string(*def.JoinFieldType)))
		joinFieldType = &ft
	}

	return &graph.EntityJoinDefinition{
		ID:              def.ID.String(),
		OrganizationID:  def.OrganizationID.String(),
		Name:            def.Name,
		Description:     description,
		LeftEntityType:  def.LeftEntityType,
		RightEntityType: def.RightEntityType,
		JoinType:        gqlJoinType,
		JoinField:       def.JoinField,
		JoinFieldType:   joinFieldType,
		LeftFilters:     convertDomainFiltersToGraph(def.LeftFilters),
		RightFilters:    convertDomainFiltersToGraph(def.RightFilters),
		SortCriteria:    convertDomainSortsToGraph(def.SortCriteria),
		CreatedAt:       def.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       def.UpdatedAt.Format(time.RFC3339),
	}
}

func convertDomainFiltersToGraph(filters []domain.JoinPropertyFilter) []*graph.PropertyFilterConfig {
	if len(filters) == 0 {
		return []*graph.PropertyFilterConfig{}
	}

	result := make([]*graph.PropertyFilterConfig, 0, len(filters))
	for _, filter := range filters {
		config := &graph.PropertyFilterConfig{
			Key:     filter.Key,
			InArray: append([]string{}, filter.InArray...),
		}
		if filter.Value != nil {
			val := *filter.Value
			config.Value = &val
		}
		if filter.Exists != nil {
			exists := *filter.Exists
			config.Exists = &exists
		}
		result = append(result, config)
	}
	return result
}

func convertDomainSortsToGraph(criteria []domain.JoinSortCriterion) []*graph.JoinSortCriterion {
	if len(criteria) == 0 {
		return []*graph.JoinSortCriterion{}
	}

	result := make([]*graph.JoinSortCriterion, 0, len(criteria))
	for _, criterion := range criteria {
		side := graph.JoinSideLeft
		if strings.EqualFold(string(criterion.Side), string(domain.JoinSideRight)) {
			side = graph.JoinSideRight
		}

		direction := graph.JoinSortDirectionAsc
		if strings.EqualFold(string(criterion.Direction), string(domain.JoinSortDesc)) {
			direction = graph.JoinSortDirectionDesc
		}

		result = append(result, &graph.JoinSortCriterion{
			Side:      side,
			Field:     criterion.Field,
			Direction: direction,
		})
	}
	return result
}

func resolvePagination(pagination *graph.PaginationInput) (int, int) {
	limit := 25
	offset := 0

	if pagination != nil {
		if pagination.Limit != nil {
			if *pagination.Limit > 0 {
				limit = *pagination.Limit
			}
		}
		if pagination.Offset != nil && *pagination.Offset >= 0 {
			offset = *pagination.Offset
		}
	}

	return limit, offset
}

func graphJoinTypeToDomain(joinType graph.JoinType) domain.JoinType {
	switch joinType {
	case graph.JoinTypeCross:
		return domain.JoinTypeCross
	case graph.JoinTypeReference:
		return domain.JoinTypeReference
	default:
		return domain.JoinTypeReference
	}
}

func sanitizeJoinType(value domain.JoinType) domain.JoinType {
	switch value {
	case domain.JoinTypeCross:
		return domain.JoinTypeCross
	case domain.JoinTypeReference:
		return domain.JoinTypeReference
	default:
		return domain.JoinTypeReference
	}
}

func (r *Resolver) ensureSchemaExists(ctx context.Context, organizationID uuid.UUID, entityType string) error {
	if entityType == "" {
		return fmt.Errorf("entity type cannot be empty")
	}
	if _, err := r.entitySchemaRepo.GetByName(ctx, organizationID, entityType); err != nil {
		return fmt.Errorf("failed to load schema %s: %w", entityType, err)
	}
	return nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}

func fieldTypePtr(value domain.FieldType) *domain.FieldType {
	v := value
	return &v
}

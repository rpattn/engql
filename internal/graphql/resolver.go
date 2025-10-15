package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/repository"

	"github.com/google/uuid"
)

// Resolver handles GraphQL queries and mutations
type Resolver struct {
	orgRepo          repository.OrganizationRepository
	entitySchemaRepo repository.EntitySchemaRepository
	entityRepo       repository.EntityRepository
	entityJoinRepo   repository.EntityJoinRepository
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
		fields := make([]*graph.FieldDefinition, 0, len(schema.Fields))
		for _, field := range schema.Fields {
			fieldType := graph.FieldType(field.Type)
			fields = append(fields, &graph.FieldDefinition{
				Type:        fieldType,
				Required:    field.Required,
				Description: &field.Description,
				Default:     &field.Default,
				Validation:  &field.Validation,
			})
		}

		result[i] = &graph.EntitySchema{
			ID:             schema.ID.String(),
			OrganizationID: schema.OrganizationID.String(),
			Name:           schema.Name,
			Description:    &schema.Description,
			Fields:         fields,
			CreatedAt:      schema.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      schema.UpdatedAt.Format(time.RFC3339),
		}
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

	fields := make([]*graph.FieldDefinition, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		fields = append(fields, toGraphFieldDefinition(field))
	}

	return &graph.EntitySchema{
		ID:             schema.ID.String(),
		OrganizationID: schema.OrganizationID.String(),
		Name:           schema.Name,
		Description:    &schema.Description,
		Fields:         fields,
		CreatedAt:      schema.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      schema.UpdatedAt.Format(time.RFC3339),
	}, nil
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

	fields := make([]*graph.FieldDefinition, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		fields = append(fields, toGraphFieldDefinition(field))
	}

	return &graph.EntitySchema{
		ID:             schema.ID.String(),
		OrganizationID: schema.OrganizationID.String(),
		Name:           schema.Name,
		Description:    &schema.Description,
		Fields:         fields,
		CreatedAt:      schema.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      schema.UpdatedAt.Format(time.RFC3339),
	}, nil
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
	entities, totalCount, err := r.entityRepo.List(ctx, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Convert to GraphQL type
	result := make([]*graph.Entity, len(entities))
	for i, entity := range entities {
		propertiesJSON, err := entity.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}

		result[i] = &graph.Entity{
			ID:             entity.ID.String(),
			OrganizationID: entity.OrganizationID.String(),
			EntityType:     entity.EntityType,
			Path:           entity.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      entity.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      entity.UpdatedAt.Format(time.RFC3339),
		}
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
		gqlEntity, err := mapDomainEntity(entity)
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

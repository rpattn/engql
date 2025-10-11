package graphql

import (
	"context"
	"fmt"
	"time"

	"graphql-engineering-api/graph"
	"graphql-engineering-api/internal/domain"
	"github.com/google/uuid"
)

// GetEntityAncestors retrieves all ancestor entities of the given entity
func (r *Resolver) GetEntityAncestors(ctx context.Context, entityID string) ([]*graph.Entity, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	// Get the entity first to find its path
	entity, err := r.entityRepo.GetByID(ctx, entityUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get ancestors using the entity's path
	ancestors, err := r.entityRepo.GetAncestors(ctx, entity.OrganizationID, entity.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity ancestors: %w", err)
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(ancestors))
	for i, ancestor := range ancestors {
		propertiesJSON, err := ancestor.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}

		result[i] = &graph.Entity{
			ID:             ancestor.ID.String(),
			OrganizationID: ancestor.OrganizationID.String(),
			EntityType:     ancestor.EntityType,
			Path:           ancestor.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      ancestor.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      ancestor.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetEntityDescendants retrieves all descendant entities of the given entity
func (r *Resolver) GetEntityDescendants(ctx context.Context, entityID string) ([]*graph.Entity, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	// Get the entity first to find its path
	entity, err := r.entityRepo.GetByID(ctx, entityUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get descendants using the entity's path
	descendants, err := r.entityRepo.GetDescendants(ctx, entity.OrganizationID, entity.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity descendants: %w", err)
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(descendants))
	for i, descendant := range descendants {
		propertiesJSON, err := descendant.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}

		result[i] = &graph.Entity{
			ID:             descendant.ID.String(),
			OrganizationID: descendant.OrganizationID.String(),
			EntityType:     descendant.EntityType,
			Path:           descendant.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      descendant.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      descendant.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetEntityChildren retrieves direct child entities of the given entity
func (r *Resolver) GetEntityChildren(ctx context.Context, entityID string) ([]*graph.Entity, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	// Get the entity first to find its path
	entity, err := r.entityRepo.GetByID(ctx, entityUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get children using the entity's path
	children, err := r.entityRepo.GetChildren(ctx, entity.OrganizationID, entity.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity children: %w", err)
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(children))
	for i, child := range children {
		propertiesJSON, err := child.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}

		result[i] = &graph.Entity{
			ID:             child.ID.String(),
			OrganizationID: child.OrganizationID.String(),
			EntityType:     child.EntityType,
			Path:           child.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      child.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      child.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetEntitySiblings retrieves sibling entities of the given entity
func (r *Resolver) GetEntitySiblings(ctx context.Context, entityID string) ([]*graph.Entity, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	// Get the entity first to find its path
	entity, err := r.entityRepo.GetByID(ctx, entityUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get siblings using the entity's path
	siblings, err := r.entityRepo.GetSiblings(ctx, entity.OrganizationID, entity.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity siblings: %w", err)
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(siblings))
	for i, sibling := range siblings {
		propertiesJSON, err := sibling.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}

		result[i] = &graph.Entity{
			ID:             sibling.ID.String(),
			OrganizationID: sibling.OrganizationID.String(),
			EntityType:     sibling.EntityType,
			Path:           sibling.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      sibling.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      sibling.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetEntityHierarchy retrieves the complete hierarchy tree for an entity
func (r *Resolver) GetEntityHierarchy(ctx context.Context, entityID string) (*graph.EntityHierarchy, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	// Get the entity first
	entity, err := r.entityRepo.GetByID(ctx, entityUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get ancestors, children, and siblings in parallel
	ancestorsChan := make(chan []domain.Entity, 1)
	childrenChan := make(chan []domain.Entity, 1)
	siblingsChan := make(chan []domain.Entity, 1)
	errorChan := make(chan error, 3)

	go func() {
		ancestors, err := r.entityRepo.GetAncestors(ctx, entity.OrganizationID, entity.Path)
		if err != nil {
			errorChan <- err
			return
		}
		ancestorsChan <- ancestors
	}()

	go func() {
		children, err := r.entityRepo.GetChildren(ctx, entity.OrganizationID, entity.Path)
		if err != nil {
			errorChan <- err
			return
		}
		childrenChan <- children
	}()

	go func() {
		siblings, err := r.entityRepo.GetSiblings(ctx, entity.OrganizationID, entity.Path)
		if err != nil {
			errorChan <- err
			return
		}
		siblingsChan <- siblings
	}()

	// Wait for all results
	ancestors := <-ancestorsChan
	children := <-childrenChan
	siblings := <-siblingsChan

	// Check for errors
	select {
	case err := <-errorChan:
		return nil, fmt.Errorf("failed to get hierarchy data: %w", err)
	default:
	}

	// Convert current entity to GraphQL format
	propertiesJSON, err := entity.GetPropertiesAsJSONB()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity properties: %w", err)
	}

	currentEntity := &graph.Entity{
		ID:             entity.ID.String(),
		OrganizationID: entity.OrganizationID.String(),
		EntityType:     entity.EntityType,
		Path:           entity.Path,
		Properties:     string(propertiesJSON),
		CreatedAt:      entity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      entity.UpdatedAt.Format(time.RFC3339),
	}

	// Convert ancestors to GraphQL format
	gqlAncestors := make([]*graph.Entity, len(ancestors))
	for i, ancestor := range ancestors {
		propertiesJSON, err := ancestor.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ancestor properties: %w", err)
		}

		gqlAncestors[i] = &graph.Entity{
			ID:             ancestor.ID.String(),
			OrganizationID: ancestor.OrganizationID.String(),
			EntityType:     ancestor.EntityType,
			Path:           ancestor.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      ancestor.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      ancestor.UpdatedAt.Format(time.RFC3339),
		}
	}

	// Convert children to GraphQL format
	gqlChildren := make([]*graph.Entity, len(children))
	for i, child := range children {
		propertiesJSON, err := child.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal child properties: %w", err)
		}

		gqlChildren[i] = &graph.Entity{
			ID:             child.ID.String(),
			OrganizationID: child.OrganizationID.String(),
			EntityType:     child.EntityType,
			Path:           child.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      child.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      child.UpdatedAt.Format(time.RFC3339),
		}
	}

	// Convert siblings to GraphQL format
	gqlSiblings := make([]*graph.Entity, len(siblings))
	for i, sibling := range siblings {
		propertiesJSON, err := sibling.GetPropertiesAsJSONB()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal sibling properties: %w", err)
		}

		gqlSiblings[i] = &graph.Entity{
			ID:             sibling.ID.String(),
			OrganizationID: sibling.OrganizationID.String(),
			EntityType:     sibling.EntityType,
			Path:           sibling.Path,
			Properties:     string(propertiesJSON),
			CreatedAt:      sibling.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      sibling.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &graph.EntityHierarchy{
		Current:   currentEntity,
		Ancestors: gqlAncestors,
		Children:  gqlChildren,
		Siblings:  gqlSiblings,
	}, nil
}

package graphql

import (
	"context"
	"fmt"
	"time"

	"graphql-engineering-api/graph"
	"graphql-engineering-api/internal/domain"
	"graphql-engineering-api/internal/middleware"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
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

	// Get ancestor IDs
	ancestors, err := r.entityRepo.GetAncestors(ctx, entity.OrganizationID, entity.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity ancestors: %w", err)
	}

	// Use dataloader if available to batch-load ancestor entities
	loadedAncestors := make(map[string]domain.Entity)
	if loader := middleware.EntityLoaderFromContext(ctx); loader != nil && len(ancestors) > 0 {
		keys := make(dataloader.Keys, len(ancestors))
		for i, a := range ancestors {
			keys[i] = dataloader.StringKey(a.ID.String())
		}

		thunk := loader.LoadMany(ctx, keys)
		results, errs := thunk()
		if len(errs) > 0 {
			// Log partial errors but continue
			for _, e := range errs {
				fmt.Printf("⚠️ dataloader error: %v\n", e)
			}
		}

		for i, r := range results {
			if r != nil {
				if e, ok := r.(domain.Entity); ok {
					loadedAncestors[ancestors[i].ID.String()] = e
				}
			}
		}
	}

	// Convert to GraphQL entities
	result := make([]*graph.Entity, len(ancestors))
	for i, ancestor := range ancestors {
		var e domain.Entity
		if loaded, ok := loadedAncestors[ancestor.ID.String()]; ok {
			e = loaded
		} else {
			e = ancestor
		}

		propsJSON, _ := e.GetPropertiesAsJSONB()
		result[i] = &graph.Entity{
			ID:             e.ID.String(),
			OrganizationID: e.OrganizationID.String(),
			EntityType:     e.EntityType,
			Path:           e.Path,
			Properties:     string(propsJSON),
			CreatedAt:      e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
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

	// Use dataloader if available to batch-load descendant entities
	loadedDescendants := make(map[string]domain.Entity)
	if loader := middleware.EntityLoaderFromContext(ctx); loader != nil && len(descendants) > 0 {
		keys := make(dataloader.Keys, len(descendants))
		for i, d := range descendants {
			keys[i] = dataloader.StringKey(d.ID.String())
		}

		thunk := loader.LoadMany(ctx, keys)
		results, errs := thunk()
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Printf("⚠️ dataloader error: %v\n", e)
			}
		}

		for i, r := range results {
			if r != nil {
				if e, ok := r.(domain.Entity); ok {
					loadedDescendants[descendants[i].ID.String()] = e
				}
			}
		}
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(descendants))
	for i, d := range descendants {
		var e domain.Entity
		if loaded, ok := loadedDescendants[d.ID.String()]; ok {
			e = loaded
		} else {
			e = d
		}

		propsJSON, _ := e.GetPropertiesAsJSONB()
		result[i] = &graph.Entity{
			ID:             e.ID.String(),
			OrganizationID: e.OrganizationID.String(),
			EntityType:     e.EntityType,
			Path:           e.Path,
			Properties:     string(propsJSON),
			CreatedAt:      e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
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

	// Use dataloader if available to batch-load child entities
	loadedChildren := make(map[string]domain.Entity)
	if loader := middleware.EntityLoaderFromContext(ctx); loader != nil && len(children) > 0 {
		keys := make(dataloader.Keys, len(children))
		for i, c := range children {
			keys[i] = dataloader.StringKey(c.ID.String())
		}

		thunk := loader.LoadMany(ctx, keys)
		results, errs := thunk()
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Printf("⚠️ dataloader error: %v\n", e)
			}
		}

		for i, r := range results {
			if r != nil {
				if e, ok := r.(domain.Entity); ok {
					loadedChildren[children[i].ID.String()] = e
				}
			}
		}
	}

	// Convert to GraphQL entities
	result := make([]*graph.Entity, len(children))
	for i, child := range children {
		var e domain.Entity
		if loaded, ok := loadedChildren[child.ID.String()]; ok {
			e = loaded
		} else {
			e = child
		}

		propsJSON, _ := e.GetPropertiesAsJSONB()
		result[i] = &graph.Entity{
			ID:             e.ID.String(),
			OrganizationID: e.OrganizationID.String(),
			EntityType:     e.EntityType,
			Path:           e.Path,
			Properties:     string(propsJSON),
			CreatedAt:      e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
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

	// Use dataloader if available to batch-load sibling entities
	loadedSiblings := make(map[string]domain.Entity)
	if loader := middleware.EntityLoaderFromContext(ctx); loader != nil && len(siblings) > 0 {
		keys := make(dataloader.Keys, len(siblings))
		for i, s := range siblings {
			keys[i] = dataloader.StringKey(s.ID.String())
		}

		thunk := loader.LoadMany(ctx, keys)
		results, errs := thunk()
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Printf("⚠️ dataloader error: %v\n", e)
			}
		}

		for i, r := range results {
			if r != nil {
				if e, ok := r.(domain.Entity); ok {
					loadedSiblings[siblings[i].ID.String()] = e
				}
			}
		}
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(siblings))
	for i, s := range siblings {
		var e domain.Entity
		if loaded, ok := loadedSiblings[s.ID.String()]; ok {
			e = loaded
		} else {
			e = s
		}

		propsJSON, _ := e.GetPropertiesAsJSONB()
		result[i] = &graph.Entity{
			ID:             e.ID.String(),
			OrganizationID: e.OrganizationID.String(),
			EntityType:     e.EntityType,
			Path:           e.Path,
			Properties:     string(propsJSON),
			CreatedAt:      e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
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

	// Get the entity itself via dataloader if available
	var entity domain.Entity
	if loader := middleware.EntityLoaderFromContext(ctx); loader != nil {
		thunk := loader.Load(ctx, dataloader.StringKey(entityID))
		result, err := thunk()
		if err != nil {
			return nil, fmt.Errorf("failed to load entity via dataloader: %w", err)
		}
		if result == nil {
			return nil, fmt.Errorf("entity not found")
		}
		e, ok := result.(domain.Entity)
		if !ok {
			return nil, fmt.Errorf("unexpected type for entity")
		}
		entity = e
	} else {
		// fallback to repo
		entity, err = r.entityRepo.GetByID(ctx, entityUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get entity: %w", err)
		}
	}

	// Collect IDs for ancestors, children, siblings
	ancestors, _ := r.entityRepo.GetAncestors(ctx, entity.OrganizationID, entity.Path)
	children, _ := r.entityRepo.GetChildren(ctx, entity.OrganizationID, entity.Path)
	siblings, _ := r.entityRepo.GetSiblings(ctx, entity.OrganizationID, entity.Path)

	// Combine all IDs to load via dataloader in one batch
	allEntities := append(append(ancestors, children...), siblings...)
	idsToLoad := make([]string, len(allEntities))
	for i, e := range allEntities {
		idsToLoad[i] = e.ID.String()
	}

	// Use dataloader to fetch all entities in one batch
	var loadedEntities map[string]domain.Entity
	if loader := middleware.EntityLoaderFromContext(ctx); loader != nil && len(idsToLoad) > 0 {
		keys := make(dataloader.Keys, len(idsToLoad))
		for i, id := range idsToLoad {
			keys[i] = dataloader.StringKey(id)
		}
		thunk := loader.LoadMany(ctx, keys)
		results, errs := thunk()
		if len(errs) > 0 {
			// optionally propagate partial errors
			for _, e := range errs {
				fmt.Printf("⚠️ dataloader error: %v\n", e)
			}
		}
		loadedEntities = make(map[string]domain.Entity)
		for i, r := range results {
			if r != nil {
				if e, ok := r.(domain.Entity); ok {
					loadedEntities[idsToLoad[i]] = e
				}
			}
		}
	}

	// Helper to convert domain.Entity -> GraphQL entity
	toGraph := func(e domain.Entity) *graph.Entity {
		propsJSON, _ := e.GetPropertiesAsJSONB()
		return &graph.Entity{
			ID:             e.ID.String(),
			OrganizationID: e.OrganizationID.String(),
			EntityType:     e.EntityType,
			Path:           e.Path,
			Properties:     string(propsJSON),
			CreatedAt:      e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
		}
	}

	// Build hierarchy
	gqlAncestors := make([]*graph.Entity, len(ancestors))
	for i, a := range ancestors {
		if loaded, ok := loadedEntities[a.ID.String()]; ok {
			gqlAncestors[i] = toGraph(loaded)
		} else {
			gqlAncestors[i] = toGraph(a)
		}
	}

	gqlChildren := make([]*graph.Entity, len(children))
	for i, c := range children {
		if loaded, ok := loadedEntities[c.ID.String()]; ok {
			gqlChildren[i] = toGraph(loaded)
		} else {
			gqlChildren[i] = toGraph(c)
		}
	}

	gqlSiblings := make([]*graph.Entity, len(siblings))
	for i, s := range siblings {
		if loaded, ok := loadedEntities[s.ID.String()]; ok {
			gqlSiblings[i] = toGraph(loaded)
		} else {
			gqlSiblings[i] = toGraph(s)
		}
	}

	currentEntity := toGraph(entity)

	return &graph.EntityHierarchy{
		Current:   currentEntity,
		Ancestors: gqlAncestors,
		Children:  gqlChildren,
		Siblings:  gqlSiblings,
	}, nil
}

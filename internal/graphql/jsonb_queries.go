package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/middleware"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
)

type contextKey string

const entityCacheContextKey contextKey = "entityCache"

func ensureEntityCache(ctx context.Context) (context.Context, map[string]*graph.Entity) {
	if ctx == nil {
		cache := make(map[string]*graph.Entity)
		return context.WithValue(context.Background(), entityCacheContextKey, cache), cache
	}

	if cached, ok := ctx.Value(entityCacheContextKey).(map[string]*graph.Entity); ok {
		return ctx, cached
	}

	cache := make(map[string]*graph.Entity)
	return context.WithValue(ctx, entityCacheContextKey, cache), cache
}

func mapDomainEntity(e domain.Entity) (*graph.Entity, error) {
	propsJSON, err := e.GetPropertiesAsJSONB()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties for entity %s: %w", e.ID, err)
	}

	return &graph.Entity{
		ID:             e.ID.String(),
		OrganizationID: e.OrganizationID.String(),
		SchemaID:       e.SchemaID.String(),
		EntityType:     e.EntityType,
		Path:           e.Path,
		Properties:     string(propsJSON),
		Version:        int(e.Version),
		CreatedAt:      e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func collectLinkedEntityIDs(props map[string]any, schema *domain.EntitySchema) []string {
	if props == nil {
		return nil
	}

	ids := make([]string, 0)
	seen := make(map[string]struct{})

	addIDs := func(value any) {
		for _, id := range normalizeLinkedIDValues(value) {
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}

	for key, value := range props {
		if isLinkedFieldName(key) {
			addIDs(value)
		}
	}

	if schema != nil {
		for _, field := range schema.Fields {
			switch field.Type {
			case domain.FieldTypeEntityReference, domain.FieldTypeEntityID:
				if value, ok := props[field.Name]; ok {
					addIDs(value)
				}
			case domain.FieldTypeEntityReferenceArray:
				if value, ok := props[field.Name]; ok {
					addIDs(value)
				}
			}
		}
	}

	return ids
}

func combineErrors(errs []error) error {
	var messages []string
	for _, err := range errs {
		if err == nil {
			continue
		}
		messages = append(messages, err.Error())
	}

	if len(messages) == 0 {
		return nil
	}

	return errors.New(strings.Join(messages, "; "))
}

// SearchEntitiesByProperty performs JSONB property-based search
func (r *Resolver) SearchEntitiesByProperty(ctx context.Context, organizationID string, propertyKey string, propertyValue string) ([]*graph.Entity, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Create a filter map for the specific property
	filter := map[string]any{
		propertyKey: propertyValue,
	}

	entities, err := r.entityRepo.FilterByProperty(ctx, orgID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to filter entities by property: %w", err)
	}

	// Convert to GraphQL format
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

	return result, nil
}

func (r *Resolver) LinkedEntities(ctx context.Context, obj *graph.Entity) ([]*graph.Entity, error) {
	if obj == nil {
		return []*graph.Entity{}, nil
	}

	if len(obj.LinkedEntities) > 0 {
		return obj.LinkedEntities, nil
	}

	ctx, cache := ensureEntityCache(ctx)

	if obj.ID != "" {
		cache[obj.ID] = obj
	}

	if err := r.hydrateLinkedEntities(ctx, []*graph.Entity{obj}); err != nil {
		return nil, err
	}

	if obj.LinkedEntities == nil {
		obj.LinkedEntities = []*graph.Entity{}
	}

	return obj.LinkedEntities, nil
}

func (r *Resolver) EntitiesByIDs(ctx context.Context, ids []string) ([]*graph.Entity, error) {
	loader := middleware.EntityLoaderFromContext(ctx)
	if loader == nil {
		return nil, fmt.Errorf("entity loader not found in context")
	}

	ctx, cache := ensureEntityCache(ctx)

	results := make([]*graph.Entity, len(ids))
	toLoad := make(dataloader.Keys, 0, len(ids))
	indices := make([]int, 0, len(ids))

	for i, id := range ids {
		if id == "" {
			continue
		}
		if cached, ok := cache[id]; ok && cached != nil {
			results[i] = cached
			continue
		}
		toLoad = append(toLoad, dataloader.StringKey(id))
		indices = append(indices, i)
	}

	var partialErrs []error

	if len(toLoad) > 0 {
		thunk := loader.LoadMany(ctx, toLoad)
		rawResults, errs := thunk()
		if len(errs) > 0 {
			partialErrs = append(partialErrs, errs...)
		}

		for idx, raw := range rawResults {
			resultIndex := indices[idx]
			if raw == nil {
				continue
			}

			entity, ok := raw.(domain.Entity)
			if !ok {
				partialErrs = append(partialErrs, fmt.Errorf("unexpected type for entity"))
				continue
			}

			gqlEntity, err := mapDomainEntity(entity)
			if err != nil {
				partialErrs = append(partialErrs, err)
				continue
			}

			results[resultIndex] = gqlEntity
			cache[gqlEntity.ID] = gqlEntity
		}
	}

	for i, id := range ids {
		if results[i] == nil {
			if cached, ok := cache[id]; ok && cached != nil {
				results[i] = cached
			}
		}
	}

	if err := r.hydrateLinkedEntities(ctx, results); err != nil {
		partialErrs = append(partialErrs, err)
	}

	if err := combineErrors(partialErrs); err != nil {
		return results, fmt.Errorf("partial errors occurred: %s", err.Error())
	}

	return results, nil
}

func (r *Resolver) hydrateLinkedEntities(ctx context.Context, parents []*graph.Entity) error {
	if len(parents) == 0 {
		return nil
	}

	ctx, cache := ensureEntityCache(ctx)

	schemaCache := make(map[string]*domain.EntitySchema)

	parentMissing := make(map[*graph.Entity][]string)
	missingSet := make(map[string]struct{})
	var errs []error

	for _, parent := range parents {
		if parent == nil {
			continue
		}

		if parent.ID != "" {
			cache[parent.ID] = parent
		}

		if len(parent.LinkedEntities) > 0 {
			for _, child := range parent.LinkedEntities {
				if child != nil && child.ID != "" {
					cache[child.ID] = child
				}
			}
			continue
		}

		var props map[string]any
		if err := json.Unmarshal([]byte(parent.Properties), &props); err != nil {
			errs = append(errs, fmt.Errorf("entity %s: invalid properties JSON: %w", parent.ID, err))
			parent.LinkedEntities = []*graph.Entity{}
			continue
		}

		var schema *domain.EntitySchema
		if parent.OrganizationID != "" && parent.EntityType != "" {
			cacheKey := parent.OrganizationID + ":" + parent.EntityType
			if cachedSchema, ok := schemaCache[cacheKey]; ok {
				schema = cachedSchema
			} else {
				if orgUUID, err := uuid.Parse(parent.OrganizationID); err == nil {
					if loadedSchema, err := r.entitySchemaRepo.GetByName(ctx, orgUUID, parent.EntityType); err == nil {
						schemaCopy := loadedSchema
						schema = &schemaCopy
					}
				}
				schemaCache[cacheKey] = schema
			}
		}

		linkedIDs := collectLinkedEntityIDs(props, schema)
		if len(linkedIDs) == 0 {
			parent.LinkedEntities = []*graph.Entity{}
			continue
		}

		for _, id := range linkedIDs {
			if id == "" {
				continue
			}
			if child, ok := cache[id]; ok && child != nil {
				parent.LinkedEntities = append(parent.LinkedEntities, child)
				continue
			}
			missingSet[id] = struct{}{}
			parentMissing[parent] = append(parentMissing[parent], id)
		}
	}

	if len(missingSet) == 0 {
		return combineErrors(errs)
	}

	missing := make([]string, 0, len(missingSet))
	for id := range missingSet {
		missing = append(missing, id)
	}

	linkedEntities, err := r.EntitiesByIDs(ctx, missing)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed loading linked entities: %w", err))
	}

	for _, entity := range linkedEntities {
		if entity != nil && entity.ID != "" {
			cache[entity.ID] = entity
		}
	}

	for parent, ids := range parentMissing {
		for _, id := range ids {
			if child, ok := cache[id]; ok && child != nil {
				parent.LinkedEntities = append(parent.LinkedEntities, child)
			}
		}
	}

	return combineErrors(errs)
}

// SearchEntitiesByMultipleProperties performs JSONB search with multiple property filters
func (r *Resolver) SearchEntitiesByMultipleProperties(ctx context.Context, organizationID string, filters map[string]any) ([]*graph.Entity, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	entities, err := r.entityRepo.FilterByProperty(ctx, orgID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to filter entities by properties: %w", err)
	}

	// Convert to GraphQL format
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

	return result, nil
}

// SearchEntitiesByPropertyRange performs range-based search on numeric properties
func (r *Resolver) SearchEntitiesByPropertyRange(ctx context.Context, organizationID string, propertyKey string, minValue *float64, maxValue *float64) ([]*graph.Entity, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// TODO: Implement pagination correctly
	entities, _, err := r.entityRepo.List(ctx, orgID, nil, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Filter by range in Go (in a production system, you'd want to do this in SQL)
	var filteredEntities []domain.Entity
	for _, entity := range entities {
		if value, exists := entity.Properties[propertyKey]; exists {
			if numValue, ok := value.(float64); ok {
				// Check if value is within range
				withinRange := true
				if minValue != nil && numValue < *minValue {
					withinRange = false
				}
				if maxValue != nil && numValue > *maxValue {
					withinRange = false
				}
				if withinRange {
					filteredEntities = append(filteredEntities, entity)
				}
			}
		}
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(filteredEntities))
	for i, entity := range filteredEntities {
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

	return result, nil
}

// SearchEntitiesByPropertyExists checks if entities have a specific property
func (r *Resolver) SearchEntitiesByPropertyExists(ctx context.Context, organizationID string, propertyKey string) ([]*graph.Entity, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Get 10 entities for the organization first
	// TODO: Implement pagination correctly
	entities, _, err := r.entityRepo.List(ctx, orgID, nil, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Filter by property existence
	var filteredEntities []domain.Entity
	for _, entity := range entities {
		if _, exists := entity.Properties[propertyKey]; exists {
			filteredEntities = append(filteredEntities, entity)
		}
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(filteredEntities))
	for i, entity := range filteredEntities {
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

	return result, nil
}

// SearchEntitiesByPropertyContains performs substring search on string properties
func (r *Resolver) SearchEntitiesByPropertyContains(ctx context.Context, organizationID string, propertyKey string, searchTerm string) ([]*graph.Entity, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// TODO: Implement pagination correctly
	entities, _, err := r.entityRepo.List(ctx, orgID, nil, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Filter by property contains
	var filteredEntities []domain.Entity
	for _, entity := range entities {
		if value, exists := entity.Properties[propertyKey]; exists {
			if strValue, ok := value.(string); ok {
				// Simple case-insensitive substring search
				if len(searchTerm) > 0 && len(strValue) > 0 {
					// Convert to lowercase for case-insensitive search
					if contains(strValue, searchTerm) {
						filteredEntities = append(filteredEntities, entity)
					}
				}
			}
		}
	}

	// Convert to GraphQL format
	result := make([]*graph.Entity, len(filteredEntities))
	for i, entity := range filteredEntities {
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

	return result, nil
}

// ValidateEntityAgainstSchema validates an entity's properties against its schema
func (r *Resolver) ValidateEntityAgainstSchema(ctx context.Context, entityID string) (*graph.ValidationResult, error) {
	entityUUID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	// Get the entity
	entity, err := r.entityRepo.GetByID(ctx, entityUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get the entity schema
	schema, err := r.entitySchemaRepo.GetByName(ctx, entity.OrganizationID, entity.EntityType)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity schema: %w", err)
	}

	// Validate properties against schema
	var errors []string
	var warnings []string

	for _, fieldDef := range schema.Fields {
		value, exists := entity.Properties[fieldDef.Name]

		// Check required fields
		if fieldDef.Required && (!exists || value == nil) {
			errors = append(errors, fmt.Sprintf("Required field '%s' is missing", fieldDef.Name))
			continue
		}

		// Skip validation for missing optional fields
		if !exists || value == nil {
			continue
		}

		// Type validation
		switch fieldDef.Type {
		case domain.FieldTypeString:
			if _, ok := value.(string); !ok {
				errors = append(errors, fmt.Sprintf("Field '%s' must be a string, got %T", fieldDef.Name, value))
			}
		case domain.FieldTypeInteger:
			if _, ok := value.(float64); !ok {
				if intVal, ok := value.(int); !ok {
					errors = append(errors, fmt.Sprintf("Field '%s' must be an integer, got %T", fieldDef.Name, value))
				} else {
					// Convert int to float64 for consistency
					entity.Properties[fieldDef.Name] = float64(intVal)
				}
			}
		case domain.FieldTypeFloat:
			if _, ok := value.(float64); !ok {
				errors = append(errors, fmt.Sprintf("Field '%s' must be a float, got %T", fieldDef.Name, value))
			}
		case domain.FieldTypeBoolean:
			if _, ok := value.(bool); !ok {
				errors = append(errors, fmt.Sprintf("Field '%s' must be a boolean, got %T", fieldDef.Name, value))
			}
		case domain.FieldTypeTimestamp:
			if strVal, ok := value.(string); ok {
				// Try to parse as timestamp
				if _, err := time.Parse(time.RFC3339, strVal); err != nil {
					warnings = append(warnings, fmt.Sprintf("Field '%s' timestamp format may be invalid: %v", fieldDef.Name, err))
				}
			} else {
				errors = append(errors, fmt.Sprintf("Field '%s' must be a timestamp string, got %T", fieldDef.Name, value))
			}
		case domain.FieldTypeJSON:
			// JSON type can be any valid JSON value, so we just check if it can be marshaled
			if _, err := json.Marshal(value); err != nil {
				errors = append(errors, fmt.Sprintf("Field '%s' contains invalid JSON: %v", fieldDef.Name, err))
			}
		default:
			warnings = append(warnings, fmt.Sprintf("Field '%s' has unsupported type '%s'", fieldDef.Name, fieldDef.Type))
		}
	}

	// Check for extra properties not defined in schema
	for propertyName := range entity.Properties {
		found := false
		for _, fieldDef := range schema.Fields {
			if fieldDef.Name == propertyName {
				found = true
				break
			}
		}
		if !found {
			warnings = append(warnings, fmt.Sprintf("Property '%s' is not defined in schema", propertyName))
		}
	}

	return &graph.ValidationResult{
		IsValid:  len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}, nil
}

// Helper function for case-insensitive substring search
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) == 0 {
		return false
	}

	// Simple case-insensitive search
	sLower := toLowerCase(s)
	substrLower := toLowerCase(substr)

	return indexOf(sLower, substrLower) >= 0
}

// Simple toLowerCase implementation
func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}

// Simple indexOf implementation
func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

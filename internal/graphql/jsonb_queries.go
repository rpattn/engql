package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"graphql-engineering-api/graph"
	"graphql-engineering-api/internal/domain"
	"graphql-engineering-api/internal/middleware"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
)

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
	// 1️⃣ Load the entity’s schema
	orgID, err := uuid.Parse(obj.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	schema, err := r.entitySchemaRepo.GetByName(ctx, orgID, obj.EntityType)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema for entity type %s: %w", obj.EntityType, err)
	}

	// 2️⃣ Parse the properties JSON
	var props map[string]any
	if err := json.Unmarshal([]byte(obj.Properties), &props); err != nil {
		return nil, fmt.Errorf("invalid properties JSON: %w", err)
	}

	// 3️⃣ Collect referenced IDs
	var refIDs []uuid.UUID
	for _, f := range schema.Fields {
		if f.Type == domain.FieldTypeEntityReference || f.Type == domain.FieldTypeEntityReferenceArray {
			val := props[f.Name]
			switch v := val.(type) {
			case string:
				if id, err := uuid.Parse(v); err == nil {
					refIDs = append(refIDs, id)
				}
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						if id, err := uuid.Parse(s); err == nil {
							refIDs = append(refIDs, id)
						}
					}
				}
			}
		}
	}

	// 4️⃣ Fetch the referenced entities
	if len(refIDs) == 0 {
		return []*graph.Entity{}, nil
	}

	linked, err := r.entityRepo.GetByIDs(ctx, refIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load linked entities: %w", err)
	}

	// 5️⃣ Map to GraphQL entities
	result := make([]*graph.Entity, len(linked))
	for i, e := range linked {
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

func (r *Resolver) EntitiesByIDs(ctx context.Context, ids []string) ([]*graph.Entity, error) {
	loader := middleware.EntityLoaderFromContext(ctx)
	if loader == nil {
		return nil, fmt.Errorf("entity loader not found in context")
	}

	// Load primary entities
	keys := make(dataloader.Keys, len(ids))
	for i, id := range ids {
		keys[i] = dataloader.StringKey(id)
	}
	thunk := loader.LoadMany(ctx, keys)
	rawResults, errs := thunk()

	var partialErrs []error
	if len(errs) > 0 {
		partialErrs = append(partialErrs, errs...)
	}

	results := make([]*graph.Entity, len(rawResults))
	linkedIDSet := make(map[string]struct{}) // collect all linked IDs

	for i, r := range rawResults {
		if r == nil {
			results[i] = nil
			continue
		}

		e, ok := r.(domain.Entity)
		if !ok {
			partialErrs = append(partialErrs, fmt.Errorf("unexpected type for entity"))
			continue
		}

		propsJSON, _ := json.Marshal(e.Properties)
		gqlEntity := &graph.Entity{
			ID:             e.ID.String(),
			OrganizationID: e.OrganizationID.String(),
			EntityType:     e.EntityType,
			Path:           e.Path,
			Properties:     string(propsJSON),
			CreatedAt:      e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
		}

		// Collect linked IDs for batch resolution
		if linkedIDsRaw, ok := e.Properties["linked_ids"]; ok {
			if linkedIDsSlice, ok := linkedIDsRaw.([]interface{}); ok {
				for _, idVal := range linkedIDsSlice {
					if idStr, ok := idVal.(string); ok {
						linkedIDSet[idStr] = struct{}{}
					}
				}
			}
		}

		results[i] = gqlEntity
	}

	// --- Batch load all linked entities at once ---
	if len(linkedIDSet) > 0 {
		allLinkedIDs := make([]string, 0, len(linkedIDSet))
		for id := range linkedIDSet {
			allLinkedIDs = append(allLinkedIDs, id)
		}

		linkedEntities, err := r.EntitiesByIDs(ctx, allLinkedIDs)
		if err != nil {
			partialErrs = append(partialErrs, err)
		}

		// Map linked ID -> entity
		linkedMap := make(map[string]*graph.Entity)
		for _, le := range linkedEntities {
			linkedMap[le.ID] = le
		}

		// Assign linked entities to each parent entity
		for _, parent := range results {
			if parent == nil {
				continue
			}
			var props map[string]interface{}
			err := json.Unmarshal([]byte(parent.Properties), &props)
			if err == nil {
				if idsRaw, ok := props["linked_ids"]; ok {
					if idsSlice, ok := idsRaw.([]interface{}); ok {
						var assigned []*graph.Entity
						for _, idVal := range idsSlice {
							if idStr, ok := idVal.(string); ok {
								if le, found := linkedMap[idStr]; found {
									assigned = append(assigned, le)
								}
							}
						}
						parent.LinkedEntities = assigned
					}
				}
			}
		}
	}

	// Return results with any partial errors
	if len(partialErrs) > 0 {
		errMsg := "partial errors occurred: "
		for _, e := range partialErrs {
			errMsg += e.Error() + "; "
		}
		return results, fmt.Errorf("%s", errMsg)
	}

	return results, nil
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
	entities, _, err := r.entityRepo.List(ctx, orgID, 10, 0)
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
	entities, _, err := r.entityRepo.List(ctx, orgID, 10, 0)
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
	entities, _, err := r.entityRepo.List(ctx, orgID, 10, 0)
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

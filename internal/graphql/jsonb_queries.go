package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"graphql-engineering-api/graph"
	"graphql-engineering-api/internal/domain"

	"github.com/google/uuid"
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

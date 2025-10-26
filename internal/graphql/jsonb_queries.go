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

type linkIdentifier struct {
	id         uuid.UUID
	hasID      bool
	reference  string
	entityType string
}

type referenceGroupKey struct {
	orgID      uuid.UUID
	entityType string
}

type referenceFieldCacheEntry struct {
	fieldName string
	found     bool
}

func (li linkIdentifier) cacheKey() string {
	if li.hasID {
		return "id:" + li.id.String()
	}
	return "ref:" + strings.ToLower(li.entityType) + ":" + li.reference
}

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

func (r *Resolver) mapDomainEntity(ctx context.Context, e domain.Entity) (*graph.Entity, error) {
	propsJSON, err := e.GetPropertiesAsJSONB()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties for entity %s: %w", e.ID, err)
	}

	gqlEntity := &graph.Entity{
		ID:             e.ID.String(),
		OrganizationID: e.OrganizationID.String(),
		SchemaID:       e.SchemaID.String(),
		EntityType:     e.EntityType,
		Path:           e.Path,
		Properties:     string(propsJSON),
		Version:        int(e.Version),
		CreatedAt:      e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
	}

	if ref, err := r.referenceValueFromEntity(ctx, e); err == nil {
		gqlEntity.ReferenceValue = ref
	} else {
		return nil, err
	}

	return gqlEntity, nil
}

func (r *Resolver) referenceValueFromEntity(ctx context.Context, entity domain.Entity) (*string, error) {
	fallback := entity.ID.String()

	if strings.TrimSpace(entity.EntityType) == "" || entity.OrganizationID == uuid.Nil {
		return &fallback, nil
	}

	if entity.Properties == nil {
		return &fallback, nil
	}

	fieldName, found, err := r.referenceFieldNameForType(ctx, nil, entity.OrganizationID, entity.EntityType)
	if err != nil {
		return nil, err
	}
	if !found {
		return &fallback, nil
	}

	raw, exists := entity.Properties[fieldName]
	if !exists {
		return &fallback, nil
	}

	value, ok := raw.(string)
	if !ok {
		return &fallback, nil
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return &fallback, nil
	}

	return &trimmed, nil
}

func collectLinkedEntityIDs(props map[string]any, schema *domain.EntitySchema) []linkIdentifier {
	if props == nil {
		return nil
	}

	var result []linkIdentifier
	seen := make(map[string]struct{})

	fieldsByName := make(map[string]domain.FieldDefinition)
	if schema != nil {
		for _, field := range schema.Fields {
			fieldsByName[strings.ToLower(field.Name)] = field
		}
	}

	for key, rawValue := range props {
		normalizedValues := normalizeLinkedIDValues(rawValue)
		if len(normalizedValues) == 0 {
			continue
		}

		lowerKey := strings.ToLower(key)
		fieldDef, found := fieldsByName[lowerKey]
		if !found && !isLinkedFieldName(key) {
			continue
		}

		preferReference := false
		switch fieldDef.Type {
		case domain.FieldTypeEntityReference, domain.FieldTypeEntityReferenceArray, domain.FieldTypeReference:
			preferReference = true
		}

		targetType := strings.TrimSpace(fieldDef.ReferenceEntityType)
		if !found && schema != nil && targetType == "" {
			targetType = schema.Name
		}

		for _, value := range normalizedValues {
			identifier, ok := buildLinkIdentifier(value, targetType, preferReference)
			if !ok {
				continue
			}
			cacheKey := identifier.cacheKey()
			if _, exists := seen[cacheKey]; exists {
				continue
			}
			seen[cacheKey] = struct{}{}
			result = append(result, identifier)
		}
	}

	return result
}

func buildLinkIdentifier(value string, targetType string, preferReference bool) (linkIdentifier, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return linkIdentifier{}, false
	}

	if preferReference {
		if targetType == "" {
			return linkIdentifier{}, false
		}
		return linkIdentifier{reference: trimmed, entityType: targetType}, true
	}

	if id, err := uuid.Parse(trimmed); err == nil {
		return linkIdentifier{id: id, hasID: true, entityType: targetType}, true
	}

	if targetType == "" {
		return linkIdentifier{}, false
	}

	return linkIdentifier{reference: trimmed, entityType: targetType}, true
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

func appendUniqueLinkedEntity(parent *graph.Entity, child *graph.Entity) {
	if parent == nil || child == nil {
		return
	}
	for _, existing := range parent.LinkedEntities {
		if existing != nil && existing.ID == child.ID {
			return
		}
	}
	parent.LinkedEntities = append(parent.LinkedEntities, child)
}

func referenceCacheKey(orgID uuid.UUID, entityType, reference string) string {
	return "ref:" + orgID.String() + ":" + strings.ToLower(entityType) + ":" + reference
}

func (r *Resolver) referenceFieldNameForType(
	ctx context.Context,
	cache map[referenceGroupKey]referenceFieldCacheEntry,
	orgID uuid.UUID,
	entityType string,
) (string, bool, error) {
	key := referenceGroupKey{orgID: orgID, entityType: strings.ToLower(entityType)}
	if cache != nil {
		if entry, ok := cache[key]; ok {
			return entry.fieldName, entry.found, nil
		}
	} else if cached, ok := r.referenceFieldCache.Load(key); ok {
		entry := cached.(referenceFieldCacheEntry)
		return entry.fieldName, entry.found, nil
	}

	schema, err := r.entitySchemaRepo.GetByName(ctx, orgID, entityType)
	if err != nil {
		return "", false, fmt.Errorf("failed to load schema for %s: %w", entityType, err)
	}

	fieldName := ""
	found := false
	for _, field := range schema.Fields {
		if field.Type == domain.FieldTypeReference {
			fieldName = field.Name
			found = true
			break
		}
	}

	entry := referenceFieldCacheEntry{fieldName: fieldName, found: found}
	if cache != nil {
		cache[key] = entry
	} else {
		r.referenceFieldCache.Store(key, entry)
	}
	return fieldName, found, nil
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
		mapped, err := r.mapDomainEntity(ctx, entity)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
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

			gqlEntity, err := r.mapDomainEntity(ctx, entity)
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
	referenceFieldCache := make(map[referenceGroupKey]referenceFieldCacheEntry)
	idParents := make(map[string][]*graph.Entity)
	referenceParents := make(map[referenceGroupKey]map[string][]*graph.Entity)
	referenceGroupTypes := make(map[referenceGroupKey]string)
	missingIDs := make(map[string]struct{})
	var errs []error

	for _, parent := range parents {
		if parent == nil {
			continue
		}

		if parent.LinkedEntities == nil {
			parent.LinkedEntities = []*graph.Entity{}
		}

		if parent.ID != "" {
			cache[parent.ID] = parent
		}

		var props map[string]any
		if err := json.Unmarshal([]byte(parent.Properties), &props); err != nil {
			errs = append(errs, fmt.Errorf("entity %s: invalid properties JSON: %w", parent.ID, err))
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

		if len(parent.LinkedEntities) > 0 {
			for _, child := range parent.LinkedEntities {
				if child != nil && child.ID != "" {
					cache[child.ID] = child
				}
			}
			continue
		}

		identifiers := collectLinkedEntityIDs(props, schema)
		if len(identifiers) == 0 {
			continue
		}

		var orgUUID uuid.UUID
		var orgParsed bool
		if parent.OrganizationID != "" {
			if parsed, err := uuid.Parse(parent.OrganizationID); err == nil {
				orgUUID = parsed
				orgParsed = true
			}
		}

		for _, identifier := range identifiers {
			if identifier.hasID {
				idKey := identifier.id.String()
				if child, ok := cache[idKey]; ok && child != nil {
					appendUniqueLinkedEntity(parent, child)
					continue
				}
				missingIDs[idKey] = struct{}{}
				idParents[idKey] = append(idParents[idKey], parent)
				continue
			}

			if identifier.reference == "" {
				continue
			}
			if !orgParsed {
				errs = append(errs, fmt.Errorf("entity %s missing valid organization id for reference %q", parent.ID, identifier.reference))
				continue
			}
			if identifier.entityType == "" {
				errs = append(errs, fmt.Errorf("entity %s link to reference %q lacks target entity type", parent.ID, identifier.reference))
				continue
			}

			refKey := referenceCacheKey(orgUUID, identifier.entityType, identifier.reference)
			if child, ok := cache[refKey]; ok && child != nil {
				appendUniqueLinkedEntity(parent, child)
				continue
			}

			group := referenceGroupKey{orgID: orgUUID, entityType: strings.ToLower(identifier.entityType)}
			if _, ok := referenceGroupTypes[group]; !ok {
				referenceGroupTypes[group] = identifier.entityType
			}
			if referenceParents[group] == nil {
				referenceParents[group] = make(map[string][]*graph.Entity)
			}
			referenceParents[group][identifier.reference] = append(referenceParents[group][identifier.reference], parent)
		}
	}

	if len(missingIDs) == 0 && len(referenceParents) == 0 {
		return combineErrors(errs)
	}

	if len(missingIDs) > 0 {
		missing := make([]string, 0, len(missingIDs))
		for id := range missingIDs {
			missing = append(missing, id)
		}

		linkedEntities, err := r.EntitiesByIDs(ctx, missing)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed loading linked entities: %w", err))
		} else {
			for _, entity := range linkedEntities {
				if entity != nil && entity.ID != "" {
					cache[entity.ID] = entity
				}
			}
		}

		for id, parents := range idParents {
			if child, ok := cache[id]; ok && child != nil {
				for _, parent := range parents {
					appendUniqueLinkedEntity(parent, child)
				}
			}
		}
	}

	for group, refMap := range referenceParents {
		actualType := referenceGroupTypes[group]
		if actualType == "" {
			continue
		}

		refField, found, err := r.referenceFieldNameForType(ctx, referenceFieldCache, group.orgID, actualType)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !found {
			errs = append(errs, fmt.Errorf("entity type %s does not declare a reference field", actualType))
			continue
		}

		references := make([]string, 0, len(refMap))
		for ref := range refMap {
			references = append(references, ref)
		}

		domainEntities, err := r.entityRepo.ListByReferences(ctx, group.orgID, actualType, references)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed loading %s references: %w", actualType, err))
			continue
		}

		resolved := make(map[string]*graph.Entity, len(domainEntities))
		for _, entity := range domainEntities {
			mapped, err := r.mapDomainEntity(ctx, entity)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			cache[mapped.ID] = mapped

			refValue := ""
			if val, ok := entity.Properties[refField]; ok {
				if str, ok := val.(string); ok {
					refValue = strings.TrimSpace(str)
				}
			}
			if refValue != "" {
				refKey := referenceCacheKey(group.orgID, actualType, refValue)
				cache[refKey] = mapped
				resolved[refValue] = mapped
			}
		}

		for refValue, parents := range refMap {
			if child, ok := resolved[refValue]; ok {
				for _, parent := range parents {
					appendUniqueLinkedEntity(parent, child)
				}
			} else {
				errs = append(errs, fmt.Errorf("no %s entity found for reference %q", actualType, refValue))
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
		mapped, err := r.mapDomainEntity(ctx, entity)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
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
        entities, _, err := r.entityRepo.List(ctx, orgID, nil, nil, 10, 0)
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
		mapped, err := r.mapDomainEntity(ctx, entity)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
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
        entities, _, err := r.entityRepo.List(ctx, orgID, nil, nil, 10, 0)
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
		mapped, err := r.mapDomainEntity(ctx, entity)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
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
        entities, _, err := r.entityRepo.List(ctx, orgID, nil, nil, 10, 0)
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
		mapped, err := r.mapDomainEntity(ctx, entity)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
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

package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"graphql-engineering-api/graph"
	"graphql-engineering-api/internal/domain"
	"graphql-engineering-api/pkg/validator"

	"github.com/google/uuid"
)

func toGraphFieldDefinition(field domain.FieldDefinition) *graph.FieldDefinition {
	desc := field.Description
	description := &desc

	def := field.Default
	defaultValue := &def

	val := field.Validation
	validation := &val

	var referenceType *string
	if field.ReferenceEntityType != "" {
		ref := field.ReferenceEntityType
		referenceType = &ref
	}

	return &graph.FieldDefinition{
		Name:                field.Name,
		Type:                graph.FieldType(field.Type),
		Required:            field.Required,
		Description:         description,
		Default:             defaultValue,
		Validation:          validation,
		ReferenceEntityType: referenceType,
	}
}

var linkedFieldCandidates = []string{
	"linked_ids",
	"linkedIds",
	"linked_entities",
	"linkedEntities",
	"linked_entity_id",
	"linkedEntityId",
	"linked_entity_ids",
	"linkedEntityIds",
}

func findLinkedFieldDefinition(fields []domain.FieldDefinition) (string, domain.FieldType, bool) {
	for _, candidate := range linkedFieldCandidates {
		for i := range fields {
			if strings.EqualFold(fields[i].Name, candidate) {
				return fields[i].Name, fields[i].Type, true
			}
		}
	}
	return "", "", false
}

func isLinkedFieldName(name string) bool {
	for _, candidate := range linkedFieldCandidates {
		if strings.EqualFold(candidate, name) {
			return true
		}
	}
	return false
}

func ensureLinkedEntityProperties(properties map[string]any, fieldName string, fieldType domain.FieldType, linkedIDs []string) error {
	if properties == nil {
		return nil
	}

	trimmed := make([]string, 0, len(linkedIDs))
	for _, id := range linkedIDs {
		if s := strings.TrimSpace(id); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 {
		return nil
	}

	switch fieldType {
	case domain.FieldTypeEntityReference:
		if len(trimmed) > 1 {
			return fmt.Errorf("linkedEntityIds provided but schema field %s expects a single reference", fieldName)
		}
		properties[fieldName] = trimmed[0]
	default:
		current := normalizeLinkedIDValues(properties[fieldName])
		current = append(current, trimmed...)
		properties[fieldName] = uniqueOrderedStrings(current)
	}
	return nil
}

func mergeLinkedIDsIntoProperties(properties map[string]any, linkedIDs []string) {
	if properties == nil {
		return
	}

	existing := normalizeLinkedIDValues(properties["linked_ids"])
	existing = append(existing, linkedIDs...)
	properties["linked_ids"] = uniqueOrderedStrings(existing)
}

func normalizeLinkedIDValues(value any) []string {
	switch v := value.(type) {
	case nil:
		return []string{}
	case string:
		id := strings.TrimSpace(v)
		if id == "" {
			return []string{}
		}
		return []string{id}
	case []string:
		return uniqueOrderedStrings(v)
	case []any:
		var collected []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					collected = append(collected, trimmed)
				}
			}
		}
		return uniqueOrderedStrings(collected)
	default:
		return []string{}
	}
}

func gatherRequestedLinkedIDs(input graph.CreateEntityInput) []string {
	var ids []string
	if input.LinkedEntityID != nil {
		if trimmed := strings.TrimSpace(*input.LinkedEntityID); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	for _, raw := range input.LinkedEntityIds {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return uniqueOrderedStrings(ids)
}

func uniqueOrderedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var result []string
	for _, val := range values {
		if val == "" {
			continue
		}
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		result = append(result, val)
	}
	return result
}

// CreateOrganization creates a new organization
func (r *Resolver) CreateOrganization(ctx context.Context, input graph.CreateOrganizationInput) (*graph.Organization, error) {
	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	org := domain.NewOrganization(input.Name, description)

	createdOrg, err := r.orgRepo.Create(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return &graph.Organization{
		ID:          createdOrg.ID.String(),
		Name:        createdOrg.Name,
		Description: &createdOrg.Description,
		CreatedAt:   createdOrg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   createdOrg.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateOrganization updates an existing organization
func (r *Resolver) UpdateOrganization(ctx context.Context, input graph.UpdateOrganizationInput) (*graph.Organization, error) {
	orgID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Get existing organization
	existingOrg, err := r.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Apply updates using immutable pattern
	updatedOrg := existingOrg
	if input.Name != nil {
		updatedOrg = updatedOrg.WithName(*input.Name)
	}
	if input.Description != nil {
		updatedOrg = updatedOrg.WithDescription(*input.Description)
	}

	// Save updated organization
	savedOrg, err := r.orgRepo.Update(ctx, updatedOrg)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return &graph.Organization{
		ID:          savedOrg.ID.String(),
		Name:        savedOrg.Name,
		Description: &savedOrg.Description,
		CreatedAt:   savedOrg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   savedOrg.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteOrganization deletes an organization
func (r *Resolver) DeleteOrganization(ctx context.Context, id string) (*bool, error) {
	orgID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	if err := r.orgRepo.Delete(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to delete organization: %w", err)
	}

	result := true
	return &result, nil
}

// CreateEntitySchema creates a new entity schema
func (r *Resolver) CreateEntitySchema(ctx context.Context, input graph.CreateEntitySchemaInput) (*graph.EntitySchema, error) {
	orgID, err := uuid.Parse(input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	// Convert input fields to domain fields
	fields := make([]domain.FieldDefinition, 0, len(input.Fields))
	for _, fieldInput := range input.Fields {
		fieldDesc := ""
		if fieldInput.Description != nil {
			fieldDesc = *fieldInput.Description
		}

		required := false
		if fieldInput.Required != nil {
			required = *fieldInput.Required
		}

		defaultValue := ""
		if fieldInput.Default != nil {
			defaultValue = *fieldInput.Default
		}

		validation := ""
		if fieldInput.Validation != nil {
			validation = *fieldInput.Validation
		}

		refEntityType := ""
		if fieldInput.ReferenceEntityType != nil {
			refEntityType = *fieldInput.ReferenceEntityType
		}

		fields = append(fields, domain.FieldDefinition{
			Name:                fieldInput.Name,
			Type:                domain.FieldType(fieldInput.Type),
			Required:            required,
			Description:         fieldDesc,
			Default:             defaultValue,
			Validation:          validation,
			ReferenceEntityType: refEntityType,
		})
	}

	schema := domain.NewEntitySchema(orgID, input.Name, description, fields)

	createdSchema, err := r.entitySchemaRepo.Create(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity schema: %w", err)
	}

	// Convert back to GraphQL format
	gqlFields := make([]*graph.FieldDefinition, 0, len(createdSchema.Fields))
	for _, field := range createdSchema.Fields {
		gqlFields = append(gqlFields, toGraphFieldDefinition(field))
	}

	return &graph.EntitySchema{
		ID:             createdSchema.ID.String(),
		OrganizationID: createdSchema.OrganizationID.String(),
		Name:           createdSchema.Name,
		Description:    &createdSchema.Description,
		Fields:         gqlFields,
		CreatedAt:      createdSchema.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      createdSchema.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateEntitySchema updates an existing entity schema
func (r *Resolver) UpdateEntitySchema(ctx context.Context, input graph.UpdateEntitySchemaInput) (*graph.EntitySchema, error) {
	schemaID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID: %w", err)
	}

	// Get existing schema
	existingSchema, err := r.entitySchemaRepo.GetByID(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity schema: %w", err)
	}

	// Apply updates using immutable pattern
	updatedSchema := existingSchema
	if input.Name != nil {
		updatedSchema = updatedSchema.WithName(*input.Name)
	}
	if input.Description != nil {
		updatedSchema = updatedSchema.WithDescription(*input.Description)
	}
	if input.Fields != nil {
		// Replace all fields
		newFields := make([]domain.FieldDefinition, 0, len(input.Fields))
		for _, fieldInput := range input.Fields {
			fieldDesc := ""
			if fieldInput.Description != nil {
				fieldDesc = *fieldInput.Description
			}

			required := false
			if fieldInput.Required != nil {
				required = *fieldInput.Required
			}

			defaultValue := ""
			if fieldInput.Default != nil {
				defaultValue = *fieldInput.Default
			}

			validation := ""
			if fieldInput.Validation != nil {
				validation = *fieldInput.Validation
			}

			refEntityType := ""
			if fieldInput.ReferenceEntityType != nil {
				refEntityType = *fieldInput.ReferenceEntityType
			}

			newFields = append(newFields, domain.FieldDefinition{
				Name:                fieldInput.Name,
				Type:                domain.FieldType(fieldInput.Type),
				Required:            required,
				Description:         fieldDesc,
				Default:             defaultValue,
				Validation:          validation,
				ReferenceEntityType: refEntityType,
			})
		}
		updatedSchema = domain.NewEntitySchema(updatedSchema.OrganizationID, updatedSchema.Name, updatedSchema.Description, newFields)
		updatedSchema.ID = schemaID // Preserve the ID
	}

	// Save updated schema
	savedSchema, err := r.entitySchemaRepo.Update(ctx, updatedSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to update entity schema: %w", err)
	}

	// Convert back to GraphQL format
	gqlFields := make([]*graph.FieldDefinition, 0, len(savedSchema.Fields))
	for _, field := range savedSchema.Fields {
		gqlFields = append(gqlFields, toGraphFieldDefinition(field))
	}

	return &graph.EntitySchema{
		ID:             savedSchema.ID.String(),
		OrganizationID: savedSchema.OrganizationID.String(),
		Name:           savedSchema.Name,
		Description:    &savedSchema.Description,
		Fields:         gqlFields,
		CreatedAt:      savedSchema.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      savedSchema.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteEntitySchema deletes an entity schema
func (r *Resolver) DeleteEntitySchema(ctx context.Context, id string) (*bool, error) {
	schemaID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID: %w", err)
	}

	if err := r.entitySchemaRepo.Delete(ctx, schemaID); err != nil {
		return nil, fmt.Errorf("failed to delete entity schema: %w", err)
	}

	result := true
	return &result, nil
}

// AddFieldToSchema adds a field to an existing entity schema
func (r *Resolver) AddFieldToSchema(ctx context.Context, schemaID string, field graph.FieldDefinitionInput) (*graph.EntitySchema, error) {
	schemaUUID, err := uuid.Parse(schemaID)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID: %w", err)
	}

	// Get existing schema
	existingSchema, err := r.entitySchemaRepo.GetByID(ctx, schemaUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity schema: %w", err)
	}

	// Add the new field
	fieldDesc := ""
	if field.Description != nil {
		fieldDesc = *field.Description
	}

	required := false
	if field.Required != nil {
		required = *field.Required
	}

	defaultValue := ""
	if field.Default != nil {
		defaultValue = *field.Default
	}

	validation := ""
	if field.Validation != nil {
		validation = *field.Validation
	}

	fieldDef := domain.FieldDefinition{
		Name:        field.Name,
		Type:        domain.FieldType(field.Type),
		Required:    required,
		Description: fieldDesc,
		Default:     defaultValue,
		Validation:  validation,
	}

	updatedSchema := existingSchema.WithField(fieldDef)

	// Save updated schema
	savedSchema, err := r.entitySchemaRepo.Update(ctx, updatedSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to update entity schema: %w", err)
	}

	// Convert back to GraphQL format
	gqlFields := make([]*graph.FieldDefinition, 0, len(savedSchema.Fields))
	for _, field := range savedSchema.Fields {
		gqlFields = append(gqlFields, toGraphFieldDefinition(field))
	}

	return &graph.EntitySchema{
		ID:             savedSchema.ID.String(),
		OrganizationID: savedSchema.OrganizationID.String(),
		Name:           savedSchema.Name,
		Description:    &savedSchema.Description,
		Fields:         gqlFields,
		CreatedAt:      savedSchema.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      savedSchema.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// RemoveFieldFromSchema removes a field from an existing entity schema
func (r *Resolver) RemoveFieldFromSchema(ctx context.Context, schemaID, fieldName string) (*graph.EntitySchema, error) {
	schemaUUID, err := uuid.Parse(schemaID)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID: %w", err)
	}

	// Get existing schema
	existingSchema, err := r.entitySchemaRepo.GetByID(ctx, schemaUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity schema: %w", err)
	}

	// Remove the field
	updatedSchema := existingSchema.WithoutField(fieldName)

	// Save updated schema
	savedSchema, err := r.entitySchemaRepo.Update(ctx, updatedSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to update entity schema: %w", err)
	}

	// Convert back to GraphQL format
	gqlFields := make([]*graph.FieldDefinition, 0, len(savedSchema.Fields))
	for _, field := range savedSchema.Fields {
		gqlFields = append(gqlFields, toGraphFieldDefinition(field))
	}

	return &graph.EntitySchema{
		ID:             savedSchema.ID.String(),
		OrganizationID: savedSchema.OrganizationID.String(),
		Name:           savedSchema.Name,
		Description:    &savedSchema.Description,
		Fields:         gqlFields,
		CreatedAt:      savedSchema.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      savedSchema.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// CreateEntity creates a new entity
func (r *Resolver) CreateEntity(ctx context.Context, input graph.CreateEntityInput) (*graph.Entity, error) {
	orgID, err := uuid.Parse(input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Parse properties JSON
	var properties map[string]any
	if err := json.Unmarshal([]byte(input.Properties), &properties); err != nil {
		return nil, fmt.Errorf("invalid properties JSON: %w", err)
	}
	if properties == nil {
		properties = make(map[string]any)
	}

	path := ""
	if input.Path != nil {
		path = *input.Path
	}

	fieldDefinitions, err := r.entitySchemaRepo.GetByName(ctx, orgID, input.EntityType)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema for entity type %s: %w", input.EntityType, err)
	}

	requestedLinkedIDs := gatherRequestedLinkedIDs(input)
	if len(requestedLinkedIDs) > 0 {
		if fieldName, fieldType, found := findLinkedFieldDefinition(fieldDefinitions.Fields); found {
			if err := ensureLinkedEntityProperties(properties, fieldName, fieldType, requestedLinkedIDs); err != nil {
				return nil, err
			}
		}
		mergeLinkedIDsIntoProperties(properties, requestedLinkedIDs)
	}

	// Convert schema fields slice -> map[string]FieldDefinition
	fieldDefsMap := make(map[string]validator.FieldDefinition)
	for _, f := range fieldDefinitions.Fields {
		var refType *string
		if f.ReferenceEntityType != "" {
			ref := f.ReferenceEntityType
			refType = &ref
		}

		fieldDefsMap[f.Name] = validator.FieldDefinition{
			Type:                graph.FieldType(strings.ToUpper(string(f.Type))),
			Required:            f.Required,
			Description:         f.Description,
			Default:             f.Default,
			Validation:          f.Validation,
			ReferenceEntityType: refType,
		}
	}

	if _, exists := properties["linked_ids"]; exists {
		if _, ok := fieldDefsMap["linked_ids"]; !ok {
			fieldDefsMap["linked_ids"] = validator.FieldDefinition{
				Type:     graph.FieldTypeEntityReferenceArray,
				Required: false,
			}
		}
	}

	validator := validator.NewJSONBValidator()
	result := validator.ValidateProperties(properties, fieldDefsMap)
	if !result.IsValid {
		return nil, fmt.Errorf("validation failed: %s", result.Errors)
	}

	entity := domain.NewEntity(orgID, input.EntityType, path, properties)

	createdEntity, err := r.entityRepo.Create(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	propertiesJSON, err := createdEntity.GetPropertiesAsJSONB()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &graph.Entity{
		ID:             createdEntity.ID.String(),
		OrganizationID: createdEntity.OrganizationID.String(),
		EntityType:     createdEntity.EntityType,
		Path:           createdEntity.Path,
		Properties:     string(propertiesJSON),
		CreatedAt:      createdEntity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      createdEntity.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateEntity updates an existing entity
func (r *Resolver) UpdateEntity(ctx context.Context, input graph.UpdateEntityInput) (*graph.Entity, error) {
	entityID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	// Get existing entity
	existingEntity, err := r.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Apply updates using immutable pattern
	updatedEntity := existingEntity
	if input.EntityType != nil {
		updatedEntity = updatedEntity.WithEntityType(*input.EntityType)
	}
	if input.Path != nil {
		updatedEntity = updatedEntity.WithPath(*input.Path)
	}
	if input.Properties != nil {
		// Parse properties JSON
		var properties map[string]any
		if err := json.Unmarshal([]byte(*input.Properties), &properties); err != nil {
			return nil, fmt.Errorf("invalid properties JSON: %w", err)
		}
		updatedEntity = updatedEntity.WithProperties(properties)
	}

	// Save updated entity
	savedEntity, err := r.entityRepo.Update(ctx, updatedEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to update entity: %w", err)
	}

	propertiesJSON, err := savedEntity.GetPropertiesAsJSONB()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &graph.Entity{
		ID:             savedEntity.ID.String(),
		OrganizationID: savedEntity.OrganizationID.String(),
		EntityType:     savedEntity.EntityType,
		Path:           savedEntity.Path,
		Properties:     string(propertiesJSON),
		CreatedAt:      savedEntity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      savedEntity.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteEntity deletes an entity
func (r *Resolver) DeleteEntity(ctx context.Context, id string) (*bool, error) {
	entityID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID: %w", err)
	}

	if err := r.entityRepo.Delete(ctx, entityID); err != nil {
		return nil, fmt.Errorf("failed to delete entity: %w", err)
	}

	result := true
	return &result, nil
}

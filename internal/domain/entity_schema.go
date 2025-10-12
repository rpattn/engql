package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FieldType represents the type of a field in an entity schema
type FieldType string

const (
	FieldTypeString               FieldType = "string"
	FieldTypeInteger              FieldType = "integer"
	FieldTypeFloat                FieldType = "float"
	FieldTypeBoolean              FieldType = "boolean"
	FieldTypeTimestamp            FieldType = "timestamp"
	FieldTypeJSON                 FieldType = "json"
	FieldTypeFileRef              FieldType = "file_reference"
	FieldTypeGeometry             FieldType = "geometry"
	FieldTypeTimeseries           FieldType = "timeseries"
	FieldTypeEntityReference      FieldType = "ENTITY_REFERENCE"
	FieldTypeEntityReferenceArray FieldType = "ENTITY_REFERENCE_ARRAY"
)

// FieldDefinition represents a field definition in a schema
type FieldDefinition struct {
	Name        string    `json:"name"`
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Description string    `json:"description,omitempty"`
	Default     string    `json:"default,omitempty"`
	Validation  string    `json:"validation,omitempty"` // Custom validation rules
}

// EntitySchema represents a schema definition for entity types
type EntitySchema struct {
	ID             uuid.UUID         `json:"id"`
	OrganizationID uuid.UUID         `json:"organization_id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Fields         []FieldDefinition `json:"fields"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// NewEntitySchema creates a new entity schema with immutable pattern
func NewEntitySchema(organizationID uuid.UUID, name, description string, fields []FieldDefinition) EntitySchema {
	now := time.Now()
	return EntitySchema{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           name,
		Description:    description,
		Fields:         copyFields(fields), // Deep copy to ensure immutability
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// WithField returns a new schema with an added/updated field
func (es EntitySchema) WithField(field FieldDefinition) EntitySchema {
	newFields := copyFields(es.Fields)

	// Check if field already exists and update it, otherwise append
	found := false
	for i, existingField := range newFields {
		if existingField.Name == field.Name {
			newFields[i] = field
			found = true
			break
		}
	}

	if !found {
		newFields = append(newFields, field)
	}

	return EntitySchema{
		ID:             es.ID,
		OrganizationID: es.OrganizationID,
		Name:           es.Name,
		Description:    es.Description,
		Fields:         newFields,
		CreatedAt:      es.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// WithoutField returns a new schema without the specified field
func (es EntitySchema) WithoutField(name string) EntitySchema {
	newFields := make([]FieldDefinition, 0, len(es.Fields))
	for _, field := range es.Fields {
		if field.Name != name {
			newFields = append(newFields, field)
		}
	}

	return EntitySchema{
		ID:             es.ID,
		OrganizationID: es.OrganizationID,
		Name:           es.Name,
		Description:    es.Description,
		Fields:         newFields,
		CreatedAt:      es.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// WithDescription returns a new schema with updated description
func (es EntitySchema) WithDescription(description string) EntitySchema {
	return EntitySchema{
		ID:             es.ID,
		OrganizationID: es.OrganizationID,
		Name:           es.Name,
		Description:    description,
		Fields:         copyFields(es.Fields),
		CreatedAt:      es.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// WithName returns a new schema with updated name
func (es EntitySchema) WithName(name string) EntitySchema {
	return EntitySchema{
		ID:             es.ID,
		OrganizationID: es.OrganizationID,
		Name:           name,
		Description:    es.Description,
		Fields:         copyFields(es.Fields),
		CreatedAt:      es.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// GetFieldsAsJSONB returns the fields as JSONB for database storage
func (es EntitySchema) GetFieldsAsJSONB() (json.RawMessage, error) {
	return json.Marshal(es.Fields)
}

// FromJSONB creates an EntitySchema from JSONB data
func FromJSONBFields(fieldsJSON json.RawMessage) ([]FieldDefinition, error) {
	var fields []FieldDefinition
	err := json.Unmarshal(fieldsJSON, &fields)
	return fields, err
}

// copyFields creates a deep copy of the fields slice to ensure immutability
func copyFields(fields []FieldDefinition) []FieldDefinition {
	if fields == nil {
		return nil
	}
	newFields := make([]FieldDefinition, len(fields))
	copy(newFields, fields)
	return newFields
}

package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FieldType represents the type of a field in an entity schema
type FieldType string

const (
	FieldTypeString     FieldType = "string"
	FieldTypeInteger    FieldType = "integer"
	FieldTypeFloat      FieldType = "float"
	FieldTypeBoolean    FieldType = "boolean"
	FieldTypeTimestamp  FieldType = "timestamp"
	FieldTypeJSON       FieldType = "json"
	FieldTypeFileRef    FieldType = "file_reference"
	FieldTypeGeometry   FieldType = "geometry"
	FieldTypeTimeseries FieldType = "timeseries"
	// FieldTypeReference marks the canonical cross-entity reference string. Only one
	// field with this type may exist per schema. When a schema wants to
	// associate the reference to another entity type it may do so via
	// ReferenceEntityType, but the association is optional.
	FieldTypeReference            FieldType = "REFERENCE"
	FieldTypeEntityReference      FieldType = "ENTITY_REFERENCE"
	FieldTypeEntityReferenceArray FieldType = "ENTITY_REFERENCE_ARRAY"
	FieldTypeEntityID             FieldType = "ENTITY_ID"
)

// FieldDefinition represents a field definition in a schema
type FieldDefinition struct {
	Name        string    `json:"name"`
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Description string    `json:"description,omitempty"`
	Default     string    `json:"default,omitempty"`
	Validation  string    `json:"validation,omitempty"` // Custom validation rules
	// ReferenceEntityType specifies the related entity type when the field holds a
	// relationship (ENTITY_REFERENCE, ENTITY_REFERENCE_ARRAY, ENTITY_ID, or
	// REFERENCE). FieldTypeReference values may omit the association when the
	// reference is standalone.
	ReferenceEntityType string `json:"referenceEntityType,omitempty"`
}

// ReferenceFieldSet captures all REFERENCE-typed fields for a schema along with
// the canonical entry used for `referenceValue`. The canonical reference is the
// first REFERENCE field that appears in the schema definition.
type ReferenceFieldSet struct {
	fields []FieldDefinition
}

// NewReferenceFieldSet constructs a ReferenceFieldSet from the provided field
// definitions while preserving declaration order.
func NewReferenceFieldSet(fields []FieldDefinition) ReferenceFieldSet {
	set := ReferenceFieldSet{}
	for _, field := range fields {
		if field.Type == FieldTypeReference {
			set.fields = append(set.fields, field)
		}
	}
	return set
}

// CanonicalField returns the schema's canonical REFERENCE field and a boolean
// indicating whether one exists.
func (s ReferenceFieldSet) CanonicalField() (FieldDefinition, bool) {
	if len(s.fields) == 0 {
		return FieldDefinition{}, false
	}
	return s.fields[0], true
}

// CanonicalName returns the name of the canonical REFERENCE field.
func (s ReferenceFieldSet) CanonicalName() (string, bool) {
	field, ok := s.CanonicalField()
	if !ok {
		return "", false
	}
	return field.Name, true
}

// Fields returns a defensive copy of the REFERENCE field definitions in
// declaration order.
func (s ReferenceFieldSet) Fields() []FieldDefinition {
	if len(s.fields) == 0 {
		return nil
	}
	clone := make([]FieldDefinition, len(s.fields))
	copy(clone, s.fields)
	return clone
}

// Names returns the ordered list of REFERENCE field names.
func (s ReferenceFieldSet) Names() []string {
	fields := s.Fields()
	if len(fields) == 0 {
		return nil
	}
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.Name != "" {
			names = append(names, field.Name)
		}
	}
	return names
}

// SchemaStatus represents lifecycle status of a schema version.
type SchemaStatus string

const (
	SchemaStatusActive     SchemaStatus = "ACTIVE"
	SchemaStatusDeprecated SchemaStatus = "DEPRECATED"
	SchemaStatusArchived   SchemaStatus = "ARCHIVED"
	SchemaStatusDraft      SchemaStatus = "DRAFT"
)

// CompatibilityLevel represents semantic version compatibility.
type CompatibilityLevel string

const (
	CompatibilityPatch CompatibilityLevel = "patch"
	CompatibilityMinor CompatibilityLevel = "minor"
	CompatibilityMajor CompatibilityLevel = "major"
)

// EntitySchema represents a schema definition for entity types
type EntitySchema struct {
	ID                uuid.UUID         `json:"id"`
	OrganizationID    uuid.UUID         `json:"organization_id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Fields            []FieldDefinition `json:"fields"`
	Version           string            `json:"version"`
	PreviousVersionID *uuid.UUID        `json:"previous_version_id,omitempty"`
	Status            SchemaStatus      `json:"status"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
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
		Version:        "1.0.0",
		Status:         SchemaStatusActive,
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
		ID:                es.ID,
		OrganizationID:    es.OrganizationID,
		Name:              es.Name,
		Description:       es.Description,
		Fields:            newFields,
		Version:           es.Version,
		PreviousVersionID: es.PreviousVersionID,
		Status:            es.Status,
		CreatedAt:         es.CreatedAt,
		UpdatedAt:         time.Now(),
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
		ID:                es.ID,
		OrganizationID:    es.OrganizationID,
		Name:              es.Name,
		Description:       es.Description,
		Fields:            newFields,
		Version:           es.Version,
		PreviousVersionID: es.PreviousVersionID,
		Status:            es.Status,
		CreatedAt:         es.CreatedAt,
		UpdatedAt:         time.Now(),
	}
}

// WithDescription returns a new schema with updated description
func (es EntitySchema) WithDescription(description string) EntitySchema {
	return EntitySchema{
		ID:                es.ID,
		OrganizationID:    es.OrganizationID,
		Name:              es.Name,
		Description:       description,
		Fields:            copyFields(es.Fields),
		Version:           es.Version,
		PreviousVersionID: es.PreviousVersionID,
		Status:            es.Status,
		CreatedAt:         es.CreatedAt,
		UpdatedAt:         time.Now(),
	}
}

// WithName returns a new schema with updated name
func (es EntitySchema) WithName(name string) EntitySchema {
	return EntitySchema{
		ID:                es.ID,
		OrganizationID:    es.OrganizationID,
		Name:              name,
		Description:       es.Description,
		Fields:            copyFields(es.Fields),
		Version:           es.Version,
		PreviousVersionID: es.PreviousVersionID,
		Status:            es.Status,
		CreatedAt:         es.CreatedAt,
		UpdatedAt:         time.Now(),
	}
}

// WithStatus returns a new schema with updated status.
func (es EntitySchema) WithStatus(status SchemaStatus) EntitySchema {
	return EntitySchema{
		ID:                es.ID,
		OrganizationID:    es.OrganizationID,
		Name:              es.Name,
		Description:       es.Description,
		Fields:            copyFields(es.Fields),
		Version:           es.Version,
		PreviousVersionID: es.PreviousVersionID,
		Status:            status,
		CreatedAt:         es.CreatedAt,
		UpdatedAt:         time.Now(),
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

// ComputeNextVersion calculates the next semantic version number based on compatibility.
func ComputeNextVersion(current string, level CompatibilityLevel) (string, error) {
	if current == "" {
		current = "1.0.0"
	}

	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", current)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version: %w", err)
	}

	switch level {
	case CompatibilityMajor:
		major++
		minor = 0
		patch = 0
	case CompatibilityMinor:
		minor++
		patch = 0
	default:
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// DetermineCompatibility compares field definitions to assess change impact.
func DetermineCompatibility(oldFields, newFields []FieldDefinition) CompatibilityLevel {
	oldMap := make(map[string]FieldDefinition, len(oldFields))
	for _, f := range oldFields {
		oldMap[strings.ToLower(f.Name)] = f
	}

	newMap := make(map[string]FieldDefinition, len(newFields))
	for _, f := range newFields {
		newMap[strings.ToLower(f.Name)] = f
	}

	majorChange := false
	minorChange := false

	for key, oldField := range oldMap {
		newField, ok := newMap[key]
		if !ok {
			majorChange = true
			continue
		}

		if oldField.Type != newField.Type {
			majorChange = true
			continue
		}
		if oldField.Required && !newField.Required {
			minorChange = true
		}
		if !oldField.Required && newField.Required {
			majorChange = true
		}
		if !strings.EqualFold(oldField.ReferenceEntityType, newField.ReferenceEntityType) {
			majorChange = true
		}
	}

	for key, newField := range newMap {
		if _, ok := oldMap[key]; ok {
			continue
		}
		if newField.Required {
			majorChange = true
		} else {
			minorChange = true
		}
	}

	if majorChange {
		return CompatibilityMajor
	}
	if minorChange {
		return CompatibilityMinor
	}
	return CompatibilityPatch
}

// NewVersionFromExisting clones the schema as a new version entry.
func NewVersionFromExisting(previous EntitySchema, updated EntitySchema, compatibility CompatibilityLevel, status SchemaStatus) (EntitySchema, error) {
	nextVersion, err := ComputeNextVersion(previous.Version, compatibility)
	if err != nil {
		return EntitySchema{}, err
	}

	now := time.Now()
	prevID := previous.ID

	return EntitySchema{
		ID:                uuid.New(),
		OrganizationID:    previous.OrganizationID,
		Name:              updated.Name,
		Description:       updated.Description,
		Fields:            copyFields(updated.Fields),
		Version:           nextVersion,
		PreviousVersionID: &prevID,
		Status:            status,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

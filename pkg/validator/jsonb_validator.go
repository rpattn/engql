package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rpattn/engql/graph"

	"github.com/google/uuid"
)

// JSONBValidator handles validation of JSONB properties against field definitions
type JSONBValidator struct{}

// NewJSONBValidator creates a new JSONB validator
func NewJSONBValidator() *JSONBValidator {
	return &JSONBValidator{}
}

// FieldDefinition represents a field definition for validation
type FieldDefinition struct {
	Type                graph.FieldType `json:"type"`
	Required            bool            `json:"required"`
	Description         string          `json:"description,omitempty"`
	Default             any             `json:"default,omitempty"`
	Validation          any             `json:"validation,omitempty"`
	ReferenceEntityType *string         `json:"referenceEntityType,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	IsValid  bool              `json:"is_valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// ValidateProperties validates entity properties against field definitions
func (jv *JSONBValidator) ValidateProperties(properties map[string]any, fieldDefinitions map[string]FieldDefinition) ValidationResult {
	result := ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	for fieldName, fieldDef := range fieldDefinitions {
		value, exists := properties[fieldName]

		// Required field missing
		if fieldDef.Required && (!exists || value == nil) {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("required field '%s' is missing", fieldName),
			})
			continue
		}

		// Skip validation for missing optional fields
		if !exists || value == nil {
			continue
		}

		// Type validation
		if err := jv.validateFieldType(fieldName, value, fieldDef.Type); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Message: err.Error(),
				Value:   value,
			})
		}

		// Custom validation rules
		if fieldDef.Validation != nil {
			if err := jv.validateCustomRules(fieldName, value, fieldDef.Validation); err != nil {
				result.Warnings = append(result.Warnings, ValidationError{
					Field:   fieldName,
					Message: err.Error(),
					Value:   value,
				})
			}
		}
	}

	// Check for extra properties not defined in schema
	for propertyName := range properties {
		if _, exists := fieldDefinitions[propertyName]; !exists {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   propertyName,
				Message: fmt.Sprintf("property '%s' is not defined in schema", propertyName),
				Value:   properties[propertyName],
			})
		}
	}

	return result
}

// normalizeFieldType ensures consistent uppercase enum comparison
func normalizeFieldType(ft graph.FieldType) graph.FieldType {
	return graph.FieldType(strings.ToUpper(string(ft)))
}

// validateFieldType validates the type of a field value
func (jv *JSONBValidator) validateFieldType(fieldName string, value any, expectedType graph.FieldType) error {
	expectedType = normalizeFieldType(expectedType)

	switch expectedType {
	case graph.FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", fieldName, value)
		}
	case graph.FieldTypeInteger:
		if !jv.isInteger(value) {
			return fmt.Errorf("field '%s' must be an integer, got %T", fieldName, value)
		}
	case graph.FieldTypeFloat:
		if !jv.isFloat(value) {
			return fmt.Errorf("field '%s' must be a float, got %T", fieldName, value)
		}
	case graph.FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean, got %T", fieldName, value)
		}
	case graph.FieldTypeTimestamp:
		switch v := value.(type) {
		case string:
			if _, err := time.Parse(time.RFC3339, v); err != nil {
				return fmt.Errorf("field '%s' must be a valid timestamp (RFC3339): %v", fieldName, err)
			}
		case time.Time:
			// already parsed; accept value
		default:
			return fmt.Errorf("field '%s' must be a timestamp string, got %T", fieldName, value)
		}
	case graph.FieldTypeJSON:
		if _, err := json.Marshal(value); err != nil {
			return fmt.Errorf("field '%s' contains invalid JSON: %v", fieldName, err)
		}
	case graph.FieldTypeFileReference:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a file reference string, got %T", fieldName, value)
		}
	case graph.FieldTypeGeometry:
		if !jv.isGeometry(value) {
			return fmt.Errorf("field '%s' must be a valid geometry, got %T", fieldName, value)
		}
	case graph.FieldTypeTimeseries:
		if !jv.isTimeseries(value) {
			return fmt.Errorf("field '%s' must be a valid timeseries, got %T", fieldName, value)
		}
	case graph.FieldTypeReference:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a reference string, got %T", fieldName, value)
		}
		if strings.TrimSpace(strVal) == "" {
			return fmt.Errorf("field '%s' must be a non-empty reference string", fieldName)
		}
	case graph.FieldTypeEntityID:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be an entity ID string, got %T", fieldName, value)
		}
		if _, err := uuid.Parse(strings.TrimSpace(strVal)); err != nil {
			return fmt.Errorf("field '%s' must be a valid UUID string: %v", fieldName, err)
		}
	case graph.FieldTypeEntityReference:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a string reference, got %T", fieldName, value)
		}
		if strings.TrimSpace(strVal) == "" {
			return fmt.Errorf("field '%s' must be a non-empty entity reference string", fieldName)
		}
	case graph.FieldTypeEntityReferenceArray:
		values, ok := value.([]interface{})
		if !ok {
			if strSlice, ok := value.([]string); ok {
				values = make([]interface{}, len(strSlice))
				for i, v := range strSlice {
					values[i] = v
				}
			} else {
				return fmt.Errorf("field '%s' must be an array of string references, got %T", fieldName, value)
			}
		}
		for _, item := range values {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("field '%s' reference values must be strings, got %T", fieldName, item)
			}
			if strings.TrimSpace(str) == "" {
				return fmt.Errorf("field '%s' contains an empty entity reference value", fieldName)
			}
		}
	default:
		return fmt.Errorf("unknown field type: %s", expectedType)
	}

	return nil
}

// validateCustomRules validates optional field rules
func (jv *JSONBValidator) validateCustomRules(fieldName string, value any, rules any) error {
	rulesMap, ok := rules.(map[string]any)
	if !ok {
		return fmt.Errorf("validation rules must be a map")
	}

	if minVal, exists := rulesMap["min"]; exists {
		if !jv.isGreaterThanOrEqual(value, minVal) {
			return fmt.Errorf("field '%s' value %v is less than minimum %v", fieldName, value, minVal)
		}
	}

	if maxVal, exists := rulesMap["max"]; exists {
		if !jv.isLessThanOrEqual(value, maxVal) {
			return fmt.Errorf("field '%s' value %v is greater than maximum %v", fieldName, value, maxVal)
		}
	}

	if minLen, exists := rulesMap["min_length"]; exists {
		if strVal, ok := value.(string); ok {
			if len(strVal) < int(minLen.(float64)) {
				return fmt.Errorf("field '%s' length %d is less than minimum %v", fieldName, len(strVal), minLen)
			}
		}
	}

	if maxLen, exists := rulesMap["max_length"]; exists {
		if strVal, ok := value.(string); ok {
			if len(strVal) > int(maxLen.(float64)) {
				return fmt.Errorf("field '%s' length %d is greater than maximum %v", fieldName, len(strVal), maxLen)
			}
		}
	}

	if pattern, exists := rulesMap["pattern"]; exists {
		if strVal, ok := value.(string); ok {
			if !strings.Contains(strings.ToLower(strVal), strings.ToLower(pattern.(string))) {
				return fmt.Errorf("field '%s' value '%s' does not match pattern '%s'", fieldName, strVal, pattern)
			}
		}
	}

	return nil
}

// Helper methods for type checking
func (jv *JSONBValidator) isInteger(value any) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return v == float64(int64(v))
	case string:
		_, err := strconv.Atoi(v)
		return err == nil
	default:
		return false
	}
}

func (jv *JSONBValidator) isFloat(value any) bool {
	switch v := value.(type) {
	case float32, float64:
		return true
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case string:
		_, err := strconv.ParseFloat(v, 64) // use v directly
		return err == nil
	default:
		return false
	}
}

func (jv *JSONBValidator) isGeometry(value any) bool {
	if _, ok := value.(string); ok {
		return true
	}
	if geomMap, ok := value.(map[string]any); ok {
		if _, hasType := geomMap["type"]; hasType {
			return true
		}
	}
	return false
}

func (jv *JSONBValidator) isTimeseries(value any) bool {
	valueSlice := reflect.ValueOf(value)
	if valueSlice.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < valueSlice.Len(); i++ {
		elem := valueSlice.Index(i).Interface()
		if elemMap, ok := elem.(map[string]any); ok {
			if _, hasTimestamp := elemMap["timestamp"]; !hasTimestamp {
				return false
			}
			if _, hasValue := elemMap["value"]; !hasValue {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func (jv *JSONBValidator) isGreaterThanOrEqual(value, min any) bool {
	switch v := value.(type) {
	case float64:
		if minFloat, ok := min.(float64); ok {
			return v >= minFloat
		}
	case int:
		if minInt, ok := min.(int); ok {
			return v >= minInt
		}
	}
	return false
}

func (jv *JSONBValidator) isLessThanOrEqual(value, max any) bool {
	switch v := value.(type) {
	case float64:
		if maxFloat, ok := max.(float64); ok {
			return v <= maxFloat
		}
	case int:
		if maxInt, ok := max.(int); ok {
			return v <= maxInt
		}
	}
	return false
}

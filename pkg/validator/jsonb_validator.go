package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// JSONBValidator handles validation of JSONB properties against field definitions
type JSONBValidator struct{}

// NewJSONBValidator creates a new JSONB validator
func NewJSONBValidator() *JSONBValidator {
	return &JSONBValidator{}
}

// FieldType represents the type of a field
type FieldType string

const (
	FieldTypeString    FieldType = "string"
	FieldTypeInteger   FieldType = "integer"
	FieldTypeFloat     FieldType = "float"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeJSON      FieldType = "json"
	FieldTypeFileRef   FieldType = "file_reference"
	FieldTypeGeometry  FieldType = "geometry"
	FieldTypeTimeseries FieldType = "timeseries"
)

// FieldDefinition represents a field definition for validation
type FieldDefinition struct {
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Description string    `json:"description,omitempty"`
	Default     any       `json:"default,omitempty"`
	Validation  any       `json:"validation,omitempty"`
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

	// Check required fields
	for fieldName, fieldDef := range fieldDefinitions {
		value, exists := properties[fieldName]

		if fieldDef.Required && (!exists || value == nil) {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("Required field '%s' is missing", fieldName),
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
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   propertyName,
				Message: fmt.Sprintf("Property '%s' is not defined in schema", propertyName),
			})
		}
	}

	return result
}

// validateFieldType validates the type of a field value
func (jv *JSONBValidator) validateFieldType(fieldName string, value any, expectedType FieldType) error {
	switch expectedType {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", fieldName, value)
		}
	case FieldTypeInteger:
		if !jv.isInteger(value) {
			return fmt.Errorf("field '%s' must be an integer, got %T", fieldName, value)
		}
	case FieldTypeFloat:
		if !jv.isFloat(value) {
			return fmt.Errorf("field '%s' must be a float, got %T", fieldName, value)
		}
	case FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean, got %T", fieldName, value)
		}
	case FieldTypeTimestamp:
		if strVal, ok := value.(string); ok {
			if _, err := time.Parse(time.RFC3339, strVal); err != nil {
				return fmt.Errorf("field '%s' must be a valid timestamp (RFC3339), got: %v", fieldName, err)
			}
		} else {
			return fmt.Errorf("field '%s' must be a timestamp string, got %T", fieldName, value)
		}
	case FieldTypeJSON:
		// JSON type can be any valid JSON value
		if _, err := json.Marshal(value); err != nil {
			return fmt.Errorf("field '%s' contains invalid JSON: %v", fieldName, err)
		}
	case FieldTypeFileRef:
		// File reference should be a string (path or ID)
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a file reference string, got %T", fieldName, value)
		}
	case FieldTypeGeometry:
		// Geometry can be a string (WKT) or an object
		if !jv.isGeometry(value) {
			return fmt.Errorf("field '%s' must be a valid geometry, got %T", fieldName, value)
		}
	case FieldTypeTimeseries:
		// Timeseries should be an array of objects with timestamp and value
		if !jv.isTimeseries(value) {
			return fmt.Errorf("field '%s' must be a valid timeseries, got %T", fieldName, value)
		}
	default:
		return fmt.Errorf("unknown field type: %s", expectedType)
	}

	return nil
}

// validateCustomRules validates custom validation rules
func (jv *JSONBValidator) validateCustomRules(fieldName string, value any, rules any) error {
	// Convert rules to map for processing
	rulesMap, ok := rules.(map[string]any)
	if !ok {
		return fmt.Errorf("validation rules must be a map")
	}

	// Check minimum value for numeric fields
	if minVal, exists := rulesMap["min"]; exists {
		if !jv.isGreaterThanOrEqual(value, minVal) {
			return fmt.Errorf("field '%s' value %v is less than minimum %v", fieldName, value, minVal)
		}
	}

	// Check maximum value for numeric fields
	if maxVal, exists := rulesMap["max"]; exists {
		if !jv.isLessThanOrEqual(value, maxVal) {
			return fmt.Errorf("field '%s' value %v is greater than maximum %v", fieldName, value, maxVal)
		}
	}

	// Check minimum length for string fields
	if minLen, exists := rulesMap["min_length"]; exists {
		if strVal, ok := value.(string); ok {
			if len(strVal) < int(minLen.(float64)) {
				return fmt.Errorf("field '%s' length %d is less than minimum %v", fieldName, len(strVal), minLen)
			}
		}
	}

	// Check maximum length for string fields
	if maxLen, exists := rulesMap["max_length"]; exists {
		if strVal, ok := value.(string); ok {
			if len(strVal) > int(maxLen.(float64)) {
				return fmt.Errorf("field '%s' length %d is greater than maximum %v", fieldName, len(strVal), maxLen)
			}
		}
	}

	// Check pattern matching for string fields
	if pattern, exists := rulesMap["pattern"]; exists {
		if strVal, ok := value.(string); ok {
			// Simple pattern matching (in production, use regexp)
			if !jv.matchesPattern(strVal, pattern.(string)) {
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
	switch value.(type) {
	case float32, float64:
		return true
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case string:
		_, err := strconv.ParseFloat(value.(string), 64)
		return err == nil
	default:
		return false
	}
}

func (jv *JSONBValidator) isGeometry(value any) bool {
	// Check if it's a string (WKT format)
	if _, ok := value.(string); ok {
		return true
	}
	
	// Check if it's a map with geometry properties
	if geomMap, ok := value.(map[string]any); ok {
		if _, hasType := geomMap["type"]; hasType {
			return true
		}
	}
	
	return false
}

func (jv *JSONBValidator) isTimeseries(value any) bool {
	// Check if it's an array
	valueSlice := reflect.ValueOf(value)
	if valueSlice.Kind() != reflect.Slice {
		return false
	}
	
	// Check if array elements have timestamp and value
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

func (jv *JSONBValidator) matchesPattern(str, pattern string) bool {
	// Simple wildcard pattern matching
	// In production, use proper regexp
	if pattern == "*" {
		return true
	}
	
	// Check if string contains the pattern
	return contains(str, pattern)
}

// contains checks if a string contains a substring (case-insensitive)
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

// toLowerCase converts a string to lowercase
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

// indexOf finds the index of a substring in a string
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

package validator

import (
	"fmt"
	"strings"

	"github.com/rpattn/engql/internal/domain"
)

var referenceCapableTypes = map[domain.FieldType]struct{}{
	domain.FieldTypeReference:            {},
	domain.FieldTypeEntityReference:      {},
	domain.FieldTypeEntityReferenceArray: {},
	domain.FieldTypeEntityID:             {},
}

// ValidateFields ensures schema field definitions satisfy cross-entity reference
// constraints. Schemas may declare at most one FieldTypeReference field, and
// fields that provide ReferenceEntityType must support reference semantics.
func ValidateFields(fields []domain.FieldDefinition) error {
	var referenceField string

	for _, field := range fields {
		trimmedRefType := strings.TrimSpace(field.ReferenceEntityType)

		if _, ok := referenceCapableTypes[field.Type]; trimmedRefType != "" && !ok {
			return fmt.Errorf("field %s cannot declare referenceEntityType because type %s does not support references", field.Name, field.Type)
		}

		if field.Type == domain.FieldTypeReference {
			if referenceField != "" {
				return fmt.Errorf("schema may declare only one REFERENCE field (found %s and %s)", referenceField, field.Name)
			}
			referenceField = field.Name
		}
	}

	return nil
}

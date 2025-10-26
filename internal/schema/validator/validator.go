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
}

// ValidateFields ensures schema field definitions satisfy cross-entity reference
// constraints. It also returns the ordered set of REFERENCE fields where the
// first entry acts as the canonical reference value for the schema.
func ValidateFields(fields []domain.FieldDefinition) (domain.ReferenceFieldSet, error) {
	set := domain.NewReferenceFieldSet(fields)

	for _, field := range fields {
		trimmedRefType := strings.TrimSpace(field.ReferenceEntityType)

		if _, ok := referenceCapableTypes[field.Type]; trimmedRefType != "" && !ok {
			return domain.ReferenceFieldSet{}, fmt.Errorf("field %s cannot declare referenceEntityType because type %s does not support references", field.Name, field.Type)
		}

		if field.Type == domain.FieldTypeEntityID {
			return domain.ReferenceFieldSet{}, fmt.Errorf("field %s cannot use deprecated type ENTITY_ID", field.Name)
		}
	}

	return set, nil
}

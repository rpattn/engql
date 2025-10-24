package validator

import (
	"testing"

	"github.com/rpattn/engql/graph"
)

func TestJSONBValidatorReferenceField(t *testing.T) {
	v := NewJSONBValidator()

	definitions := map[string]FieldDefinition{
		"reference": {
			Type:     graph.FieldTypeReference,
			Required: true,
		},
	}

	result := v.ValidateProperties(map[string]any{"reference": ""}, definitions)
	if result.IsValid {
		t.Fatalf("expected reference field to reject empty string")
	}

	result = v.ValidateProperties(map[string]any{"reference": "   "}, definitions)
	if result.IsValid {
		t.Fatalf("expected reference field to reject whitespace value")
	}

	result = v.ValidateProperties(map[string]any{"reference": "abc-123"}, definitions)
	if !result.IsValid {
		t.Fatalf("expected reference field to accept non-empty string, got errors: %+v", result.Errors)
	}
}

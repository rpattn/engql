package validator

import (
	"testing"

	"github.com/rpattn/engql/internal/domain"
)

func TestValidateFields_AllowsSingleReferenceField(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "id", Type: domain.FieldTypeString},
		{Name: "owner", Type: domain.FieldTypeReference, ReferenceEntityType: "accounts"},
	}

	if err := ValidateFields(fields); err != nil {
		t.Fatalf("expected validation to pass, got error: %v", err)
	}
}

func TestValidateFields_MultipleReferenceFields(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "primary", Type: domain.FieldTypeReference, ReferenceEntityType: "accounts"},
		{Name: "secondary", Type: domain.FieldTypeReference, ReferenceEntityType: "users"},
	}

	if err := ValidateFields(fields); err == nil {
		t.Fatalf("expected error for multiple reference fields")
	}
}

func TestValidateFields_ReferenceMissingTarget(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "ref", Type: domain.FieldTypeReference},
	}

	if err := ValidateFields(fields); err == nil {
		t.Fatalf("expected error when reference field lacks target entity type")
	}
}

func TestValidateFields_InvalidReferenceEntityTypeUsage(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "name", Type: domain.FieldTypeString, ReferenceEntityType: "accounts"},
	}

	if err := ValidateFields(fields); err == nil {
		t.Fatalf("expected error when non-reference field declares referenceEntityType")
	}
}

func TestValidateFields_ReferenceEntityTypeAllowed(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "owner", Type: domain.FieldTypeEntityReference, ReferenceEntityType: "accounts"},
		{Name: "ownerIds", Type: domain.FieldTypeEntityReferenceArray, ReferenceEntityType: "accounts"},
		{Name: "ownerID", Type: domain.FieldTypeEntityID, ReferenceEntityType: "accounts"},
	}

	if err := ValidateFields(fields); err != nil {
		t.Fatalf("expected validation to pass for standard reference types, got %v", err)
	}
}

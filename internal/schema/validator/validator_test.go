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

	if _, err := ValidateFields(fields); err != nil {
		t.Fatalf("expected validation to pass, got error: %v", err)
	}
}

func TestValidateFields_MultipleReferenceFields(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "primary", Type: domain.FieldTypeReference, ReferenceEntityType: "accounts"},
		{Name: "secondary", Type: domain.FieldTypeReference, ReferenceEntityType: "users"},
	}

	set, err := ValidateFields(fields)
	if err != nil {
		t.Fatalf("expected validation to allow multiple reference fields, got %v", err)
	}

	canonical, ok := set.CanonicalName()
	if !ok {
		t.Fatalf("expected canonical reference to be returned")
	}
	if canonical != "primary" {
		t.Fatalf("expected first reference field to be canonical, got %s", canonical)
	}

	names := set.Names()
	if len(names) != 2 {
		t.Fatalf("expected both reference fields to be included in summary, got %v", names)
	}
}

func TestValidateFields_ReferenceFieldMayOmitTarget(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "ref", Type: domain.FieldTypeReference},
	}

	if _, err := ValidateFields(fields); err != nil {
		t.Fatalf("expected validation to allow reference fields without a target, got %v", err)
	}
}

func TestValidateFields_InvalidReferenceEntityTypeUsage(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "name", Type: domain.FieldTypeString, ReferenceEntityType: "accounts"},
	}

	if _, err := ValidateFields(fields); err == nil {
		t.Fatalf("expected error when non-reference field declares referenceEntityType")
	}
}

func TestValidateFields_ReferenceEntityTypeAllowed(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "owner", Type: domain.FieldTypeEntityReference, ReferenceEntityType: "accounts"},
		{Name: "ownerIds", Type: domain.FieldTypeEntityReferenceArray, ReferenceEntityType: "accounts"},
	}

	if _, err := ValidateFields(fields); err != nil {
		t.Fatalf("expected validation to pass for standard reference types, got %v", err)
	}
}

func TestValidateFields_RejectsEntityID(t *testing.T) {
	fields := []domain.FieldDefinition{
		{Name: "legacy", Type: domain.FieldTypeEntityID, ReferenceEntityType: "accounts"},
	}

	if _, err := ValidateFields(fields); err == nil {
		t.Fatalf("expected validation to reject ENTITY_ID fields")
	}
}

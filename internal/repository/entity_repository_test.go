package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestEnsureReferenceNormalization_TrimsWhitespace(t *testing.T) {
	repo := &entityRepository{}
	schemaID := uuid.New()
	repo.referenceFieldCache.Store(schemaID, referenceFieldCacheEntry{canonical: "reference", names: []string{"reference"}})

	props := map[string]any{"reference": "  ABC-123  "}
	if err := repo.ensureReferenceNormalization(context.Background(), schemaID, props, true); err != nil {
		t.Fatalf("unexpected error normalising reference: %v", err)
	}

	value, ok := props["reference"].(string)
	if !ok {
		t.Fatalf("reference property not stored as string: %#v", props["reference"])
	}
	if value != "ABC-123" {
		t.Fatalf("expected trimmed reference value, got %q", value)
	}
}

func TestEnsureReferenceNormalization_EmptyStrictError(t *testing.T) {
	repo := &entityRepository{}
	schemaID := uuid.New()
	repo.referenceFieldCache.Store(schemaID, referenceFieldCacheEntry{canonical: "reference", names: []string{"reference"}})

	props := map[string]any{"reference": "   "}
	if err := repo.ensureReferenceNormalization(context.Background(), schemaID, props, true); err == nil {
		t.Fatalf("expected error for empty reference value")
	}
}

func TestEnsureReferenceNormalization_NonStrictAllowsEmpty(t *testing.T) {
	repo := &entityRepository{}
	schemaID := uuid.New()
	repo.referenceFieldCache.Store(schemaID, referenceFieldCacheEntry{canonical: "reference", names: []string{"reference"}})

	props := map[string]any{"reference": "   "}
	if err := repo.ensureReferenceNormalization(context.Background(), schemaID, props, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, ok := props["reference"].(string)
	if !ok {
		t.Fatalf("reference property not stored as string: %#v", props["reference"])
	}
	if value != "" {
		t.Fatalf("expected empty string after normalisation, got %q", value)
	}
}

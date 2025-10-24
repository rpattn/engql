package graphql

import (
	"testing"

	"github.com/google/uuid"

	"github.com/rpattn/engql/graph"
)

func TestConvertReferenceValuesToIDsHandlesValidUUIDs(t *testing.T) {
	parent := &graph.Entity{ID: "parent"}
	refID := uuid.New().String()

	refMap := map[string][]*graph.Entity{
		refID: {parent},
	}
	idParents := make(map[string][]*graph.Entity)
	missingIDs := make(map[string]struct{})

	handled, invalid := convertReferenceValuesToIDs(refMap, idParents, missingIDs)

	if !handled {
		t.Fatalf("expected references to be converted to IDs")
	}
	if len(invalid) != 0 {
		t.Fatalf("expected no invalid references, got %v", invalid)
	}

	if _, ok := missingIDs[refID]; !ok {
		t.Fatalf("expected %s to be recorded as missing id", refID)
	}

	parents, ok := idParents[refID]
	if !ok {
		t.Fatalf("expected parent slice for id %s", refID)
	}
	if len(parents) != 1 || parents[0] != parent {
		t.Fatalf("unexpected parents slice %#v", parents)
	}
}

func TestConvertReferenceValuesToIDsReportsInvalidValues(t *testing.T) {
	invalidValue := "not-a-uuid"
	refMap := map[string][]*graph.Entity{
		invalidValue: {nil},
	}

	idParents := make(map[string][]*graph.Entity)
	missingIDs := make(map[string]struct{})

	handled, invalid := convertReferenceValuesToIDs(refMap, idParents, missingIDs)

	if handled {
		t.Fatalf("expected conversion to fail for invalid values")
	}
	if len(invalid) != 1 || invalid[0] != invalidValue {
		t.Fatalf("unexpected invalid references %v", invalid)
	}

	if len(idParents) != 0 {
		t.Fatalf("expected no id parents, got %#v", idParents)
	}
	if len(missingIDs) != 0 {
		t.Fatalf("expected no missing ids, got %#v", missingIDs)
	}
}

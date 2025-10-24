package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEntitySnapshotCanonicalText(t *testing.T) {
	schemaID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	snapshot := EntitySnapshot{
		Path:       "root.node",
		SchemaID:   schemaID,
		EntityType: "Example",
		Version:    1,
		Properties: map[string]any{
			"name": "base",
			"metadata": map[string]any{
				"color": "red",
				"size":  float64(10),
			},
			"tags": []any{"alpha", "beta"},
		},
	}

	lines, err := snapshot.CanonicalText()
	if err != nil {
		t.Fatalf("unexpected error generating canonical text: %v", err)
	}

	expected := []string{
		"Path: root.node",
		"SchemaID: 123e4567-e89b-12d3-a456-426614174000",
		"EntityType: Example",
		"Version: 1",
		"Properties:",
		"  metadata.color: \"red\"",
		"  metadata.size: 10",
		"  name: \"base\"",
		"  tags[0]: \"alpha\"",
		"  tags[1]: \"beta\"",
	}

	if len(lines) != len(expected) {
		t.Fatalf("expected %d canonical lines, got %d\n%v", len(expected), len(lines), lines)
	}

	for idx, line := range expected {
		if lines[idx] != line {
			t.Errorf("line %d mismatch: expected %q got %q", idx, line, lines[idx])
		}
	}
}

func TestDiffEntitySnapshots(t *testing.T) {
	entityID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeffffffff")

	base := EntitySnapshot{
		Path:       "root.node",
		SchemaID:   entityID,
		EntityType: "Example",
		Version:    1,
		Properties: map[string]any{
			"name":     "Base",
			"metadata": map[string]any{"color": "red"},
		},
	}

	target := EntitySnapshot{
		Path:       "root.node",
		SchemaID:   entityID,
		EntityType: "Example",
		Version:    2,
		Properties: map[string]any{
			"name":     "Target",
			"metadata": map[string]any{"color": "blue"},
			"count":    float64(2),
		},
	}

	diff, err := DiffEntitySnapshots("base", &base, "target", &target)
	if err != nil {
		t.Fatalf("unexpected diff error: %v", err)
	}

	if diff == "" {
		t.Fatalf("expected diff output, got empty string")
	}

	if !strings.Contains(diff, "-  metadata.color: \"red\"") {
		t.Errorf("diff missing base metadata change: %s", diff)
	}

	if !strings.Contains(diff, "+  metadata.color: \"blue\"") {
		t.Errorf("diff missing target metadata change: %s", diff)
	}

	if !strings.Contains(diff, "+  count: 2") {
		t.Errorf("diff missing added property: %s", diff)
	}
}

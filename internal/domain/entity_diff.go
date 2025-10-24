package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// EntitySnapshot represents the minimal data required to compute diffs between entity versions.
type EntitySnapshot struct {
	Path       string
	SchemaID   uuid.UUID
	EntityType string
	Properties map[string]any
	Version    int64
}

// NewEntitySnapshotFromEntity creates a snapshot from the current entity record.
func NewEntitySnapshotFromEntity(entity Entity) EntitySnapshot {
	return EntitySnapshot{
		Path:       entity.Path,
		SchemaID:   entity.SchemaID,
		EntityType: entity.EntityType,
		Properties: cloneProperties(entity.Properties),
		Version:    entity.Version,
	}
}

// NewEntitySnapshotFromHistory creates a snapshot from a historical entity record.
func NewEntitySnapshotFromHistory(history EntityHistory) EntitySnapshot {
	return EntitySnapshot{
		Path:       history.Path,
		SchemaID:   history.SchemaID,
		EntityType: history.EntityType,
		Properties: cloneProperties(history.Properties),
		Version:    history.Version,
	}
}

// CanonicalText flattens the snapshot into a deterministic set of lines suitable for diffing.
func (s EntitySnapshot) CanonicalText() ([]string, error) {
	lines := []string{
		fmt.Sprintf("Path: %s", s.Path),
		fmt.Sprintf("SchemaID: %s", s.SchemaID),
		fmt.Sprintf("EntityType: %s", s.EntityType),
		fmt.Sprintf("Version: %d", s.Version),
		"Properties:",
	}

	flattened := map[string]string{}
	if len(s.Properties) > 0 {
		if err := flattenProperties("", s.Properties, flattened); err != nil {
			return nil, err
		}
	}

	if len(flattened) == 0 {
		lines = append(lines, "  (empty)")
		return lines, nil
	}

	keys := make([]string, 0, len(flattened))
	for key := range flattened {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("  %s: %s", key, flattened[key]))
	}

	return lines, nil
}

// DiffEntitySnapshots produces a unified diff between two snapshots using the provided labels.
func DiffEntitySnapshots(baseLabel string, base *EntitySnapshot, targetLabel string, target *EntitySnapshot) (string, error) {
	baseString, err := canonicalString(base)
	if err != nil {
		return "", err
	}

	targetString, err := canonicalString(target)
	if err != nil {
		return "", err
	}

	return buildUnifiedDiff(baseLabel, targetLabel, baseString, targetString), nil
}

func canonicalString(snapshot *EntitySnapshot) (string, error) {
	if snapshot == nil {
		return "", nil
	}

	lines, err := snapshot.CanonicalText()
	if err != nil {
		return "", err
	}

	return strings.Join(lines, "\n") + "\n", nil
}

func flattenProperties(prefix string, value any, acc map[string]string) error {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			if prefix != "" {
				acc[prefix] = "{}"
			}
			return nil
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPrefix := key
			if prefix != "" {
				nextPrefix = prefix + "." + key
			}
			if err := flattenProperties(nextPrefix, typed[key], acc); err != nil {
				return err
			}
		}
	case []any:
		if len(typed) == 0 {
			if prefix != "" {
				acc[prefix] = "[]"
			}
			return nil
		}
		for idx, item := range typed {
			nextPrefix := fmt.Sprintf("%s[%d]", prefix, idx)
			if prefix == "" {
				nextPrefix = fmt.Sprintf("[%d]", idx)
			}
			if err := flattenProperties(nextPrefix, item, acc); err != nil {
				return err
			}
		}
	case nil:
		if prefix != "" {
			acc[prefix] = "null"
		}
	default:
		if prefix == "" {
			return fmt.Errorf("property key missing for value %v", typed)
		}
		encoded, err := json.Marshal(typed)
		if err != nil {
			acc[prefix] = fmt.Sprintf("%v", typed)
		} else {
			acc[prefix] = string(encoded)
		}
	}

	return nil
}

func cloneProperties(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

type diffOp struct {
	prefix string
	line   string
}

func buildUnifiedDiff(baseLabel, targetLabel, baseContent, targetContent string) string {
	baseLines := splitLines(baseContent)
	targetLines := splitLines(targetContent)

	ops := diffLines(baseLines, targetLines)

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("--- %s\n", baseLabel))
	builder.WriteString(fmt.Sprintf("+++ %s\n", targetLabel))
	builder.WriteString("@@ -0,0 +0,0 @@\n")
	for _, operation := range ops {
		builder.WriteString(operation.prefix)
		builder.WriteString(operation.line)
		builder.WriteString("\n")
	}

	return builder.String()
}

func splitLines(input string) []string {
	lines := strings.Split(input, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func diffLines(base, target []string) []diffOp {
	m := len(base)
	n := len(target)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if base[i] == target[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	ops := make([]diffOp, 0, m+n)
	i, j := 0, 0
	for i < m && j < n {
		if base[i] == target[j] {
			ops = append(ops, diffOp{prefix: " ", line: base[i]})
			i++
			j++
			continue
		}

		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{prefix: "-", line: base[i]})
			i++
		} else {
			ops = append(ops, diffOp{prefix: "+", line: target[j]})
			j++
		}
	}

	for i < m {
		ops = append(ops, diffOp{prefix: "-", line: base[i]})
		i++
	}

	for j < n {
		ops = append(ops, diffOp{prefix: "+", line: target[j]})
		j++
	}

	return ops
}

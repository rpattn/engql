package validator

import (
	"fmt"
	"strings"
)

// PathManager handles hierarchical path operations for ltree
type PathManager struct{}

// NewPathManager creates a new path manager instance
func NewPathManager() *PathManager {
	return &PathManager{}
}

// GeneratePath creates a new hierarchical path
// parentPath: the parent's path (empty for root)
// index: the position within the parent's children
func (pm *PathManager) GeneratePath(parentPath string, index int) string {
	if parentPath == "" {
		return fmt.Sprintf("%d", index)
	}
	return fmt.Sprintf("%s.%d", parentPath, index)
}

// GetParentPath extracts the parent path from a given path
func (pm *PathManager) GetParentPath(path string) string {
	if path == "" {
		return ""
	}
	
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return ""
	}
	
	return path[:lastDot]
}

// GetPathDepth returns the depth level of a path
func (pm *PathManager) GetPathDepth(path string) int {
	if path == "" {
		return 0
	}
	
	return strings.Count(path, ".") + 1
}

// IsAncestorOf checks if the first path is an ancestor of the second
func (pm *PathManager) IsAncestorOf(ancestorPath, descendantPath string) bool {
	if ancestorPath == "" {
		return true // Root is ancestor of all
	}
	
	return strings.HasPrefix(descendantPath, ancestorPath+".")
}

// IsDescendantOf checks if the first path is a descendant of the second
func (pm *PathManager) IsDescendantOf(descendantPath, ancestorPath string) bool {
	return pm.IsAncestorOf(ancestorPath, descendantPath)
}

// IsSiblingOf checks if two paths are siblings (same parent)
func (pm *PathManager) IsSiblingOf(path1, path2 string) bool {
	return pm.GetParentPath(path1) == pm.GetParentPath(path2) && path1 != path2
}

// IsDirectChildOf checks if the first path is a direct child of the second
func (pm *PathManager) IsDirectChildOf(childPath, parentPath string) bool {
	return pm.GetParentPath(childPath) == parentPath
}

// GetPathComponents splits a path into its components
func (pm *PathManager) GetPathComponents(path string) []string {
	if path == "" {
		return []string{}
	}
	
	return strings.Split(path, ".")
}

// ValidatePath validates that a path follows the correct ltree format
func (pm *PathManager) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	components := pm.GetPathComponents(path)
	for i, component := range components {
		if component == "" {
			return fmt.Errorf("path component %d is empty", i)
		}
		
		// Check if component is numeric (ltree requirement)
		for _, char := range component {
			if char < '0' || char > '9' {
				return fmt.Errorf("path component %d contains non-numeric character: %c", i, char)
			}
		}
	}
	
	return nil
}

// GetNextSiblingPath generates the path for the next sibling
func (pm *PathManager) GetNextSiblingPath(currentPath string) string {
	components := pm.GetPathComponents(currentPath)
	if len(components) == 0 {
		return "1"
	}
	
	// Increment the last component
	lastComponent := components[len(components)-1]
	// Parse as integer and increment
	lastIndex := 0
	for _, char := range lastComponent {
		lastIndex = lastIndex*10 + int(char-'0')
	}
	lastIndex++
	
	// Rebuild the path
	if len(components) == 1 {
		return fmt.Sprintf("%d", lastIndex)
	}
	
	parentPath := strings.Join(components[:len(components)-1], ".")
	return fmt.Sprintf("%s.%d", parentPath, lastIndex)
}

// GetChildPaths generates paths for direct children of a given path
func (pm *PathManager) GetChildPaths(parentPath string, count int) []string {
	paths := make([]string, count)
	for i := 0; i < count; i++ {
		paths[i] = pm.GeneratePath(parentPath, i+1)
	}
	return paths
}

// PathComparator provides comparison functions for sorting paths
type PathComparator struct{}

// NewPathComparator creates a new path comparator
func NewPathComparator() *PathComparator {
	return &PathComparator{}
}

// ComparePaths compares two paths for sorting
// Returns -1 if path1 < path2, 0 if equal, 1 if path1 > path2
func (pc *PathComparator) ComparePaths(path1, path2 string) int {
	components1 := strings.Split(path1, ".")
	components2 := strings.Split(path2, ".")
	
	minLen := len(components1)
	if len(components2) < minLen {
		minLen = len(components2)
	}
	
	// Compare common components
	for i := 0; i < minLen; i++ {
		comp1 := parseInt(components1[i])
		comp2 := parseInt(components2[i])
		
		if comp1 < comp2 {
			return -1
		} else if comp1 > comp2 {
			return 1
		}
	}
	
	// If all common components are equal, shorter path comes first
	if len(components1) < len(components2) {
		return -1
	} else if len(components1) > len(components2) {
		return 1
	}
	
	return 0
}

// parseInt safely parses a string to integer
func parseInt(s string) int {
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		}
	}
	return result
}

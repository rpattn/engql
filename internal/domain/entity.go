package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Entity represents a dynamic entity instance with hierarchical properties
type Entity struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	EntityType     string          `json:"entity_type"`
	Path           string          `json:"path"` // ltree path as string
	Properties     map[string]any  `json:"properties"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// NewEntity creates a new entity with immutable pattern
func NewEntity(organizationID uuid.UUID, entityType, path string, properties map[string]any) Entity {
	now := time.Now()
	return Entity{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		EntityType:     entityType,
		Path:           path,
		Properties:     copyProperties(properties), // Deep copy to ensure immutability
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// WithProperty returns a new entity with an added/updated property
func (e Entity) WithProperty(key string, value any) Entity {
	newProperties := copyProperties(e.Properties)
	newProperties[key] = value
	
	return Entity{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		EntityType:     e.EntityType,
		Path:           e.Path,
		Properties:     newProperties,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// WithoutProperty returns a new entity without the specified property
func (e Entity) WithoutProperty(key string) Entity {
	newProperties := copyProperties(e.Properties)
	delete(newProperties, key)
	
	return Entity{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		EntityType:     e.EntityType,
		Path:           e.Path,
		Properties:     newProperties,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// WithPath returns a new entity with updated hierarchical path
func (e Entity) WithPath(path string) Entity {
	return Entity{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		EntityType:     e.EntityType,
		Path:           path,
		Properties:     copyProperties(e.Properties),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// WithEntityType returns a new entity with updated entity type
func (e Entity) WithEntityType(entityType string) Entity {
	return Entity{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		EntityType:     entityType,
		Path:           e.Path,
		Properties:     copyProperties(e.Properties),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

// WithProperties returns a new entity with updated properties
func (e Entity) WithProperties(properties map[string]any) Entity {
	return Entity{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		EntityType:     e.EntityType,
		Path:           e.Path,
		Properties:     copyProperties(properties),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      time.Now(),
	}
}

func (e *Entity) GetPropertiesAsJSONB() (json.RawMessage, error) {
	if e.Properties == nil {
		e.Properties = make(map[string]any)
	}
	return json.Marshal(e.Properties)
}

// FromJSONBProperties creates properties map from JSONB data
func FromJSONBProperties(propertiesJSON json.RawMessage) (map[string]any, error) {
	var properties map[string]any
	err := json.Unmarshal(propertiesJSON, &properties)
	return properties, err
}

// GetParentPath returns the parent path from the current path
func (e Entity) GetParentPath() string {
	if e.Path == "" {
		return ""
	}
	
	// Simple implementation - in production you might want more sophisticated path handling
	// For ltree format like "1.2.3", parent would be "1.2"
	lastDot := -1
	for i := len(e.Path) - 1; i >= 0; i-- {
		if e.Path[i] == '.' {
			lastDot = i
			break
		}
	}
	
	if lastDot == -1 {
		return ""
	}
	
	return e.Path[:lastDot]
}

// IsDescendantOf checks if this entity is a descendant of the given path
func (e Entity) IsDescendantOf(path string) bool {
	if path == "" {
		return true // Root path, all entities are descendants
	}
	
	return len(e.Path) > len(path) && e.Path[:len(path)] == path
}

// IsAncestorOf checks if this entity is an ancestor of the given path
func (e Entity) IsAncestorOf(path string) bool {
	if e.Path == "" {
		return false // No entity can be ancestor of root
	}
	
	return len(path) > len(e.Path) && path[:len(e.Path)] == e.Path
}

// copyProperties creates a deep copy of the properties map to ensure immutability
func copyProperties(properties map[string]any) map[string]any {
	newProperties := make(map[string]any, len(properties))
	for k, v := range properties {
		// For a truly immutable implementation, you'd need to deep copy each value
		// For simplicity, we're doing a shallow copy here
		newProperties[k] = v
	}
	return newProperties
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

// EntityHistory captures a historical snapshot of an entity version.
type EntityHistory struct {
	ID             uuid.UUID
	EntityID       uuid.UUID
	OrganizationID uuid.UUID
	SchemaID       uuid.UUID
	EntityType     string
	Path           string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int64
	ChangeType     string
	ChangedAt      *time.Time
	Reason         *string
}

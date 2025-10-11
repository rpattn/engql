package domain

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant/organization in the system
type Organization struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewOrganization creates a new organization with immutable pattern
func NewOrganization(name, description string) Organization {
	now := time.Now()
	return Organization{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// WithDescription returns a new organization with updated description
func (o Organization) WithDescription(description string) Organization {
	return Organization{
		ID:          o.ID,
		Name:        o.Name,
		Description: description,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   time.Now(),
	}
}

// WithName returns a new organization with updated name
func (o Organization) WithName(name string) Organization {
	return Organization{
		ID:          o.ID,
		Name:        name,
		Description: o.Description,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   time.Now(),
	}
}

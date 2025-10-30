package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type contextKey string

const organizationIDKey contextKey = "organizationID"

// ContextWithOrganizationID returns a new context that carries the authenticated organization scope.
func ContextWithOrganizationID(ctx context.Context, id uuid.UUID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, organizationIDKey, id)
}

// OrganizationIDFromContext retrieves the authenticated organization scope from the context, if any.
func OrganizationIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.Nil, false
	}
	value := ctx.Value(organizationIDKey)
	if value == nil {
		return uuid.Nil, false
	}
	id, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	if id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// EnforceOrganizationScope ensures the provided organization matches the authenticated scope when present.
func EnforceOrganizationScope(ctx context.Context, organizationID uuid.UUID) error {
	if organizationID == uuid.Nil {
		return fmt.Errorf("organizationId is required")
	}
	scopedID, ok := OrganizationIDFromContext(ctx)
	if !ok {
		return nil
	}
	if scopedID != organizationID {
		return fmt.Errorf("organizationId %s does not match authenticated scope", organizationID)
	}
	return nil
}

package graphql

import "time"
import "encoding/json"

import "graphql-engineering-api/internal/domain"
import "graphql-engineering-api/graph"

// Safely dereference strings
func stringOrEmpty(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// Convert domain → GraphQL field types
func convertDomainFieldsToGraph(fields []domain.FieldDefinition) []*graph.FieldDefinition {
	result := make([]*graph.FieldDefinition, len(fields))
	for i, f := range fields {
		result[i] = &graph.FieldDefinition{
			Name:        f.Name,
			Type:        graph.FieldType(f.Type),
			Required:    f.Required,
			Description: &f.Description,
			Default:     &f.Default,
			Validation:  &f.Validation,
		}
	}
	return result
}

func convertEntitiesToGraph(entities []domain.Entity) []*graph.Entity {
	result := make([]*graph.Entity, len(entities))
	for i, e := range entities {
		result[i] = convertEntityToGraph(&e)
	}
	return result
}

func convertEntityToGraph(e *domain.Entity) *graph.Entity {
	// Convert map to JSON string
	propsJSON, err := json.Marshal(e.Properties)
	if err != nil {
		// fallback to empty JSON object on error
		propsJSON = []byte("{}")
	}

	return &graph.Entity{
		ID:             e.ID.String(),
		OrganizationID: e.OrganizationID.String(),
		EntityType:     e.EntityType,
		Path:           e.Path,
		Properties:     string(propsJSON), // now it's a string
		CreatedAt:      e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      e.UpdatedAt.Format(time.RFC3339),
	}
}

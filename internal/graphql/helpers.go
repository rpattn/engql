package graphql

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

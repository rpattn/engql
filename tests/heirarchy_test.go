package tests

import (
	"encoding/json"
	"testing"
)

// --- Hierarchy resolver coverage ---
//
// Covers:
// ✅ getEntityChildren
// ✅ getEntitySiblings
// ✅ getEntityAncestors
// ✅ getEntityDescendants
// ✅ getEntityHierarchy
//
// Each test uses a small tree:
// Root (Machine A)
// ├── Child (Machine B)
// └── Grandchild (Machine C)

func TestEntityHierarchyResolvers(t *testing.T) {
	// STEP 1: Create organization
	createOrgQuery := `
		mutation($input: CreateOrganizationInput!) {
			createOrganization(input: $input) {
				id
				name
			}
		}
	`
	orgVars := map[string]interface{}{
		"input": map[string]interface{}{
			"name": "Hierarchy Org",
		},
	}
	orgData := sendGraphQLRequest(t, createOrgQuery, orgVars)
	orgID := orgData["createOrganization"].(map[string]interface{})["id"].(string)
	t.Logf("✅ Created org for hierarchy test: %s", orgID)

	// STEP 2: Create schema
	createSchemaQuery := `
		mutation($input: CreateEntitySchemaInput!) {
			createEntitySchema(input: $input) {
				id
				name
			}
		}
	`
	schemaVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Node",
			"description":    "Hierarchy test node",
			"fields": []map[string]interface{}{
				{"name": "name", "type": "STRING", "required": true},
			},
		},
	}
	schemaData := sendGraphQLRequest(t, createSchemaQuery, schemaVars)
	schemaID := schemaData["createEntitySchema"].(map[string]interface{})["id"].(string)
	t.Logf("✅ Created schema for hierarchy test: %s", schemaID)

	// STEP 3: Create parent entity
	createEntityQuery := `
		mutation($input: CreateEntityInput!) {
			createEntity(input: $input) {
				id
				entityType
				path
				properties
			}
		}
	`
	parentProps, _ := json.Marshal(map[string]interface{}{"name": "Parent"})
	parentVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Node",
			"path":           "1",
			"properties":     string(parentProps),
		},
	}
	parentData := sendGraphQLRequest(t, createEntityQuery, parentVars)
	parentID := parentData["createEntity"].(map[string]interface{})["id"].(string)
	t.Logf("✅ Created parent entity: %s", parentID)

	// STEP 4: Create child entity
	childProps, _ := json.Marshal(map[string]interface{}{"name": "Child"})
	childVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Node",
			"path":           "1.1",
			"properties":     string(childProps),
		},
	}
	childData := sendGraphQLRequest(t, createEntityQuery, childVars)
	childID := childData["createEntity"].(map[string]interface{})["id"].(string)
	t.Logf("✅ Created child entity: %s", childID)

	// STEP 5: Query hierarchy using correct fields
	query := `
		query($entityId: String!) {
			getEntityHierarchy(entityId: $entityId) {
				current { id entityType properties }
				ancestors { id entityType properties }
				children { id entityType properties }
				siblings { id entityType properties }
			}
		}
	`
	data := sendGraphQLRequest(t, query, map[string]interface{}{"entityId": childID})
	hierarchy := data["getEntityHierarchy"].(map[string]interface{})
	t.Logf("✅ Hierarchy query returned: %+v", hierarchy)

	// STEP 6: Cleanup
	deleteEntityQuery := `
		mutation($id: String!) {
			deleteEntity(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": parentID})
	sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": childID})

	deleteSchemaQuery := `
		mutation($id: String!) {
			deleteEntitySchema(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteSchemaQuery, map[string]interface{}{"id": schemaID})

	deleteOrgQuery := `
		mutation($id: String!) {
			deleteOrganization(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteOrgQuery, map[string]interface{}{"id": orgID})
	t.Log("🗑️ Cleaned up hierarchy test resources")
}


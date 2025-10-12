package tests

import (
	"encoding/json"
	"testing"
)

func TestGetEntitiesByIDs_E2E(t *testing.T) {
	// STEP 1: Create an organization
	createOrgQuery := `
		mutation CreateOrg($input: CreateOrganizationInput!) {
			createOrganization(input: $input) {
				id
				name
			}
		}
	`
	orgVars := map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Test Org for GetEntitiesByIDs",
			"description": "E2E test org for multi-ID retrieval",
		},
	}
	orgData := sendGraphQLRequest(t, createOrgQuery, orgVars)
	org := orgData["createOrganization"].(map[string]interface{})
	orgID := org["id"].(string)
	t.Logf("✅ Created organization: %s", orgID)

	// STEP 2: Create a schema
	createSchemaQuery := `
		mutation CreateSchema($input: CreateEntitySchemaInput!) {
			createEntitySchema(input: $input) {
				id
				name
			}
		}
	`
	schemaVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Part",
			"description":    "Part schema for GetEntitiesByIDs test",
			"fields": []map[string]interface{}{
				{"name": "name", "type": "STRING", "required": true},
			},
		},
	}
	schemaData := sendGraphQLRequest(t, createSchemaQuery, schemaVars)
	schema := schemaData["createEntitySchema"].(map[string]interface{})
	schemaID := schema["id"].(string)
	t.Logf("✅ Created entity schema: %s", schemaID)

	// STEP 3: Create multiple entities
	createEntityQuery := `
		mutation CreateEntity($input: CreateEntityInput!) {
			createEntity(input: $input) {
				id
				properties
			}
		}
	`
	createEntity := func(name string) string {
		props, _ := json.Marshal(map[string]interface{}{
			"name": name,
		})
		entityVars := map[string]interface{}{
			"input": map[string]interface{}{
				"organizationId": orgID,
				"entityType":     "Part",
				"path":           "1",
				"properties":     string(props),
			},
		}
		data := sendGraphQLRequest(t, createEntityQuery, entityVars)
		entity := data["createEntity"].(map[string]interface{})
		return entity["id"].(string)
	}

	entityID1 := createEntity("Bolt M8")
	entityID2 := createEntity("Nut M8")
	t.Logf("✅ Created entities: %s, %s", entityID1, entityID2)

	// STEP 4: Query them using a batch endpoint
	getEntitiesQuery := `
		query GetEntitiesByIDs($ids: [String!]!) {
			entitiesByIDs(ids: $ids) {
				id
				entityType
				properties
			}
		}
	`
	queryVars := map[string]interface{}{
		"ids": []string{entityID1, entityID2},
	}
	data := sendGraphQLRequest(t, getEntitiesQuery, queryVars)
	entities := data["entitiesByIDs"].([]interface{})
	if len(entities) != 2 {
		t.Fatalf("❌ Expected 2 entities, got %d", len(entities))
	}
	t.Logf("✅ Retrieved %d entities by IDs", len(entities))

	// STEP 5: Cleanup
	deleteEntityQuery := `
		mutation DeleteEntity($id: String!) {
			deleteEntity(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": entityID1})
	sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": entityID2})
	t.Logf("🗑️ Deleted entities")

	deleteSchemaQuery := `
		mutation DeleteSchema($id: String!) {
			deleteEntitySchema(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteSchemaQuery, map[string]interface{}{"id": schemaID})

	deleteOrgQuery := `
		mutation DeleteOrg($id: String!) {
			deleteOrganization(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteOrgQuery, map[string]interface{}{"id": orgID})
	t.Logf("🧹 Cleaned up organization, schema, and entities")
}

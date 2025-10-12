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

func TestEntitiesByIDs_LinkedEntities_E2E(t *testing.T) {
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
			"name":        "Linked Entities Org",
			"description": "E2E test org for linked entities",
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
			"name":           "Component",
			"description":    "Component schema with linked entities",
			"fields": []map[string]interface{}{
				{"name": "name", "type": "STRING", "required": true},
				{"name": "linked_ids", "type": "JSON", "required": false},
			},
		},
	}
	schemaData := sendGraphQLRequest(t, createSchemaQuery, schemaVars)
	schema := schemaData["createEntitySchema"].(map[string]interface{})
	schemaID := schema["id"].(string)
	t.Logf("✅ Created entity schema: %s", schemaID)

	// STEP 3: Create entities
	createEntityQuery := `
		mutation CreateEntity($input: CreateEntityInput!) {
			createEntity(input: $input) {
				id
				properties
			}
		}
	`
	createEntity := func(name string, linkedIDs []string) string {
		props, _ := json.Marshal(map[string]interface{}{
			"name":       name,
			"linked_ids": linkedIDs,
		})
		entityVars := map[string]interface{}{
			"input": map[string]interface{}{
				"organizationId": orgID,
				"entityType":     "Component",
				"path":           "1",
				"properties":     string(props),
			},
		}
		data := sendGraphQLRequest(t, createEntityQuery, entityVars)
		entity := data["createEntity"].(map[string]interface{})
		return entity["id"].(string)
	}

	entityAID := createEntity("Entity A", nil)
	entityBID := createEntity("Entity B", []string{entityAID})
	t.Logf("✅ Created entities: %s -> %s linked to %s", entityAID, entityBID, entityAID)

	// STEP 4: Query entities by IDs
	getEntitiesQuery := `
		query GetEntitiesByIDs($ids: [String!]!) {
			entitiesByIDs(ids: $ids) {
				id
				properties
				linkedEntities {
					id
					properties
				}
			}
		}
	`
	queryVars := map[string]interface{}{"ids": []string{entityBID}}
	data := sendGraphQLRequest(t, getEntitiesQuery, queryVars)

	entities, ok := data["entitiesByIDs"].([]interface{})
	if !ok || len(entities) != 1 {
		t.Fatalf("❌ Expected 1 entity, got %v", len(entities))
	}

	entity, ok := entities[0].(map[string]interface{})
	if !ok {
		t.Fatalf("❌ Entity type assertion failed")
	}

	linkedRaw, ok := entity["linkedEntities"].([]interface{})
	if !ok || len(linkedRaw) != 1 {
		t.Fatalf("❌ Expected 1 linked entity, got %v", len(linkedRaw))
	}

	linkedEntity, ok := linkedRaw[0].(map[string]interface{})
	if !ok {
		t.Fatalf("❌ Linked entity type assertion failed")
	}

	if linkedEntity["id"] != entityAID {
		t.Fatalf("❌ Linked entity ID mismatch, got %v, want %v", linkedEntity["id"], entityAID)
	}

	t.Logf("✅ Linked entities resolved correctly")

	// STEP 5: Cleanup
	deleteEntityQuery := `
		mutation DeleteEntity($id: String!) {
			deleteEntity(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": entityAID})
	sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": entityBID})
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

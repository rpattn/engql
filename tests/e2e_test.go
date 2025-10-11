package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const graphqlURL = "http://localhost:8080/query"

// helper for sending GraphQL requests
func sendGraphQLRequest(t *testing.T, query string, vars map[string]interface{}) map[string]interface{} {
	reqBody, err := json.Marshal(GraphQLRequest{Query: query, Variables: vars})
	if err != nil {
		t.Fatalf("❌ Error marshaling GraphQL request: %v", err)
	}

	resp, err := http.Post(graphqlURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("❌ HTTP request error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("❌ Error reading response body: %v", err)
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		t.Fatalf("❌ Error parsing GraphQL response: %v\nRaw: %s", err, string(body))
	}

	if len(gqlResp.Errors) > 0 {
		t.Fatalf("❌ GraphQL returned errors: %v", gqlResp.Errors)
	}

	return gqlResp.Data
}

func TestFullE2EFlow(t *testing.T) {
	// STEP 1: Create an organization
	createOrgQuery := `
		mutation CreateOrg($input: CreateOrganizationInput!) {
			createOrganization(input: $input) {
				id
				name
				description
			}
		}
	`
	orgVars := map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Test Org E2E",
			"description": "End-to-end test organization",
		},
	}
	orgData := sendGraphQLRequest(t, createOrgQuery, orgVars)
	org := orgData["createOrganization"].(map[string]interface{})
	orgID := org["id"].(string)
	t.Logf("✅ Created organization: %s", orgID)

	// STEP 2: Create an entity schema for that org
	createSchemaQuery := `
		mutation CreateSchema($input: CreateEntitySchemaInput!) {
			createEntitySchema(input: $input) {
				id
				name
				description
			}
		}
	`
	schemaVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Component",
			"description":    "A test component schema",
			"fields": []map[string]interface{}{
				{"name": "name", "type": "STRING", "required": true},
				{"name": "material", "type": "STRING", "required": false},
				{"name": "weight", "type": "FLOAT", "required": false},
			},
		},
	}
	schemaData := sendGraphQLRequest(t, createSchemaQuery, schemaVars)
	schema := schemaData["createEntitySchema"].(map[string]interface{})
	schemaID := schema["id"].(string)
	t.Logf("✅ Created entity schema: %s", schemaID)

	// STEP 3: Create an entity using that schema
	createEntityQuery := `
		mutation CreateEntity($input: CreateEntityInput!) {
			createEntity(input: $input) {
				id
				entityType
				properties
			}
		}
	`
	// Convert properties map → JSON string
	props, _ := json.Marshal(map[string]interface{}{
		"name":     "Steel Bracket",
		"material": "Steel",
		"weight":   2.5,
	})

	entityVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Component",
			"path":           "1",
			"properties":     string(props), // 👈 GraphQL expects string
		},
	}
	entityData := sendGraphQLRequest(t, createEntityQuery, entityVars)
	entity := entityData["createEntity"].(map[string]interface{})
	entityID := entity["id"].(string)
	t.Logf("✅ Created entity: %s", entityID)

	// STEP 4: Query the entity
	getEntityQuery := `
		query GetEntity($id: String!) {
			entity(id: $id) {
				id
				entityType
				properties
			}
		}
	`
	entityLookup := sendGraphQLRequest(t, getEntityQuery, map[string]interface{}{"id": entityID})
	t.Logf("✅ Queried entity: %+v", entityLookup)

	// STEP 5: Search entities by property
	searchQuery := `
		query Search($orgID: String!, $filters: String!) {
			searchEntitiesByMultipleProperties(organizationId: $orgID, filters: $filters) {
				id
				entityType
				properties
			}
		}
	`
	filtersJSON := `{"material": "Steel"}`
	searchVars := map[string]interface{}{
		"orgID":   orgID,
		"filters": filtersJSON,
	}
	searchData := sendGraphQLRequest(t, searchQuery, searchVars)
	results := searchData["searchEntitiesByMultipleProperties"].([]interface{})
	t.Logf("✅ Found %d entities with material=Steel", len(results))

	// STEP 6: Clean up — delete entity, schema, and org
	deleteEntityQuery := `
		mutation DeleteEntity($id: String!) {
			deleteEntity(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": entityID})
	t.Logf("🗑️ Deleted entity %s", entityID)

	deleteSchemaQuery := `
		mutation DeleteSchema($id: String!) {
			deleteEntitySchema(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteSchemaQuery, map[string]interface{}{"id": schemaID})
	t.Logf("🗑️ Deleted schema %s", schemaID)

	deleteOrgQuery := `
		mutation DeleteOrg($id: String!) {
			deleteOrganization(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteOrgQuery, map[string]interface{}{"id": orgID})
	t.Logf("🗑️ Deleted organization %s", orgID)
}

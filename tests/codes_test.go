package tests

import (
	"encoding/json"
	"testing"
)

// TestStyleEntityHierarchyFlow creates a hierarchy of entities with dot-separated tags,
// and verifies ancestors and children retrieval.
func TestStyleEntityHierarchyFlow(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Test Org Hierarchy"},
	}
	orgData := sendGraphQLRequest(t, createOrgQuery, orgVars)
	orgID := orgData["createOrganization"].(map[string]interface{})["id"].(string)
	t.Logf("✅ Created organization: %s", orgID)

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
			"name":           "TestEntity",
			"description":    "Test hierarchy schema",
			"fields": []map[string]interface{}{
				{"name": "tag", "type": "STRING", "required": true},
				{"name": "description", "type": "STRING", "required": false},
			},
		},
	}
	schemaData := sendGraphQLRequest(t, createSchemaQuery, schemaVars)
	schemaID := schemaData["createEntitySchema"].(map[string]interface{})["id"].(string)
	t.Logf("✅ Created schema: %s", schemaID)

	// STEP 3: Create hierarchical entities with dot-separated tags
	createEntityQuery := `
		mutation($input: CreateEntityInput!) {
			createEntity(input: $input) {
				id
				entityType
				properties
				path
			}
		}
	`
	entities := []struct {
		tag  string
		path string
		desc string
	}{
		{"UMD80", "1", "Root entity"},
		{"UMD80TR001", "1.1", "Child of TEST1"},
		{"UMD80UM001", "1.2", "Child of TEST1"},
		{"UMD80UM001TR001", "1.1.A", "Grandchild of TEST1.1"},
	}

	var createdIDs []string
	for _, e := range entities {
		propsJSON, _ := json.Marshal(map[string]interface{}{"tag": e.tag, "description": e.desc})
		vars := map[string]interface{}{
			"input": map[string]interface{}{
				"organizationId": orgID,
				"entityType":     "TestEntity",
				"path":           e.path,
				"properties":     string(propsJSON),
			},
		}
		data := sendGraphQLRequest(t, createEntityQuery, vars)
		entity := data["createEntity"].(map[string]interface{})
		createdIDs = append(createdIDs, entity["id"].(string))
		t.Logf("✅ Created entity: %s (%s)", e.tag, entity["id"].(string))
	}

	// STEP 4: Verify ancestors of TEST1.1.A
	getAncestorsQuery := `
		query($id: String!) {
			getEntityAncestors(entityId: $id) {
				id
				properties
				path
			}
		}
	`
	entityId := createdIDs[3] // TEST1.1.A
	ancestorData := sendGraphQLRequest(t, getAncestorsQuery, map[string]interface{}{"id": entityId})
	ancestors := ancestorData["getEntityAncestors"].([]interface{})
	if len(ancestors) != 2 {
		t.Fatalf("❌ Expected 2 ancestors for TEST1.1.A, got %d", len(ancestors))
	}

	t.Logf("🧩 Ancestors of grandchild:")
	for i, a := range ancestors {
		rawProps := a.(map[string]interface{})["properties"]
		var props map[string]interface{}
		switch v := rawProps.(type) {
		case string:
			if err := json.Unmarshal([]byte(v), &props); err != nil {
				t.Fatalf("❌ Failed to parse properties JSON: %v", err)
			}
		case map[string]interface{}:
			props = v
		}
		t.Logf("   → %d: %s", i+1, props["tag"])
	}

	// STEP 5: Verify children of TEST1
	getChildrenQuery := `
		query($id: String!) {
			getEntityChildren(entityId: $id) {
				id
				properties
				path
			}
		}
	`
	childrenData := sendGraphQLRequest(t, getChildrenQuery, map[string]interface{}{"id": createdIDs[0]})
	children := childrenData["getEntityChildren"].([]interface{})
	if len(children) != 2 {
		t.Fatalf("❌ Expected 2 children for TEST1, got %d", len(children))
	}
	t.Logf("🧩 Children of parent:")
	for i, c := range children {
		rawProps := c.(map[string]interface{})["properties"]
		var props map[string]interface{}
		switch v := rawProps.(type) {
		case string:
			if err := json.Unmarshal([]byte(v), &props); err != nil {
				t.Fatalf("❌ Failed to parse properties JSON: %v", err)
			}
		case map[string]interface{}:
			props = v
		}
		t.Logf("   → %d: %s", i+1, props["tag"])
	}

	// STEP 6: Cleanup
	deleteEntityQuery := `mutation($id: String!) { deleteEntity(id: $id) }`
	for _, id := range createdIDs {
		sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": id})
	}
	deleteSchemaQuery := `mutation($id: String!) { deleteEntitySchema(id: $id) }`
	sendGraphQLRequest(t, deleteSchemaQuery, map[string]interface{}{"id": schemaID})
	deleteOrgQuery := `mutation($id: String!) { deleteOrganization(id: $id) }`
	sendGraphQLRequest(t, deleteOrgQuery, map[string]interface{}{"id": orgID})
	t.Logf("🗑️ Cleaned up organization and entities")
}

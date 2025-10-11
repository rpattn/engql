package tests

import (
	"encoding/json"
	"testing"
)
// ------------------------------------------------------------
// Extended GraphQL Resolver Tests
// ------------------------------------------------------------

func TestResolverCoverage(t *testing.T) {
	// STEP 1: Create org
	createOrgQuery := `
		mutation ($input: CreateOrganizationInput!) {
			createOrganization(input: $input) { id name description }
		}
	`
	orgVars := map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Coverage Org",
			"description": "For resolver coverage tests",
		},
	}
	org := sendGraphQLRequest(t, createOrgQuery, orgVars)["createOrganization"].(map[string]interface{})
	orgID := org["id"].(string)
	t.Logf("✅ Org created: %s", orgID)

	// STEP 2: Create schema
	createSchemaQuery := `
		mutation ($input: CreateEntitySchemaInput!) {
			createEntitySchema(input: $input) { id name description }
		}
	`
	schemaVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Machine",
			"description":    "Schema for test machines",
			"fields": []map[string]interface{}{
				{"name": "serial", "type": "STRING", "required": true},
				{"name": "status", "type": "STRING", "required": false},
				{"name": "weight", "type": "FLOAT", "required": false},
			},
		},
	}
	schema := sendGraphQLRequest(t, createSchemaQuery, schemaVars)["createEntitySchema"].(map[string]interface{})
	schemaID := schema["id"].(string)
	t.Logf("✅ Schema created: %s", schemaID)

	// STEP 3: Add and remove field
	addFieldQuery := `
		mutation ($schemaId: String!, $field: FieldDefinitionInput!) {
			addFieldToSchema(schemaId: $schemaId, field: $field) { id name fields { name type } }
		}
	`
	sendGraphQLRequest(t, addFieldQuery, map[string]interface{}{
		"schemaId": schemaID,
		"field": map[string]interface{}{
			"name":     "power",
			"type":     "FLOAT",
			"required": false,
		},
	})
	t.Logf("✅ Added field to schema")

	removeFieldQuery := `
		mutation ($schemaId: String!, $fieldName: String!) {
			removeFieldFromSchema(schemaId: $schemaId, fieldName: $fieldName) { id name }
		}
	`
	sendGraphQLRequest(t, removeFieldQuery, map[string]interface{}{
		"schemaId":  schemaID,
		"fieldName": "power",
	})
	t.Logf("✅ Removed field from schema")

	// STEP 4: Create multiple entities
	createEntityQuery := `
		mutation ($input: CreateEntityInput!) {
			createEntity(input: $input) { id entityType properties }
		}
	`

	createEntity := func(serial string, weight float64, status string) string {
		props, _ := json.Marshal(map[string]interface{}{
			"serial": serial,
			"weight": weight,
			"status": status,
		})
		entityVars := map[string]interface{}{
			"input": map[string]interface{}{
				"organizationId": orgID,
				"entityType":     "Machine",
				"path":           "root",
				"properties":     string(props),
			},
		}
		resp := sendGraphQLRequest(t, createEntityQuery, entityVars)
		id := resp["createEntity"].(map[string]interface{})["id"].(string)
		t.Logf("✅ Created entity: %s", id)
		return id
	}

	entity1 := createEntity("SN-100", 10.5, "active")
	entity2 := createEntity("SN-200", 15.0, "inactive")

	// STEP 5: Query entity by ID
	getEntityQuery := `
		query ($id: String!) {
			entity(id: $id) { id entityType properties }
		}
	`
	entityData := sendGraphQLRequest(t, getEntityQuery, map[string]interface{}{"id": entity1})
	t.Logf("✅ Fetched entity: %+v", entityData["entity"])

	// STEP 6: Query entities list
	getEntitiesQuery := `
		query ($org: String!) {
			entitiesByType(organizationId: $org, entityType: "Machine") {
				id entityType
			}
		}
	`
	list := sendGraphQLRequest(t, getEntitiesQuery, map[string]interface{}{"org": orgID})
	t.Logf("✅ Found entities: %+v", list["entitiesByType"])

	// STEP 7: Search by property
	searchQuery := `
		query ($orgID: String!, $filters: String!) {
			searchEntitiesByMultipleProperties(organizationId: $orgID, filters: $filters) { id entityType }
		}
	`
	filtersJSON := `{"status":"active"}`
	results := sendGraphQLRequest(t, searchQuery, map[string]interface{}{
		"orgID":   orgID,
		"filters": filtersJSON,
	})
	t.Logf("✅ Search results: %+v", results["searchEntitiesByMultipleProperties"])

	// STEP 8: Validate entity against schema
	validateQuery := `
		query ($id: String!) {
			validateEntityAgainstSchema(entityId: $id) { isValid errors }
		}
	`
	val := sendGraphQLRequest(t, validateQuery, map[string]interface{}{"id": entity1})
	t.Logf("✅ Validation result: %+v", val["validateEntityAgainstSchema"])

	// STEP 9: Delete entities + schema + org
	deleteEntity := func(id string) {
		q := `mutation ($id: String!) { deleteEntity(id: $id) }`
		sendGraphQLRequest(t, q, map[string]interface{}{"id": id})
		t.Logf("🗑️ Deleted entity: %s", id)
	}
	deleteEntity(entity1)
	deleteEntity(entity2)

	deleteSchemaQuery := `mutation ($id: String!) { deleteEntitySchema(id: $id) }`
	sendGraphQLRequest(t, deleteSchemaQuery, map[string]interface{}{"id": schemaID})
	t.Logf("🗑️ Deleted schema: %s", schemaID)

	deleteOrgQuery := `mutation ($id: String!) { deleteOrganization(id: $id) }`
	sendGraphQLRequest(t, deleteOrgQuery, map[string]interface{}{"id": orgID})
	t.Logf("🗑️ Deleted org: %s", orgID)
}

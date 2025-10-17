package tests

import (
	"encoding/json"
	"reflect"
	"testing"
) //fmt time

func TestSchemaVersioningAndEntityRollback(t *testing.T) {
	createOrgMutation := `
		mutation ($input: CreateOrganizationInput!) {
			createOrganization(input: $input) { id name }
		}
	`
	orgResp := sendGraphQLRequest(t, createOrgMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Versioned Org 2",
			"description": "Organization for versioning test",
		},
	})
	org := orgResp["createOrganization"].(map[string]interface{})
	orgID := org["id"].(string)

	createSchemaMutation := `
		mutation ($input: CreateEntitySchemaInput!) {
			createEntitySchema(input: $input) { id name version status }
		}
	`
	schemaResp := sendGraphQLRequest(t, createSchemaMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Component",
			"description":    "Component schema",
			"fields": []map[string]interface{}{
				{
					"name":     "name",
					"type":     "STRING",
					"required": true,
				},
				{
					"name":     "description",
					"type":     "STRING",
					"required": false,
				},
			},
		},
	})
	createdSchema := schemaResp["createEntitySchema"].(map[string]interface{})
	schemaID := createdSchema["id"].(string)
	if version := createdSchema["version"].(string); version != "1.0.0" {
		t.Fatalf("expected initial schema version 1.0.0, got %s", version)
	}

	addFieldMutation := `
		mutation ($id: String!, $field: FieldDefinitionInput!) {
			addFieldToSchema(schemaId: $id, field: $field) { id version }
		}
	`
	addFieldResp := sendGraphQLRequest(t, addFieldMutation, map[string]interface{}{
		"id": schemaID,
		"field": map[string]interface{}{
			"name":     "notes",
			"type":     "STRING",
			"required": false,
		},
	})
	addedSchema := addFieldResp["addFieldToSchema"].(map[string]interface{})
	if version := addedSchema["version"].(string); version != "1.1.0" {
		t.Fatalf("expected schema version 1.1.0 after additive change, got %s", version)
	}
	schemaID = addedSchema["id"].(string) // <-- UPDATE ID

	updateSchemaMutation := `
		mutation ($input: UpdateEntitySchemaInput!) {
			updateEntitySchema(input: $input) { id version }
		}
	`
	updateResp := sendGraphQLRequest(t, updateSchemaMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"id": schemaID,
			"fields": []map[string]interface{}{
				{
					"name":     "name",
					"type":     "STRING",
					"required": true,
				},
				{
					"name":     "description",
					"type":     "STRING",
					"required": true,
				},
				{
					"name":     "notes",
					"type":     "STRING",
					"required": false,
				},
			},
		},
	})
	updatedSchema := updateResp["updateEntitySchema"].(map[string]interface{})
	if version := updatedSchema["version"].(string); version != "2.0.0" {
		t.Fatalf("expected schema version 2.0.0 after breaking change, got %s", version)
	}
	schemaID = updatedSchema["id"].(string) // <-- UPDATE ID

	listVersionsQuery := `
		query ($org: String!, $name: String!) {
			entitySchemaVersions(organizationId: $org, name: $name) {
				version
				status
			}
		}
	`
	versionsResp := sendGraphQLRequest(t, listVersionsQuery, map[string]interface{}{
		"org":  orgID,
		"name": "Component",
	})
	versions := versionsResp["entitySchemaVersions"].([]interface{})
	if len(versions) != 3 {
		t.Fatalf("expected 3 schema versions, got %d", len(versions))
	}
	firstVersion := versions[0].(map[string]interface{})
	if firstVersion["version"].(string) != "2.0.0" {
		t.Errorf("expected latest version to be 2.0.0, got %s", firstVersion["version"])
	}

	createEntityMutation := `
		mutation ($input: CreateEntityInput!) {
			createEntity(input: $input) { id schemaId version properties }
		}
	`
	initialProperties := `{"name":"Core","description":"Initial","notes":"n1"}`
	entityResp := sendGraphQLRequest(t, createEntityMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Component",
			"properties":     initialProperties,
		},
	})
	entity := entityResp["createEntity"].(map[string]interface{})
	entityID := entity["id"].(string)
	if entity["version"].(float64) != 1 {
		t.Fatalf("expected entity version 1, got %v", entity["version"])
	}

	updateEntityMutation := `
		mutation ($input: UpdateEntityInput!) {
			updateEntity(input: $input) { id version properties }
		}
	`
	updatedProperties := `{"name":"Core","description":"Updated","notes":"n2"}`
	updateEntityResp := sendGraphQLRequest(t, updateEntityMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"id":         entityID,
			"properties": updatedProperties,
		},
	})
	entityAfterUpdate := updateEntityResp["updateEntity"].(map[string]interface{})
	if entityAfterUpdate["version"].(float64) != 2 {
		t.Fatalf("expected entity version 2 after update, got %v", entityAfterUpdate["version"])
	}

	rollbackMutation := `
		mutation ($id: String!, $version: Int!, $reason: String) {
			rollbackEntity(id: $id, toVersion: $version, reason: $reason) { id version properties }
		}
	`
	rollbackResp := sendGraphQLRequest(t, rollbackMutation, map[string]interface{}{
		"id":      entityID,
		"version": 1,
		"reason":  "restore baseline",
	})
	rolledEntity := rollbackResp["rollbackEntity"].(map[string]interface{})
	if rolledEntity["version"].(float64) != 3 {
		t.Fatalf("expected entity version 3 after rollback, got %v", rolledEntity["version"])
	}
	if !jsonPropsEqual(t, rolledEntity["properties"].(string), initialProperties) {
		t.Fatalf("expected properties %s after rollback, got %s", initialProperties, rolledEntity["properties"].(string))
	}

	deleteSchemaMutation := `mutation($id: String!) { deleteEntitySchema(id: $id) }`
	deleteResp := sendGraphQLRequest(t, deleteSchemaMutation, map[string]interface{}{"id": schemaID})
	if deleted, ok := deleteResp["deleteEntitySchema"].(bool); !ok || !deleted {
		t.Fatal("expected deleteEntitySchema to succeed")
	}
	//fmt.Printf("DeleteEntitySchema called for ID=%s\n", schemaID)

	versionsAfterDelete := sendGraphQLRequest(t, listVersionsQuery, map[string]interface{}{
		"org":  orgID,
		"name": "Component",
	})["entitySchemaVersions"].([]interface{})
	if len(versionsAfterDelete) != 3 {
		t.Fatalf("expected 4 schema versions after archive, got %d", len(versionsAfterDelete))
	}
	archivedVersion := versionsAfterDelete[0].(map[string]interface{})
	if status := archivedVersion["status"].(string); status != "ARCHIVED" {
		t.Fatalf("expected latest version status ARCHIVED, got %s", status)
	}

	//x, _ := json.MarshalIndent(archivedVersion, "", "  ")
	//t.Logf("Schemas after archival:\n%s", x)

	//time.Sleep(100 * time.Millisecond)

	listSchemasQuery := `query ($org: String!) { entitySchemas(organizationId: $org) { id name description status } }`
	listResp := sendGraphQLRequest(t, listSchemasQuery, map[string]interface{}{"org": orgID})
	list := listResp["entitySchemas"].([]interface{})

	//b, _ := json.MarshalIndent(list, "", "  ")
	//t.Logf("Active schemas after archival:\n%s", b)

	if len(list) != 0 {
		t.Fatalf("expected no active schemas after archival, got %d", len(list))
	}
}

func jsonPropsEqual(t *testing.T, actual, expected string) bool {
	var actMap map[string]any
	var expMap map[string]any
	if err := json.Unmarshal([]byte(actual), &actMap); err != nil {
		t.Fatalf("failed to unmarshal actual properties %s: %v", actual, err)
	}
	if err := json.Unmarshal([]byte(expected), &expMap); err != nil {
		t.Fatalf("failed to unmarshal expected properties %s: %v", expected, err)
	}
	return reflect.DeepEqual(actMap, expMap)
}

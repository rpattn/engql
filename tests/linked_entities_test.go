package tests

import (
	"encoding/json"
	"testing"
)

func TestLinkedEntitiesAutoResolution(t *testing.T) {
	createOrg := `
		mutation ($input: CreateOrganizationInput!) {
			createOrganization(input: $input) { id name }
		}
	`
	orgVars := map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Linked Entities Org",
			"description": "Org for linked entity resolver coverage",
		},
	}
	orgResp := sendGraphQLRequest(t, createOrg, orgVars)
	org := orgResp["createOrganization"].(map[string]interface{})
	orgID := org["id"].(string)
	t.Logf("[setup] created organization %s", orgID)

	createSchema := `
		mutation ($input: CreateEntitySchemaInput!) {
			createEntitySchema(input: $input) {
				id
				name
				fields { name type referenceEntityType }
			}
		}
	`
	schemaVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Component",
			"description":    "Component schema with entity references",
			"fields": []map[string]interface{}{
				{
					"name":     "name",
					"type":     "STRING",
					"required": true,
				},
				{
					"name":                "linked_ids",
					"type":                "ENTITY_REFERENCE_ARRAY",
					"required":            false,
					"referenceEntityType": "Component",
				},
			},
		},
	}
	schemaResp := sendGraphQLRequest(t, createSchema, schemaVars)
	schema := schemaResp["createEntitySchema"].(map[string]interface{})
	schemaID := schema["id"].(string)
	t.Logf("[setup] created schema %s with ENTITY_REFERENCE_ARRAY field", schemaID)

	createEntity := `
		mutation ($input: CreateEntityInput!) {
			createEntity(input: $input) {
				id
				properties
			}
		}
	`

	parentProps, _ := json.Marshal(map[string]interface{}{
		"name": "Parent Component",
	})
	parentVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Component",
			"path":           "root.parent",
			"properties":     string(parentProps),
		},
	}
	parentResp := sendGraphQLRequest(t, createEntity, parentVars)
	parent := parentResp["createEntity"].(map[string]interface{})
	parentID := parent["id"].(string)
	t.Logf("[setup] created parent component %s", parentID)

	childProps, _ := json.Marshal(map[string]interface{}{
		"name": "Child Component",
	})
	childVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Component",
			"path":           "root.parent.child",
			"properties":     string(childProps),
			"linkedEntityId": parentID,
		},
	}
	childResp := sendGraphQLRequest(t, createEntity, childVars)
	child := childResp["createEntity"].(map[string]interface{})
	childID := child["id"].(string)
	t.Logf("[setup] created child component %s linked to %s", childID, parentID)

	var storedProps map[string]interface{}
	if err := json.Unmarshal([]byte(child["properties"].(string)), &storedProps); err != nil {
		t.Fatalf("? failed to parse child properties JSON: %v", err)
	}
	rawLinked, ok := storedProps["linked_ids"].([]interface{})
	if !ok || len(rawLinked) != 1 || rawLinked[0].(string) != parentID {
		t.Fatalf("? expected linked_ids to contain parent id, got %#v", storedProps["linked_ids"])
	}
	t.Log("[assert] linked_ids persisted to child properties")

	entitiesByIDs := `
		query ($ids: [String!]!) {
			entitiesByIDs(ids: $ids) {
				id
				linkedEntities { id }
			}
		}
	`
	byIDsResp := sendGraphQLRequest(t, entitiesByIDs, map[string]interface{}{"ids": []string{childID}})
	byIDsEntities := byIDsResp["entitiesByIDs"].([]interface{})
	if len(byIDsEntities) != 1 {
		t.Fatalf("? expected 1 entity, got %d", len(byIDsEntities))
	}
	byIDs := byIDsEntities[0].(map[string]interface{})
	linkedSlice, ok := byIDs["linkedEntities"].([]interface{})
	if !ok || len(linkedSlice) != 1 {
		t.Fatalf("? expected 1 linked entity via entitiesByIDs, got %#v", byIDs["linkedEntities"])
	}
	linkedEntity := linkedSlice[0].(map[string]interface{})
	if linkedEntity["id"] != parentID {
		t.Fatalf("? linked entity mismatch, got %v want %v", linkedEntity["id"], parentID)
	}
	t.Log("[assert] entitiesByIDs resolver hydrated linked entities")

	entitiesByType := `
		query ($org: String!, $type: String!) {
			entitiesByType(organizationId: $org, entityType: $type) {
				id
				linkedEntities { id }
			}
		}
	`
	typeResp := sendGraphQLRequest(t, entitiesByType, map[string]interface{}{
		"org":  orgID,
		"type": "Component",
	})
	typeEntities := typeResp["entitiesByType"].([]interface{})
	var childFound, parentFound bool
	for _, raw := range typeEntities {
		entity := raw.(map[string]interface{})
		switch entity["id"] {
		case childID:
			childFound = true
			linked := entity["linkedEntities"].([]interface{})
			if len(linked) != 1 || linked[0].(map[string]interface{})["id"] != parentID {
				t.Fatalf("? entitiesByType did not resolve linked entity for child: %#v", entity["linkedEntities"])
			}
		case parentID:
			parentFound = true
			if len(entity["linkedEntities"].([]interface{})) != 0 {
				t.Fatalf("? parent should not have linked entities, got %#v", entity["linkedEntities"])
			}
		}
	}
	if !childFound || !parentFound {
		t.Fatalf("? entitiesByType response missing parent (%v) or child (%v)", parentFound, childFound)
	}
	t.Log("[assert] entitiesByType returns both nodes with resolved links")

	deleteEntity := `
		mutation ($id: String!) {
			deleteEntity(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteEntity, map[string]interface{}{"id": childID})
	sendGraphQLRequest(t, deleteEntity, map[string]interface{}{"id": parentID})
	t.Log("[cleanup] removed child and parent entities")

	deleteSchema := `
		mutation ($id: String!) {
			deleteEntitySchema(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteSchema, map[string]interface{}{"id": schemaID})
	t.Log("[cleanup] removed schema")

	deleteOrg := `
		mutation ($id: String!) {
			deleteOrganization(id: $id)
		}
	`
	sendGraphQLRequest(t, deleteOrg, map[string]interface{}{"id": orgID})
	t.Log("[cleanup] removed organization")
}

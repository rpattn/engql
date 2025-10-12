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

	secondaryProps, _ := json.Marshal(map[string]interface{}{
		"name": "Secondary Component",
	})
	secondaryVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Component",
			"path":           "root.secondary",
			"properties":     string(secondaryProps),
		},
	}
	secondaryResp := sendGraphQLRequest(t, createEntity, secondaryVars)
	secondary := secondaryResp["createEntity"].(map[string]interface{})
	secondaryID := secondary["id"].(string)
	t.Logf("[setup] created secondary component %s", secondaryID)

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
			"linkedEntityIds": []string{
				parentID,
				secondaryID,
			},
		},
	}
	childResp := sendGraphQLRequest(t, createEntity, childVars)
	child := childResp["createEntity"].(map[string]interface{})
	childID := child["id"].(string)
	t.Logf("[setup] created child component %s linked to %s and %s", childID, parentID, secondaryID)

	var storedProps map[string]interface{}
	if err := json.Unmarshal([]byte(child["properties"].(string)), &storedProps); err != nil {
		t.Fatalf("? failed to parse child properties JSON: %v", err)
	}
	rawLinked, ok := storedProps["linked_ids"].([]interface{})
	if !ok {
		t.Fatalf("? expected linked_ids array in child properties, got %#v", storedProps["linked_ids"])
	}
	if len(rawLinked) != 2 {
		t.Fatalf("? expected two linked ids, got %#v", rawLinked)
	}
	linkedSet := make(map[string]struct{})
	for _, item := range rawLinked {
		id, ok := item.(string)
		if !ok {
			t.Fatalf("? linked_ids entry not string: %#v", item)
		}
		linkedSet[id] = struct{}{}
	}
	if _, ok := linkedSet[parentID]; !ok {
		t.Fatalf("? linked_ids missing primary parent %s", parentID)
	}
	if _, ok := linkedSet[secondaryID]; !ok {
		t.Fatalf("? linked_ids missing secondary parent %s", secondaryID)
	}
	t.Log("[assert] linked_ids persisted to child properties with both parents")

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
	if !ok || len(linkedSlice) != 2 {
		t.Fatalf("? expected 2 linked entities via entitiesByIDs, got %#v", byIDs["linkedEntities"])
	}
	foundIDs := make(map[string]struct{})
	for _, item := range linkedSlice {
		entity := item.(map[string]interface{})
		foundIDs[entity["id"].(string)] = struct{}{}
	}
	if _, ok := foundIDs[parentID]; !ok {
		t.Fatalf("? entitiesByIDs missing parent %s", parentID)
	}
	if _, ok := foundIDs[secondaryID]; !ok {
		t.Fatalf("? entitiesByIDs missing secondary parent %s", secondaryID)
	}
	t.Log("[assert] entitiesByIDs resolver hydrated both linked entities")

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
	var childFound, parentFound, secondaryFound bool
	for _, raw := range typeEntities {
		entity := raw.(map[string]interface{})
		switch entity["id"] {
		case childID:
			childFound = true
			linked := entity["linkedEntities"].([]interface{})
			if len(linked) != 2 {
				t.Fatalf("? entitiesByType expected 2 linked entities for child, got %#v", linked)
			}
			childLinks := make(map[string]struct{})
			for _, item := range linked {
				link := item.(map[string]interface{})
				childLinks[link["id"].(string)] = struct{}{}
			}
			if _, ok := childLinks[parentID]; !ok {
				t.Fatalf("? entitiesByType child missing parent %s", parentID)
			}
			if _, ok := childLinks[secondaryID]; !ok {
				t.Fatalf("? entitiesByType child missing secondary parent %s", secondaryID)
			}
		case parentID:
			parentFound = true
			if len(entity["linkedEntities"].([]interface{})) != 0 {
				t.Fatalf("? parent should not have linked entities, got %#v", entity["linkedEntities"])
			}
		case secondaryID:
			secondaryFound = true
			if len(entity["linkedEntities"].([]interface{})) != 0 {
				t.Fatalf("? secondary parent should not have linked entities, got %#v", entity["linkedEntities"])
			}
		}
	}
	if !childFound || !parentFound || !secondaryFound {
		t.Fatalf("? entitiesByType response missing parent (%v), secondary (%v) or child (%v)", parentFound, secondaryFound, childFound)
	}
	t.Log("[assert] entitiesByType returns nodes with resolved links")

	deleteEntity := `
		mutation ($id: String!) {
			deleteEntity(id: $id)
		}
	`
	for _, target := range []struct {
		name string
		id   string
	}{
		{"child", childID},
		{"parent", parentID},
		{"secondary parent", secondaryID},
	} {
		resp := sendGraphQLRequest(t, deleteEntity, map[string]interface{}{"id": target.id})
		if success, ok := resp["deleteEntity"].(bool); !ok || !success {
			t.Fatalf("? failed to delete %s entity %s", target.name, target.id)
		}
	}
	t.Log("[cleanup] removed child and parent entities")

	deleteSchema := `
		mutation ($id: String!) {
			deleteEntitySchema(id: $id)
		}
	`
	deleteSchemaResp := sendGraphQLRequest(t, deleteSchema, map[string]interface{}{"id": schemaID})
	if success, ok := deleteSchemaResp["deleteEntitySchema"].(bool); !ok || !success {
		t.Fatalf("? failed to delete entity schema %s", schemaID)
	}
	t.Log("[cleanup] removed schema")

	deleteOrg := `
		mutation ($id: String!) {
			deleteOrganization(id: $id)
		}
	`
	deleteOrgResp := sendGraphQLRequest(t, deleteOrg, map[string]interface{}{"id": orgID})
	if success, ok := deleteOrgResp["deleteOrganization"].(bool); !ok || !success {
		t.Fatalf("? failed to delete organization %s", orgID)
	}
	t.Log("[cleanup] removed organization")
}

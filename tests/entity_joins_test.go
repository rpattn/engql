package tests

import (
	"encoding/json"
	"testing"
)

func TestEntityJoinDefinitionLifecycle(t *testing.T) {
	createOrgMutation := `
		mutation ($input: CreateOrganizationInput!) {
			createOrganization(input: $input) { id name }
		}
	`
	orgData := sendGraphQLRequest(t, createOrgMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Join Definition Org",
			"description": "Org used for join definition lifecycle test",
		},
	})
	org := orgData["createOrganization"].(map[string]interface{})
	orgID := org["id"].(string)
	t.Logf("[setup] created organization %s", orgID)

	createSchemaMutation := `
		mutation ($input: CreateEntitySchemaInput!) {
			createEntitySchema(input: $input) { id name }
		}
	`

	teamSchemaVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Team",
			"description":    "Team schema for join tests",
			"fields": []map[string]interface{}{
				{
					"name":     "name",
					"type":     "STRING",
					"required": true,
				},
			},
		},
	}
	teamSchemaResp := sendGraphQLRequest(t, createSchemaMutation, teamSchemaVars)
	teamSchema := teamSchemaResp["createEntitySchema"].(map[string]interface{})
	teamSchemaID := teamSchema["id"].(string)
	t.Logf("[setup] created team schema %s", teamSchemaID)

	componentSchemaVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "Component",
			"description":    "Component schema referencing teams",
			"fields": []map[string]interface{}{
				{
					"name":     "name",
					"type":     "STRING",
					"required": true,
				},
				{
					"name":                "owner",
					"type":                "ENTITY_REFERENCE",
					"required":            true,
					"referenceEntityType": "Team",
				},
			},
		},
	}
	componentSchemaResp := sendGraphQLRequest(t, createSchemaMutation, componentSchemaVars)
	componentSchema := componentSchemaResp["createEntitySchema"].(map[string]interface{})
	componentSchemaID := componentSchema["id"].(string)
	t.Logf("[setup] created component schema %s", componentSchemaID)

	createEntityMutation := `
		mutation ($input: CreateEntityInput!) {
			createEntity(input: $input) { id properties }
		}
	`

	teamProps, _ := json.Marshal(map[string]interface{}{
		"name": "Team Atlas",
	})
	teamEntityVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Team",
			"path":           "teams.atlas",
			"properties":     string(teamProps),
		},
	}
	teamEntityResp := sendGraphQLRequest(t, createEntityMutation, teamEntityVars)
	teamEntity := teamEntityResp["createEntity"].(map[string]interface{})
	teamID := teamEntity["id"].(string)
	t.Logf("[setup] created team entity %s", teamID)

	componentProps, _ := json.Marshal(map[string]interface{}{
		"name":  "Component Falcon",
		"owner": teamID,
	})
	componentVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"entityType":     "Component",
			"path":           "components.falcon",
			"properties":     string(componentProps),
		},
	}
	componentResp := sendGraphQLRequest(t, createEntityMutation, componentVars)
	componentEntity := componentResp["createEntity"].(map[string]interface{})
	componentID := componentEntity["id"].(string)
	t.Logf("[setup] created component entity %s", componentID)

	createJoinMutation := `
		mutation ($input: CreateEntityJoinDefinitionInput!) {
			createEntityJoinDefinition(input: $input) {
				id
				name
				joinType
				joinField
				joinFieldType
				leftFilters { key value }
				sortCriteria { side field direction }
			}
		}
	`
	createJoinVars := map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId":  orgID,
			"name":            "ComponentsToTeams",
			"description":     "Join component owners to team definitions",
			"leftEntityType":  "Component",
			"rightEntityType": "Team",
			"joinField":       "owner",
			"leftFilters": []map[string]interface{}{
				{"key": "name", "value": "Component Falcon"},
			},
			"sortCriteria": []map[string]interface{}{
				{"side": "LEFT", "field": "name", "direction": "ASC"},
			},
		},
	}
	createJoinResp := sendGraphQLRequest(t, createJoinMutation, createJoinVars)
	joinData := createJoinResp["createEntityJoinDefinition"].(map[string]interface{})
	joinID := joinData["id"].(string)
	if joinData["joinFieldType"].(string) != "ENTITY_REFERENCE" {
		t.Fatalf("? expected join field type ENTITY_REFERENCE, got %s", joinData["joinFieldType"])
	}
	if joinData["joinType"].(string) != "REFERENCE" {
		t.Fatalf("? expected join type REFERENCE, got %s", joinData["joinType"])
	}
	t.Logf("[assert] created join definition %s", joinID)

	listJoinsQuery := `
		query ($org: String!) {
			entityJoinDefinitions(organizationId: $org) {
				id
				name
				joinType
				leftEntityType
				rightEntityType
				joinField
			}
		}
	`
	listResp := sendGraphQLRequest(t, listJoinsQuery, map[string]interface{}{"org": orgID})
	definitions := listResp["entityJoinDefinitions"].([]interface{})
	if len(definitions) != 1 {
		t.Fatalf("? expected 1 join definition, got %d", len(definitions))
	}
	listed := definitions[0].(map[string]interface{})
	if listed["id"].(string) != joinID {
		t.Fatalf("? expected join ID %s, got %s", joinID, listed["id"])
	}
	t.Log("[assert] join definition listed with correct metadata")

	executeJoinQuery := `
		query ($input: ExecuteEntityJoinInput!) {
			executeEntityJoin(input: $input) {
				edges {
					left { id properties }
					right { id properties }
				}
				pageInfo { totalCount hasNextPage hasPreviousPage }
			}
		}
	`
	executeResp := sendGraphQLRequest(t, executeJoinQuery, map[string]interface{}{
		"input": map[string]interface{}{
			"joinId": joinID,
			"pagination": map[string]interface{}{
				"limit":  10,
				"offset": 0,
			},
		},
	})
	executeData := executeResp["executeEntityJoin"].(map[string]interface{})
	edges := executeData["edges"].([]interface{})
	if len(edges) != 1 {
		t.Fatalf("? expected 1 join edge, got %d", len(edges))
	}
	firstEdge := edges[0].(map[string]interface{})
	left := firstEdge["left"].(map[string]interface{})
	right := firstEdge["right"].(map[string]interface{})
	if left["id"].(string) != componentID {
		t.Fatalf("? expected left entity %s, got %s", componentID, left["id"])
	}
	if right["id"].(string) != teamID {
		t.Fatalf("? expected right entity %s, got %s", teamID, right["id"])
	}
	pageInfo := executeData["pageInfo"].(map[string]interface{})
	if int(pageInfo["totalCount"].(float64)) != 1 {
		t.Fatalf("? expected totalCount 1, got %v", pageInfo["totalCount"])
	}
	if pageInfo["hasNextPage"].(bool) {
		t.Fatalf("? expected hasNextPage false, got true")
	}
	t.Log("[assert] executeEntityJoin returned expected edge")

	updateJoinMutation := `
		mutation ($input: UpdateEntityJoinDefinitionInput!) {
			updateEntityJoinDefinition(input: $input) {
				id
				name
				sortCriteria { field direction }
			}
		}
	`
	updateResp := sendGraphQLRequest(t, updateJoinMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"id":   joinID,
			"name": "ComponentsJoinedToTeams",
			"sortCriteria": []map[string]interface{}{
				{"side": "RIGHT", "field": "name", "direction": "DESC"},
			},
		},
	})
	updated := updateResp["updateEntityJoinDefinition"].(map[string]interface{})
	if updated["name"].(string) != "ComponentsJoinedToTeams" {
		t.Fatalf("? expected updated name, got %s", updated["name"])
	}
	sortRules := updated["sortCriteria"].([]interface{})
	if len(sortRules) != 1 || sortRules[0].(map[string]interface{})["field"].(string) != "name" {
		t.Fatalf("? expected updated sortCriteria, got %#v", sortRules)
	}
	t.Log("[assert] join definition updated")

	deleteJoinMutation := `
		mutation ($id: String!) {
			deleteEntityJoinDefinition(id: $id)
		}
	`
	deleteJoinResp := sendGraphQLRequest(t, deleteJoinMutation, map[string]interface{}{"id": joinID})
	if !deleteJoinResp["deleteEntityJoinDefinition"].(bool) {
		t.Fatalf("? expected join definition deletion to succeed")
	}
	t.Log("[cleanup] deleted join definition")

	deleteEntityMutation := `
		mutation ($id: String!) { deleteEntity(id: $id) }
	`
	for _, target := range []string{componentID, teamID} {
		sendGraphQLRequest(t, deleteEntityMutation, map[string]interface{}{"id": target})
	}

	deleteSchemaMutation := `
		mutation ($id: String!) { deleteEntitySchema(id: $id) }
	`
	for _, schemaID := range []string{componentSchemaID, teamSchemaID} {
		sendGraphQLRequest(t, deleteSchemaMutation, map[string]interface{}{"id": schemaID})
	}

	deleteOrgMutation := `
		mutation ($id: String!) { deleteOrganization(id: $id) }
	`
	sendGraphQLRequest(t, deleteOrgMutation, map[string]interface{}{"id": orgID})
	t.Log("[cleanup] removed organization")
}

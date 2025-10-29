package tests

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestTransformationExecutionQuery(t *testing.T) {
	createOrgQuery := `
        mutation ($input: CreateOrganizationInput!) {
            createOrganization(input: $input) { id }
        }
    `
	orgResp := sendGraphQLRequest(t, createOrgQuery, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Transformation Org",
			"description": "Integration test org",
		},
	})
	orgID := orgResp["createOrganization"].(map[string]interface{})["id"].(string)

	t.Cleanup(func() {
		deleteOrgQuery := `mutation ($id: String!) { deleteOrganization(id: $id) }`
		sendGraphQLRequest(t, deleteOrgQuery, map[string]interface{}{"id": orgID})
	})

	createSchemaQuery := `
        mutation ($input: CreateEntitySchemaInput!) {
            createEntitySchema(input: $input) { id }
        }
    `
	schemaResp := sendGraphQLRequest(t, createSchemaQuery, map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "User",
			"description":    "User schema for transformation tests",
			"fields": []map[string]interface{}{
				{"name": "firstName", "type": "STRING", "required": true},
				{"name": "status", "type": "STRING", "required": true},
			},
		},
	})
	schemaID := schemaResp["createEntitySchema"].(map[string]interface{})["id"].(string)

	t.Cleanup(func() {
		deleteSchemaQuery := `mutation ($id: String!) { deleteEntitySchema(id: $id) }`
		sendGraphQLRequest(t, deleteSchemaQuery, map[string]interface{}{"id": schemaID})
	})

	createEntityQuery := `
        mutation ($input: CreateEntityInput!) {
            createEntity(input: $input) { id }
        }
    `

	createEntity := func(firstName, status string) string {
		props, _ := json.Marshal(map[string]any{
			"firstName": firstName,
			"status":    status,
		})
		resp := sendGraphQLRequest(t, createEntityQuery, map[string]interface{}{
			"input": map[string]interface{}{
				"organizationId": orgID,
				"entityType":     "User",
				"path":           "root",
				"properties":     string(props),
			},
		})
		return resp["createEntity"].(map[string]interface{})["id"].(string)
	}

	entity1 := createEntity("Alice", "active")
	entity2 := createEntity("Bob", "inactive")
	entity3 := createEntity("Charlie", "active")

	t.Cleanup(func() {
		deleteEntityQuery := `mutation ($id: String!) { deleteEntity(id: $id) }`
		for _, id := range []string{entity1, entity2, entity3} {
			sendGraphQLRequest(t, deleteEntityQuery, map[string]interface{}{"id": id})
		}
	})

	loadNodeID := uuid.NewString()
	materializeNodeID := uuid.NewString()

	createTransformationQuery := `
        mutation ($input: CreateEntityTransformationInput!) {
            createEntityTransformation(input: $input) { id }
        }
    `

	transformationResp := sendGraphQLRequest(t, createTransformationQuery, map[string]interface{}{
		"input": map[string]interface{}{
			"organizationId": orgID,
			"name":           "User table",
			"description":    "materialize users",
			"nodes": []map[string]interface{}{
				{
					"id":   loadNodeID,
					"name": "load-users",
					"type": "LOAD",
					"load": map[string]interface{}{
						"alias":      "users",
						"entityType": "User",
					},
				},
				{
					"id":     materializeNodeID,
					"name":   "materialize-users",
					"type":   "MATERIALIZE",
					"inputs": []string{loadNodeID},
					"materialize": map[string]interface{}{
						"outputs": []map[string]interface{}{
							{
								"alias": "table",
								"fields": []map[string]interface{}{
									{"sourceAlias": "users", "sourceField": "firstName", "outputField": "firstName"},
									{"sourceAlias": "users", "sourceField": "status", "outputField": "status"},
								},
							},
						},
					},
				},
			},
		},
	})
	transformationID := transformationResp["createEntityTransformation"].(map[string]interface{})["id"].(string)

	t.Cleanup(func() {
		deleteTransformationQuery := `mutation ($id: String!) { deleteEntityTransformation(id: $id) }`
		sendGraphQLRequest(t, deleteTransformationQuery, map[string]interface{}{"id": transformationID})
	})

	executionQuery := `
        query (
            $id: String!
            $filters: [TransformationExecutionFilterInput!]
            $sort: TransformationExecutionSortInput
            $pagination: PaginationInput
        ) {
            transformationExecution(
                transformationId: $id
                filters: $filters
                sort: $sort
                pagination: $pagination
            ) {
                columns { key alias field label sourceAlias sourceField }
                rows { values { columnKey value } }
                pageInfo { totalCount hasNextPage hasPreviousPage }
            }
        }
    `

	baseResp := sendGraphQLRequest(t, executionQuery, map[string]interface{}{"id": transformationID})
	baseResult := baseResp["transformationExecution"].(map[string]interface{})

	columns := baseResult["columns"].([]interface{})
	if len(columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(columns))
	}

	firstColumn := columns[0].(map[string]interface{})
	if firstColumn["key"].(string) != "table.firstName" {
		t.Fatalf("unexpected column key %v", firstColumn["key"])
	}

	rows := baseResult["rows"].([]interface{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	names := make(map[string]struct{})
	for _, raw := range rows {
		row := raw.(map[string]interface{})
		values := row["values"].([]interface{})
		valueMap := make(map[string]string)
		for _, v := range values {
			cell := v.(map[string]interface{})
			key := cell["columnKey"].(string)
			if val, ok := cell["value"].(string); ok {
				valueMap[key] = val
			}
		}
		name := valueMap["table.firstName"]
		names[name] = struct{}{}
	}

	for _, expected := range []string{"Alice", "Bob", "Charlie"} {
		if _, ok := names[expected]; !ok {
			t.Fatalf("missing row for %s", expected)
		}
	}

	filterResp := sendGraphQLRequest(t, executionQuery, map[string]interface{}{
		"id": transformationID,
		"filters": []map[string]interface{}{
			{"alias": "table", "field": "status", "value": "active"},
		},
	})
	filterRows := filterResp["transformationExecution"].(map[string]interface{})["rows"].([]interface{})
	if len(filterRows) != 2 {
		t.Fatalf("expected 2 filtered rows, got %d", len(filterRows))
	}

	sortResp := sendGraphQLRequest(t, executionQuery, map[string]interface{}{
		"id": transformationID,
		"sort": map[string]interface{}{
			"alias":     "table",
			"field":     "firstName",
			"direction": "DESC",
		},
	})
	sortRows := sortResp["transformationExecution"].(map[string]interface{})["rows"].([]interface{})
	firstSorted := sortRows[0].(map[string]interface{})
	firstSortedValues := firstSorted["values"].([]interface{})
	var firstName string
	for _, v := range firstSortedValues {
		cell := v.(map[string]interface{})
		if cell["columnKey"].(string) == "table.firstName" {
			if value, ok := cell["value"].(string); ok {
				firstName = value
			}
		}
	}
	if firstName != "Charlie" {
		t.Fatalf("expected first row to be Charlie after sort, got %s", firstName)
	}

	paginationResp := sendGraphQLRequest(t, executionQuery, map[string]interface{}{
		"id": transformationID,
		"sort": map[string]interface{}{
			"alias":     "table",
			"field":     "firstName",
			"direction": "ASC",
		},
		"pagination": map[string]interface{}{
			"limit":  1,
			"offset": 1,
		},
	})

	paginationResult := paginationResp["transformationExecution"].(map[string]interface{})
	pagedRows := paginationResult["rows"].([]interface{})
	if len(pagedRows) != 1 {
		t.Fatalf("expected 1 paginated row, got %d", len(pagedRows))
	}

	pageInfo := paginationResult["pageInfo"].(map[string]interface{})
	if pageInfo["totalCount"].(float64) != 3 {
		t.Fatalf("expected total count 3, got %v", pageInfo["totalCount"])
	}
	if hasNext, _ := pageInfo["hasNextPage"].(bool); !hasNext {
		t.Fatalf("expected next page to be true")
	}
	if hasPrev, _ := pageInfo["hasPreviousPage"].(bool); !hasPrev {
		t.Fatalf("expected previous page to be true")
	}
}

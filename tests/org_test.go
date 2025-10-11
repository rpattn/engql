package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type GraphQLRequest struct {
	Query string `json:"query"`
}

type GraphQLResponse struct {
	Data   map[string]interface{}   `json:"data"`
	Errors []map[string]interface{} `json:"errors"`
}

func TestOrganizations(t *testing.T) {
	// Arrange: Build GraphQL query
	query := `
		query {
			organizations {
				id
				name
				description
				createdAt
			}
		}
	`

	req := GraphQLRequest{Query: query}
	reqBody, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Error marshaling request: %v", err)
	}

	// Act: Send HTTP request
	resp, err := http.Post("http://localhost:8080/query", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response: %v", err)
	}

	// Assert: Parse and verify response
	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		t.Fatalf("Error parsing response: %v\nRaw response: %s", err, string(body))
	}

	if len(graphqlResp.Errors) > 0 {
		t.Fatalf("GraphQL returned errors: %v", graphqlResp.Errors)
	}

	if graphqlResp.Data == nil {
		t.Fatalf("Expected data, got nil")
	}

	t.Logf("✅ Success! Response: %+v", graphqlResp.Data)
}

package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const baseURL = "http://localhost:8080/query"

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   map[string]interface{}   `json:"data"`
	Errors []map[string]interface{} `json:"errors"`
}

// Helper to execute GraphQL queries/mutations
func gqlRequest(t *testing.T, query string, variables map[string]interface{}) GraphQLResponse {
	t.Helper()

	reqBody, err := json.Marshal(GraphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		t.Fatalf("failed to parse response: %v\nRaw: %s", err, string(body))
	}

	if len(gqlResp.Errors) > 0 {
		t.Fatalf("GraphQL returned errors: %+v", gqlResp.Errors)
	}

	return gqlResp
}

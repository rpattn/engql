package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JoinSide string

const (
	JoinSideLeft  JoinSide = "LEFT"
	JoinSideRight JoinSide = "RIGHT"
)

type JoinSortDirection string

const (
	JoinSortAsc  JoinSortDirection = "ASC"
	JoinSortDesc JoinSortDirection = "DESC"
)

// JoinPropertyFilter mirrors the GraphQL-level filter structure for persistence
type JoinPropertyFilter struct {
	Key     string   `json:"key"`
	Value   *string  `json:"value,omitempty"`
	Exists  *bool    `json:"exists,omitempty"`
	InArray []string `json:"inArray,omitempty"`
}

type JoinSortCriterion struct {
	Side      JoinSide          `json:"side"`
	Field     string            `json:"field"`
	Direction JoinSortDirection `json:"direction"`
}

type EntityJoinDefinition struct {
	ID              uuid.UUID            `json:"id"`
	OrganizationID  uuid.UUID            `json:"organization_id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	LeftEntityType  string               `json:"left_entity_type"`
	RightEntityType string               `json:"right_entity_type"`
	JoinField       string               `json:"join_field"`
	JoinFieldType   FieldType            `json:"join_field_type"`
	LeftFilters     []JoinPropertyFilter `json:"left_filters"`
	RightFilters    []JoinPropertyFilter `json:"right_filters"`
	SortCriteria    []JoinSortCriterion  `json:"sort_criteria"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

type JoinExecutionOptions struct {
	LeftFilters  []JoinPropertyFilter
	RightFilters []JoinPropertyFilter
	SortCriteria []JoinSortCriterion
	Limit        int
	Offset       int
}

type EntityJoinEdge struct {
	Left  Entity
	Right Entity
}

// Helper utilities for encoding/decoding filter data to JSONB blobs used by persistence.
func FiltersToJSONB(filters []JoinPropertyFilter) (json.RawMessage, error) {
	if filters == nil {
		filters = []JoinPropertyFilter{}
	}
	return json.Marshal(filters)
}

func FiltersFromJSONB(data json.RawMessage) ([]JoinPropertyFilter, error) {
	if len(data) == 0 {
		return []JoinPropertyFilter{}, nil
	}

	var filters []JoinPropertyFilter
	if err := json.Unmarshal(data, &filters); err != nil {
		return nil, err
	}
	if filters == nil {
		filters = []JoinPropertyFilter{}
	}
	return filters, nil
}

func SortCriteriaToJSONB(criteria []JoinSortCriterion) (json.RawMessage, error) {
	if criteria == nil {
		criteria = []JoinSortCriterion{}
	}
	return json.Marshal(criteria)
}

func SortCriteriaFromJSONB(data json.RawMessage) ([]JoinSortCriterion, error) {
	if len(data) == 0 {
		return []JoinSortCriterion{}, nil
	}

	var criteria []JoinSortCriterion
	if err := json.Unmarshal(data, &criteria); err != nil {
		return nil, err
	}
	if criteria == nil {
		criteria = []JoinSortCriterion{}
	}
	return criteria, nil
}

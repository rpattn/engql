package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

type EntityTransformationNodeType string

const (
	TransformationNodeLoad        EntityTransformationNodeType = "LOAD"
	TransformationNodeFilter      EntityTransformationNodeType = "FILTER"
	TransformationNodeProject     EntityTransformationNodeType = "PROJECT"
	TransformationNodeJoin        EntityTransformationNodeType = "JOIN"
	TransformationNodeLeftJoin    EntityTransformationNodeType = "LEFT_JOIN"
	TransformationNodeAntiJoin    EntityTransformationNodeType = "ANTI_JOIN"
	TransformationNodeUnion       EntityTransformationNodeType = "UNION"
	TransformationNodeSort        EntityTransformationNodeType = "SORT"
	TransformationNodePaginate    EntityTransformationNodeType = "PAGINATE"
	TransformationNodeMaterialize EntityTransformationNodeType = "MATERIALIZE"
)

type EntityTransformation struct {
	ID             uuid.UUID                  `json:"id"`
	OrganizationID uuid.UUID                  `json:"organization_id"`
	Name           string                     `json:"name"`
	Description    string                     `json:"description"`
	Nodes          []EntityTransformationNode `json:"nodes"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

type EntityTransformationNode struct {
	ID     uuid.UUID                    `json:"id"`
	Name   string                       `json:"name"`
	Type   EntityTransformationNodeType `json:"type"`
	Inputs []uuid.UUID                  `json:"inputs"`

	Load        *EntityTransformationLoadConfig        `json:"load,omitempty"`
	Filter      *EntityTransformationFilterConfig      `json:"filter,omitempty"`
	Project     *EntityTransformationProjectConfig     `json:"project,omitempty"`
	Join        *EntityTransformationJoinConfig        `json:"join,omitempty"`
	Materialize *EntityTransformationMaterializeConfig `json:"materialize,omitempty"`
	Sort        *EntityTransformationSortConfig        `json:"sort,omitempty"`
	Paginate    *EntityTransformationPaginateConfig    `json:"paginate,omitempty"`
}

type EntityTransformationLoadConfig struct {
	Alias      string           `json:"alias"`
	EntityType string           `json:"entityType"`
	Filters    []PropertyFilter `json:"filters,omitempty"`
}

type EntityTransformationFilterConfig struct {
	Alias   string           `json:"alias"`
	Filters []PropertyFilter `json:"filters,omitempty"`
}

type EntityTransformationProjectConfig struct {
	Alias  string   `json:"alias"`
	Fields []string `json:"fields"`
}

type EntityTransformationJoinConfig struct {
	LeftAlias  string `json:"leftAlias"`
	RightAlias string `json:"rightAlias"`
	OnField    string `json:"onField"`
}

type EntityTransformationMaterializeConfig struct {
	Outputs []EntityTransformationMaterializeOutput `json:"outputs"`
}

type EntityTransformationMaterializeOutput struct {
	Alias  string                                        `json:"alias"`
	Fields []EntityTransformationMaterializeFieldMapping `json:"fields"`
}

type EntityTransformationMaterializeFieldMapping struct {
	SourceAlias string `json:"sourceAlias"`
	SourceField string `json:"sourceField"`
	OutputField string `json:"outputField"`
}

type EntityTransformationSortConfig struct {
	Alias     string            `json:"alias"`
	Field     string            `json:"field"`
	Direction JoinSortDirection `json:"direction"`
}

type EntityTransformationPaginateConfig struct {
	Limit  *int `json:"limit,omitempty"`
	Offset *int `json:"offset,omitempty"`
}

type EntityTransformationExecutionOptions struct {
	Limit  int
	Offset int
}

type EntityTransformationRecord struct {
	Entities map[string]*Entity
}

type EntityTransformationExecutionResult struct {
	Records    []EntityTransformationRecord
	TotalCount int
}

func (t EntityTransformation) NodeByID(id uuid.UUID) (EntityTransformationNode, bool) {
	for _, node := range t.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return EntityTransformationNode{}, false
}

func (t EntityTransformation) TopologicallySortedNodes() ([]EntityTransformationNode, error) {
	indegree := make(map[uuid.UUID]int)
	adjacency := make(map[uuid.UUID][]uuid.UUID)
	for _, node := range t.Nodes {
		indegree[node.ID] = indegree[node.ID]
		for _, input := range node.Inputs {
			indegree[node.ID]++
			adjacency[input] = append(adjacency[input], node.ID)
		}
	}

	var queue []uuid.UUID
	for _, node := range t.Nodes {
		if indegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i].String() < queue[j].String() })

	var result []EntityTransformationNode
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		node, ok := t.NodeByID(currentID)
		if !ok {
			return nil, fmt.Errorf("node %s not found", currentID)
		}
		result = append(result, node)
		for _, next := range adjacency[currentID] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(t.Nodes) {
		return nil, fmt.Errorf("transformation graph contains cycles")
	}
	return result, nil
}

func EntityTransformationNodesToJSON(nodes []EntityTransformationNode) (json.RawMessage, error) {
	if nodes == nil {
		nodes = []EntityTransformationNode{}
	}
	return json.Marshal(nodes)
}

func EntityTransformationNodesFromJSON(data json.RawMessage) ([]EntityTransformationNode, error) {
	if len(data) == 0 {
		return []EntityTransformationNode{}, nil
	}
	var nodes []EntityTransformationNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, err
	}
	if nodes == nil {
		nodes = []EntityTransformationNode{}
	}
	return nodes, nil
}

func (r EntityTransformationRecord) Clone() EntityTransformationRecord {
	cloned := make(map[string]*Entity, len(r.Entities))
	for key, value := range r.Entities {
		if value == nil {
			cloned[key] = nil
			continue
		}
		entityCopy := *value
		entityCopy.Properties = copyProperties(entityCopy.Properties)
		cloned[key] = &entityCopy
	}
	return EntityTransformationRecord{Entities: cloned}
}

func ApplyPropertyFilters(entity *Entity, filters []PropertyFilter) bool {
	if entity == nil {
		return false
	}
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		value, ok := entity.Properties[filter.Key]
		if filter.Exists != nil {
			if *filter.Exists {
				if !ok {
					return false
				}
			} else {
				if ok {
					if filter.Value == "" && len(filter.InArray) == 0 {
						if !propertyValueIsEmpty(value) {
							return false
						}
					} else {
						return false
					}
				}
			}
		}
		if filter.Value != "" {
			if !ok {
				return false
			}
			if fmt.Sprintf("%v", value) != filter.Value {
				return false
			}
		}
		if len(filter.InArray) > 0 {
			if !ok {
				return false
			}
			matched := false
			for _, candidate := range filter.InArray {
				if fmt.Sprintf("%v", value) == candidate {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	return true
}

func propertyValueIsEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	case *string:
		if v == nil {
			return true
		}
		return *v == ""
	case fmt.Stringer:
		return v.String() == ""
	case []byte:
		return len(v) == 0
	default:
		return false
	}
}

func ProjectEntity(entity *Entity, fields []string) *Entity {
	if entity == nil {
		return nil
	}
	if len(fields) == 0 {
		return entity
	}
	allowed := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		allowed[field] = struct{}{}
	}
	projected := entity.WithProperties(entity.Properties)
	for key := range projected.Properties {
		if _, ok := allowed[key]; !ok {
			delete(projected.Properties, key)
		}
	}
	return &projected
}

func SortRecords(records []EntityTransformationRecord, alias string, field string, direction JoinSortDirection) {
	sort.SliceStable(records, func(i, j int) bool {
		left := records[i].Entities[alias]
		right := records[j].Entities[alias]
		var leftValue string
		var rightValue string
		if left != nil {
			leftValue = fmt.Sprintf("%v", left.Properties[field])
		}
		if right != nil {
			rightValue = fmt.Sprintf("%v", right.Properties[field])
		}
		if direction == JoinSortDesc {
			return leftValue > rightValue
		}
		return leftValue < rightValue
	})
}

func PaginateRecords(records []EntityTransformationRecord, limit, offset int) []EntityTransformationRecord {
	if offset >= len(records) {
		return []EntityTransformationRecord{}
	}
	end := len(records)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return records[offset:end]
}

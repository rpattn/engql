package domain

// SortDirection represents ordering direction for sortable fields.
type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

// EntitySortField enumerates fields that can be sorted when listing entities.
type EntitySortField string

const (
	EntitySortFieldCreatedAt  EntitySortField = "created_at"
	EntitySortFieldUpdatedAt  EntitySortField = "updated_at"
	EntitySortFieldEntityType EntitySortField = "entity_type"
	EntitySortFieldPath       EntitySortField = "path"
	EntitySortFieldVersion    EntitySortField = "version"
	EntitySortFieldProperty   EntitySortField = "property"
)

// EntitySort captures ordering preferences for entity listings.
type EntitySort struct {
	Field       EntitySortField
	Direction   SortDirection
	PropertyKey string
}

package domain

// EntityFilter represents filtering options for listing entities.
type EntityFilter struct {
	EntityType      string
	PropertyFilters []PropertyFilter
	TextSearch      string
}

// PropertyFilter represents a property-level filter.
type PropertyFilter struct {
	Key     string
	Value   string
	Exists  *bool
	InArray []string
}

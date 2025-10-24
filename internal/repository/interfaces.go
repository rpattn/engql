package repository

import (
	"context"
	"time"

	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
)

// OrganizationRepository defines the interface for organization operations
type OrganizationRepository interface {
	Create(ctx context.Context, org domain.Organization) (domain.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	GetByName(ctx context.Context, name string) (domain.Organization, error)
	List(ctx context.Context) ([]domain.Organization, error)
	Update(ctx context.Context, org domain.Organization) (domain.Organization, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EntitySchemaRepository defines the interface for entity schema operations
type EntitySchemaRepository interface {
	Create(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.EntitySchema, error)
	GetByName(ctx context.Context, organizationID uuid.UUID, name string) (domain.EntitySchema, error)
	List(ctx context.Context, organizationID uuid.UUID) ([]domain.EntitySchema, error)
	ListVersions(ctx context.Context, organizationID uuid.UUID, name string) ([]domain.EntitySchema, error)
	CreateVersion(ctx context.Context, schema domain.EntitySchema) (domain.EntitySchema, error)
	Exists(ctx context.Context, organizationID uuid.UUID, name string) (bool, error)
	ArchiveSchema(ctx context.Context, schemaID uuid.UUID) error
}

// EntityRepository defines the interface for entity operations
type EntityRepository interface {
	Create(ctx context.Context, entity domain.Entity) (domain.Entity, error)
	CreateBatch(ctx context.Context, items []EntityBatchItem, opts EntityBatchOptions) (EntityBatchResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Entity, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Entity, error)
	GetHistoryByVersion(ctx context.Context, entityID uuid.UUID, version int64) (domain.EntityHistory, error)
	ListHistory(ctx context.Context, entityID uuid.UUID) ([]domain.EntityHistory, error)
	List(ctx context.Context, organizationID uuid.UUID, filter *domain.EntityFilter, limit int, offset int) ([]domain.Entity, int, error)
	ListByType(ctx context.Context, organizationID uuid.UUID, entityType string) ([]domain.Entity, error)
	GetByReference(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValue string) (domain.Entity, error)
	ListByReferences(ctx context.Context, organizationID uuid.UUID, entityType string, referenceValues []string) ([]domain.Entity, error)
	Update(ctx context.Context, entity domain.Entity) (domain.Entity, error)
	Delete(ctx context.Context, id uuid.UUID) error
	RollbackEntity(ctx context.Context, id string, toVersion int64, reason string) error

	// Hierarchical operations
	GetAncestors(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error)
	GetDescendants(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error)
	GetChildren(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error)
	GetSiblings(ctx context.Context, organizationID uuid.UUID, path string) ([]domain.Entity, error)

	// JSONB filtering operations
	FilterByProperty(ctx context.Context, organizationID uuid.UUID, filter map[string]any) ([]domain.Entity, error)

	// Count operations
	Count(ctx context.Context, organizationID uuid.UUID) (int64, error)
	CountByType(ctx context.Context, organizationID uuid.UUID, entityType string) (int64, error)

	// Batch ingestion tracking
	ListIngestBatches(ctx context.Context, organizationID *uuid.UUID, statuses []string, limit int, offset int) ([]IngestBatchRecord, error)
	GetIngestBatchStats(ctx context.Context, organizationID *uuid.UUID) (IngestBatchStats, error)
}

// EntityBatchItem represents one row destined for batch insertion.
type EntityBatchItem struct {
	OrganizationID uuid.UUID
	SchemaID       uuid.UUID
	EntityType     string
	Path           string
	PropertiesJSON []byte
}

// EntityBatchOptions carries metadata about the staged batch.
type EntityBatchOptions struct {
	SourceFile string
}

// EntityBatchResult returns metadata about a staged batch.
type EntityBatchResult struct {
	BatchID    uuid.UUID
	RowsStaged int
}

// IngestBatchRecord captures persisted batch lifecycle data.
type IngestBatchRecord struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SchemaID       uuid.UUID
	EntityType     string
	FileName       *string
	RowsStaged     int
	RowsFlushed    int
	SkipValidation bool
	Status         string
	ErrorMessage   *string
	EnqueuedAt     time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	UpdatedAt      time.Time
}

// IngestBatchStats aggregates batch activity metrics.
type IngestBatchStats struct {
	TotalBatches      int64
	InProgressBatches int64
	CompletedBatches  int64
	FailedBatches     int64
	TotalRowsStaged   int64
	TotalRowsFlushed  int64
}

// EntityJoinRepository defines operations for persisted join definitions and executions
type EntityJoinRepository interface {
	Create(ctx context.Context, join domain.EntityJoinDefinition) (domain.EntityJoinDefinition, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.EntityJoinDefinition, error)
	ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]domain.EntityJoinDefinition, error)
	Update(ctx context.Context, join domain.EntityJoinDefinition) (domain.EntityJoinDefinition, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ExecuteJoin(ctx context.Context, join domain.EntityJoinDefinition, options domain.JoinExecutionOptions) ([]domain.EntityJoinEdge, int64, error)
}

// IngestionLogRepository stores ingestion errors for observability.
type IngestionLogRepository interface {
	Record(ctx context.Context, entry domain.IngestionLogEntry) error
	List(ctx context.Context, organizationID uuid.UUID, schemaName string, fileName string, limit int, offset int) ([]domain.IngestionLogEntry, error)
}

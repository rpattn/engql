package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EntityExportJobType enumerates supported export job types.
type EntityExportJobType string

const (
	EntityExportJobTypeEntityType     EntityExportJobType = "ENTITY_TYPE"
	EntityExportJobTypeTransformation EntityExportJobType = "TRANSFORMATION"
)

// EntityExportJobStatus captures lifecycle state for an export job.
type EntityExportJobStatus string

const (
	EntityExportJobStatusPending   EntityExportJobStatus = "PENDING"
	EntityExportJobStatusRunning   EntityExportJobStatus = "RUNNING"
	EntityExportJobStatusCompleted EntityExportJobStatus = "COMPLETED"
	EntityExportJobStatusFailed    EntityExportJobStatus = "FAILED"
)

// EntityExportJob mirrors persisted export job metadata for dashboards and workers.
type EntityExportJob struct {
	ID                    uuid.UUID                             `json:"id"`
	OrganizationID        uuid.UUID                             `json:"organization_id"`
	JobType               EntityExportJobType                   `json:"job_type"`
	EntityType            *string                               `json:"entity_type,omitempty"`
	TransformationID      *uuid.UUID                            `json:"transformation_id,omitempty"`
	Transformation        *EntityTransformation                 `json:"transformation_definition,omitempty"`
	TransformationOptions *EntityTransformationExecutionOptions `json:"transformation_options,omitempty"`
	Filters               []PropertyFilter                      `json:"filters"`
	RowsRequested         int                                   `json:"rows_requested"`
	RowsExported          int                                   `json:"rows_exported"`
	BytesWritten          int64                                 `json:"bytes_written"`
	FilePath              *string                               `json:"file_path,omitempty"`
	FileMimeType          *string                               `json:"file_mime_type,omitempty"`
	FileByteSize          *int64                                `json:"file_byte_size,omitempty"`
	Status                EntityExportJobStatus                 `json:"status"`
	ErrorMessage          *string                               `json:"error_message,omitempty"`
	EnqueuedAt            time.Time                             `json:"enqueued_at"`
	StartedAt             *time.Time                            `json:"started_at,omitempty"`
	CompletedAt           *time.Time                            `json:"completed_at,omitempty"`
	UpdatedAt             time.Time                             `json:"updated_at"`
}

// FiltersToJSON marshals property filters into the JSONB layout stored in Postgres.
func (j EntityExportJob) FiltersToJSON() (json.RawMessage, error) {
	filters := j.Filters
	if filters == nil {
		filters = []PropertyFilter{}
	}
	return json.Marshal(filters)
}

// EntityExportFiltersFromJSON unmarshals persisted filter JSON into property filters.
func EntityExportFiltersFromJSON(data []byte) ([]PropertyFilter, error) {
	if len(data) == 0 {
		return []PropertyFilter{}, nil
	}
	var filters []PropertyFilter
	if err := json.Unmarshal(data, &filters); err != nil {
		return nil, err
	}
	if filters == nil {
		filters = []PropertyFilter{}
	}
	return filters, nil
}

// TransformationToJSON marshals the snapshot transformation definition for storage.
func (j EntityExportJob) TransformationToJSON() (json.RawMessage, error) {
	if j.Transformation == nil {
		return nil, nil
	}
	return json.Marshal(j.Transformation)
}

// TransformationFromJSON hydrates a stored transformation snapshot.
func TransformationFromJSON(data []byte) (*EntityTransformation, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var transformation EntityTransformation
	if err := json.Unmarshal(data, &transformation); err != nil {
		return nil, err
	}
	return &transformation, nil
}

// TransformationOptionsToJSON marshals execution options for persistence.
func (j EntityExportJob) TransformationOptionsToJSON() (json.RawMessage, error) {
	if j.TransformationOptions == nil {
		return nil, nil
	}
	return json.Marshal(j.TransformationOptions)
}

// TransformationOptionsFromJSON unmarshals stored execution options.
func TransformationOptionsFromJSON(data []byte) (*EntityTransformationExecutionOptions, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var options EntityTransformationExecutionOptions
	if err := json.Unmarshal(data, &options); err != nil {
		return nil, err
	}
	return &options, nil
}

// EntityExportLog captures row-level failures that occur while exporting.
type EntityExportLog struct {
	ID             uuid.UUID `json:"id"`
	ExportJobID    uuid.UUID `json:"export_job_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	RowIdentifier  *string   `json:"row_identifier,omitempty"`
	ErrorMessage   string    `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
}

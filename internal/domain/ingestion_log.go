package domain

import (
	"time"

	"github.com/google/uuid"
)

// IngestionLogEntry captures row level issues that occur during ingestion.
type IngestionLogEntry struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	SchemaName     string    `json:"schema_name"`
	FileName       string    `json:"file_name"`
	RowNumber      *int      `json:"row_number,omitempty"`
	ErrorMessage   string    `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
}

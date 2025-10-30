package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rpattn/engql/internal/db"
	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type entityExportRepository struct {
	queries *db.Queries
}

// ErrExportJobStatusConflict indicates that a job cannot transition to the requested state.
var ErrExportJobStatusConflict = errors.New("export job status conflict")

// NewEntityExportRepository wires a repository for managing export jobs.
func NewEntityExportRepository(queries *db.Queries) EntityExportRepository {
	return &entityExportRepository{queries: queries}
}

func (r *entityExportRepository) Create(ctx context.Context, job domain.EntityExportJob) (domain.EntityExportJob, error) {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}

	filtersJSON, err := job.FiltersToJSON()
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("marshal export filters: %w", err)
	}

	transformationJSON, err := job.TransformationToJSON()
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("marshal transformation snapshot: %w", err)
	}

	optionsJSON, err := job.TransformationOptionsToJSON()
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("marshal transformation options: %w", err)
	}

	entityType := pgtype.Text{}
	if job.EntityType != nil && *job.EntityType != "" {
		entityType = pgtype.Text{String: *job.EntityType, Valid: true}
	}

	transformationID := pgtype.UUID{}
	if job.TransformationID != nil {
		transformationID = pgtype.UUID{Valid: true}
		copy(transformationID.Bytes[:], (*job.TransformationID)[:])
	}

	rowsRequested := job.RowsRequested
	if rowsRequested < 0 {
		rowsRequested = 0
	}

	if err := r.queries.InsertEntityExportJob(ctx, db.InsertEntityExportJobParams{
		ID:                       job.ID,
		OrganizationID:           job.OrganizationID,
		JobType:                  string(job.JobType),
		EntityType:               entityType,
		TransformationID:         transformationID,
		Filters:                  filtersJSON,
		RowsRequested:            int32(rowsRequested),
		TransformationDefinition: transformationJSON,
		TransformationOptions:    optionsJSON,
	}); err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("insert export job: %w", err)
	}

	return r.GetByID(ctx, job.ID)
}

func (r *entityExportRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.EntityExportJob, error) {
	row, err := r.queries.GetEntityExportJobByID(ctx, id)
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("get export job: %w", err)
	}
	return mapEntityExportJob(row)
}

func (r *entityExportRepository) List(ctx context.Context, organizationID *uuid.UUID, statuses []domain.EntityExportJobStatus, limit int, offset int) ([]domain.EntityExportJob, error) {
	if len(statuses) == 0 {
		return []domain.EntityExportJob{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	statusValues := make([]string, len(statuses))
	for i, status := range statuses {
		statusValues[i] = string(status)
	}

	rows, err := r.queries.ListEntityExportJobsByStatus(ctx, db.ListEntityExportJobsByStatusParams{
		Statuses:       statusValues,
		OrganizationID: toPGUUID(organizationID),
		PageOffset:     int32(offset),
		PageLimit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list export jobs: %w", err)
	}

	jobs := make([]domain.EntityExportJob, 0, len(rows))
	for _, row := range rows {
		job, mapErr := mapEntityExportJob(row)
		if mapErr != nil {
			return nil, mapErr
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (r *entityExportRepository) MarkRunning(ctx context.Context, id uuid.UUID) error {
	affected, err := r.queries.MarkEntityExportJobRunning(ctx, id)
	if err != nil {
		return fmt.Errorf("mark export job running: %w", err)
	}
	if affected == 0 {
		return ErrExportJobStatusConflict
	}
	return nil
}

func (r *entityExportRepository) UpdateProgress(ctx context.Context, id uuid.UUID, rowsExported int, bytesWritten int64, rowsRequested *int) error {
	if rowsExported < 0 {
		rowsExported = 0
	}
	if bytesWritten < 0 {
		bytesWritten = 0
	}
	requestedParam := pgtype.Int4{}
	if rowsRequested != nil {
		requested := max(*rowsRequested, rowsExported)
		if requested < 0 {
			requested = 0
		}
		if requested > math.MaxInt32 {
			requested = math.MaxInt32
		}
		requestedParam = pgtype.Int4{Int32: int32(requested), Valid: true}
	}
	if err := r.queries.UpdateEntityExportJobProgress(ctx, db.UpdateEntityExportJobProgressParams{
		RowsExported:  int32(rowsExported),
		RowsRequested: requestedParam,
		BytesWritten:  bytesWritten,
		ID:            id,
	}); err != nil {
		return fmt.Errorf("update export progress: %w", err)
	}
	return nil
}

func (r *entityExportRepository) MarkCompleted(ctx context.Context, id uuid.UUID, result EntityExportResult) error {
	filePath := pgtype.Text{}
	if result.FilePath != nil && *result.FilePath != "" {
		filePath = pgtype.Text{String: *result.FilePath, Valid: true}
	}
	fileMime := pgtype.Text{}
	if result.FileMimeType != nil && *result.FileMimeType != "" {
		fileMime = pgtype.Text{String: *result.FileMimeType, Valid: true}
	}
	fileSize := pgtype.Int8{}
	if result.FileByteSize != nil {
		fileSize = pgtype.Int8{Int64: *result.FileByteSize, Valid: true}
	}

	if err := r.queries.MarkEntityExportJobCompleted(ctx, db.MarkEntityExportJobCompletedParams{
		RowsExported: int32(max(result.RowsExported, 0)),
		BytesWritten: max64(result.BytesWritten, 0),
		FilePath:     filePath,
		FileMimeType: fileMime,
		FileByteSize: fileSize,
		ID:           id,
	}); err != nil {
		return fmt.Errorf("mark export job completed: %w", err)
	}
	return nil
}

func (r *entityExportRepository) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	msg := pgtype.Text{}
	if errorMessage != "" {
		msg = pgtype.Text{String: errorMessage, Valid: true}
	}
	if err := r.queries.MarkEntityExportJobFailed(ctx, db.MarkEntityExportJobFailedParams{
		ErrorMessage: msg,
		ID:           id,
	}); err != nil {
		return fmt.Errorf("mark export job failed: %w", err)
	}
	return nil
}

func (r *entityExportRepository) MarkCancelled(ctx context.Context, id uuid.UUID, reason string) error {
	msg := pgtype.Text{}
	if strings.TrimSpace(reason) != "" {
		msg = pgtype.Text{String: reason, Valid: true}
	}
	affected, err := r.queries.MarkEntityExportJobCancelled(ctx, db.MarkEntityExportJobCancelledParams{
		ErrorMessage: msg,
		ID:           id,
	})
	if err != nil {
		return fmt.Errorf("mark export job cancelled: %w", err)
	}
	if affected == 0 {
		return ErrExportJobStatusConflict
	}
	return nil
}

func (r *entityExportRepository) RecordLog(ctx context.Context, entry domain.EntityExportLog) error {
	rowIdentifier := pgtype.Text{}
	if entry.RowIdentifier != nil && *entry.RowIdentifier != "" {
		rowIdentifier = pgtype.Text{String: *entry.RowIdentifier, Valid: true}
	}
	if err := r.queries.InsertEntityExportLog(ctx, db.InsertEntityExportLogParams{
		ExportJobID:    entry.ExportJobID,
		OrganizationID: entry.OrganizationID,
		RowIdentifier:  rowIdentifier,
		ErrorMessage:   entry.ErrorMessage,
	}); err != nil {
		return fmt.Errorf("record export log: %w", err)
	}
	return nil
}

func (r *entityExportRepository) ListLogs(ctx context.Context, jobID uuid.UUID, limit int, offset int) ([]domain.EntityExportLog, error) {
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.queries.ListEntityExportLogsForJob(ctx, db.ListEntityExportLogsForJobParams{
		ExportJobID: jobID,
		PageLimit:   int32(limit),
		PageOffset:  int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list export logs: %w", err)
	}
	logs := make([]domain.EntityExportLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, mapEntityExportLog(row))
	}
	return logs, nil
}

func mapEntityExportJob(row db.EntityExportJob) (domain.EntityExportJob, error) {
	filters, err := domain.EntityExportFiltersFromJSON(row.Filters)
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("unmarshal export filters: %w", err)
	}

	transformation, err := domain.TransformationFromJSON(row.TransformationDefinition)
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("unmarshal transformation snapshot: %w", err)
	}

	options, err := domain.TransformationOptionsFromJSON(row.TransformationOptions)
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("unmarshal transformation options: %w", err)
	}

	var entityType *string
	if row.EntityType.Valid {
		value := row.EntityType.String
		entityType = &value
	}

	var transformationID *uuid.UUID
	if row.TransformationID.Valid {
		parsed, convErr := uuid.FromBytes(row.TransformationID.Bytes[:])
		if convErr != nil {
			return domain.EntityExportJob{}, fmt.Errorf("invalid transformation identifier: %w", convErr)
		}
		transformationID = &parsed
	}

	if !row.EnqueuedAt.Valid {
		return domain.EntityExportJob{}, fmt.Errorf("export job missing enqueue timestamp")
	}
	enqueuedAt := row.EnqueuedAt.Time

	var startedAt *time.Time
	if row.StartedAt.Valid {
		value := row.StartedAt.Time
		startedAt = &value
	}

	var completedAt *time.Time
	if row.CompletedAt.Valid {
		value := row.CompletedAt.Time
		completedAt = &value
	}

	var filePath *string
	if row.FilePath.Valid {
		value := row.FilePath.String
		filePath = &value
	}

	var fileMime *string
	if row.FileMimeType.Valid {
		value := row.FileMimeType.String
		fileMime = &value
	}

	var fileSize *int64
	if row.FileByteSize.Valid {
		value := row.FileByteSize.Int64
		fileSize = &value
	}

	var errorMessage *string
	if row.ErrorMessage.Valid {
		value := row.ErrorMessage.String
		errorMessage = &value
	}

	bytesWritten := row.BytesWritten

	return domain.EntityExportJob{
		ID:                    row.ID,
		OrganizationID:        row.OrganizationID,
		JobType:               domain.EntityExportJobType(row.JobType),
		EntityType:            entityType,
		TransformationID:      transformationID,
		Filters:               filters,
		RowsRequested:         int(row.RowsRequested),
		RowsExported:          int(row.RowsExported),
		BytesWritten:          bytesWritten,
		FilePath:              filePath,
		FileMimeType:          fileMime,
		FileByteSize:          fileSize,
		Status:                domain.EntityExportJobStatus(row.Status),
		ErrorMessage:          errorMessage,
		EnqueuedAt:            enqueuedAt,
		StartedAt:             startedAt,
		CompletedAt:           completedAt,
		UpdatedAt:             row.UpdatedAt,
		Transformation:        transformation,
		TransformationOptions: options,
	}, nil
}

func mapEntityExportLog(row db.EntityExportLog) domain.EntityExportLog {
	var rowIdentifier *string
	if row.RowIdentifier.Valid {
		value := row.RowIdentifier.String
		rowIdentifier = &value
	}

	return domain.EntityExportLog{
		ID:             row.ID,
		ExportJobID:    row.ExportJobID,
		OrganizationID: row.OrganizationID,
		RowIdentifier:  rowIdentifier,
		ErrorMessage:   row.ErrorMessage,
		CreatedAt:      row.CreatedAt,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

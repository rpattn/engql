package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/auth"
	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/export"

	"github.com/google/uuid"
)

func (r *Resolver) QueueEntityTypeExport(ctx context.Context, input graph.QueueEntityTypeExportInput) (*graph.EntityExportJob, error) {
	if r.exportService == nil {
		return nil, fmt.Errorf("export service is not configured")
	}
	orgID, err := uuid.Parse(input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organizationId: %w", err)
	}
	if err := auth.EnforceOrganizationScope(ctx, orgID); err != nil {
		return nil, err
	}
	req := export.EntityTypeExportRequest{
		OrganizationID: orgID,
		EntityType:     input.EntityType,
		Filters:        graphFiltersToDomain(input.Filters),
	}
	job, err := r.exportService.QueueEntityTypeExport(ctx, req)
	if err != nil {
		return nil, err
	}
	return toGraphEntityExportJob(job), nil
}

func (r *Resolver) QueueTransformationExport(ctx context.Context, input graph.QueueTransformationExportInput) (*graph.EntityExportJob, error) {
	if r.exportService == nil {
		return nil, fmt.Errorf("export service is not configured")
	}
	orgID, err := uuid.Parse(input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organizationId: %w", err)
	}
	if err := auth.EnforceOrganizationScope(ctx, orgID); err != nil {
		return nil, err
	}
	transformationID, err := uuid.Parse(input.TransformationID)
	if err != nil {
		return nil, fmt.Errorf("invalid transformationId: %w", err)
	}
	options := domain.EntityTransformationExecutionOptions{}
	if input.Options != nil {
		if input.Options.Limit != nil {
			options.Limit = *input.Options.Limit
		}
		if input.Options.Offset != nil {
			options.Offset = *input.Options.Offset
		}
	}
	req := export.TransformationExportRequest{
		OrganizationID:   orgID,
		TransformationID: transformationID,
		Filters:          graphFiltersToDomain(input.Filters),
		Options:          options,
	}
	job, err := r.exportService.QueueTransformationExport(ctx, req)
	if err != nil {
		return nil, err
	}
	return toGraphEntityExportJob(job), nil
}

func (r *Resolver) CancelEntityExportJob(ctx context.Context, id string) (*graph.EntityExportJob, error) {
	if r.exportService == nil {
		return nil, fmt.Errorf("export service is not configured")
	}
	jobID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid export job id: %w", err)
	}
	existing, err := r.exportService.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := auth.EnforceOrganizationScope(ctx, existing.OrganizationID); err != nil {
		return nil, err
	}
	job, err := r.exportService.CancelJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return toGraphEntityExportJob(job), nil
}

func (r *Resolver) ListEntityExportJobs(ctx context.Context, organizationID string, statuses []graph.EntityExportJobStatus, limit *int, offset *int) ([]*graph.EntityExportJob, error) {
	if r.exportService == nil {
		return nil, fmt.Errorf("export service is not configured")
	}
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organizationId: %w", err)
	}
	if err := auth.EnforceOrganizationScope(ctx, orgID); err != nil {
		return nil, err
	}
	domainStatuses := make([]domain.EntityExportJobStatus, 0, len(statuses))
	for _, status := range statuses {
		domainStatuses = append(domainStatuses, domain.EntityExportJobStatus(status))
	}
	if len(domainStatuses) == 0 {
		domainStatuses = []domain.EntityExportJobStatus{
			domain.EntityExportJobStatusPending,
			domain.EntityExportJobStatusRunning,
			domain.EntityExportJobStatusCompleted,
			domain.EntityExportJobStatusCancelled,
			domain.EntityExportJobStatusFailed,
		}
	}
	pageLimit := 20
	if limit != nil && *limit > 0 {
		pageLimit = *limit
	}
	pageOffset := 0
	if offset != nil && *offset >= 0 {
		pageOffset = *offset
	}
	jobs, err := r.exportService.ListJobs(ctx, &orgID, domainStatuses, pageLimit, pageOffset)
	if err != nil {
		return nil, err
	}
	result := make([]*graph.EntityExportJob, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, toGraphEntityExportJob(job))
	}
	return result, nil
}

func (r *Resolver) GetEntityExportJob(ctx context.Context, id string) (*graph.EntityExportJob, error) {
	if r.exportService == nil {
		return nil, fmt.Errorf("export service is not configured")
	}
	jobID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid export job id: %w", err)
	}
	job, err := r.exportService.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := auth.EnforceOrganizationScope(ctx, job.OrganizationID); err != nil {
		return nil, err
	}
	return toGraphEntityExportJob(job), nil
}

func (r *Resolver) ResolveEntityExportJobDownloadURL(ctx context.Context, obj *graph.EntityExportJob) (*string, error) {
	if r.exportService == nil {
		return nil, fmt.Errorf("export service is not configured")
	}
	jobID, err := uuid.Parse(obj.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid export job id: %w", err)
	}
	job, err := r.exportService.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if err := auth.EnforceOrganizationScope(ctx, job.OrganizationID); err != nil {
		return nil, err
	}
	return r.exportService.BuildDownloadURL(job)
}

func toGraphEntityExportJob(job domain.EntityExportJob) *graph.EntityExportJob {
	result := &graph.EntityExportJob{
		ID:             job.ID.String(),
		OrganizationID: job.OrganizationID.String(),
		JobType:        graph.EntityExportJobType(job.JobType),
		Status:         graph.EntityExportJobStatus(job.Status),
		RowsRequested:  job.RowsRequested,
		RowsExported:   job.RowsExported,
		BytesWritten:   int(job.BytesWritten),
		Filters:        domainFiltersToGraph(job.Filters),
		EnqueuedAt:     job.EnqueuedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      job.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if job.EntityType != nil {
		result.EntityType = job.EntityType
	}
	if job.TransformationID != nil {
		id := job.TransformationID.String()
		result.TransformationID = &id
	}
	if job.ErrorMessage != nil {
		result.ErrorMessage = job.ErrorMessage
	}
	if job.FileMimeType != nil {
		result.FileMimeType = job.FileMimeType
	}
	if job.FileByteSize != nil {
		size := int(*job.FileByteSize)
		result.FileByteSize = &size
	}
	if job.Transformation != nil {
		result.TransformationDefinition = mapTransformationToGraph(*job.Transformation)
	}
	if job.StartedAt != nil {
		started := job.StartedAt.UTC().Format(time.RFC3339)
		result.StartedAt = &started
	}
	if job.CompletedAt != nil {
		completed := job.CompletedAt.UTC().Format(time.RFC3339)
		result.CompletedAt = &completed
	}
	return result
}

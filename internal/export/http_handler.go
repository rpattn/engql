package export

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/rpattn/engql/internal/auth"
	"github.com/rpattn/engql/internal/domain"
)

type Handler struct {
	service *Service
}

func NewHTTPHandler(service *Service) http.Handler {
	return &Handler{service: service}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/"):
		h.handleDownload(w, r)
		return
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/logs"):
		h.handleListLogs(w, r)
		return
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/entity-type"):
		h.handleQueueEntityType(w, r)
		return
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transformation"):
		h.handleQueueTransformation(w, r)
		return
	case r.Method == http.MethodPost:
		h.handleQueue(w, r)
		return
	case r.Method == http.MethodGet && (strings.HasSuffix(r.URL.Path, "/jobs") || strings.HasSuffix(r.URL.Path, "/batches") || strings.HasSuffix(r.URL.Path, "/exports")):
		h.handleListJobs(w, r)
		return
	case r.Method == http.MethodGet:
		h.handleListJobs(w, r)
		return
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
}

type entityTypeQueuePayload struct {
	OrganizationID string                `json:"organizationId"`
	EntityType     string                `json:"entityType"`
	Filters        []propertyFilterInput `json:"filters"`
}

type transformationQueuePayload struct {
	OrganizationID   string                      `json:"organizationId"`
	TransformationID string                      `json:"transformationId"`
	Filters          []propertyFilterInput       `json:"filters"`
	Options          *transformationOptionsInput `json:"options"`
}

type queueExportPayload struct {
	OrganizationID   string                      `json:"organizationId"`
	JobType          string                      `json:"jobType"`
	EntityType       *string                     `json:"entityType"`
	TransformationID *string                     `json:"transformationId"`
	Filters          []propertyFilterInput       `json:"filters"`
	Options          *transformationOptionsInput `json:"options"`
}

type propertyFilterInput struct {
	Key     string   `json:"key"`
	Value   *string  `json:"value"`
	Exists  *bool    `json:"exists"`
	InArray []string `json:"inArray"`
}

type transformationOptionsInput struct {
	Limit  *int `json:"limit"`
	Offset *int `json:"offset"`
}

func (h *Handler) handleQueueEntityType(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload entityTypeQueuePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}
	orgID, err := uuid.Parse(strings.TrimSpace(payload.OrganizationID))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid organizationId: %v", err), http.StatusBadRequest)
		return
	}
	if err := auth.EnforceOrganizationScope(r.Context(), orgID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	req := EntityTypeExportRequest{
		OrganizationID: orgID,
		EntityType:     payload.EntityType,
		Filters:        toDomainFilters(payload.Filters),
	}
	job, err := h.service.QueueEntityTypeExport(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (h *Handler) handleQueueTransformation(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload transformationQueuePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}
	orgID, err := uuid.Parse(strings.TrimSpace(payload.OrganizationID))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid organizationId: %v", err), http.StatusBadRequest)
		return
	}
	if err := auth.EnforceOrganizationScope(r.Context(), orgID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	transformationID, err := uuid.Parse(strings.TrimSpace(payload.TransformationID))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid transformationId: %v", err), http.StatusBadRequest)
		return
	}
	options := toExecutionOptions(payload.Options)
	req := TransformationExportRequest{
		OrganizationID:   orgID,
		TransformationID: transformationID,
		Filters:          toDomainFilters(payload.Filters),
		Options:          options,
	}
	job, err := h.service.QueueTransformationExport(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (h *Handler) handleQueue(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload queueExportPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}
	orgID, err := uuid.Parse(strings.TrimSpace(payload.OrganizationID))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid organizationId: %v", err), http.StatusBadRequest)
		return
	}
	if err := auth.EnforceOrganizationScope(r.Context(), orgID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	jobType := strings.ToUpper(strings.TrimSpace(payload.JobType))
	switch domain.EntityExportJobType(jobType) {
	case domain.EntityExportJobTypeEntityType:
		entityType := ""
		if payload.EntityType != nil {
			entityType = *payload.EntityType
		}
		req := EntityTypeExportRequest{
			OrganizationID: orgID,
			EntityType:     entityType,
			Filters:        toDomainFilters(payload.Filters),
		}
		job, queueErr := h.service.QueueEntityTypeExport(r.Context(), req)
		if queueErr != nil {
			http.Error(w, queueErr.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, job)
	case domain.EntityExportJobTypeTransformation:
		if payload.TransformationID == nil {
			http.Error(w, "transformationId is required", http.StatusBadRequest)
			return
		}
		transformationID, err := uuid.Parse(strings.TrimSpace(*payload.TransformationID))
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid transformationId: %v", err), http.StatusBadRequest)
			return
		}
		req := TransformationExportRequest{
			OrganizationID:   orgID,
			TransformationID: transformationID,
			Filters:          toDomainFilters(payload.Filters),
			Options:          toExecutionOptions(payload.Options),
		}
		job, queueErr := h.service.QueueTransformationExport(r.Context(), req)
		if queueErr != nil {
			http.Error(w, queueErr.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, job)
	default:
		http.Error(w, fmt.Sprintf("unsupported jobType %q", payload.JobType), http.StatusBadRequest)
	}
}

func (h *Handler) handleListJobs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var organizationID *uuid.UUID
	if raw := strings.TrimSpace(query.Get("organizationId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid organizationId: %v", err), http.StatusBadRequest)
			return
		}
		organizationID = &id
		if err := auth.EnforceOrganizationScope(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}
	statuses := parseStatuses(query["status"])
	if len(statuses) == 0 {
		statuses = []domain.EntityExportJobStatus{
			domain.EntityExportJobStatusPending,
			domain.EntityExportJobStatusRunning,
			domain.EntityExportJobStatusCompleted,
			domain.EntityExportJobStatusFailed,
		}
	}
	limit := 20
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			http.Error(w, "offset must be zero or positive", http.StatusBadRequest)
			return
		}
		offset = parsed
	}
	jobs, err := h.service.ListJobs(r.Context(), organizationID, statuses, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("list jobs: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *Handler) handleListLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	jobIDRaw := strings.TrimSpace(query.Get("jobId"))
	if jobIDRaw == "" {
		http.Error(w, "jobId is required", http.StatusBadRequest)
		return
	}
	jobID, err := uuid.Parse(jobIDRaw)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid jobId: %v", err), http.StatusBadRequest)
		return
	}
	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("job not found: %v", err), http.StatusNotFound)
		return
	}
	if err := auth.EnforceOrganizationScope(r.Context(), job.OrganizationID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	limit := 200
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			http.Error(w, "offset must be zero or positive", http.StatusBadRequest)
			return
		}
		offset = parsed
	}
	logs, err := h.service.ListLogs(r.Context(), jobID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("list logs: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func toDomainFilters(inputs []propertyFilterInput) []domain.PropertyFilter {
	if len(inputs) == 0 {
		return []domain.PropertyFilter{}
	}
	filters := make([]domain.PropertyFilter, 0, len(inputs))
	for _, input := range inputs {
		key := strings.TrimSpace(input.Key)
		if key == "" {
			continue
		}
		filter := domain.PropertyFilter{Key: key}
		if input.Value != nil {
			filter.Value = *input.Value
		}
		if input.Exists != nil {
			filter.Exists = input.Exists
		}
		if len(input.InArray) > 0 {
			filter.InArray = append([]string(nil), input.InArray...)
		}
		filters = append(filters, filter)
	}
	return filters
}

func toExecutionOptions(input *transformationOptionsInput) domain.EntityTransformationExecutionOptions {
	opts := domain.EntityTransformationExecutionOptions{}
	if input == nil {
		return opts
	}
	if input.Limit != nil {
		opts.Limit = *input.Limit
	}
	if input.Offset != nil {
		opts.Offset = *input.Offset
	}
	return opts
}

func parseStatuses(values []string) []domain.EntityExportJobStatus {
	if len(values) == 0 {
		return nil
	}
	result := make([]domain.EntityExportJobStatus, 0, len(values))
	for _, raw := range values {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			trimmed := strings.ToUpper(strings.TrimSpace(part))
			switch domain.EntityExportJobStatus(trimmed) {
			case domain.EntityExportJobStatusPending,
				domain.EntityExportJobStatusRunning,
				domain.EntityExportJobStatusCompleted,
				domain.EntityExportJobStatusFailed:
				result = append(result, domain.EntityExportJobStatus(trimmed))
			}
		}
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	idx := strings.LastIndex(path, "/")
	if idx == -1 || idx == len(path)-1 {
		http.Error(w, "missing export identifier", http.StatusBadRequest)
		return
	}
	idSegment := path[idx+1:]
	jobID, err := uuid.Parse(idSegment)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid export identifier: %v", err), http.StatusBadRequest)
		return
	}
	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("job not found: %v", err), http.StatusNotFound)
		return
	}
	if err := auth.EnforceOrganizationScope(r.Context(), job.OrganizationID); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if err := h.service.ValidateDownloadToken(jobID, token); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	file, err := h.service.OpenJobFile(job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	defer file.Close()

	filename := filepath.Base(strings.TrimSpace(*job.FilePath))
	if filename == "" {
		filename = fmt.Sprintf("export-%s.csv", jobID.String())
	}
	contentType := "application/octet-stream"
	if job.FileMimeType != nil && strings.TrimSpace(*job.FileMimeType) != "" {
		contentType = *job.FileMimeType
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if job.FileByteSize != nil && *job.FileByteSize > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(*job.FileByteSize, 10))
	}
	http.ServeContent(w, r, filename, job.UpdatedAt, file)
}

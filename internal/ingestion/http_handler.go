package ingestion

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rpattn/engql/internal/domain"
)

// Handler exposes ingestion as an HTTP endpoint.
type Handler struct {
	service *Service
}

// NewHTTPHandler wraps the service with a POST endpoint.
func NewHTTPHandler(service *Service) http.Handler {
	return &Handler{service: service}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/logs"):
		h.handleLogs(w, r)
		return
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/batches"):
		h.handleBatches(w, r)
		return
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/preview"):
		h.handlePreview(w, r)
		return
	case r.Method == http.MethodPost:
		h.handleIngest(w, r)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

type uploadPayload struct {
	fileName        string
	fileData        []byte
	organizationID  uuid.UUID
	schemaName      string
	description     string
	headerRowIndex  *int
	columnOverrides map[string]domain.FieldType
	skipValidation  bool
}

func (h *Handler) handleIngest(w http.ResponseWriter, r *http.Request) {
	payload, err := parseUploadPayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req := Request{
		OrganizationID:       payload.organizationID,
		SchemaName:           payload.schemaName,
		Description:          payload.description,
		FileName:             payload.fileName,
		HeaderRowIndex:       payload.headerRowIndex,
		ColumnOverrides:      payload.columnOverrides,
		Data:                 bytes.NewReader(payload.fileData),
		SkipEntityValidation: payload.skipValidation,
	}

	summary, err := h.service.Ingest(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request) {
	payload, err := parseUploadPayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limitRaw := strings.TrimSpace(r.FormValue("previewLimit"))
	var limit int
	if limitRaw != "" {
		parsed, convErr := strconv.Atoi(limitRaw)
		if convErr != nil {
			http.Error(w, fmt.Sprintf("invalid preview limit: %v", convErr), http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	req := PreviewRequest{
		OrganizationID:  payload.organizationID,
		SchemaName:      payload.schemaName,
		FileName:        payload.fileName,
		HeaderRowIndex:  payload.headerRowIndex,
		ColumnOverrides: payload.columnOverrides,
		Data:            bytes.NewReader(payload.fileData),
		Limit:           limit,
	}

	result, err := h.service.Preview(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleBatches(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var organizationID *uuid.UUID
	if raw := strings.TrimSpace(query.Get("organizationId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid organizationId: %v", err), http.StatusBadRequest)
			return
		}
		organizationID = &id
	}

	limit := 20
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = value
	}

	offset := 0
	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		value, err := strconv.Atoi(rawOffset)
		if err != nil || value < 0 {
			http.Error(w, "offset must be zero or positive", http.StatusBadRequest)
			return
		}
		offset = value
	}

	overview, err := h.service.GetBatchOverview(r.Context(), organizationID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, overview)
}

func (h *Handler) handleLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	orgRaw := strings.TrimSpace(query.Get("organizationId"))
	if orgRaw == "" {
		http.Error(w, "organizationId is required", http.StatusBadRequest)
		return
	}
	organizationID, err := uuid.Parse(orgRaw)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid organizationId: %v", err), http.StatusBadRequest)
		return
	}

	schemaName := strings.TrimSpace(query.Get("schemaName"))
	if schemaName == "" {
		http.Error(w, "schemaName is required", http.StatusBadRequest)
		return
	}

	fileName := strings.TrimSpace(query.Get("fileName"))
	if fileName == "" {
		http.Error(w, "fileName is required", http.StatusBadRequest)
		return
	}

	limit := 100
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		value, convErr := strconv.Atoi(rawLimit)
		if convErr != nil || value <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = value
	}

	offset := 0
	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		value, convErr := strconv.Atoi(rawOffset)
		if convErr != nil || value < 0 {
			http.Error(w, "offset must be zero or positive", http.StatusBadRequest)
			return
		}
		offset = value
	}

	logs, err := h.service.GetBatchLogs(r.Context(), organizationID, schemaName, fileName, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

func parseUploadPayload(r *http.Request) (uploadPayload, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return uploadPayload{}, fmt.Errorf("invalid form data: %w", err)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return uploadPayload{}, fmt.Errorf("file required: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return uploadPayload{}, fmt.Errorf("failed to read file: %w", err)
	}

	orgIDRaw := strings.TrimSpace(r.FormValue("organizationId"))
	if orgIDRaw == "" {
		return uploadPayload{}, errors.New("organizationId is required")
	}
	orgID, err := uuid.Parse(orgIDRaw)
	if err != nil {
		return uploadPayload{}, fmt.Errorf("invalid organization id: %w", err)
	}

	schemaName := strings.TrimSpace(r.FormValue("schemaName"))
	if schemaName == "" {
		return uploadPayload{}, errors.New("schemaName is required")
	}

	description := strings.TrimSpace(r.FormValue("description"))

	headerRowIndex, err := parseHeaderRowIndex(r.FormValue("headerRowIndex"))
	if err != nil {
		return uploadPayload{}, err
	}

	columnOverrides, err := parseColumnOverrides(r.FormValue("columnTypes"))
	if err != nil {
		return uploadPayload{}, err
	}

	skipValidation, err := parseSkipValidation(r.FormValue("skipValidation"))
	if err != nil {
		return uploadPayload{}, err
	}

	return uploadPayload{
		fileName:        header.Filename,
		fileData:        data,
		organizationID:  orgID,
		schemaName:      schemaName,
		description:     description,
		headerRowIndex:  headerRowIndex,
		columnOverrides: columnOverrides,
		skipValidation:  skipValidation,
	}, nil
}

func parseHeaderRowIndex(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid headerRowIndex: %w", err)
	}
	if value < 0 {
		return nil, errors.New("headerRowIndex cannot be negative")
	}
	return &value, nil
}

func parseColumnOverrides(raw string) (map[string]domain.FieldType, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var input map[string]string
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil, fmt.Errorf("invalid columnTypes payload: %w", err)
	}

	overrides := make(map[string]domain.FieldType, len(input))
	for key, value := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		fieldType, err := normalizeFieldType(value)
		if err != nil {
			return nil, err
		}
		overrides[key] = fieldType
	}
	return overrides, nil
}

func parseSkipValidation(raw string) (bool, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", "0", "false", "off", "no":
		return false, nil
	case "1", "true", "on", "yes":
		return true, nil
	default:
		return false, fmt.Errorf("invalid skipValidation flag: %q", raw)
	}
}

func normalizeFieldType(raw string) (domain.FieldType, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "string":
		return domain.FieldTypeString, nil
	case "int", "integer":
		return domain.FieldTypeInteger, nil
	case "float", "double", "decimal":
		return domain.FieldTypeFloat, nil
	case "bool", "boolean":
		return domain.FieldTypeBoolean, nil
	case "timestamp", "datetime":
		return domain.FieldTypeTimestamp, nil
	case "json":
		return domain.FieldTypeJSON, nil
	default:
		return "", fmt.Errorf("unsupported column type %q", raw)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

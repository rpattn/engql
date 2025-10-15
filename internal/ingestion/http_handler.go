package ingestion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, fmt.Sprintf("invalid form data: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("file required: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	orgIDRaw := strings.TrimSpace(r.FormValue("organizationId"))
	orgID, err := uuid.Parse(orgIDRaw)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid organization id: %v", err), http.StatusBadRequest)
		return
	}

	schemaName := strings.TrimSpace(r.FormValue("schemaName"))
	if schemaName == "" {
		http.Error(w, "schemaName is required", http.StatusBadRequest)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read file: %v", err), http.StatusBadRequest)
		return
	}

	req := Request{
		OrganizationID: orgID,
		SchemaName:     schemaName,
		Description:    description,
		FileName:       header.Filename,
		Data:           bytes.NewReader(data),
	}

	summary, err := h.service.Ingest(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

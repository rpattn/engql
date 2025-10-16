package ingestion

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"
	"github.com/rpattn/engql/pkg/validator"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

var (
	// ErrUnsupportedFormat is returned when an uploaded file is not supported.
	ErrUnsupportedFormat = errors.New("unsupported file format")

	byteOrderMark = []byte{0xEF, 0xBB, 0xBF}

	timeLayouts = []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05.000000000",
		"2006/01/02",
		"01/02/2006",
		"02/01/2006",
	}
)

// Service ingests tabular data into entity schemas.
type Service struct {
	schemaRepo repository.EntitySchemaRepository
	entityRepo repository.EntityRepository
	logRepo    repository.IngestionLogRepository
	validator  *validator.JSONBValidator
}

// NewService creates a new ingestion service.
func NewService(
	schemaRepo repository.EntitySchemaRepository,
	entityRepo repository.EntityRepository,
	logRepo repository.IngestionLogRepository,
) *Service {
	return &Service{
		schemaRepo: schemaRepo,
		entityRepo: entityRepo,
		logRepo:    logRepo,
		validator:  validator.NewJSONBValidator(),
	}
}

// Request describes the ingestion input.
type Request struct {
	OrganizationID  uuid.UUID
	SchemaName      string
	Description     string
	FileName        string
	HeaderRowIndex  *int
	ColumnOverrides map[string]domain.FieldType
	Data            io.Reader
}

// PreviewRequest describes the preview input prior to ingestion.
type PreviewRequest struct {
	OrganizationID  uuid.UUID
	SchemaName      string
	FileName        string
	HeaderRowIndex  *int
	ColumnOverrides map[string]domain.FieldType
	Data            io.Reader
	Limit           int
}

// PreviewHeader summarizes column level metadata for previews.
type PreviewHeader struct {
	Name          string `json:"name"`
	OriginalLabel string `json:"originalLabel"`
	DetectedType  string `json:"detectedType"`
	EffectiveType string `json:"effectiveType"`
	Required      bool   `json:"required"`
	Overridden    bool   `json:"overridden"`
}

// PreviewRow captures sample data and validation feedback.
type PreviewRow struct {
	RowNumber int               `json:"rowNumber"`
	Values    map[string]string `json:"values"`
	Errors    []string          `json:"errors,omitempty"`
}

// HeaderCandidate represents a potential header row option.
type HeaderCandidate struct {
	Index   int      `json:"index"`
	Values  []string `json:"values"`
	Current bool     `json:"current"`
}

// PreviewResult returns preview metadata back to clients.
type PreviewResult struct {
	TotalRows        int               `json:"totalRows"`
	InvalidRows      int               `json:"invalidRows"`
	Headers          []PreviewHeader   `json:"headers"`
	Rows             []PreviewRow      `json:"rows"`
	SchemaChanges    []SchemaChange    `json:"schemaChanges"`
	HeaderCandidates []HeaderCandidate `json:"headerCandidates"`
}

// SchemaChange highlights schema level adjustments or conflicts.
type SchemaChange struct {
	Field        string `json:"field,omitempty"`
	ExistingType string `json:"existingType,omitempty"`
	DetectedType string `json:"detectedType,omitempty"`
	Message      string `json:"message"`
}

// Summary returns ingestion level metrics.
type Summary struct {
	TotalRows         int            `json:"totalRows"`
	ValidRows         int            `json:"validRows"`
	InvalidRows       int            `json:"invalidRows"`
	NewFieldsDetected []string       `json:"newFieldsDetected"`
	SchemaChanges     []SchemaChange `json:"schemaChanges"`
	SchemaCreated     bool           `json:"schemaCreated"`
}

type tableData struct {
	headers        []string
	rawHeaders     []string
	rows           [][]string
	headerRowIndex int
}

// Ingest reads the uploaded file, updates the schema, and persists valid entities.
func (s *Service) Ingest(ctx context.Context, req Request) (Summary, error) {
	summary := Summary{
		NewFieldsDetected: []string{},
		SchemaChanges:     []SchemaChange{},
	}

	if req.OrganizationID == uuid.Nil {
		return summary, errors.New("organization id is required")
	}
	if strings.TrimSpace(req.SchemaName) == "" {
		return summary, errors.New("schema name is required")
	}
	if req.Data == nil {
		return summary, errors.New("data reader is required")
	}

	payload, err := io.ReadAll(req.Data)
	if err != nil {
		return summary, fmt.Errorf("failed to read upload: %w", err)
	}
	if len(payload) == 0 {
		return summary, errors.New("file is empty")
	}

	table, _, err := parseTable(req.FileName, payload, req.HeaderRowIndex)
	if err != nil {
		return summary, err
	}
	if len(table.headers) == 0 {
		return summary, errors.New("no header row detected")
	}

	detectedFields := inferFieldDefinitions(table)
	detectedFields = applyOverridesToDefinitions(detectedFields, req.ColumnOverrides)
	if len(detectedFields) == 0 {
		return summary, errors.New("no fields inferred from data set")
	}

	summary.TotalRows = len(table.rows)

	exists, err := s.schemaRepo.Exists(ctx, req.OrganizationID, req.SchemaName)
	if err != nil {
		return summary, fmt.Errorf("failed to check schema existence: %w", err)
	}

	var workingSchema domain.EntitySchema
	if exists {
		workingSchema, err = s.schemaRepo.GetByName(ctx, req.OrganizationID, req.SchemaName)
		if err != nil {
			return summary, fmt.Errorf("failed to load schema: %w", err)
		}
	} else {
		workingSchema = domain.NewEntitySchema(req.OrganizationID, req.SchemaName, req.Description, detectedFields)
		created, err := s.schemaRepo.Create(ctx, workingSchema)
		if err != nil {
			return summary, fmt.Errorf("failed to create schema: %w", err)
		}
		workingSchema = created
		summary.SchemaCreated = true
		summary.SchemaChanges = append(summary.SchemaChanges, SchemaChange{
			Message: fmt.Sprintf("schema %s created", req.SchemaName),
		})
	}

	fieldMap := make(map[string]domain.FieldDefinition)
	for _, field := range workingSchema.Fields {
		fieldMap[field.Name] = field
	}

	baseSchema := workingSchema
	var schemaUpdated bool
	for _, detected := range detectedFields {
		existing, found := fieldMap[detected.Name]
		if !found {
			workingSchema = workingSchema.WithField(detected)
			fieldMap[detected.Name] = detected
			summary.NewFieldsDetected = append(summary.NewFieldsDetected, detected.Name)
			schemaUpdated = true
			continue
		}

		if !fieldTypesCompatible(existing.Type, detected.Type) {
			message := fmt.Sprintf("field %s type mismatch: existing=%s, detected=%s", detected.Name, existing.Type, detected.Type)
			summary.SchemaChanges = append(summary.SchemaChanges, SchemaChange{
				Field:        detected.Name,
				ExistingType: string(existing.Type),
				DetectedType: string(detected.Type),
				Message:      message,
			})
			s.logIngestionError(ctx, req, nil, errors.New(message))
		}

		if detected.Required && !existing.Required {
			updated := existing
			updated.Required = true
			workingSchema = workingSchema.WithField(updated)
			fieldMap[updated.Name] = updated
			schemaUpdated = true
			summary.SchemaChanges = append(summary.SchemaChanges, SchemaChange{
				Field:   detected.Name,
				Message: "promoted to required based on data inference",
			})
		}
	}

	if schemaUpdated && !summary.SchemaCreated {
		compatibility := domain.DetermineCompatibility(baseSchema.Fields, workingSchema.Fields)
		nextVersion, err := domain.NewVersionFromExisting(baseSchema, workingSchema, compatibility, domain.SchemaStatusActive)
		if err != nil {
			return summary, fmt.Errorf("failed to prepare schema version: %w", err)
		}

		persisted, err := s.schemaRepo.CreateVersion(ctx, nextVersion)
		if err != nil {
			return summary, fmt.Errorf("failed to persist schema version: %w", err)
		}
		workingSchema = persisted

		fieldMap = make(map[string]domain.FieldDefinition)
		for _, field := range workingSchema.Fields {
			fieldMap[field.Name] = field
		}

		summary.SchemaChanges = append(summary.SchemaChanges, SchemaChange{
			Message: fmt.Sprintf("schema %s updated to version %s (%s)", workingSchema.Name, workingSchema.Version, compatibility),
		})
	}

	if summary.TotalRows == 0 {
		return summary, nil
	}

	validatorDefs := buildValidatorDefinitions(workingSchema.Fields)
	usedPaths := make(map[string]int)

	for rowIdx, row := range table.rows {
		rowNumber := table.headerRowIndex + rowIdx + 2 // include header row (1-based)
		properties := make(map[string]any)
		rowValid := true

		for colIdx, header := range table.headers {
			if colIdx >= len(row) {
				continue
			}

			fieldDef, ok := fieldMap[header]
			if !ok {
				// Column not part of schema; skip silently to avoid failing ingestion.
				continue
			}

			raw := strings.TrimSpace(row[colIdx])
			if raw == "" {
				continue
			}

			coerced, coerceErr := coerceValue(fieldDef.Type, raw)
			if coerceErr != nil {
				rowValid = false
				s.summaryRowError(ctx, req, rowNumber, fmt.Errorf("field %s: %w", header, coerceErr))
				break
			}
			properties[fieldDef.Name] = coerced
		}

		if !rowValid {
			summary.InvalidRows++
			continue
		}

		validationResult := s.validator.ValidateProperties(properties, validatorDefs)
		if !validationResult.IsValid {
			rowValid = false
			var messages []string
			for _, validationErr := range validationResult.Errors {
				messages = append(messages, fmt.Sprintf("%s: %s", validationErr.Field, validationErr.Message))
			}

			if len(validationResult.Warnings) > 0 {
				for _, warning := range validationResult.Warnings {
					messages = append(messages, fmt.Sprintf("warning %s: %s", warning.Field, warning.Message))
				}
			}

			s.summaryRowError(ctx, req, rowNumber, errors.New(strings.Join(messages, "; ")))
			summary.InvalidRows++
			continue
		}

		path := generatePath(workingSchema.Name, row, rowIdx, usedPaths)
		entity := domain.NewEntity(req.OrganizationID, workingSchema.ID, workingSchema.Name, path, properties)

		if _, err := s.entityRepo.Create(ctx, entity); err != nil {
			s.summaryRowError(ctx, req, rowNumber, fmt.Errorf("failed to insert entity: %w", err))
			summary.InvalidRows++
			continue
		}

		summary.ValidRows++
	}

	return summary, nil
}

// Preview runs validations against a limited set of rows without persisting entities.
func (s *Service) Preview(ctx context.Context, req PreviewRequest) (PreviewResult, error) {
	result := PreviewResult{
		Headers:          []PreviewHeader{},
		Rows:             []PreviewRow{},
		SchemaChanges:    []SchemaChange{},
		HeaderCandidates: []HeaderCandidate{},
	}

	if req.OrganizationID == uuid.Nil {
		return result, errors.New("organization id is required")
	}
	if strings.TrimSpace(req.SchemaName) == "" {
		return result, errors.New("schema name is required")
	}
	if req.Data == nil {
		return result, errors.New("data reader is required")
	}

	payload, err := io.ReadAll(req.Data)
	if err != nil {
		return result, fmt.Errorf("failed to read upload: %w", err)
	}
	if len(payload) == 0 {
		return result, errors.New("file is empty")
	}

	table, records, err := parseTable(req.FileName, payload, req.HeaderRowIndex)
	if err != nil {
		return result, err
	}

	result.HeaderCandidates = buildHeaderCandidates(records, 10, table.headerRowIndex)

	if len(table.headers) == 0 {
		return result, errors.New("no header row detected")
	}

	autoDetected := inferFieldDefinitions(table)
	detectedFields := applyOverridesToDefinitions(autoDetected, req.ColumnOverrides)

	exists, err := s.schemaRepo.Exists(ctx, req.OrganizationID, req.SchemaName)
	if err != nil {
		return result, fmt.Errorf("failed to check schema existence: %w", err)
	}

	var workingSchema domain.EntitySchema
	if exists {
		workingSchema, err = s.schemaRepo.GetByName(ctx, req.OrganizationID, req.SchemaName)
		if err != nil {
			return result, fmt.Errorf("failed to load schema: %w", err)
		}
	} else {
		workingSchema = domain.NewEntitySchema(req.OrganizationID, req.SchemaName, "", detectedFields)
		result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
			Message: fmt.Sprintf("schema %s would be created", req.SchemaName),
		})
	}

	fieldMap := make(map[string]domain.FieldDefinition)
	for _, field := range workingSchema.Fields {
		fieldMap[field.Name] = field
	}

	baseSchema := workingSchema
	var schemaUpdated bool

	for _, detected := range detectedFields {
		existing, found := fieldMap[detected.Name]
		if !found {
			workingSchema = workingSchema.WithField(detected)
			fieldMap[detected.Name] = detected
			schemaUpdated = true
			result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
				Field:   detected.Name,
				Message: "new field detected",
			})
			continue
		}

		if !fieldTypesCompatible(existing.Type, detected.Type) {
			message := fmt.Sprintf("field %s type mismatch: existing=%s, detected=%s", detected.Name, existing.Type, detected.Type)
			result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
				Field:        detected.Name,
				ExistingType: string(existing.Type),
				DetectedType: string(detected.Type),
				Message:      message,
			})
		}

		if detected.Required && !existing.Required {
			updated := existing
			updated.Required = true
			workingSchema = workingSchema.WithField(updated)
			fieldMap[updated.Name] = updated
			schemaUpdated = true
			result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
				Field:   detected.Name,
				Message: "would be promoted to required based on data inference",
			})
		}
	}

	if schemaUpdated && exists {
		compatibility := domain.DetermineCompatibility(baseSchema.Fields, workingSchema.Fields)
		result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
			Message: fmt.Sprintf("schema %s would be updated (%s)", workingSchema.Name, compatibility),
		})
	}

	result.TotalRows = len(table.rows)

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	validatorDefs := buildValidatorDefinitions(workingSchema.Fields)
	invalidRows := 0

	for rowIdx, row := range table.rows {
		rowNumber := table.headerRowIndex + rowIdx + 2
		rowValues := make(map[string]string, len(table.headers))
		for colIdx, header := range table.headers {
			if colIdx < len(row) {
				rowValues[header] = strings.TrimSpace(row[colIdx])
			} else {
				rowValues[header] = ""
			}
		}

		var rowErrors []string
		properties := make(map[string]any)

		for colIdx, header := range table.headers {
			if colIdx >= len(row) {
				continue
			}

			fieldDef, ok := fieldMap[header]
			if !ok {
				continue
			}

			raw := strings.TrimSpace(row[colIdx])
			if raw == "" {
				continue
			}

			coerced, coerceErr := coerceValue(fieldDef.Type, raw)
			if coerceErr != nil {
				rowErrors = append(rowErrors, fmt.Sprintf("field %s: %v", header, coerceErr))
				break
			}
			properties[fieldDef.Name] = coerced
		}

		if len(rowErrors) == 0 {
			validationResult := s.validator.ValidateProperties(properties, validatorDefs)
			if !validationResult.IsValid {
				for _, validationErr := range validationResult.Errors {
					rowErrors = append(rowErrors, fmt.Sprintf("%s: %s", validationErr.Field, validationErr.Message))
				}
				for _, warning := range validationResult.Warnings {
					rowErrors = append(rowErrors, fmt.Sprintf("warning %s: %s", warning.Field, warning.Message))
				}
			}
		}

		if len(rowErrors) > 0 {
			invalidRows++
		}

		if rowIdx < limit {
			previewRow := PreviewRow{
				RowNumber: rowNumber,
				Values:    rowValues,
			}
			if len(rowErrors) > 0 {
				previewRow.Errors = rowErrors
			}
			result.Rows = append(result.Rows, previewRow)
		}
	}

	result.InvalidRows = invalidRows

	detectedByName := make(map[string]domain.FieldDefinition, len(autoDetected))
	for _, field := range autoDetected {
		detectedByName[field.Name] = field
	}

	for idx, header := range table.headers {
		detectedField, ok := detectedByName[header]
		var detectedType domain.FieldType
		var detectedRequired bool
		if ok {
			detectedType = detectedField.Type
			detectedRequired = detectedField.Required
		}

		effectiveField, ok := fieldMap[header]
		effectiveType := detectedType
		required := detectedRequired
		if ok {
			effectiveType = effectiveField.Type
			required = effectiveField.Required
		}

		previewHeader := PreviewHeader{
			Name:          header,
			OriginalLabel: "",
			DetectedType:  string(detectedType),
			EffectiveType: string(effectiveType),
			Required:      required,
			Overridden:    req.ColumnOverrides != nil && req.ColumnOverrides[header] != "",
		}
		if idx < len(table.rawHeaders) {
			previewHeader.OriginalLabel = table.rawHeaders[idx]
		}
		result.Headers = append(result.Headers, previewHeader)
	}

	return result, nil
}

func parseTable(fileName string, payload []byte, headerRowIndex *int) (tableData, [][]string, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".csv":
		return parseCSV(payload, headerRowIndex)
	case ".xlsx":
		return parseExcel(payload, headerRowIndex)
	default:
		return tableData{}, nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
}

func parseCSV(payload []byte, headerRowIndex *int) (tableData, [][]string, error) {
	reader := bufio.NewReader(bytes.NewReader(payload))
	if prefix, err := reader.Peek(len(byteOrderMark)); err == nil && bytes.Equal(prefix, byteOrderMark) {
		_, _ = reader.Discard(len(byteOrderMark))
	}

	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		return tableData{}, nil, fmt.Errorf("failed to read csv: %w", err)
	}

	table, err := normalizeTable(records, headerRowIndex)
	if err != nil {
		return tableData{}, nil, err
	}
	return table, records, nil
}

func parseExcel(payload []byte, headerRowIndex *int) (tableData, [][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(payload))
	if err != nil {
		return tableData{}, nil, fmt.Errorf("failed to open xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return tableData{}, nil, errors.New("excel file has no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return tableData{}, nil, fmt.Errorf("failed to read rows from xlsx: %w", err)
	}

	table, err := normalizeTable(rows, headerRowIndex)
	if err != nil {
		return tableData{}, nil, err
	}
	return table, rows, nil
}

func normalizeTable(records [][]string, headerRowIndex *int) (tableData, error) {
	if len(records) == 0 {
		return tableData{}, errors.New("no rows found in file")
	}

	var headerRow []string
	var dataRows [][]string
	headerIndex := -1

	if headerRowIndex != nil {
		if *headerRowIndex < 0 || *headerRowIndex >= len(records) {
			return tableData{}, fmt.Errorf("header row index %d out of range", *headerRowIndex)
		}
		selected := cleanRow(records[*headerRowIndex])
		if len(selected) == 0 {
			return tableData{}, fmt.Errorf("selected header row %d is empty", *headerRowIndex+1)
		}
		headerRow = records[*headerRowIndex]
		headerIndex = *headerRowIndex
		for idx := *headerRowIndex + 1; idx < len(records); idx++ {
			row := records[idx]
			if len(cleanRow(row)) == 0 {
				continue
			}
			dataRows = append(dataRows, row)
		}
	} else {
		for idx, row := range records {
			if len(cleanRow(row)) == 0 {
				continue
			}
			if headerRow == nil {
				headerRow = row
				headerIndex = idx
				continue
			}
			dataRows = append(dataRows, row)
		}
	}

	if headerRow == nil {
		return tableData{}, errors.New("header row could not be detected")
	}

	headers := sanitizeHeaders(headerRow)
	rawHeaders := make([]string, len(headerRow))
	for i, value := range headerRow {
		rawHeaders[i] = strings.TrimSpace(value)
	}

	for i := range dataRows {
		dataRows[i] = padRow(dataRows[i], len(headers))
	}

	dataRows = filterEmptyRows(dataRows)

	return tableData{
		headers:        headers,
		rawHeaders:     rawHeaders,
		rows:           dataRows,
		headerRowIndex: headerIndex,
	}, nil
}

func buildHeaderCandidates(records [][]string, limit int, currentIndex int) []HeaderCandidate {
	if limit <= 0 {
		limit = 10
	}

	candidates := make([]HeaderCandidate, 0, limit)
	for idx, row := range records {
		if len(cleanRow(row)) == 0 {
			continue
		}

		values := make([]string, len(row))
		for i, cell := range row {
			values[i] = strings.TrimSpace(cell)
		}

		candidates = append(candidates, HeaderCandidate{
			Index:   idx,
			Values:  values,
			Current: idx == currentIndex,
		})

		if len(candidates) >= limit {
			break
		}
	}

	return candidates
}

func cleanRow(row []string) []string {
	var cleaned []string
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			cleaned = append(cleaned, cell)
		}
	}
	return cleaned
}

func sanitizeHeaders(raw []string) []string {
	headers := make([]string, len(raw))
	seen := make(map[string]int)

	for idx, value := range raw {
		name := strings.TrimSpace(value)
		name = strings.ReplaceAll(name, " ", "_")
		name = strings.ReplaceAll(name, ".", "_")
		name = strings.ReplaceAll(name, "-", "_")
		name = strings.Trim(name, "_")
		if name == "" {
			name = fmt.Sprintf("column_%d", idx+1)
		}

		base := name
		count := seen[base]
		if count > 0 {
			name = fmt.Sprintf("%s_%d", base, count+1)
		}
		seen[base] = count + 1

		headers[idx] = name
	}

	return headers
}

func padRow(row []string, length int) []string {
	if len(row) >= length {
		return row[:length]
	}
	padded := make([]string, length)
	copy(padded, row)
	for i := len(row); i < length; i++ {
		padded[i] = ""
	}
	return padded
}

func filterEmptyRows(rows [][]string) [][]string {
	var filtered [][]string
	for _, row := range rows {
		keep := false
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				keep = true
				break
			}
		}
		if keep {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func inferFieldDefinitions(table tableData) []domain.FieldDefinition {
	definitions := make([]domain.FieldDefinition, 0, len(table.headers))
	for idx, header := range table.headers {
		fieldType, required := profileColumn(idx, table.rows)
		definitions = append(definitions, domain.FieldDefinition{
			Name:     header,
			Type:     fieldType,
			Required: required,
		})
	}
	return definitions
}

func applyOverridesToDefinitions(fields []domain.FieldDefinition, overrides map[string]domain.FieldType) []domain.FieldDefinition {
	if len(fields) == 0 || len(overrides) == 0 {
		return fields
	}
	overridden := make([]domain.FieldDefinition, len(fields))
	for idx, field := range fields {
		if override, ok := overrides[field.Name]; ok && override != "" {
			field.Type = override
		}
		overridden[idx] = field
	}
	return overridden
}

func profileColumn(col int, rows [][]string) (domain.FieldType, bool) {
	isBool := true
	isInt := true
	isFloat := true
	isTimestamp := true
	allPresent := true
	hasValue := false

	for _, row := range rows {
		if col >= len(row) {
			allPresent = false
			continue
		}

		value := strings.TrimSpace(row[col])
		if value == "" {
			allPresent = false
			continue
		}

		hasValue = true

		if !looksLikeBool(value) {
			isBool = false
		}
		if !looksLikeInt(value) {
			isInt = false
		}
		if !looksLikeFloat(value) {
			isFloat = false
		}
		if !looksLikeTimestamp(value) {
			isTimestamp = false
		}
	}

	switch {
	case isBool && hasValue:
		return domain.FieldTypeBoolean, allPresent && hasValue
	case isInt && hasValue:
		return domain.FieldTypeInteger, allPresent && hasValue
	case isFloat && hasValue:
		return domain.FieldTypeFloat, allPresent && hasValue
	case isTimestamp && hasValue:
		return domain.FieldTypeTimestamp, allPresent && hasValue
	default:
		return domain.FieldTypeString, allPresent && hasValue
	}
}

func looksLikeBool(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "true" || value == "false" {
		return true
	}
	if value == "1" || value == "0" {
		return true
	}
	if value == "yes" || value == "no" {
		return true
	}
	_, err := strconv.ParseBool(value)
	return err == nil
}

func looksLikeInt(value string) bool {
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return true
	}
	// Allow float representations that can be losslessly converted to int.
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return math.Mod(f, 1) == 0
	}
	return false
}

func looksLikeFloat(value string) bool {
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func looksLikeTimestamp(value string) bool {
	_, err := parseTimestamp(value)
	return err == nil
}

func fieldTypesCompatible(existing, detected domain.FieldType) bool {
	if existing == detected {
		return true
	}
	// Allow float detections for integer fields.
	if existing == domain.FieldTypeFloat && detected == domain.FieldTypeInteger {
		return true
	}
	return false
}

func buildValidatorDefinitions(fields []domain.FieldDefinition) map[string]validator.FieldDefinition {
	defs := make(map[string]validator.FieldDefinition, len(fields))
	for _, field := range fields {
		var refType *string
		if field.ReferenceEntityType != "" {
			ref := field.ReferenceEntityType
			refType = &ref
		}
		var validation any
		if trimmed := strings.TrimSpace(field.Validation); trimmed != "" {
			var parsed any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				validation = parsed
			}
		}
		defs[field.Name] = validator.FieldDefinition{
			Type:                graph.FieldType(strings.ToUpper(string(field.Type))),
			Required:            field.Required,
			Description:         field.Description,
			Default:             field.Default,
			Validation:          validation,
			ReferenceEntityType: refType,
		}
	}
	return defs
}

func coerceValue(fieldType domain.FieldType, raw string) (any, error) {
	switch fieldType {
	case domain.FieldTypeString:
		return raw, nil
	case domain.FieldTypeInteger:
		if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return i, nil
		}
		if f, err := strconv.ParseFloat(raw, 64); err == nil && math.Mod(f, 1) == 0 {
			return int64(f), nil
		}
		return nil, fmt.Errorf("unable to coerce %q to integer", raw)
	case domain.FieldTypeFloat:
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return f, nil
		}
		return nil, fmt.Errorf("unable to coerce %q to float", raw)
	case domain.FieldTypeBoolean:
		value := strings.ToLower(strings.TrimSpace(raw))
		switch value {
		case "1", "yes", "y":
			return true, nil
		case "0", "no", "n":
			return false, nil
		}
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("unable to coerce %q to boolean", raw)
		}
		return boolVal, nil
	case domain.FieldTypeTimestamp:
		ts, err := parseTimestamp(raw)
		if err != nil {
			return nil, fmt.Errorf("unable to coerce %q to timestamp: %w", raw, err)
		}
		return ts, nil
	case domain.FieldTypeJSON:
		var out any
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return nil, fmt.Errorf("invalid json payload: %w", err)
		}
		return out, nil
	default:
		// Fallback for unknown types; best effort interpretation.
		return raw, nil
	}
}

func parseTimestamp(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range timeLayouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp format")
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return ""
	}
	return value
}

func generatePath(schemaName string, row []string, index int, used map[string]int) string {
	base := slugify(schemaName)
	if base == "" {
		base = "entity"
	}

	var candidate string
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell != "" {
			candidate = cell
			break
		}
	}

	child := slugify(candidate)
	if child == "" {
		child = fmt.Sprintf("row_%d", index+1)
	}

	path := fmt.Sprintf("%s.%s", base, child)
	if count, ok := used[path]; ok {
		count++
		used[path] = count
		path = fmt.Sprintf("%s_%d", path, count)
	} else {
		used[path] = 1
	}
	return path
}

func (s *Service) summaryRowError(ctx context.Context, req Request, rowNumber int, err error) {
	s.logIngestionError(ctx, req, &rowNumber, err)
}

func (s *Service) logIngestionError(ctx context.Context, req Request, rowNumber *int, err error) {
	if s.logRepo == nil || err == nil {
		return
	}
	entry := domain.IngestionLogEntry{
		OrganizationID: req.OrganizationID,
		SchemaName:     req.SchemaName,
		FileName:       req.FileName,
		ErrorMessage:   err.Error(),
	}
	if rowNumber != nil {
		entry.RowNumber = rowNumber
	}
	_ = s.logRepo.Record(ctx, entry)
}

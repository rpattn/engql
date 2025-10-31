package export

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"
	"github.com/rpattn/engql/internal/transformations"
)

type workerFunc func(context.Context, domain.EntityExportJob) error

var errJobNotRunnable = errors.New("export job is no longer runnable")

type Service struct {
	organizations          repository.OrganizationRepository
	schemaRepo             repository.EntitySchemaRepository
	entityRepo             repository.EntityRepository
	exportRepo             repository.EntityExportRepository
	transformationRepo     repository.EntityTransformationRepository
	transformationExecutor *transformations.Executor

	exportDir  string
	jobTimeout time.Duration
	pageSize   int
	now        func() time.Time

	downloadSigner *downloadSigner

	workerCancels sync.Map // map[uuid.UUID]context.CancelFunc
}

type Option func(*Service)

func WithExportDirectory(dir string) Option {
	return func(s *Service) {
		if strings.TrimSpace(dir) != "" {
			s.exportDir = filepath.Clean(dir)
		}
	}
}

func WithJobTimeout(timeout time.Duration) Option {
	return func(s *Service) {
		if timeout > 0 {
			s.jobTimeout = timeout
		}
	}
}

func WithPageSize(size int) Option {
	return func(s *Service) {
		if size > 0 {
			s.pageSize = size
		}
	}
}

// WithDownloadTokenTTL customizes the TTL for generated download links.
func WithDownloadTokenTTL(ttl time.Duration) Option {
	return func(s *Service) {
		if ttl > 0 {
			s.downloadSigner = newDownloadSigner(ttl)
		}
	}
}

func NewService(
	organizations repository.OrganizationRepository,
	schemaRepo repository.EntitySchemaRepository,
	entityRepo repository.EntityRepository,
	exportRepo repository.EntityExportRepository,
	transformationRepo repository.EntityTransformationRepository,
	opts ...Option,
) *Service {
	service := &Service{
		organizations:          organizations,
		schemaRepo:             schemaRepo,
		entityRepo:             entityRepo,
		exportRepo:             exportRepo,
		transformationRepo:     transformationRepo,
		transformationExecutor: transformations.NewExecutor(entityRepo, schemaRepo),
		exportDir:              filepath.Join(os.TempDir(), "engql-exports"),
		jobTimeout:             30 * time.Minute,
		pageSize:               1000,
		now:                    time.Now,
	}
	for _, opt := range opts {
		opt(service)
	}
	if service.pageSize <= 0 {
		service.pageSize = 1000
	}
	if service.jobTimeout <= 0 {
		service.jobTimeout = 30 * time.Minute
	}
	if strings.TrimSpace(service.exportDir) == "" {
		service.exportDir = filepath.Join(os.TempDir(), "engql-exports")
	}
	if service.downloadSigner == nil {
		service.downloadSigner = newDownloadSigner(5 * time.Minute)
	}
	if service.now == nil {
		service.now = time.Now
	}
	return service
}

type EntityTypeExportRequest struct {
	OrganizationID uuid.UUID
	EntityType     string
	Filters        []domain.PropertyFilter
}

type TransformationExportRequest struct {
	OrganizationID   uuid.UUID
	TransformationID uuid.UUID
	Filters          []domain.PropertyFilter
	Options          domain.EntityTransformationExecutionOptions
}

func (s *Service) QueueEntityTypeExport(ctx context.Context, req EntityTypeExportRequest) (domain.EntityExportJob, error) {
	if req.OrganizationID == uuid.Nil {
		return domain.EntityExportJob{}, errors.New("organization ID is required")
	}
	entityType := strings.TrimSpace(req.EntityType)
	if entityType == "" {
		return domain.EntityExportJob{}, errors.New("entity type is required")
	}
	if _, err := s.organizations.GetByID(ctx, req.OrganizationID); err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("validate organization: %w", err)
	}
	if _, err := s.schemaRepo.GetByName(ctx, req.OrganizationID, entityType); err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("resolve schema %s: %w", entityType, err)
	}
	filter := &domain.EntityFilter{EntityType: entityType, PropertyFilters: append([]domain.PropertyFilter(nil), req.Filters...)}
	_, total, err := s.entityRepo.List(ctx, req.OrganizationID, filter, nil, 1, 0)
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("estimate export rows: %w", err)
	}
	rowsRequested := total
	job := domain.EntityExportJob{
		OrganizationID: req.OrganizationID,
		JobType:        domain.EntityExportJobTypeEntityType,
		EntityType:     &entityType,
		Filters:        append([]domain.PropertyFilter(nil), req.Filters...),
		RowsRequested:  rowsRequested,
	}
	persisted, err := s.exportRepo.Create(ctx, job)
	if err != nil {
		return domain.EntityExportJob{}, err
	}
	s.launchWorker(persisted, s.runEntityTypeExport)
	return persisted, nil
}

func (s *Service) QueueTransformationExport(ctx context.Context, req TransformationExportRequest) (domain.EntityExportJob, error) {
	if req.OrganizationID == uuid.Nil {
		return domain.EntityExportJob{}, errors.New("organization ID is required")
	}
	if req.TransformationID == uuid.Nil {
		return domain.EntityExportJob{}, errors.New("transformation ID is required")
	}
	if _, err := s.organizations.GetByID(ctx, req.OrganizationID); err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("validate organization: %w", err)
	}
	transformation, err := s.transformationRepo.GetByID(ctx, req.TransformationID)
	if err != nil {
		return domain.EntityExportJob{}, fmt.Errorf("load transformation: %w", err)
	}
	transformationCopy := transformation
	optionsCopy := req.Options
	rowsRequested := 0
	if optionsCopy.Limit > 0 {
		rowsRequested = optionsCopy.Limit
	}
	job := domain.EntityExportJob{
		OrganizationID:        req.OrganizationID,
		JobType:               domain.EntityExportJobTypeTransformation,
		TransformationID:      &req.TransformationID,
		Transformation:        &transformationCopy,
		TransformationOptions: &optionsCopy,
		Filters:               append([]domain.PropertyFilter(nil), req.Filters...),
		RowsRequested:         rowsRequested,
	}
	persisted, err := s.exportRepo.Create(ctx, job)
	if err != nil {
		return domain.EntityExportJob{}, err
	}
	s.launchWorker(persisted, s.runTransformationExport)
	return persisted, nil
}

func (s *Service) ListJobs(ctx context.Context, organizationID *uuid.UUID, statuses []domain.EntityExportJobStatus, limit, offset int) ([]domain.EntityExportJob, error) {
	return s.exportRepo.List(ctx, organizationID, statuses, limit, offset)
}

func (s *Service) ListLogs(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]domain.EntityExportLog, error) {
	return s.exportRepo.ListLogs(ctx, jobID, limit, offset)
}

// GetJob returns the metadata for a single export job.
func (s *Service) GetJob(ctx context.Context, id uuid.UUID) (domain.EntityExportJob, error) {
	if id == uuid.Nil {
		return domain.EntityExportJob{}, errors.New("job ID is required")
	}
	return s.exportRepo.GetByID(ctx, id)
}

// BuildDownloadURL signs a short-lived download URL for completed export files.
func (s *Service) BuildDownloadURL(job domain.EntityExportJob) (*string, error) {
	if job.Status != domain.EntityExportJobStatusCompleted {
		return nil, nil
	}
	if job.FilePath == nil || strings.TrimSpace(*job.FilePath) == "" {
		return nil, nil
	}
	if s.downloadSigner == nil {
		return nil, errors.New("download signer not configured")
	}
	token := s.downloadSigner.Sign(job.ID, s.now())
	values := url.Values{}
	values.Set("token", token)
	download := fmt.Sprintf("/exports/files/%s?%s", job.ID.String(), values.Encode())
	return &download, nil
}

// ValidateDownloadToken ensures the token is valid for the given job.
func (s *Service) ValidateDownloadToken(jobID uuid.UUID, token string) error {
	if s.downloadSigner == nil {
		return errors.New("download signer not configured")
	}
	return s.downloadSigner.Verify(jobID, token, s.now())
}

// OpenJobFile opens the completed export file for streaming to the client.
func (s *Service) OpenJobFile(job domain.EntityExportJob) (*os.File, error) {
	if job.Status != domain.EntityExportJobStatusCompleted {
		return nil, errors.New("export is not completed")
	}
	if job.FilePath == nil || strings.TrimSpace(*job.FilePath) == "" {
		return nil, errors.New("export file is unavailable")
	}
	file, err := os.Open(*job.FilePath)
	if err != nil {
		return nil, fmt.Errorf("open export file: %w", err)
	}
	return file, nil
}

// CancelJob requests cancellation for a pending or running export job.
func (s *Service) CancelJob(ctx context.Context, id uuid.UUID) (domain.EntityExportJob, error) {
	if id == uuid.Nil {
		return domain.EntityExportJob{}, errors.New("job ID is required")
	}
	job, err := s.exportRepo.GetByID(ctx, id)
	if err != nil {
		return domain.EntityExportJob{}, err
	}
	if job.Status != domain.EntityExportJobStatusPending && job.Status != domain.EntityExportJobStatusRunning {
		return job, fmt.Errorf("export job in status %s cannot be cancelled", job.Status)
	}
	reason := "Cancelled by user"
	if err := s.exportRepo.MarkCancelled(ctx, id, reason); err != nil {
		if errors.Is(err, repository.ErrExportJobStatusConflict) {
			updated, getErr := s.exportRepo.GetByID(ctx, id)
			if getErr != nil {
				return domain.EntityExportJob{}, getErr
			}
			return updated, nil
		}
		return domain.EntityExportJob{}, err
	}
	if cancel, ok := s.workerCancels.LoadAndDelete(id); ok {
		if fn, okCast := cancel.(context.CancelFunc); okCast {
			fn()
		}
	}
	return s.exportRepo.GetByID(ctx, id)
}

func (s *Service) launchWorker(job domain.EntityExportJob, run workerFunc) {
	baseCtx, baseCancel := context.WithCancel(context.Background())
	ctx := baseCtx
	cancelFunc := baseCancel
	if s.jobTimeout > 0 {
		timeoutCtx, timeoutCancel := context.WithTimeout(baseCtx, s.jobTimeout)
		ctx = timeoutCtx
		cancelFunc = func() {
			timeoutCancel()
			baseCancel()
		}
	}
	s.workerCancels.Store(job.ID, cancelFunc)
	go func() {
		defer func() {
			cancelFunc()
			s.workerCancels.Delete(job.ID)
		}()
		defer func() {
			if rec := recover(); rec != nil {
				err := fmt.Errorf("panic: %v", rec)
				log.Printf("[export] panic while processing job %s: %v", job.ID, rec)
				s.failJob(context.Background(), job.ID, err)
			}
		}()
		if err := run(ctx, job); err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				log.Printf("[export] job %s cancelled", job.ID)
			case errors.Is(err, errJobNotRunnable):
				log.Printf("[export] job %s not runnable, skipping", job.ID)
			default:
				s.failJob(ctx, job.ID, err)
			}
		}
	}()
}

func (s *Service) failJob(ctx context.Context, jobID uuid.UUID, err error) {
	if err == nil {
		return
	}
	if ctx == nil || ctx.Err() != nil {
		ctx = context.Background()
	}
	message := truncateError(err)
	if markErr := s.exportRepo.MarkFailed(ctx, jobID, message); markErr != nil {
		log.Printf("[export] failed to mark job %s as failed: %v (original error: %v)", jobID, markErr, err)
		return
	}
	log.Printf("[export] job %s failed: %v", jobID, err)
}

func (s *Service) runEntityTypeExport(ctx context.Context, job domain.EntityExportJob) error {
	if job.EntityType == nil || strings.TrimSpace(*job.EntityType) == "" {
		return errors.New("export job missing entity type")
	}
	if err := s.exportRepo.MarkRunning(ctx, job.ID); err != nil {
		if errors.Is(err, repository.ErrExportJobStatusConflict) {
			return errJobNotRunnable
		}
		return fmt.Errorf("mark export job running: %w", err)
	}
	schema, err := s.schemaRepo.GetByName(ctx, job.OrganizationID, *job.EntityType)
	if err != nil {
		return fmt.Errorf("load schema %s: %w", *job.EntityType, err)
	}
	if err := s.ensureExportDirectory(); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(s.exportDir, fmt.Sprintf("%s-*.csv", job.ID))
	if err != nil {
		return fmt.Errorf("create temp export file: %w", err)
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}()

	buffered := bufio.NewWriterSize(tempFile, 1<<20) // 1 MiB buffer for streaming writes
	counter := &countingWriter{writer: buffered}
	csvWriter := csv.NewWriter(counter)

	headers := schemaFieldNames(schema.Fields)
	rows := make([]string, len(headers))
	const gcInterval = 500000
	nextGCTrigger := gcInterval
	if len(headers) > 0 {
		if err := csvWriter.Write(headers); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush header: %w", err)
	}
	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("flush buffered header: %w", err)
	}

	rowsExported := 0
	rowsTarget := job.RowsRequested
	offset := 0
	pageSize := s.pageSize
	filters := append([]domain.PropertyFilter(nil), job.Filters...)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		entities, total, err := s.entityRepo.List(ctx, job.OrganizationID, &domain.EntityFilter{EntityType: *job.EntityType, PropertyFilters: filters}, nil, pageSize, offset)
		if err != nil {
			return fmt.Errorf("list entities: %w", err)
		}
		if offset == 0 && total > 0 {
			rowsTarget = total
		}
		if len(entities) == 0 {
			break
		}
		batchSize := len(entities)
		for _, entity := range entities {
			for i, field := range headers {
				rows[i] = formatValue(entity.Properties[field])
			}
			if err := csvWriter.Write(rows); err != nil {
				return fmt.Errorf("write entity row: %w", err)
			}
			rowsExported++
			if gcInterval > 0 && rowsExported >= nextGCTrigger {
				runtime.GC()
				nextGCTrigger += gcInterval
			}
		}
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("flush rows: %w", err)
		}
		if err := buffered.Flush(); err != nil {
			return fmt.Errorf("flush buffered rows: %w", err)
		}
		var requestedPtr *int
		if rowsTarget > 0 {
			requestedPtr = &rowsTarget
		}
		if err := s.exportRepo.UpdateProgress(ctx, job.ID, rowsExported, counter.count, requestedPtr); err != nil {
			return fmt.Errorf("update export progress: %w", err)
		}
		shouldBreak := false
		if rowsTarget > 0 && rowsExported >= rowsTarget {
			shouldBreak = true
		}
		if !shouldBreak && batchSize < pageSize {
			shouldBreak = true
		}
		for i := range entities {
			// Release entity batch memory promptly for streaming exports.
			entities[i].Properties = nil
		}
		entities = nil // Drop reference to entity batch to allow GC.
		if shouldBreak {
			break
		}
		offset += pageSize
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("final flush: %w", err)
	}
	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("final buffered flush: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync export file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close export file: %w", err)
	}

	finalPath := filepath.Join(s.exportDir, s.finalFileName(job))
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("promote export file: %w", err)
	}
	cleanup = false
	info, err := os.Stat(finalPath)
	if err != nil {
		return fmt.Errorf("stat export file: %w", err)
	}
	size := info.Size()
	mime := "text/csv"
	bytesWritten := counter.count
	if bytesWritten == 0 {
		bytesWritten = size
	}
	if err := s.exportRepo.MarkCompleted(ctx, job.ID, repository.EntityExportResult{
		RowsExported: rowsExported,
		BytesWritten: bytesWritten,
		FilePath:     &finalPath,
		FileMimeType: &mime,
		FileByteSize: &size,
	}); err != nil {
		return fmt.Errorf("mark export completed: %w", err)
	}
	log.Printf("[export] job %s completed (rows=%d path=%s)", job.ID, rowsExported, finalPath)
	return nil
}

func (s *Service) runTransformationExport(ctx context.Context, job domain.EntityExportJob) error {
	if err := s.exportRepo.MarkRunning(ctx, job.ID); err != nil {
		if errors.Is(err, repository.ErrExportJobStatusConflict) {
			return errJobNotRunnable
		}
		return fmt.Errorf("mark export job running: %w", err)
	}
	transformation := job.Transformation
	if transformation == nil && job.TransformationID != nil {
		loaded, err := s.transformationRepo.GetByID(ctx, *job.TransformationID)
		if err != nil {
			return fmt.Errorf("load transformation %s: %w", job.TransformationID, err)
		}
		transformation = &loaded
	}
	if transformation == nil {
		return errors.New("export job missing transformation definition")
	}
	materializeConfig, err := findMaterializeConfig(*transformation)
	if err != nil {
		return err
	}
	columns := buildMaterializeColumns(materializeConfig)
	if err := s.ensureExportDirectory(); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(s.exportDir, fmt.Sprintf("%s-*.csv", job.ID))
	if err != nil {
		return fmt.Errorf("create temp export file: %w", err)
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}()

	buffered := bufio.NewWriterSize(tempFile, 1<<20) // 1 MiB buffer for streaming writes
	counter := &countingWriter{writer: buffered}
	csvWriter := csv.NewWriter(counter)

	if len(columns) > 0 {
		headers := make([]string, len(columns))
		for i, column := range columns {
			headers[i] = column.header
		}
		if err := csvWriter.Write(headers); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush header: %w", err)
	}
	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("flush buffered header: %w", err)
	}

	options := domain.EntityTransformationExecutionOptions{}
	if job.TransformationOptions != nil {
		options = *job.TransformationOptions
	}
	baseOffset := options.Offset
	if baseOffset < 0 {
		baseOffset = 0
	}
	requested := options.Limit
	if requested < 0 {
		requested = 0
	}
	rowsTarget := requested
	rowsExported := 0
	totalCount := 0

	rowBuffer := make([]string, len(columns))
	const gcInterval = 500000
	nextGCTrigger := gcInterval

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		limit := s.pageSize
		if requested > 0 {
			remaining := requested - rowsExported
			if remaining <= 0 {
				break
			}
			if remaining < limit {
				limit = remaining
			}
		}
		if limit <= 0 {
			break
		}
		pageOptions := domain.EntityTransformationExecutionOptions{Limit: limit, Offset: baseOffset + rowsExported}
		result, err := s.transformationExecutor.ExecuteStreaming(ctx, *transformation, pageOptions)
		if err != nil {
			return fmt.Errorf("execute transformation: %w", err)
		}
		if rowsExported == 0 {
			totalCount = result.TotalCount
			if rowsTarget == 0 && totalCount > 0 {
				remaining := totalCount - baseOffset
				if remaining < 0 {
					remaining = 0
				}
				rowsTarget = remaining
			}
		}
		if len(result.Records) == 0 {
			break
		}
		batchSize := len(result.Records)
		for _, record := range result.Records {
			for i, column := range columns {
				rowBuffer[i] = ""
				if entity := record.Entities[column.alias]; entity != nil {
					rowBuffer[i] = formatValue(entity.Properties[column.field])
				}
			}
			if err := csvWriter.Write(rowBuffer); err != nil {
				return fmt.Errorf("write transformation row: %w", err)
			}
			rowsExported++
			if gcInterval > 0 && rowsExported >= nextGCTrigger {
				runtime.GC()
				nextGCTrigger += gcInterval
			}
		}
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("flush rows: %w", err)
		}
		if err := buffered.Flush(); err != nil {
			return fmt.Errorf("flush buffered rows: %w", err)
		}
		var rowsPtr *int
		if rowsTarget > 0 {
			rowsPtr = &rowsTarget
		}
		if err := s.exportRepo.UpdateProgress(ctx, job.ID, rowsExported, counter.count, rowsPtr); err != nil {
			return fmt.Errorf("update export progress: %w", err)
		}
		shouldBreak := false
		if rowsTarget > 0 && rowsExported >= rowsTarget {
			shouldBreak = true
		}
		if !shouldBreak && batchSize < limit {
			shouldBreak = true
		}
		for i := range result.Records {
			// Release transformation batch memory promptly for streaming exports.
			for alias := range result.Records[i].Entities {
				result.Records[i].Entities[alias] = nil
			}
			result.Records[i].Entities = nil
		}
		result.Records = nil // Drop reference to transformation batch to allow GC.
		if shouldBreak {
			break
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("final flush: %w", err)
	}
	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("final buffered flush: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync export file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close export file: %w", err)
	}
	finalPath := filepath.Join(s.exportDir, s.finalFileName(job))
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("promote export file: %w", err)
	}
	cleanup = false
	info, err := os.Stat(finalPath)
	if err != nil {
		return fmt.Errorf("stat export file: %w", err)
	}
	size := info.Size()
	bytesWritten := counter.count
	if bytesWritten == 0 {
		bytesWritten = size
	}
	mime := "text/csv"
	if err := s.exportRepo.MarkCompleted(ctx, job.ID, repository.EntityExportResult{
		RowsExported: rowsExported,
		BytesWritten: bytesWritten,
		FilePath:     &finalPath,
		FileMimeType: &mime,
		FileByteSize: &size,
	}); err != nil {
		return fmt.Errorf("mark export completed: %w", err)
	}
	log.Printf("[export] transformation job %s completed (rows=%d path=%s)", job.ID, rowsExported, finalPath)
	return nil
}

func (s *Service) ensureExportDirectory() error {
	if strings.TrimSpace(s.exportDir) == "" {
		return errors.New("export directory is not configured")
	}
	if err := os.MkdirAll(s.exportDir, 0o755); err != nil {
		return fmt.Errorf("ensure export directory: %w", err)
	}
	return nil
}

func (s *Service) finalFileName(job domain.EntityExportJob) string {
	var base string
	switch job.JobType {
	case domain.EntityExportJobTypeEntityType:
		if job.EntityType != nil && strings.TrimSpace(*job.EntityType) != "" {
			base = sanitizeFileComponent(*job.EntityType)
		}
	case domain.EntityExportJobTypeTransformation:
		base = "transformation"
	}
	if base == "" {
		base = "entity-export"
	}
	return fmt.Sprintf("%s-%s.csv", base, job.ID.String())
}

func schemaFieldNames(fields []domain.FieldDefinition) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != "" {
			names = append(names, field.Name)
		}
	}
	return names
}

type materializeColumn struct {
	alias  string
	field  string
	header string
}

func findMaterializeConfig(transformation domain.EntityTransformation) (*domain.EntityTransformationMaterializeConfig, error) {
	var config *domain.EntityTransformationMaterializeConfig
	for i := range transformation.Nodes {
		node := transformation.Nodes[i]
		if node.Type != domain.TransformationNodeMaterialize || node.Materialize == nil {
			continue
		}
		copyConfig := *node.Materialize
		config = &copyConfig
	}
	if config == nil {
		return nil, fmt.Errorf("transformation %s missing materialize node", transformation.ID)
	}
	return config, nil
}

func buildMaterializeColumns(config *domain.EntityTransformationMaterializeConfig) []materializeColumn {
	if config == nil {
		return []materializeColumn{}
	}
	columns := make([]materializeColumn, 0)
	for _, output := range config.Outputs {
		alias := strings.TrimSpace(output.Alias)
		if alias == "" {
			continue
		}
		for _, field := range output.Fields {
			targetField := strings.TrimSpace(field.OutputField)
			if targetField == "" {
				continue
			}

			header := targetField
			if alias != "" {
				header = fmt.Sprintf("%s.%s", alias, targetField)
			}
			columns = append(columns, materializeColumn{
				alias:  alias,
				field:  targetField,
				header: header,
			})
		}
	}
	return columns
}

func sanitizeFileComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	builder := strings.Builder{}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		case r == ' ':
			builder.WriteRune('-')
		default:
			builder.WriteRune('-')
		}
	}
	result := builder.String()
	result = strings.Trim(result, "-")
	if result == "" {
		return "export"
	}
	return result
}

type countingWriter struct {
	writer *bufio.Writer
	count  int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.writer.Write(p)
	c.count += int64(n)
	return n, err
}

func formatValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case time.Time:
		return v.UTC().Format(time.RFC3339)
	case *time.Time:
		if v == nil {
			return ""
		}
		return v.UTC().Format(time.RFC3339)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case json.Number:
		return v.String()
	case float32, float64, int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case []byte:
		return string(v)
	case map[string]any, []any:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(encoded)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	const maxLen = 512
	msg := err.Error()
	if len(msg) > maxLen {
		return msg[:maxLen]
	}
	return msg
}

type downloadSigner struct {
	secret []byte
	ttl    time.Duration
}

func newDownloadSigner(ttl time.Duration) *downloadSigner {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &downloadSigner{secret: []byte(uuid.New().String()), ttl: ttl}
}

func (s *downloadSigner) Sign(jobID uuid.UUID, now time.Time) string {
	expires := now.Add(s.ttl).Unix()
	payload := fmt.Sprintf("%s:%d", jobID.String(), expires)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	raw := fmt.Sprintf("%s:%s", payload, signature)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func (s *downloadSigner) Verify(jobID uuid.UUID, token string, now time.Time) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("missing download token")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("decode token: %w", err)
	}
	parts := strings.Split(string(decoded), ":")
	if len(parts) != 3 {
		return errors.New("invalid token format")
	}
	if parts[0] != jobID.String() {
		return errors.New("token does not match export job")
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid token expiration: %w", err)
	}
	if now.Unix() > expires {
		return errors.New("download token expired")
	}
	payload := fmt.Sprintf("%s:%s", parts[0], parts[1])
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	expected := mac.Sum(nil)
	provided, err := hex.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("invalid token signature: %w", err)
	}
	if !hmac.Equal(expected, provided) {
		return errors.New("invalid download token")
	}
	return nil
}

-- Track background export jobs for entities.

-- name: InsertEntityExportJob :exec
INSERT INTO entity_export_jobs (
    id,
    organization_id,
    job_type,
    entity_type,
    transformation_id,
    filters,
    rows_requested,
    status
) VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(job_type),
    sqlc.arg(entity_type),
    sqlc.arg(transformation_id),
    sqlc.arg(filters),
    sqlc.arg(rows_requested),
    'PENDING'
);

-- name: MarkEntityExportJobRunning :exec
UPDATE entity_export_jobs
SET status = 'RUNNING',
    started_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: UpdateEntityExportJobProgress :exec
UPDATE entity_export_jobs
SET rows_exported = sqlc.arg(rows_exported),
    rows_requested = GREATEST(
        rows_requested,
        sqlc.arg(rows_exported),
        COALESCE(sqlc.narg(rows_requested)::INTEGER, rows_requested)
    ),
    bytes_written = sqlc.arg(bytes_written),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: MarkEntityExportJobCompleted :exec
UPDATE entity_export_jobs
SET status = 'COMPLETED',
    rows_exported = sqlc.arg(rows_exported),
    bytes_written = sqlc.arg(bytes_written),
    file_path = sqlc.arg(file_path),
    file_mime_type = sqlc.arg(file_mime_type),
    file_byte_size = sqlc.arg(file_byte_size),
    rows_requested = GREATEST(rows_requested, sqlc.arg(rows_exported)),
    completed_at = NOW(),
    updated_at = NOW(),
    error_message = NULL
WHERE id = sqlc.arg(id);

-- name: MarkEntityExportJobFailed :exec
UPDATE entity_export_jobs
SET status = 'FAILED',
    error_message = sqlc.arg(error_message),
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: GetEntityExportJobByID :one
SELECT
    id,
    organization_id,
    job_type,
    entity_type,
    transformation_id,
    filters,
    rows_requested,
    rows_exported,
    bytes_written,
    file_path,
    file_mime_type,
    file_byte_size,
    status,
    error_message,
    enqueued_at,
    started_at,
    completed_at,
    updated_at
FROM entity_export_jobs
WHERE id = sqlc.arg(id);

-- name: ListEntityExportJobsByStatus :many
SELECT
    id,
    organization_id,
    job_type,
    entity_type,
    transformation_id,
    filters,
    rows_requested,
    rows_exported,
    bytes_written,
    file_path,
    file_mime_type,
    file_byte_size,
    status,
    error_message,
    enqueued_at,
    started_at,
    completed_at,
    updated_at
FROM entity_export_jobs
WHERE status = ANY(sqlc.arg(statuses)::text[])
  AND (sqlc.narg(organization_id)::uuid IS NULL OR organization_id = sqlc.narg(organization_id))
ORDER BY enqueued_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: InsertEntityExportLog :exec
INSERT INTO entity_export_logs (
    export_job_id,
    organization_id,
    row_identifier,
    error_message
) VALUES (
    sqlc.arg(export_job_id),
    sqlc.arg(organization_id),
    sqlc.arg(row_identifier),
    sqlc.arg(error_message)
);

-- name: ListEntityExportLogsForJob :many
SELECT
    id,
    export_job_id,
    organization_id,
    row_identifier,
    error_message,
    created_at
FROM entity_export_logs
WHERE export_job_id = sqlc.arg(export_job_id)
ORDER BY created_at ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

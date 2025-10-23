-- Track background flush batches for staged entity ingestion.

-- name: InsertEntityIngestBatch :exec
INSERT INTO entity_ingest_batches (
    id,
    organization_id,
    schema_id,
    entity_type,
    file_name,
    rows_staged,
    skip_validation,
    status
) VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(schema_id),
    sqlc.arg(entity_type),
    sqlc.arg(file_name),
    sqlc.arg(rows_staged),
    sqlc.arg(skip_validation),
    'PENDING'
);

-- name: MarkEntityIngestBatchFlushing :exec
UPDATE entity_ingest_batches
SET status = 'FLUSHING',
    started_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: MarkEntityIngestBatchCompleted :exec
UPDATE entity_ingest_batches
SET status = 'COMPLETED',
    rows_flushed = sqlc.arg(rows_flushed),
    completed_at = NOW(),
    updated_at = NOW(),
    error_message = NULL
WHERE id = sqlc.arg(id);

-- name: MarkEntityIngestBatchFailed :exec
UPDATE entity_ingest_batches
SET status = 'FAILED',
    error_message = sqlc.arg(error_message),
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ListEntityIngestBatchesByStatus :many
SELECT
    id,
    organization_id,
    schema_id,
    entity_type,
    file_name,
    rows_staged,
    rows_flushed,
    skip_validation,
    status,
    error_message,
    enqueued_at,
    started_at,
    completed_at,
    updated_at
FROM entity_ingest_batches
WHERE status = ANY(sqlc.arg(statuses)::text[])
  AND (sqlc.narg(organization_id)::uuid IS NULL OR organization_id = sqlc.narg(organization_id))
ORDER BY enqueued_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: EntityIngestBatchStats :one
SELECT
    COUNT(*) AS total_batches,
    COUNT(*) FILTER (WHERE status IN ('PENDING', 'FLUSHING')) AS in_progress_batches,
    COUNT(*) FILTER (WHERE status = 'COMPLETED') AS completed_batches,
    COUNT(*) FILTER (WHERE status = 'FAILED') AS failed_batches,
    COALESCE(SUM(rows_staged), 0)::bigint AS total_rows_staged,
    COALESCE(SUM(rows_flushed), 0)::bigint AS total_rows_flushed
FROM entity_ingest_batches
WHERE (sqlc.narg(organization_id)::uuid IS NULL OR organization_id = sqlc.narg(organization_id));

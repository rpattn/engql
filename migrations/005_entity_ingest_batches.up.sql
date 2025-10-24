CREATE TABLE entity_ingest_batches (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    schema_id UUID NOT NULL REFERENCES entity_schemas(id) ON DELETE CASCADE,
    entity_type VARCHAR(255) NOT NULL,
    file_name TEXT,
    rows_staged INTEGER NOT NULL,
    rows_flushed INTEGER NOT NULL DEFAULT 0,
    skip_validation BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL,
    error_message TEXT,
    enqueued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT entity_ingest_batches_status_check CHECK (
        status IN ('PENDING', 'FLUSHING', 'COMPLETED', 'FAILED')
    )
);

CREATE INDEX entity_ingest_batches_status_idx
    ON entity_ingest_batches (status, enqueued_at DESC);

CREATE INDEX entity_ingest_batches_org_idx
    ON entity_ingest_batches (organization_id, enqueued_at DESC);

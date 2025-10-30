CREATE TABLE entity_export_jobs (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL,
    entity_type VARCHAR(255),
    transformation_id UUID REFERENCES entity_transformations(id) ON DELETE SET NULL,
    filters JSONB DEFAULT '[]'::jsonb NOT NULL,
    rows_requested INTEGER DEFAULT 0 NOT NULL,
    rows_exported INTEGER DEFAULT 0 NOT NULL,
    file_path TEXT,
    file_mime_type TEXT,
    file_byte_size BIGINT,
    status TEXT NOT NULL,
    error_message TEXT,
    enqueued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT entity_export_jobs_status_check CHECK (
        status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED')
    ),
    CONSTRAINT entity_export_jobs_type_check CHECK (
        job_type IN ('ENTITY_TYPE', 'TRANSFORMATION')
    )
);

CREATE TABLE entity_export_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    export_job_id UUID NOT NULL REFERENCES entity_export_jobs(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    row_identifier TEXT,
    error_message TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE INDEX entity_export_jobs_status_idx
    ON entity_export_jobs (status, enqueued_at DESC);

CREATE INDEX entity_export_jobs_org_idx
    ON entity_export_jobs (organization_id, enqueued_at DESC);

CREATE INDEX entity_export_logs_job_idx
    ON entity_export_logs (export_job_id);

CREATE INDEX entity_export_logs_org_idx
    ON entity_export_logs (organization_id);

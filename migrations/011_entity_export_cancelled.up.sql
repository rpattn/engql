ALTER TABLE entity_export_jobs
    DROP CONSTRAINT IF EXISTS entity_export_jobs_status_check;

ALTER TABLE entity_export_jobs
    ADD CONSTRAINT entity_export_jobs_status_check
    CHECK (
        status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')
    );

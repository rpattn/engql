-- Temporary staging table used for high-throughput COPY operations.
-- Unlogged to avoid WAL overhead and kept free of indexes/triggers.

CREATE UNLOGGED TABLE IF NOT EXISTS entities_ingest (
    organization_id UUID NOT NULL,
    schema_id UUID NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    path LTREE,
    properties JSONB NOT NULL DEFAULT '{}'::jsonb
);

COMMENT ON TABLE entities_ingest IS 'Staging table for COPY-based entity ingestion';

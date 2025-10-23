ALTER TABLE entities_ingest
    ADD COLUMN batch_id UUID NOT NULL DEFAULT uuid_generate_v4(),
    ADD COLUMN enqueued_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE entities_ingest
    ALTER COLUMN batch_id DROP DEFAULT;

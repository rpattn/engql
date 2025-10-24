ALTER TABLE entities_ingest
    DROP COLUMN IF EXISTS enqueued_at,
    DROP COLUMN IF EXISTS batch_id;

CREATE TABLE IF NOT EXISTS ingestion_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    schema_name VARCHAR(255) NOT NULL,
    file_name TEXT NOT NULL,
    row_number INT,
    error_message TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingestion_logs_org ON ingestion_logs (organization_id);
CREATE INDEX IF NOT EXISTS idx_ingestion_logs_org_schema ON ingestion_logs (organization_id, schema_name);

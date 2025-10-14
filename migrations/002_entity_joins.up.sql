CREATE TABLE IF NOT EXISTS entity_joins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    left_entity_type VARCHAR(255) NOT NULL,
    right_entity_type VARCHAR(255) NOT NULL,
    join_field VARCHAR(255) NOT NULL,
    join_field_type VARCHAR(64) NOT NULL,
    left_filters JSONB NOT NULL DEFAULT '[]',
    right_filters JSONB NOT NULL DEFAULT '[]',
    sort_criteria JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, name)
);

CREATE INDEX IF NOT EXISTS entity_joins_org_idx ON entity_joins (organization_id);
CREATE INDEX IF NOT EXISTS entity_joins_left_type_idx ON entity_joins (left_entity_type);
CREATE INDEX IF NOT EXISTS entity_joins_right_type_idx ON entity_joins (right_entity_type);

DROP TRIGGER IF EXISTS update_entity_joins_updated_at ON entity_joins;
CREATE TRIGGER update_entity_joins_updated_at
    BEFORE UPDATE ON entity_joins
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

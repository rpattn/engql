-- =============================================
--  Base schema for ENGQL
-- =============================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS ltree;

-- =============================================
--  FUNCTIONS
-- =============================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION bump_entity_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.version := COALESCE(OLD.version, 0) + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION archive_entity_change()
RETURNS TRIGGER AS $$
DECLARE
    audit_reason TEXT;
    change_kind TEXT := TG_OP;
BEGIN
    audit_reason := current_setting('app.reason', true);
    IF audit_reason IS NULL OR audit_reason = '' THEN
        audit_reason := TG_OP;
    ELSIF audit_reason ~* '^ROLLBACK' THEN
        change_kind := 'ROLLBACK';
    END IF;

    INSERT INTO entities_history (
        entity_id,
        organization_id,
        schema_id,
        entity_type,
        path,
        properties,
        created_at,
        updated_at,
        version,
        change_type,
        changed_at,
        reason
    )
    VALUES (
        OLD.id,
        OLD.organization_id,
        OLD.schema_id,
        OLD.entity_type,
        OLD.path,
        OLD.properties,
        OLD.created_at,
        OLD.updated_at,
        OLD.version,
        change_kind,
        NOW(),
        audit_reason
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION validate_entity_properties()
RETURNS TRIGGER AS $$
DECLARE
    schema_fields JSONB;
    field_def JSONB;
    field_value JSONB;
    i INT;
    field_name TEXT;
BEGIN
    SELECT fields INTO schema_fields
    FROM entity_schemas
    WHERE organization_id = NEW.organization_id
      AND name = NEW.entity_type;

    IF schema_fields IS NULL THEN
        RETURN NEW;
    END IF;

    FOR i IN 0 .. jsonb_array_length(schema_fields) - 1 LOOP
        field_def := schema_fields->i;
        field_name := field_def->>'name';
        field_value := NEW.properties->field_name;

        IF (field_def->>'required')::boolean AND (field_value IS NULL OR field_value = 'null') THEN
            RAISE EXCEPTION 'Required field % is missing or null', field_name;
        END IF;
    END LOOP;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================
--  TABLES
-- =============================================

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE entity_schemas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    fields JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    version TEXT DEFAULT '1.0.0' NOT NULL,
    previous_version_id UUID,
    status TEXT DEFAULT 'ACTIVE' NOT NULL,
    CONSTRAINT entity_schemas_status_check CHECK (status IN ('ACTIVE', 'DEPRECATED', 'ARCHIVED', 'DRAFT')),
    CONSTRAINT entity_schemas_version_format_check CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$')
);

CREATE TABLE entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    entity_type VARCHAR(255) NOT NULL,
    path LTREE,
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    schema_id UUID NOT NULL REFERENCES entity_schemas(id),
    version BIGINT DEFAULT 1 NOT NULL
);

CREATE TABLE entities_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    schema_id UUID NOT NULL REFERENCES entity_schemas(id) ON DELETE CASCADE,
    entity_type VARCHAR(255) NOT NULL,
    path LTREE,
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    version BIGINT NOT NULL,
    change_type TEXT NOT NULL CHECK (change_type IN ('UPDATE', 'DELETE', 'ROLLBACK')),
    changed_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    reason TEXT
);

CREATE TABLE entity_joins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    left_entity_type VARCHAR(255) NOT NULL,
    right_entity_type VARCHAR(255) NOT NULL,
    join_field VARCHAR(255),
    join_field_type VARCHAR(64),
    left_filters JSONB DEFAULT '[]'::jsonb NOT NULL,
    right_filters JSONB DEFAULT '[]'::jsonb NOT NULL,
    sort_criteria JSONB DEFAULT '[]'::jsonb NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    join_type VARCHAR(32) DEFAULT 'REFERENCE' NOT NULL,
    UNIQUE(organization_id, name)
);

CREATE TABLE ingestion_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    schema_name VARCHAR(255) NOT NULL,
    file_name TEXT NOT NULL,
    row_number INTEGER,
    error_message TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================
--  INDEXES
-- =============================================

CREATE UNIQUE INDEX entity_schemas_org_name_version_idx
    ON entity_schemas (organization_id, name, version);

CREATE UNIQUE INDEX entities_history_entity_version_idx
    ON entities_history (entity_id, version);

CREATE INDEX idx_entities_created_at ON entities (created_at);
CREATE INDEX idx_entities_org_type ON entities (organization_id, entity_type);
CREATE INDEX idx_entities_path ON entities USING GIST (path);
CREATE INDEX idx_entities_properties ON entities USING GIN (properties);
CREATE INDEX idx_entity_schemas_org_name ON entity_schemas (organization_id, name);
CREATE INDEX idx_organizations_name ON organizations (name);

-- =============================================
--  TRIGGERS
-- =============================================

CREATE TRIGGER update_organizations_updated_at
BEFORE UPDATE ON organizations
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_entity_schemas_updated_at
BEFORE UPDATE ON entity_schemas
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_entities_updated_at
BEFORE UPDATE ON entities
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER entities_version_bump
BEFORE UPDATE ON entities
FOR EACH ROW EXECUTE FUNCTION bump_entity_version();

CREATE TRIGGER entities_archive_change
BEFORE UPDATE OR DELETE ON entities
FOR EACH ROW EXECUTE FUNCTION archive_entity_change();

CREATE TRIGGER validate_entity_properties_trigger
BEFORE INSERT OR UPDATE ON entities
FOR EACH ROW EXECUTE FUNCTION validate_entity_properties();
-- =============================================
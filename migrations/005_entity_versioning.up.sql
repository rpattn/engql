BEGIN;

-- Extend entity_schemas with append-only metadata
ALTER TABLE entity_schemas
    ADD COLUMN IF NOT EXISTS version TEXT;

ALTER TABLE entity_schemas
    ALTER COLUMN version SET DEFAULT '1.0.0';

UPDATE entity_schemas SET version = COALESCE(version, '1.0.0');

ALTER TABLE entity_schemas
    ALTER COLUMN version SET NOT NULL;

ALTER TABLE entity_schemas
    ADD COLUMN IF NOT EXISTS previous_version_id UUID;

ALTER TABLE entity_schemas
    ADD COLUMN IF NOT EXISTS status TEXT;

ALTER TABLE entity_schemas
    ALTER COLUMN status SET DEFAULT 'ACTIVE';

UPDATE entity_schemas SET status = COALESCE(status, 'ACTIVE');

ALTER TABLE entity_schemas
    ALTER COLUMN status SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'entity_schemas_version_format_check'
    ) THEN
        ALTER TABLE entity_schemas
            ADD CONSTRAINT entity_schemas_version_format_check
            CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$');
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'entity_schemas_status_check'
    ) THEN
        ALTER TABLE entity_schemas
            ADD CONSTRAINT entity_schemas_status_check
            CHECK (status IN ('ACTIVE', 'DEPRECATED', 'ARCHIVED', 'DRAFT'));
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'entity_schemas_previous_version_fk'
    ) THEN
        ALTER TABLE entity_schemas
            ADD CONSTRAINT entity_schemas_previous_version_fk
            FOREIGN KEY (previous_version_id) REFERENCES entity_schemas(id);
    END IF;
END;
$$;

-- Allow multiple versions per schema name while keeping each version unique
ALTER TABLE entities
    DROP CONSTRAINT IF EXISTS entities_organization_id_entity_type_fkey;

ALTER TABLE entity_schemas
    DROP CONSTRAINT IF EXISTS entity_schemas_organization_id_name_key;

CREATE UNIQUE INDEX IF NOT EXISTS entity_schemas_org_name_version_idx
    ON entity_schemas (organization_id, name, version);

-- Ensure legacy rows have an explicit status
UPDATE entity_schemas SET status = 'ACTIVE' WHERE status IS NULL;

-- Add schema version reference and version tracking to entities
ALTER TABLE entities
    ADD COLUMN IF NOT EXISTS schema_id UUID;

ALTER TABLE entities
    ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 1;

UPDATE entities e
SET schema_id = s.id
FROM entity_schemas s
WHERE e.organization_id = s.organization_id
  AND e.entity_type = s.name
  AND e.schema_id IS NULL;

ALTER TABLE entities
    ALTER COLUMN schema_id SET NOT NULL;

UPDATE entities
SET version = COALESCE(version, 1);

ALTER TABLE entities
    ALTER COLUMN version SET DEFAULT 1;

ALTER TABLE entities
    ALTER COLUMN version SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'entities_schema_id_fkey'
    ) THEN
        ALTER TABLE entities
            ADD CONSTRAINT entities_schema_id_fkey
            FOREIGN KEY (schema_id) REFERENCES entity_schemas(id);
    END IF;
END;
$$;

-- Append-only history table for entity changes
CREATE TABLE IF NOT EXISTS entities_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    schema_id UUID NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    path LTREE,
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    version BIGINT NOT NULL,
    change_type TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT,
    CHECK (change_type IN ('UPDATE', 'DELETE', 'ROLLBACK'))
);

CREATE UNIQUE INDEX IF NOT EXISTS entities_history_entity_version_idx
    ON entities_history (entity_id, version);

ALTER TABLE entities_history
    DROP CONSTRAINT IF EXISTS entities_history_schema_id_fkey;

ALTER TABLE entities_history
    ADD CONSTRAINT entities_history_schema_id_fkey
    FOREIGN KEY (schema_id) REFERENCES entity_schemas(id) ON DELETE CASCADE;

-- Trigger functions to maintain version history
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

DROP TRIGGER IF EXISTS entities_version_bump ON entities;
CREATE TRIGGER entities_version_bump
BEFORE UPDATE ON entities
FOR EACH ROW
EXECUTE FUNCTION bump_entity_version();

DROP TRIGGER IF EXISTS entities_archive_change ON entities;
CREATE TRIGGER entities_archive_change
BEFORE UPDATE OR DELETE ON entities
FOR EACH ROW
EXECUTE FUNCTION archive_entity_change();

-- Validate entity properties against the referenced schema version
CREATE OR REPLACE FUNCTION validate_entity_properties()
RETURNS TRIGGER AS $$
DECLARE
    schema_fields JSONB;
    field_def JSONB;
    field_value JSONB;
    i INT;
    field_name TEXT;
BEGIN
    SELECT name, fields INTO NEW.entity_type, schema_fields
    FROM entity_schemas
    WHERE id = NEW.schema_id;

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

COMMIT;

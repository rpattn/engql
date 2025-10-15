BEGIN;

-- Restore original validate_entity_properties implementation
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

DROP TRIGGER IF EXISTS entities_archive_change ON entities;
DROP TRIGGER IF EXISTS entities_version_bump ON entities;

DROP FUNCTION IF EXISTS archive_entity_change();
DROP FUNCTION IF EXISTS bump_entity_version();

DROP TABLE IF EXISTS entities_history;

ALTER TABLE entities
    DROP CONSTRAINT IF EXISTS entities_schema_id_fkey;

ALTER TABLE entities
    DROP COLUMN IF EXISTS schema_id,
    DROP COLUMN IF EXISTS version;

ALTER TABLE entities
    ADD CONSTRAINT entities_organization_id_entity_type_fkey
    FOREIGN KEY (organization_id, entity_type) REFERENCES entity_schemas(organization_id, name);

DROP INDEX IF EXISTS entity_schemas_org_name_version_idx;

ALTER TABLE entity_schemas
    DROP CONSTRAINT IF EXISTS entity_schemas_previous_version_fk,
    DROP CONSTRAINT IF EXISTS entity_schemas_status_check,
    DROP CONSTRAINT IF EXISTS entity_schemas_version_format_check;

ALTER TABLE entity_schemas
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS previous_version_id,
    DROP COLUMN IF EXISTS status;

ALTER TABLE entity_schemas
    ADD CONSTRAINT entity_schemas_organization_id_name_key
    UNIQUE (organization_id, name);

COMMIT;

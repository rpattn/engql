-- Restore original entity property validation trigger implementation.

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

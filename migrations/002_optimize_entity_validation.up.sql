-- Optimize entity property validation for high-volume COPY operations.
-- Allows skipping validation when the application has already enforced it
-- and avoids repeated lookups by schema name.

CREATE OR REPLACE FUNCTION validate_entity_properties()
RETURNS TRIGGER AS $$
DECLARE
    schema_fields JSONB;
    field_rec RECORD;
    field_name TEXT;
    skip_setting TEXT;
BEGIN
    skip_setting := current_setting('app.skip_entity_property_validation', true);
    IF skip_setting IS NOT NULL THEN
        skip_setting := lower(skip_setting);
        IF skip_setting IN ('1', 'on', 'true', 't', 'yes') THEN
            RETURN NEW;
        END IF;
    END IF;

    SELECT fields
      INTO schema_fields
      FROM entity_schemas
     WHERE id = NEW.schema_id;

    IF schema_fields IS NULL OR jsonb_typeof(schema_fields) <> 'array' THEN
        RETURN NEW;
    END IF;

    FOR field_rec IN
        SELECT value
          FROM jsonb_array_elements(schema_fields) AS t(value)
         WHERE COALESCE((value->>'required')::boolean, false)
    LOOP
        field_name := field_rec.value->>'name';

        IF field_name IS NULL OR field_name = '' THEN
            CONTINUE;
        END IF;

        IF NOT (NEW.properties ? field_name) THEN
            RAISE EXCEPTION 'Required field % is missing or null', field_name;
        END IF;

        IF NEW.properties->field_name IS NULL OR NEW.properties->>field_name = 'null' THEN
            RAISE EXCEPTION 'Required field % is missing or null', field_name;
        END IF;
    END LOOP;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

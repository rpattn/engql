-- Ensure unique reference values per schema by creating dynamic indexes.
CREATE OR REPLACE FUNCTION refresh_entity_reference_indexes()
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    schema_rec RECORD;
    index_name TEXT;
    required_indexes TEXT[] := ARRAY[]::TEXT[];
BEGIN
    FOR schema_rec IN
        SELECT es.id AS schema_id,
               es.name AS schema_name,
               (field ->> 'name') AS reference_field
        FROM entity_schemas es
        CROSS JOIN LATERAL jsonb_array_elements(es.fields) AS field
        WHERE jsonb_typeof(es.fields) = 'array'
          AND upper(field ->> 'type') = 'REFERENCE'
          AND (field ->> 'name') IS NOT NULL
    LOOP
        index_name := format('entities_ref_idx_%s', replace(schema_rec.schema_id::TEXT, '-', '_'));
        required_indexes := array_append(required_indexes, index_name);

        EXECUTE format(
            'CREATE UNIQUE INDEX IF NOT EXISTS %I ON entities (organization_id, entity_type, (properties ->> %L)) '
            || 'WHERE entity_type = %L AND properties ? %L',
            index_name,
            schema_rec.reference_field,
            schema_rec.schema_name,
            schema_rec.reference_field
        );
    END LOOP;

    FOR schema_rec IN
        SELECT indexname
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND indexname LIKE 'entities_ref_idx_%'
    LOOP
        IF NOT schema_rec.indexname = ANY(required_indexes) THEN
            EXECUTE format('DROP INDEX IF EXISTS %I', schema_rec.indexname);
        END IF;
    END LOOP;
END;
$$;

CREATE OR REPLACE FUNCTION trigger_refresh_entity_reference_indexes()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM refresh_entity_reference_indexes();
    RETURN NULL;
END;
$$;

CREATE TRIGGER entity_schemas_refresh_reference_indexes
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON entity_schemas
FOR EACH STATEMENT EXECUTE FUNCTION trigger_refresh_entity_reference_indexes();

SELECT refresh_entity_reference_indexes();

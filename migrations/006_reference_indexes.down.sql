DROP TRIGGER IF EXISTS entity_schemas_refresh_reference_indexes ON entity_schemas;
DROP FUNCTION IF EXISTS trigger_refresh_entity_reference_indexes();

DO $$
DECLARE
    idx RECORD;
BEGIN
    FOR idx IN
        SELECT indexname
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND indexname LIKE 'entities_ref_idx_%'
    LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', idx.indexname);
    END LOOP;
END;
$$;

DROP FUNCTION IF EXISTS refresh_entity_reference_indexes();

ALTER TABLE entity_export_jobs
    ADD COLUMN transformation_definition JSONB,
    ADD COLUMN transformation_options JSONB;

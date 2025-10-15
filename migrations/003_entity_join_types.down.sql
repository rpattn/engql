ALTER TABLE entity_joins
    ALTER COLUMN join_field SET NOT NULL;

ALTER TABLE entity_joins
    ALTER COLUMN join_field_type SET NOT NULL;

ALTER TABLE entity_joins
    DROP COLUMN join_type;

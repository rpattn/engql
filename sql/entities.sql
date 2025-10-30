-- name: CreateEntity :one
INSERT INTO entities (organization_id, schema_id, entity_type, path, properties)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at;

-- name: GetEntity :one
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE id = $1;

-- name: ListEntities :many
SELECT
    id,
    organization_id,
    schema_id,
    entity_type,
    path,
    properties,
    version,
    created_at,
    updated_at,
    COUNT(*) OVER() AS total_count
FROM entities
WHERE organization_id = sqlc.arg(organization_id)
  AND (
        sqlc.arg(entity_type)::text = ''
        OR entity_type = sqlc.arg(entity_type)::text
    )
  AND (
        COALESCE(array_length(sqlc.arg(property_keys)::text[], 1), 0) = 0
        OR (
            SELECT bool_and(COALESCE((properties ->> filters.key) ILIKE filters.value, false))
            FROM (
                SELECT keys.key, values.value
                FROM unnest(COALESCE(sqlc.arg(property_keys)::text[], ARRAY[]::text[])) WITH ORDINALITY AS keys(key, ord)
                JOIN unnest(COALESCE(sqlc.arg(property_values)::text[], ARRAY[]::text[])) WITH ORDINALITY AS values(value, ord)
                  ON keys.ord = values.ord
            ) AS filters
        )
    )
  AND (
        sqlc.arg(text_search)::text = ''
        OR entity_type ILIKE sqlc.arg(text_search)::text
        OR path::text ILIKE sqlc.arg(text_search)::text
        OR properties::text ILIKE sqlc.arg(text_search)::text
    )
ORDER BY
    CASE
        WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_direction)::text = 'asc'
            THEN created_at
    END ASC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_direction)::text = 'desc'
            THEN created_at
    END DESC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'updated_at' AND sqlc.arg(sort_direction)::text = 'asc'
            THEN updated_at
    END ASC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'updated_at' AND sqlc.arg(sort_direction)::text = 'desc'
            THEN updated_at
    END DESC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'entity_type' AND sqlc.arg(sort_direction)::text = 'asc'
            THEN LOWER(entity_type)
    END ASC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'entity_type' AND sqlc.arg(sort_direction)::text = 'desc'
            THEN LOWER(entity_type)
    END DESC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'path' AND sqlc.arg(sort_direction)::text = 'asc'
            THEN path::text
    END ASC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'path' AND sqlc.arg(sort_direction)::text = 'desc'
            THEN path::text
    END DESC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'version' AND sqlc.arg(sort_direction)::text = 'asc'
            THEN version
    END ASC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'version' AND sqlc.arg(sort_direction)::text = 'desc'
            THEN version
    END DESC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'property' AND sqlc.arg(sort_direction)::text = 'asc'
            THEN LOWER(COALESCE(properties ->> sqlc.arg(sort_property)::text, ''))
    END ASC,
    CASE
        WHEN sqlc.arg(sort_field)::text = 'property' AND sqlc.arg(sort_direction)::text = 'desc'
            THEN LOWER(COALESCE(properties ->> sqlc.arg(sort_property)::text, ''))
    END DESC,
    created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListEntitiesByType :many
SELECT
    id,
    organization_id,
    schema_id,
    entity_type,
    path,
    properties,
    version,
    created_at,
    updated_at
FROM entities
WHERE organization_id = sqlc.arg(organization_id)
  AND entity_type = sqlc.arg(entity_type)
ORDER BY created_at DESC;

-- name: GetEntityByReference :one
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE organization_id = $1
  AND entity_type = $2
  AND properties ->> sqlc.arg(field_name)::text = sqlc.arg(reference_value)::text
LIMIT 1;

-- name: ListEntitiesByReferences :many
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE organization_id = $1
  AND entity_type = $2
  AND properties ->> sqlc.arg(field_name)::text = ANY(sqlc.arg(reference_values)::text[]);

-- name: UpdateEntity :one
UPDATE entities
SET schema_id = $2, entity_type = $3, path = $4, properties = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at;

-- name: DeleteEntity :exec
DELETE FROM entities
WHERE id = $1;

-- name: GetEntityAncestors :many
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE organization_id = $1
  AND path @> $2::ltree
  AND path <> $2::ltree
ORDER BY nlevel(path);

-- name: GetEntityDescendants :many
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE organization_id = $1 AND path ~ ($2 || '.*')::lquery;

-- name: GetEntityChildren :many
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE organization_id = $1 AND path ~ ($2 || '.*{1}')::lquery;

-- name: GetEntitySiblings :many
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE organization_id = $1 AND path ~ ($2 || '.*{1}')::lquery;

-- name: FilterEntitiesByProperty :many
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE organization_id = $1 
AND properties @> $2;

-- name: GetEntityCount :one
SELECT COUNT(*)
FROM entities
WHERE organization_id = $1;

-- name: GetEntityCountByType :one
SELECT COUNT(*)
FROM entities
WHERE organization_id = $1 AND entity_type = $2;

-- name: GetEntitiesByIDs :many
SELECT id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at
FROM entities
WHERE id = ANY(@ids::uuid[]);

-- name: GetEntityHistoryByVersion :one
SELECT id, entity_id, organization_id, schema_id, entity_type, path, properties, created_at, updated_at, version, change_type, changed_at, reason
FROM entities_history
WHERE entity_id = $1 AND version = $2;

-- name: ListEntityHistory :many
SELECT id, entity_id, organization_id, schema_id, entity_type, path, properties, created_at, updated_at, version, change_type, changed_at, reason
FROM entities_history
WHERE entity_id = $1
ORDER BY version DESC;

-- name: GetMaxEntityHistoryVersion :one
SELECT COALESCE(MAX(version), 0)::BIGINT
FROM entities_history
WHERE entity_id = $1;

-- name: UpsertEntityFromHistory :exec
INSERT INTO entities (id, organization_id, schema_id, entity_type, path, properties, version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
ON CONFLICT (id) DO UPDATE
SET schema_id = EXCLUDED.schema_id,
    entity_type = EXCLUDED.entity_type,
    path = EXCLUDED.path,
    properties = EXCLUDED.properties,
    updated_at = NOW();

-- name: InsertEntityHistoryRecord :exec
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
    reason
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);


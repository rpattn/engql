-- name: CreateEntity :one
INSERT INTO entities (organization_id, entity_type, path, properties)
VALUES ($1, $2, $3, $4)
RETURNING id, organization_id, entity_type, path, properties, created_at, updated_at;

-- name: GetEntity :one
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at
FROM entities
WHERE id = $1;

-- name: ListEntities :many
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at,
       COUNT(*) OVER() AS total_count
FROM entities
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListEntitiesByType :many
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at
FROM entities
WHERE organization_id = $1 AND entity_type = $2
ORDER BY created_at DESC;

-- name: UpdateEntity :one
UPDATE entities
SET entity_type = $2, path = $3, properties = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, organization_id, entity_type, path, properties, created_at, updated_at;

-- name: DeleteEntity :exec
DELETE FROM entities
WHERE id = $1;

-- name: GetEntityAncestors :many
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at
FROM entities
WHERE organization_id = $1
  AND path @> $2::ltree
  AND path <> $2::ltree
ORDER BY nlevel(path);

-- name: GetEntityDescendants :many
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at
FROM entities
WHERE organization_id = $1 AND path ~ ($2 || '.*')::lquery;

-- name: GetEntityChildren :many
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at
FROM entities
WHERE organization_id = $1 AND path ~ ($2 || '.*{1}')::lquery;

-- name: GetEntitySiblings :many
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at
FROM entities
WHERE organization_id = $1 AND path ~ ($2 || '.*{1}')::lquery;

-- name: FilterEntitiesByProperty :many
SELECT id, organization_id, entity_type, path, properties, created_at, updated_at
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

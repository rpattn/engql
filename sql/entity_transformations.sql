-- name: CreateEntityTransformation :one
INSERT INTO entity_transformations (
    id,
    organization_id,
    name,
    description,
    nodes
)
VALUES ($1, $2, $3, $4, $5)
RETURNING
    id,
    organization_id,
    name,
    description,
    nodes,
    created_at,
    updated_at;

-- name: GetEntityTransformation :one
SELECT
    id,
    organization_id,
    name,
    description,
    nodes,
    created_at,
    updated_at
FROM entity_transformations
WHERE id = $1;

-- name: ListEntityTransformationsByOrganization :many
SELECT
    id,
    organization_id,
    name,
    description,
    nodes,
    created_at,
    updated_at
FROM entity_transformations
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: UpdateEntityTransformation :one
UPDATE entity_transformations
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    nodes = COALESCE($4, nodes),
    updated_at = NOW()
WHERE id = $1
RETURNING
    id,
    organization_id,
    name,
    description,
    nodes,
    created_at,
    updated_at;

-- name: DeleteEntityTransformation :exec
DELETE FROM entity_transformations
WHERE id = $1;

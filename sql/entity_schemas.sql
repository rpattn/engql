-- name: CreateEntitySchema :one
INSERT INTO entity_schemas (organization_id, name, description, fields)
VALUES ($1, $2, $3, $4)
RETURNING id, organization_id, name, description, fields, created_at, updated_at;

-- name: GetEntitySchema :one
SELECT id, organization_id, name, description, fields, created_at, updated_at
FROM entity_schemas
WHERE id = $1;

-- name: GetEntitySchemaByName :one
SELECT id, organization_id, name, description, fields, created_at, updated_at
FROM entity_schemas
WHERE organization_id = $1 AND name = $2;

-- name: ListEntitySchemas :many
SELECT id, organization_id, name, description, fields, created_at, updated_at
FROM entity_schemas
WHERE organization_id = $1
ORDER BY name;

-- name: UpdateEntitySchema :one
UPDATE entity_schemas
SET name = $2, description = $3, fields = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, organization_id, name, description, fields, created_at, updated_at;

-- name: DeleteEntitySchema :exec
DELETE FROM entity_schemas
WHERE id = $1;

-- name: SchemaExists :one
SELECT EXISTS(
    SELECT 1 FROM entity_schemas
    WHERE organization_id = $1 AND name = $2
);

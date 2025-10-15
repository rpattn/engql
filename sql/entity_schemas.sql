-- name: CreateEntitySchema :one
INSERT INTO entity_schemas (
    organization_id,
    name,
    description,
    fields,
    version,
    previous_version_id,
    status
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, organization_id, name, description, fields, version, previous_version_id, status, created_at, updated_at;

-- name: GetEntitySchema :one
SELECT id, organization_id, name, description, fields, version, previous_version_id, status, created_at, updated_at
FROM entity_schemas
WHERE id = $1;

-- name: GetEntitySchemaByName :one
SELECT id, organization_id, name, description, fields, version, previous_version_id, status, created_at, updated_at
FROM entity_schemas
WHERE organization_id = $1 AND name = $2 AND status <> 'ARCHIVED'
ORDER BY created_at DESC
LIMIT 1;

-- name: ListEntitySchemas :many
SELECT DISTINCT ON (organization_id, name)
    id,
    organization_id,
    name,
    description,
    fields,
    version,
    previous_version_id,
    status,
    created_at,
    updated_at
FROM entity_schemas
WHERE organization_id = $1 AND status <> 'ARCHIVED'
ORDER BY organization_id, name, created_at DESC;

-- name: ListEntitySchemaVersions :many
SELECT id, organization_id, name, description, fields, version, previous_version_id, status, created_at, updated_at
FROM entity_schemas
WHERE organization_id = $1 AND name = $2
ORDER BY created_at DESC;

-- name: GetEntitySchemaVersionByNumber :one
SELECT id, organization_id, name, description, fields, version, previous_version_id, status, created_at, updated_at
FROM entity_schemas
WHERE organization_id = $1 AND name = $2 AND version = $3
ORDER BY created_at DESC
LIMIT 1;

-- name: SchemaExists :one
SELECT EXISTS(
    SELECT 1 FROM entity_schemas
    WHERE organization_id = $1 AND name = $2
);

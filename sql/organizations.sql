-- name: CreateOrganization :one
INSERT INTO organizations (name, description)
VALUES ($1, $2)
RETURNING id, name, description, created_at, updated_at;

-- name: GetOrganization :one
SELECT id, name, description, created_at, updated_at
FROM organizations
WHERE id = $1;

-- name: GetOrganizationByName :one
SELECT id, name, description, created_at, updated_at
FROM organizations
WHERE name = $1;

-- name: ListOrganizations :many
SELECT id, name, description, created_at, updated_at
FROM organizations
ORDER BY name;

-- name: UpdateOrganization :one
UPDATE organizations
SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, created_at, updated_at;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;

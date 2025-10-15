-- name: CreateEntityJoin :one
INSERT INTO entity_joins (
    organization_id,
    name,
    description,
    left_entity_type,
    right_entity_type,
    join_field,
    join_field_type,
    join_type,
    left_filters,
    right_filters,
    sort_criteria
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING
    id,
    organization_id,
    name,
    description,
    left_entity_type,
    right_entity_type,
    join_field,
    join_field_type,
    join_type,
    left_filters,
    right_filters,
    sort_criteria,
    created_at,
    updated_at;

-- name: GetEntityJoin :one
SELECT
    id,
    organization_id,
    name,
    description,
    left_entity_type,
    right_entity_type,
    join_field,
    join_field_type,
    join_type,
    left_filters,
    right_filters,
    sort_criteria,
    created_at,
    updated_at
FROM entity_joins
WHERE id = $1;

-- name: ListEntityJoinsByOrganization :many
SELECT
    id,
    organization_id,
    name,
    description,
    left_entity_type,
    right_entity_type,
    join_field,
    join_field_type,
    join_type,
    left_filters,
    right_filters,
    sort_criteria,
    created_at,
    updated_at
FROM entity_joins
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: UpdateEntityJoin :one
UPDATE entity_joins
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    left_entity_type = COALESCE($4, left_entity_type),
    right_entity_type = COALESCE($5, right_entity_type),
    join_field = COALESCE($6, join_field),
    join_field_type = COALESCE($7, join_field_type),
    join_type = COALESCE($8, join_type),
    left_filters = COALESCE($9, left_filters),
    right_filters = COALESCE($10, right_filters),
    sort_criteria = COALESCE($11, sort_criteria),
    updated_at = NOW()
WHERE id = $1
RETURNING
    id,
    organization_id,
    name,
    description,
    left_entity_type,
    right_entity_type,
    join_field,
    join_field_type,
    join_type,
    left_filters,
    right_filters,
    sort_criteria,
    created_at,
    updated_at;

-- name: DeleteEntityJoin :exec
DELETE FROM entity_joins
WHERE id = $1;

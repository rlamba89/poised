-- Option lists are scoped to a hospital (OPT-07, TEN-01).

-- name: ListOptionLists :many
SELECT l.id, l.name, l.options, l.updated_at, u.name AS updated_by_name
FROM option_lists l
JOIN users u ON u.id = l.updated_by
WHERE l.hospital_id = $1
ORDER BY lower(l.name);

-- name: CreateOptionList :one
INSERT INTO option_lists (hospital_id, name, options, updated_by)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateOptionList :execrows
UPDATE option_lists
SET name = $3, options = $4, updated_by = $5, updated_at = now()
WHERE id = $1 AND hospital_id = $2;

-- name: DeleteOptionList :execrows
DELETE FROM option_lists WHERE id = $1 AND hospital_id = $2;

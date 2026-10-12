-- Option lists (OPT-07) belong to a trust: hospital_id NULL is the whole trust's, as every list
-- made so far is; otherwise only that hospital sees it. Names are unique in the trust.

-- name: ListOptionLists :many
SELECT l.id, l.name, l.options, l.updated_at, u.name AS updated_by_name
FROM option_lists l
JOIN users u ON u.id = l.updated_by
WHERE l.org_id = sqlc.arg(org_id) AND (l.hospital_id IS NULL OR l.hospital_id = sqlc.arg(hospital_id)::uuid)
ORDER BY lower(l.name);

-- name: CreateOptionList :one
INSERT INTO option_lists (org_id, name, options, updated_by)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateOptionList :execrows
UPDATE option_lists
SET name = sqlc.arg(name), options = sqlc.arg(options), updated_by = sqlc.arg(updated_by), updated_at = now()
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
  AND (hospital_id IS NULL OR hospital_id = sqlc.arg(hospital_id)::uuid);

-- name: DeleteOptionList :execrows
DELETE FROM option_lists
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
  AND (hospital_id IS NULL OR hospital_id = sqlc.arg(hospital_id)::uuid);

-- name: GetOptionListOwner :one
-- Whose the list is: NULL for the trust's, else the hospital whose own it is.
SELECT hospital_id FROM option_lists
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
  AND (hospital_id IS NULL OR hospital_id = sqlc.arg(hospital_id)::uuid);

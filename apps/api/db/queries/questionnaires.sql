-- The slice has one version per questionnaire, so "latest version" is that draft.

-- name: ListQuestionnaires :many
SELECT q.id, q.name, q.description, q.created_by, q.created_at,
       creator.name AS created_by_name,
       v.version_no, v.status, v.updated_at,
       editor.name AS updated_by_name,
       count(*) OVER () AS total
FROM questionnaires q
JOIN users creator ON creator.id = q.created_by
JOIN LATERAL (
    SELECT * FROM questionnaire_versions qv
    WHERE qv.questionnaire_id = q.id
    ORDER BY qv.version_no DESC LIMIT 1
) v ON true
JOIN users editor ON editor.id = v.updated_by
WHERE q.hospital_id = sqlc.arg(hospital_id)
  AND (sqlc.arg(search)::text = ''
       OR q.name ILIKE '%' || sqlc.arg(search) || '%'
       OR q.description ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY v.updated_at DESC, q.name
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: LockHospitalQuestionnaires :exec
-- Serialises creates in one hospital so the unique-draft-name check can't race.
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg(hospital_id)::uuid::text));

-- name: DraftNameExists :one
SELECT EXISTS (
    SELECT 1 FROM questionnaires q
    JOIN questionnaire_versions v ON v.questionnaire_id = q.id
    WHERE q.hospital_id = sqlc.arg(hospital_id)
      AND v.status = 'draft'
      AND lower(q.name) = lower(sqlc.arg(name))
      AND q.id <> sqlc.arg(exclude_id)
);

-- name: CreateQuestionnaire :one
INSERT INTO questionnaires (hospital_id, name, description, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: CreateVersion :one
INSERT INTO questionnaire_versions (questionnaire_id, version_no, status, updated_by)
VALUES ($1, $2, 'draft', $3)
RETURNING id;

-- name: GetQuestionnaire :one
SELECT q.id, q.hospital_id, q.name, q.description, q.created_by, q.created_at,
       v.id AS version_id, v.version_no, v.status, v.updated_at,
       editor.name AS updated_by_name
FROM questionnaires q
JOIN LATERAL (
    SELECT * FROM questionnaire_versions qv
    WHERE qv.questionnaire_id = q.id
    ORDER BY qv.version_no DESC LIMIT 1
) v ON true
JOIN users editor ON editor.id = v.updated_by
WHERE q.id = $1 AND q.hospital_id = $2;

-- name: HasNonDraftVersion :one
SELECT EXISTS (
    SELECT 1 FROM questionnaire_versions WHERE questionnaire_id = $1 AND status <> 'draft'
);

-- name: DeleteQuestionnaire :exec
DELETE FROM questionnaires WHERE id = $1 AND hospital_id = $2;

-- name: TouchVersion :exec
UPDATE questionnaire_versions SET updated_by = $2, updated_at = now() WHERE id = $1;

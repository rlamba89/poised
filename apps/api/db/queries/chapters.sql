-- Every chapter lookup joins up to the questionnaire, so it is scoped to the trust and hospital.

-- name: ListChapters :many
SELECT c.id, c.position, c.name, c.description, c.icon, c.audience, c.revision, c.updated_at
FROM chapters c
WHERE c.version_id = $1
ORDER BY c.position, c.name;

-- name: GetChapterMeta :one
SELECT c.id, c.version_id, c.name, c.description, c.icon, c.audience, c.revision,
       v.status AS version_status, v.questionnaire_id, q.name AS questionnaire_name,
       q.hospital_id AS owner_hospital_id
FROM chapters c
JOIN questionnaire_versions v ON v.id = c.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE c.id = sqlc.arg(id) AND q.org_id = sqlc.arg(org_id)
  AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid);

-- name: GetChapter :one
SELECT c.id, c.version_id, c.position, c.name, c.description, c.icon, c.audience,
       c.content, c.revision, c.updated_at,
       v.status AS version_status, v.questionnaire_id, q.name AS questionnaire_name,
       q.hospital_id AS owner_hospital_id
FROM chapters c
JOIN questionnaire_versions v ON v.id = c.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE c.id = sqlc.arg(id) AND q.org_id = sqlc.arg(org_id)
  AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid);

-- name: CreateChapter :one
INSERT INTO chapters (version_id, position, name, updated_by)
VALUES ($1, (SELECT coalesce(max(position), 0) + 1 FROM chapters WHERE version_id = $1), $2, $3)
RETURNING id;

-- name: UpdateChapterMeta :exec
UPDATE chapters
SET name = $2, description = $3, icon = $4, audience = $5, updated_by = $6, updated_at = now()
WHERE id = $1;

-- name: SetChapterPosition :exec
UPDATE chapters SET position = $3 WHERE id = $1 AND version_id = $2;

-- name: DeleteChapter :exec
DELETE FROM chapters WHERE id = $1;

-- name: SaveChapterContent :one
-- Saves only if nobody saved since the caller loaded `revision` (LCY-04).
UPDATE chapters
SET content = sqlc.arg(content), revision = revision + 1, updated_by = sqlc.arg(updated_by), updated_at = now()
WHERE id = sqlc.arg(id) AND revision = sqlc.arg(revision)
RETURNING revision;

-- name: ListVersionContents :many
-- A version's Question Sets with their content, for the key checks (S23).
SELECT c.name, c.audience, c.content
FROM chapters c
WHERE c.version_id = $1
ORDER BY c.position, c.name;

-- name: ListTrustPublishedContents :many
-- Every Question Set of the trust's published and retired versions (S23): where each question
-- key is used, and its options as published. Newest version first within each questionnaire.
SELECT q.id AS questionnaire_id, q.name AS questionnaire_name, v.id AS version_id, v.version_no,
       c.name, c.audience, c.content
FROM chapters c
JOIN questionnaire_versions v ON v.id = c.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE q.org_id = $1 AND v.status IN ('published', 'retired')
ORDER BY q.name, q.id, v.version_no DESC, c.position, c.name;

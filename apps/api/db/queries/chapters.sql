-- Every chapter lookup joins up to the questionnaire so it is scoped to the hospital.

-- name: ListChapters :many
SELECT c.id, c.position, c.name, c.description, c.icon, c.audience, c.revision, c.updated_at
FROM chapters c
WHERE c.version_id = $1
ORDER BY c.position, c.name;

-- name: GetChapterMeta :one
SELECT c.id, c.version_id, c.name, c.description, c.icon, c.audience, c.revision,
       v.status AS version_status, v.questionnaire_id, q.name AS questionnaire_name
FROM chapters c
JOIN questionnaire_versions v ON v.id = c.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE c.id = $1 AND q.hospital_id = $2;

-- name: GetChapter :one
SELECT c.id, c.version_id, c.position, c.name, c.description, c.icon, c.audience,
       c.content, c.revision, c.updated_at,
       v.status AS version_status, v.questionnaire_id, q.name AS questionnaire_name
FROM chapters c
JOIN questionnaire_versions v ON v.id = c.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE c.id = $1 AND q.hospital_id = $2;

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

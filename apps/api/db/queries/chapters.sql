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
WHERE c.id = sqlc.arg(id) AND q.org_id = sqlc.arg(org_id)::uuid
  AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid);

-- name: GetChapter :one
SELECT c.id, c.version_id, c.position, c.name, c.description, c.icon, c.audience,
       c.content, c.revision, c.updated_at,
       v.status AS version_status, v.questionnaire_id, q.name AS questionnaire_name,
       q.hospital_id AS owner_hospital_id
FROM chapters c
JOIN questionnaire_versions v ON v.id = c.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE c.id = sqlc.arg(id) AND q.org_id = sqlc.arg(org_id)::uuid
  AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid);

-- Every change to a version's Question Sets is made only into a draft, holding the version's row
-- FOR SHARE (S44a, verification 1): a change waits for a Publish in progress (which holds it FOR
-- UPDATE) and is then refused (no row), so what Publish checks is what it publishes.

-- name: CreateChapter :one
INSERT INTO chapters (version_id, position, name, updated_by)
SELECT v.id, (SELECT coalesce(max(position), 0) + 1 FROM chapters WHERE version_id = v.id), sqlc.arg(name), sqlc.arg(updated_by)::uuid
FROM questionnaire_versions v
WHERE v.id = sqlc.arg(version_id) AND v.status = 'draft'
FOR SHARE OF v
RETURNING id;

-- name: UpdateChapterMeta :execrows
WITH draft AS (
    SELECT v.id FROM questionnaire_versions v JOIN chapters c ON c.version_id = v.id
    WHERE c.id = sqlc.arg(id) AND v.status = 'draft'
    FOR SHARE OF v
)
UPDATE chapters ch
SET name = sqlc.arg(name), description = sqlc.arg(description), icon = sqlc.arg(icon), audience = sqlc.arg(audience),
    updated_by = sqlc.arg(updated_by)::uuid, updated_at = now()
WHERE ch.id = sqlc.arg(id) AND ch.version_id IN (SELECT draft.id FROM draft);

-- name: LockDraftVersion :one
-- Taken in a transaction that changes several of a draft's Question Sets (the new order): no row
-- once it isn't a draft. NO KEY UPDATE, not SHARE (S44a, verification 2): the transaction then
-- updates this row (TouchVersion), and upgrading a SHARE lock while a content save holds SHARE and
-- waits for a set this transaction moved is a deadlock. It still conflicts with Publish's FOR UPDATE.
SELECT id FROM questionnaire_versions WHERE id = $1 AND status = 'draft' FOR NO KEY UPDATE;

-- name: SetChapterPosition :exec
UPDATE chapters SET position = $3 WHERE id = $1 AND version_id = $2;

-- name: DeleteChapter :execrows
WITH draft AS (
    SELECT v.id FROM questionnaire_versions v JOIN chapters c ON c.version_id = v.id
    WHERE c.id = sqlc.arg(id) AND v.status = 'draft'
    FOR SHARE OF v
)
DELETE FROM chapters ch WHERE ch.id = sqlc.arg(id) AND ch.version_id IN (SELECT draft.id FROM draft);

-- name: SaveChapterContent :one
-- Saves only if nobody saved since the caller loaded `revision` (LCY-04), and only into a draft:
-- the version's row is taken FOR SHARE (S44a), so a save waits for a Publish in progress (which
-- holds it FOR UPDATE) and is then refused, and Publish waits for a save in progress.
WITH draft AS (
    SELECT v.id FROM questionnaire_versions v JOIN chapters c ON c.version_id = v.id
    WHERE c.id = sqlc.arg(id) AND v.status = 'draft'
    FOR SHARE OF v
)
UPDATE chapters ch
SET content = sqlc.arg(content), revision = ch.revision + 1, updated_by = sqlc.arg(updated_by)::uuid, updated_at = now()
WHERE ch.id = sqlc.arg(id) AND ch.revision = sqlc.arg(revision) AND ch.version_id IN (SELECT draft.id FROM draft)
RETURNING ch.revision;

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
WHERE q.org_id = $1::uuid AND v.status IN ('published', 'retired')
ORDER BY q.name, q.id, v.version_no DESC, c.position, c.name;

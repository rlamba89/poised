-- A questionnaire shows its latest version: the draft if there is one, else the last published.
-- Questionnaires belong to a trust (org_id). hospital_id NULL means every hospital of the trust
-- uses it; otherwise only that hospital sees it. Each lookup is scoped to the trust and hospital.

-- name: ListQuestionnaires :many
SELECT q.id, q.hospital_id, q.name, q.description, q.created_by, q.created_at,
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
WHERE q.org_id = sqlc.arg(org_id) AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid)
  AND (sqlc.arg(search)::text = ''
       OR q.name ILIKE '%' || sqlc.arg(search) || '%'
       OR q.description ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY v.updated_at DESC, q.name
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: LockOrgQuestionnaires :exec
-- Serialises creates in one trust so the unique-draft-name check can't race.
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg(org_id)::uuid::text));

-- name: DraftNameExists :one
SELECT EXISTS (
    SELECT 1 FROM questionnaires q
    JOIN questionnaire_versions v ON v.questionnaire_id = q.id
    WHERE q.org_id = sqlc.arg(org_id)
      AND v.status = 'draft'
      AND lower(q.name) = lower(sqlc.arg(name))
      AND q.id <> sqlc.arg(exclude_id)
);

-- name: CreateQuestionnaire :one
-- A new questionnaire is the trust's, for all its hospitals.
INSERT INTO questionnaires (org_id, name, description, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: CreateVersion :one
INSERT INTO questionnaire_versions (questionnaire_id, version_no, status, updated_by)
VALUES ($1, $2, 'draft', $3)
RETURNING id;

-- name: GetQuestionnaire :one
SELECT q.id, q.org_id, q.hospital_id, q.name, q.description, q.created_by, q.created_at,
       v.id AS version_id, v.version_no, v.status, v.updated_at,
       editor.name AS updated_by_name
FROM questionnaires q
JOIN LATERAL (
    SELECT * FROM questionnaire_versions qv
    WHERE qv.questionnaire_id = q.id
    ORDER BY qv.version_no DESC LIMIT 1
) v ON true
JOIN users editor ON editor.id = v.updated_by
WHERE q.id = sqlc.arg(id) AND q.org_id = sqlc.arg(org_id)
  AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid);

-- name: HasNonDraftVersion :one
SELECT EXISTS (
    SELECT 1 FROM questionnaire_versions WHERE questionnaire_id = $1 AND status <> 'draft'
);

-- name: DeleteQuestionnaire :exec
DELETE FROM questionnaires WHERE id = $1 AND org_id = $2;

-- name: TouchVersion :exec
UPDATE questionnaire_versions SET updated_by = $2, updated_at = now() WHERE id = $1;

-- name: PublishVersion :one
-- Publishes the draft and retires the version published before it (LCY-01), in one statement.
-- Returns 0 when the version is no longer a draft.
WITH pub AS (
    UPDATE questionnaire_versions qv
    SET status = 'published', updated_by = sqlc.arg(updated_by), updated_at = now()
    WHERE qv.id = sqlc.arg(version_id) AND qv.status = 'draft'
    RETURNING qv.questionnaire_id
), retired AS (
    UPDATE questionnaire_versions
    SET status = 'retired'
    WHERE questionnaire_id IN (SELECT questionnaire_id FROM pub) AND status = 'published'
)
SELECT count(*) FROM pub;

-- name: CreateNextVersion :one
-- A new draft copied from version `from_version_id` (LCY-06/07): same chapters, same content,
-- so stable IDs and test cases carry over. Chapters get new row ids.
WITH v AS (
    INSERT INTO questionnaire_versions (questionnaire_id, version_no, status, updated_by)
    SELECT prev.questionnaire_id, prev.version_no + 1, 'draft', sqlc.arg(updated_by)
    FROM questionnaire_versions prev WHERE prev.id = sqlc.arg(from_version_id)
    RETURNING questionnaire_versions.id
), c AS (
    INSERT INTO chapters (version_id, position, name, description, icon, audience, content, updated_by)
    SELECT v.id, ch.position, ch.name, ch.description, ch.icon, ch.audience, ch.content, sqlc.arg(updated_by)
    FROM chapters ch, v
    WHERE ch.version_id = sqlc.arg(from_version_id)
)
SELECT id FROM v;

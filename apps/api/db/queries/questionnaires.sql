-- A questionnaire shows its latest version: the draft if there is one, else the last published.
-- Questionnaires belong to a trust (org_id). hospital_id NULL means every hospital of the trust
-- uses it; otherwise only that hospital sees it. Each lookup is scoped to the trust and hospital.

-- name: ListQuestionnaires :many
SELECT q.id, q.hospital_id, q.name, q.description, q.created_by::uuid AS created_by, q.created_at,
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
WHERE q.org_id = sqlc.arg(org_id)::uuid AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid)
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
    WHERE q.org_id = sqlc.arg(org_id)::uuid
      AND v.status = 'draft'
      AND lower(q.name) = lower(sqlc.arg(name))
      AND q.id <> sqlc.arg(exclude_id)
);

-- name: CreateQuestionnaire :one
-- A new questionnaire is the trust's, for all its hospitals.
INSERT INTO questionnaires (org_id, name, description, created_by)
VALUES (sqlc.arg(org_id)::uuid, sqlc.arg(name), sqlc.arg(description), sqlc.arg(created_by)::uuid)
RETURNING id;

-- name: CreateVersion :one
-- A new questionnaire's first draft, with the platform's default ASA rules (S40).
INSERT INTO questionnaire_versions (questionnaire_id, version_no, status, updated_by, rules)
VALUES (sqlc.arg(questionnaire_id), sqlc.arg(version_no), 'draft', sqlc.arg(updated_by)::uuid, sqlc.arg(rules)::jsonb)
RETURNING id;

-- name: GetQuestionnaire :one
SELECT q.id, q.org_id::uuid AS org_id, q.hospital_id, q.name, q.description, q.created_by::uuid AS created_by, q.created_at,
       v.id AS version_id, v.version_no, v.status, v.updated_at,
       editor.name AS updated_by_name, v.rules, v.rules_revision
FROM questionnaires q
JOIN LATERAL (
    SELECT * FROM questionnaire_versions qv
    WHERE qv.questionnaire_id = q.id
    ORDER BY qv.version_no DESC LIMIT 1
) v ON true
JOIN users editor ON editor.id = v.updated_by
WHERE q.id = sqlc.arg(id) AND q.org_id = sqlc.arg(org_id)::uuid
  AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid);

-- name: HasNonDraftVersion :one
SELECT EXISTS (
    SELECT 1 FROM questionnaire_versions WHERE questionnaire_id = $1 AND status <> 'draft'
);

-- name: DeleteQuestionnaire :exec
DELETE FROM questionnaires WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)::uuid;

-- name: TouchVersion :exec
-- Only a draft (S44c, S44a verification 3): a save's touch that waited for Publish (it runs after
-- the save, as its own statement) mustn't overwrite the publisher's "last changed by".
UPDATE questionnaire_versions SET updated_by = sqlc.arg(updated_by)::uuid, updated_at = now() WHERE id = sqlc.arg(id) AND status = 'draft';

-- name: LockVersionForPublish :one
-- Publish re-reads the version under its lock (S44a, S41 verification): a rules save in progress is
-- waited for, and one sent meanwhile waits and is then refused (it saves only a draft), so the rules
-- checked are the ones published. A Question Set save takes this row FOR SHARE for the same reason.
SELECT status, rules FROM questionnaire_versions WHERE id = $1 FOR UPDATE;

-- name: PublishVersion :one
-- Publishes the draft and retires the version published before it (LCY-01), in one statement.
-- Returns 0 when the version is no longer a draft.
WITH pub AS (
    UPDATE questionnaire_versions qv
    SET status = 'published', updated_by = sqlc.arg(updated_by)::uuid, updated_at = now()
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
-- so stable IDs and test cases carry over. Chapters get new row ids. The ASA rules are copied too;
-- a version from before rules existed gives the new draft the platform's defaults (S40 answers 1).
WITH v AS (
    INSERT INTO questionnaire_versions (questionnaire_id, version_no, status, updated_by, rules)
    SELECT prev.questionnaire_id, prev.version_no + 1, 'draft', sqlc.arg(updated_by)::uuid,
           coalesce(prev.rules, sqlc.arg(default_rules)::jsonb)
    FROM questionnaire_versions prev WHERE prev.id = sqlc.arg(from_version_id)
    RETURNING questionnaire_versions.id
), c AS (
    INSERT INTO chapters (version_id, position, name, description, icon, audience, content, updated_by)
    SELECT v.id, ch.position, ch.name, ch.description, ch.icon, ch.audience, ch.content, sqlc.arg(updated_by)::uuid
    FROM chapters ch, v
    WHERE ch.version_id = sqlc.arg(from_version_id)
)
SELECT id FROM v;

-- name: SaveVersionRules :one
-- Saves a draft's ASA rules (S40) only if nobody saved them since the caller loaded `revision`.
-- No row: the version isn't a draft any more, or the revision is stale.
UPDATE questionnaire_versions
SET rules = sqlc.arg(rules)::jsonb, rules_revision = rules_revision + 1,
    updated_by = sqlc.arg(updated_by)::uuid, updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'draft' AND rules_revision = sqlc.arg(revision)
RETURNING rules_revision;

-- name: VersionStatus :one
SELECT status FROM questionnaire_versions WHERE id = $1;

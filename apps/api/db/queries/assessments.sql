-- Assessments (plan C7, session S47): the platform's assessment questionnaires (no trust; migration
-- 00038) and the episode forms a clinician adds from them, with the clinician's answers.

-- name: LockAssessmentDefinition :exec
-- Two clinicians adding the same assessment at once publish its version once.
SELECT pg_advisory_xact_lock(hashtext('assessment:' || sqlc.arg(assessment)::text));

-- name: MatchingAssessmentVersion :one
-- The assessment's newest version, published or retired, whose one chapter's content is this one
-- (jsonb equality: key order and spacing don't matter). Any version, not only the latest: two
-- builds serving at once (an overlapping deploy, a rollback) then reuse each other's versions
-- instead of each publishing its own on every add (S47 verification 4).
SELECT v.id AS version_id
FROM questionnaires q
JOIN questionnaire_versions v ON v.questionnaire_id = q.id AND v.status IN ('published', 'retired')
JOIN chapters c ON c.version_id = v.id
WHERE q.assessment = sqlc.arg(assessment) AND c.content = sqlc.arg(content)::jsonb
ORDER BY v.version_no DESC
LIMIT 1;

-- name: AssessmentQuestionnaire :one
-- The assessment's questionnaire, made the first time it's needed, with the name Go has now: a
-- renamed assessment's questionnaire takes its new name with its new version (S47 verification 4).
INSERT INTO questionnaires (assessment, name, description)
VALUES (sqlc.arg(assessment), sqlc.arg(name), '')
ON CONFLICT (assessment) WHERE assessment IS NOT NULL DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: RetireAssessmentVersions :exec
UPDATE questionnaire_versions SET status = 'retired', updated_at = now()
WHERE questionnaire_id = $1 AND status = 'published';

-- name: PublishAssessmentVersion :one
-- A new published version of the assessment, with its content as one clinician Question Set.
WITH v AS (
    INSERT INTO questionnaire_versions (questionnaire_id, version_no, status)
    SELECT sqlc.arg(questionnaire_id), coalesce(max(version_no), 0) + 1, 'published'
    FROM questionnaire_versions WHERE questionnaire_id = sqlc.arg(questionnaire_id)
    RETURNING id
), c AS (
    INSERT INTO chapters (version_id, position, name, audience, content)
    SELECT v.id, 1, sqlc.arg(name), 'clinician', sqlc.arg(content)::jsonb FROM v
    RETURNING id, version_id
)
SELECT c.version_id, c.id AS chapter_id FROM c;

-- name: AddAssessmentForm :one
-- The assessment on the episode: no row if it already has one.
INSERT INTO episode_forms (episode_id, version_id, kind, assessment)
VALUES (sqlc.arg(episode_id), sqlc.arg(version_id), 'assessment', sqlc.arg(assessment))
ON CONFLICT (episode_id, assessment) WHERE kind = 'assessment' DO NOTHING
RETURNING id;

-- name: ListAssessmentForms :many
-- The episode's assessments, oldest first, each with its version's content and the clinician's
-- row (none until the first save). Confirmed: validated_at, by updated_by.
SELECT f.id, f.assessment::text AS assessment, q.name, f.created_at, v.version_no, c.id AS chapter_id, c.content,
       a.data, a.sources, coalesce(a.revision, 0)::int AS revision, a.updated_at, a.validated_at AS confirmed_at,
       u.name AS updated_by_name
FROM episode_forms f
JOIN questionnaire_versions v ON v.id = f.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
JOIN chapters c ON c.version_id = v.id
LEFT JOIN episode_answers a ON a.episode_form_id = f.id AND a.chapter_id = c.id AND a.actor = 'clinician'
LEFT JOIN users u ON u.id = a.updated_by
WHERE f.episode_id = $1 AND f.kind = 'assessment'
ORDER BY f.created_at, f.id;

-- name: GetAssessmentForm :one
-- One of the episode's assessments, with its chapter.
SELECT f.id, f.assessment::text AS assessment, c.id AS chapter_id, c.content
FROM episode_forms f
JOIN chapters c ON c.version_id = f.version_id
WHERE f.id = sqlc.arg(id) AND f.episode_id = sqlc.arg(episode_id) AND f.kind = 'assessment';

-- name: SaveAssessmentAnswers :one
-- The clinician's answers and their sources. `confirmed` stamps the row (Confirm: who, when); a
-- save without it clears the stamp, so a changed item needs a new Confirm. Returns the new
-- revision, or no row unless `revision` is the row's current one (0: no row yet), checked again
-- by ON CONFLICT … WHERE, as SaveClinicianAnswers. The caller holds the episode's lock.
INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data, sources, updated_by, validated_at, revision)
SELECT sqlc.arg(form_id), sqlc.arg(chapter_id), 'clinician', sqlc.arg(data), sqlc.arg(sources)::jsonb, sqlc.arg(updated_by),
       CASE WHEN sqlc.arg(confirmed)::boolean THEN now() END, 1
WHERE sqlc.arg(revision)::int = coalesce((SELECT a.revision FROM episode_answers a
      WHERE a.episode_form_id = sqlc.arg(form_id) AND a.chapter_id = sqlc.arg(chapter_id) AND a.actor = 'clinician'), 0)
ON CONFLICT (episode_form_id, chapter_id, actor) DO UPDATE
SET data = EXCLUDED.data, sources = EXCLUDED.sources, updated_by = EXCLUDED.updated_by, updated_at = now(),
    validated_at = EXCLUDED.validated_at, revision = episode_answers.revision + 1
WHERE episode_answers.revision = sqlc.arg(revision)
RETURNING revision;

-- Patient records belong to the trust (org_id); an episode is at one of its hospitals. Every
-- episode lookup is scoped to the hospital, or to the hospitals the caller can see.

-- name: ListPublishedVersions :many
-- The HQs an episode can be given: each questionnaire's published version.
SELECT v.id AS version_id, v.version_no, q.name
FROM questionnaire_versions v
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE q.org_id = sqlc.arg(org_id) AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid)
  AND v.status = 'published'
ORDER BY q.name;

-- name: CreateEpisode :one
-- The episode and its HQ form on the version, in one statement (S18). Creates nothing (no row)
-- unless the patient is the trust's and the version is published and one this hospital uses.
WITH e AS (
    INSERT INTO episodes (org_id, hospital_id, patient_id, procedure, anaesthetic, consultant, hospital_number, created_by)
    SELECT sqlc.arg(org_id), sqlc.arg(hospital_id), p.id, sqlc.arg(procedure), sqlc.arg(anaesthetic),
           sqlc.arg(consultant), sqlc.arg(hospital_number), sqlc.arg(created_by)
    FROM patients p, questionnaire_versions v
    JOIN questionnaires q ON q.id = v.questionnaire_id
    WHERE p.id = sqlc.arg(patient_id) AND p.org_id = sqlc.arg(org_id)
      AND v.id = sqlc.arg(version_id) AND v.status = 'published'
      AND q.org_id = sqlc.arg(org_id) AND (q.hospital_id IS NULL OR q.hospital_id = sqlc.arg(hospital_id)::uuid)
    RETURNING id
), hq AS (
    INSERT INTO episode_forms (episode_id, version_id, kind)
    SELECT e.id, sqlc.arg(version_id), 'hq' FROM e
)
SELECT id FROM e;

-- name: ListEpisodes :many
-- One trust's episodes: at every hospital when all_hospitals (a trust-wide member's "All
-- hospitals", STF-03), else only at the hospitals listed.
SELECT e.id, e.hospital_id, h.name AS hospital_name, e.status, e.procedure, e.created_at,
       e.patient_id, p.first_name, p.last_name, e.hospital_number,
       q.name AS questionnaire_name, v.version_no
FROM episodes e
JOIN hospitals h ON h.id = e.hospital_id
JOIN patients p ON p.id = e.patient_id
JOIN episode_forms f ON f.episode_id = e.id AND f.kind = 'hq'
JOIN questionnaire_versions v ON v.id = f.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE e.org_id = sqlc.arg(org_id) AND h.status = 'active' -- a suspended hospital's are kept, not shown (ORG-03)
  AND (sqlc.arg(all_hospitals)::boolean OR e.hospital_id = ANY (sqlc.arg(hospital_ids)::uuid[]))
  AND (sqlc.arg(status)::text = '' OR e.status = sqlc.arg(status))
ORDER BY e.created_at DESC;

-- name: GetEpisode :one
-- With its HQ form (S18): the version, the patient's submit and who sent it keep their names
-- from before forms, so the episode's JSON is unchanged; hq_form_id holds the answers.
SELECT e.id, e.status, e.procedure, e.anaesthetic, e.consultant, e.nurse_asa, e.anaesthetist_asa,
       f.submitted_at AS patient_submitted_at, e.review_completed_at, e.created_at, f.version_id, e.hospital_number,
       e.invited_at, e.reminders_sent, e.not_responding_at, -- the follow-up (S16)
       reviewer.name AS review_completed_by_name, submitter.name AS patient_submitted_by_name,
       p.id AS patient_id, p.first_name, p.last_name, p.date_of_birth, p.sex, p.national_id, p.national_id_type,
       p.mobile, p.email,
       q.name AS questionnaire_name, v.version_no, f.id AS hq_form_id
FROM episodes e
JOIN patients p ON p.id = e.patient_id
JOIN episode_forms f ON f.episode_id = e.id AND f.kind = 'hq'
JOIN questionnaire_versions v ON v.id = f.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
LEFT JOIN users reviewer ON reviewer.id = e.review_completed_by
LEFT JOIN users submitter ON submitter.id = f.submitted_by
WHERE e.id = $1 AND e.hospital_id = $2;

-- name: UpdateEpisode :execrows
-- Changes nothing if the status is no longer `loaded_status`, the one the caller read: the
-- workflow (the patient's submit, completing the review) may have moved it on since.
UPDATE episodes
SET status = sqlc.arg(status), procedure = sqlc.arg(procedure), anaesthetic = sqlc.arg(anaesthetic),
    consultant = sqlc.arg(consultant), nurse_asa = sqlc.arg(nurse_asa), anaesthetist_asa = sqlc.arg(anaesthetist_asa)
WHERE id = sqlc.arg(id) AND status = sqlc.arg(loaded_status);

-- name: AddEpisodeEvent :exec
INSERT INTO episode_events (episode_id, kind, text, user_id) VALUES ($1, $2, $3, $4);

-- name: AddAutomaticEpisodeEvent :exec
-- An entry written by the jobs (S16): a reminder, "not responding".
INSERT INTO episode_events (episode_id, kind, text, automatic) VALUES ($1, 'event', $2, true);

-- name: ListEpisodeEvents :many
-- user_name is NULL for the patient's entries and the automatic ones.
SELECT ev.id, ev.kind, ev.text, ev.created_at, u.name AS user_name, ev.automatic
FROM episode_events ev
LEFT JOIN users u ON u.id = ev.user_id
WHERE ev.episode_id = $1
ORDER BY ev.created_at, ev.id;

-- name: GetEpisodeForm :one
-- An episode's HQ form as the patient fills it in: for the patient's session (the handler checks
-- the episode is theirs) or staff completing it on their behalf (it checks the hospital).
SELECT e.id, e.status, f.submitted_at AS patient_submitted_at, f.version_id, f.id AS form_id,
       e.org_id, e.hospital_id, e.patient_id, p.account_id, -- for the checks and the audit log, never sent to the patient
       p.first_name, p.last_name, p.date_of_birth, p.sex,
       q.name AS questionnaire_name, h.name AS hospital_name, h.timezone,
       (o.status = 'active' AND h.status = 'active')::boolean AS open -- false while suspended (ORG-03)
FROM episodes e
JOIN orgs o ON o.id = e.org_id
JOIN hospitals h ON h.id = e.hospital_id
JOIN patients p ON p.id = e.patient_id
JOIN episode_forms f ON f.episode_id = e.id AND f.kind = 'hq'
JOIN questionnaire_versions v ON v.id = f.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE e.id = $1;

-- name: ListVersionChapters :many
SELECT id, name, description, icon, audience, content
FROM chapters WHERE version_id = $1
ORDER BY position, name;

-- name: GetVersionChapter :one
SELECT id, audience, content FROM chapters WHERE id = $1 AND version_id = $2;

-- name: ListEpisodeAnswers :many
-- One form's answers, both actors'.
SELECT a.chapter_id, a.actor, a.data, a.updated_at, a.validated_at, a.revision, u.name AS updated_by_name
FROM episode_answers a
LEFT JOIN users u ON u.id = a.updated_by
WHERE a.episode_form_id = $1;

-- name: SavePatientAnswers :one
-- entered_by is the staff member completing the HQ on the patient's behalf (PAT-09), NULL for the
-- patient. Once staff have entered a set's answers it stays marked with the last of them, even if
-- the patient saves it again, so the answers never look all the patient's own.
-- Returns the new revision, or no row if nothing was saved:
--   - once the form has been submitted, the patient's answers are frozen;
--   - `revision` must be the row's current one, or 0 if there's no row yet (see SaveClinicianAnswers).
INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data, updated_by, revision)
SELECT f.id, sqlc.arg(chapter_id), 'patient', sqlc.arg(data), sqlc.arg(entered_by), 1
FROM episode_forms f WHERE f.id = sqlc.arg(form_id) AND f.status = 'open'
  AND sqlc.arg(revision)::int = coalesce((SELECT a.revision FROM episode_answers a
      WHERE a.episode_form_id = f.id AND a.chapter_id = sqlc.arg(chapter_id) AND a.actor = 'patient'), 0)
ON CONFLICT (episode_form_id, chapter_id, actor) DO UPDATE
SET data = EXCLUDED.data, updated_at = now(), revision = episode_answers.revision + 1,
    updated_by = coalesce(EXCLUDED.updated_by, episode_answers.updated_by)
WHERE episode_answers.revision = sqlc.arg(revision)
RETURNING revision;

-- name: SubmitPatientHQ :execrows
-- Submits the HQ form, freezing the patient's answers, and moves its episode to Ready for review.
-- submitted_by: the staff member who sent it on the patient's behalf (PAT-09), NULL for the patient.
-- No row: the form was already submitted.
WITH submitted AS (
    UPDATE episode_forms hq SET status = 'submitted', submitted_at = now(), submitted_by = sqlc.arg(submitted_by)
    WHERE hq.id = sqlc.arg(form_id) AND hq.kind = 'hq' AND hq.status = 'open'
    RETURNING hq.episode_id
)
UPDATE episodes e SET status = 'ready_for_review' FROM submitted WHERE e.id = submitted.episode_id;

-- name: SaveClinicianAnswers :one
-- The clinician's copy of a Question Set's answers, which is final. `validated` stamps it
-- ("Validated by … on …"); a later save without it clears the stamp. Returns the new revision, or
-- no row if nothing was saved:
--   - unless the episode is Ready for review: before that the patient is still filling in, after
--     it the review is complete;
--   - unless `revision` is the one the caller started from: the row's current revision, or 0 if
--     there's no row yet. The SELECT checks it against the rows as they were when the statement
--     started; ON CONFLICT … WHERE checks it again against the row as it is now, so of two saves
--     sent at once from the same revision, only the first is applied.
INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data, updated_by, validated_at, revision)
SELECT f.id, sqlc.arg(chapter_id), 'clinician', sqlc.arg(data), sqlc.arg(updated_by),
       CASE WHEN sqlc.arg(validated)::boolean THEN now() END, 1
FROM episode_forms f JOIN episodes e ON e.id = f.episode_id
WHERE f.id = sqlc.arg(form_id) AND e.status = 'ready_for_review'
  AND sqlc.arg(revision)::int = coalesce((SELECT a.revision FROM episode_answers a
      WHERE a.episode_form_id = f.id AND a.chapter_id = sqlc.arg(chapter_id) AND a.actor = 'clinician'), 0)
ON CONFLICT (episode_form_id, chapter_id, actor) DO UPDATE
SET data = EXCLUDED.data, updated_by = EXCLUDED.updated_by, updated_at = now(), validated_at = EXCLUDED.validated_at,
    revision = episode_answers.revision + 1
WHERE episode_answers.revision = sqlc.arg(revision)
RETURNING revision;

-- name: CompleteReview :execrows
UPDATE episodes SET status = 'ready_for_poa', review_completed_by = $2, review_completed_at = now()
WHERE id = $1 AND status = 'ready_for_review';

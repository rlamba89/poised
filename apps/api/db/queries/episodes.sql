-- Every episode lookup is scoped to the hospital.

-- name: CreatePatient :one
INSERT INTO patients (hospital_id, first_name, last_name, date_of_birth, sex, hospital_number, phone, email)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: ListPatients :many
SELECT id, first_name, last_name, date_of_birth, sex, hospital_number, phone, email
FROM patients WHERE hospital_id = $1
ORDER BY last_name, first_name;

-- name: ListPublishedVersions :many
-- The HQs an episode can be given: each questionnaire's published version.
SELECT v.id AS version_id, v.version_no, q.name
FROM questionnaire_versions v
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE q.hospital_id = $1 AND v.status = 'published'
ORDER BY q.name;

-- name: CreateEpisode :one
-- Creates nothing (no row) unless the patient and a published version both belong to the hospital.
INSERT INTO episodes (hospital_id, patient_id, version_id, procedure, anaesthetic, consultant, patient_token, created_by)
SELECT sqlc.arg(hospital_id), p.id, v.id, sqlc.arg(procedure), sqlc.arg(anaesthetic), sqlc.arg(consultant),
       sqlc.arg(patient_token), sqlc.arg(created_by)
FROM patients p, questionnaire_versions v
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE p.id = sqlc.arg(patient_id) AND p.hospital_id = sqlc.arg(hospital_id)
  AND v.id = sqlc.arg(version_id) AND v.status = 'published' AND q.hospital_id = sqlc.arg(hospital_id)
RETURNING id;

-- name: ListEpisodes :many
SELECT e.id, e.status, e.procedure, e.created_at,
       p.first_name, p.last_name, p.hospital_number,
       q.name AS questionnaire_name, v.version_no
FROM episodes e
JOIN patients p ON p.id = e.patient_id
JOIN questionnaire_versions v ON v.id = e.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE e.hospital_id = sqlc.arg(hospital_id)
  AND (sqlc.arg(status)::text = '' OR e.status = sqlc.arg(status))
ORDER BY e.created_at DESC;

-- name: GetEpisode :one
SELECT e.id, e.status, e.procedure, e.anaesthetic, e.consultant, e.nurse_asa, e.anaesthetist_asa,
       e.patient_token, e.patient_submitted_at, e.review_completed_at, e.created_at, e.version_id,
       reviewer.name AS review_completed_by_name,
       p.id AS patient_id, p.first_name, p.last_name, p.date_of_birth, p.sex, p.hospital_number, p.phone, p.email,
       q.name AS questionnaire_name, v.version_no
FROM episodes e
JOIN patients p ON p.id = e.patient_id
JOIN questionnaire_versions v ON v.id = e.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
LEFT JOIN users reviewer ON reviewer.id = e.review_completed_by
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

-- name: ListEpisodeEvents :many
SELECT ev.id, ev.kind, ev.text, ev.created_at, u.name AS user_name
FROM episode_events ev
LEFT JOIN users u ON u.id = ev.user_id
WHERE ev.episode_id = $1
ORDER BY ev.created_at, ev.id;

-- name: GetEpisodeByToken :one
-- The patient's link: no sign-in, the token is the key.
SELECT e.id, e.status, e.patient_submitted_at, e.version_id,
       p.first_name, p.last_name, p.date_of_birth, p.sex,
       q.name AS questionnaire_name
FROM episodes e
JOIN patients p ON p.id = e.patient_id
JOIN questionnaire_versions v ON v.id = e.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE e.patient_token = $1;

-- name: ListVersionChapters :many
SELECT id, name, description, icon, audience, content
FROM chapters WHERE version_id = $1
ORDER BY position, name;

-- name: GetVersionChapter :one
SELECT id, audience, content FROM chapters WHERE id = $1 AND version_id = $2;

-- name: ListEpisodeAnswers :many
SELECT a.chapter_id, a.actor, a.data, a.updated_at, a.validated_at, a.revision, u.name AS updated_by_name
FROM episode_answers a
LEFT JOIN users u ON u.id = a.updated_by
WHERE a.episode_id = $1;

-- name: SavePatientAnswers :one
-- Returns the new revision, or no row if nothing was saved:
--   - once the patient has submitted, their answers are frozen;
--   - `revision` must be the row's current one, or 0 if there's no row yet (see SaveClinicianAnswers).
INSERT INTO episode_answers (episode_id, chapter_id, actor, data, revision)
SELECT e.id, sqlc.arg(chapter_id), 'patient', sqlc.arg(data), 1
FROM episodes e WHERE e.id = sqlc.arg(episode_id) AND e.patient_submitted_at IS NULL
  AND sqlc.arg(revision)::int = coalesce((SELECT a.revision FROM episode_answers a
      WHERE a.episode_id = e.id AND a.chapter_id = sqlc.arg(chapter_id) AND a.actor = 'patient'), 0)
ON CONFLICT (episode_id, chapter_id, actor) DO UPDATE
SET data = EXCLUDED.data, updated_at = now(), revision = episode_answers.revision + 1
WHERE episode_answers.revision = sqlc.arg(revision)
RETURNING revision;

-- name: SubmitPatientHQ :execrows
UPDATE episodes SET patient_submitted_at = now(), status = 'ready_for_review'
WHERE id = $1 AND patient_submitted_at IS NULL;

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
INSERT INTO episode_answers (episode_id, chapter_id, actor, data, updated_by, validated_at, revision)
SELECT e.id, sqlc.arg(chapter_id), 'clinician', sqlc.arg(data), sqlc.arg(updated_by),
       CASE WHEN sqlc.arg(validated)::boolean THEN now() END, 1
FROM episodes e WHERE e.id = sqlc.arg(episode_id) AND e.status = 'ready_for_review'
  AND sqlc.arg(revision)::int = coalesce((SELECT a.revision FROM episode_answers a
      WHERE a.episode_id = e.id AND a.chapter_id = sqlc.arg(chapter_id) AND a.actor = 'clinician'), 0)
ON CONFLICT (episode_id, chapter_id, actor) DO UPDATE
SET data = EXCLUDED.data, updated_by = EXCLUDED.updated_by, updated_at = now(), validated_at = EXCLUDED.validated_at,
    revision = episode_answers.revision + 1
WHERE episode_answers.revision = sqlc.arg(revision)
RETURNING revision;

-- name: CompleteReview :execrows
UPDATE episodes SET status = 'ready_for_poa', review_completed_by = $2, review_completed_at = now()
WHERE id = $1 AND status = 'ready_for_review';

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
INSERT INTO episodes (hospital_id, patient_id, version_id, procedure, anaesthetic, consultant, created_by)
SELECT sqlc.arg(hospital_id), p.id, v.id, sqlc.arg(procedure), sqlc.arg(anaesthetic), sqlc.arg(consultant),
       sqlc.arg(created_by)
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
       e.patient_submitted_at, e.review_completed_at, e.created_at, e.version_id,
       reviewer.name AS review_completed_by_name,
       p.id AS patient_id, p.first_name, p.last_name, p.date_of_birth, p.sex, p.hospital_number, p.phone, p.email,
       q.name AS questionnaire_name, v.version_no
FROM episodes e
JOIN patients p ON p.id = e.patient_id
JOIN questionnaire_versions v ON v.id = e.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
LEFT JOIN users reviewer ON reviewer.id = e.review_completed_by
WHERE e.id = $1 AND e.hospital_id = $2;

-- name: UpdateEpisode :exec
UPDATE episodes
SET status = $2, procedure = $3, anaesthetic = $4, consultant = $5, nurse_asa = $6, anaesthetist_asa = $7
WHERE id = $1;

-- name: AddEpisodeEvent :exec
INSERT INTO episode_events (episode_id, kind, text, user_id) VALUES ($1, $2, $3, $4);

-- name: ListEpisodeEvents :many
SELECT ev.id, ev.kind, ev.text, ev.created_at, u.name AS user_name
FROM episode_events ev
LEFT JOIN users u ON u.id = ev.user_id
WHERE ev.episode_id = $1
ORDER BY ev.created_at, ev.id;

-- name: GetPatientEpisode :one
-- The episode of a signed-in patient (their session names it).
SELECT e.id, e.status, e.patient_submitted_at, e.version_id,
       p.first_name, p.last_name, p.date_of_birth, p.sex,
       q.name AS questionnaire_name
FROM episodes e
JOIN patients p ON p.id = e.patient_id
JOIN questionnaire_versions v ON v.id = e.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE e.id = $1;

-- name: ListVersionChapters :many
SELECT id, name, description, icon, audience, content
FROM chapters WHERE version_id = $1
ORDER BY position, name;

-- name: GetVersionChapter :one
SELECT id, audience, content FROM chapters WHERE id = $1 AND version_id = $2;

-- name: ListEpisodeAnswers :many
SELECT a.chapter_id, a.actor, a.data, a.updated_at, a.validated_at, u.name AS updated_by_name
FROM episode_answers a
LEFT JOIN users u ON u.id = a.updated_by
WHERE a.episode_id = $1;

-- name: SavePatientAnswers :execrows
-- Saves nothing once the patient has submitted: their answers are then frozen.
INSERT INTO episode_answers (episode_id, chapter_id, actor, data)
SELECT e.id, sqlc.arg(chapter_id), 'patient', sqlc.arg(data)
FROM episodes e WHERE e.id = sqlc.arg(episode_id) AND e.patient_submitted_at IS NULL
ON CONFLICT (episode_id, chapter_id, actor) DO UPDATE SET data = EXCLUDED.data, updated_at = now();

-- name: SubmitPatientHQ :execrows
UPDATE episodes SET patient_submitted_at = now(), status = 'ready_for_review'
WHERE id = $1 AND patient_submitted_at IS NULL;

-- name: SaveClinicianAnswers :execrows
-- The clinician's copy of a Question Set's answers, which is final. `validated` stamps it
-- ("Validated by … on …"); a later save without it clears the stamp. Saves nothing unless the
-- episode is Ready for review: before that the patient is still filling in, after it the review is complete.
INSERT INTO episode_answers (episode_id, chapter_id, actor, data, updated_by, validated_at)
SELECT e.id, sqlc.arg(chapter_id), 'clinician', sqlc.arg(data), sqlc.arg(updated_by),
       CASE WHEN sqlc.arg(validated)::boolean THEN now() END
FROM episodes e WHERE e.id = sqlc.arg(episode_id) AND e.status = 'ready_for_review'
ON CONFLICT (episode_id, chapter_id, actor) DO UPDATE
SET data = EXCLUDED.data, updated_by = EXCLUDED.updated_by, updated_at = now(), validated_at = EXCLUDED.validated_at;

-- name: CompleteReview :execrows
UPDATE episodes SET status = 'ready_for_poa', review_completed_by = $2, review_completed_at = now()
WHERE id = $1 AND status = 'ready_for_review';

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

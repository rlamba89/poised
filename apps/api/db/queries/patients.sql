-- The trust's patient records (plan C2, S13). A record is the trust's (org_id); "this hospital's
-- patients" are those with at least one episode here, and only they are ever listed or shown at
-- a hospital (decisions.md, 11 Oct).

-- name: SearchPatients :many
-- This hospital's patients matching a search (patient.ParseSearch): the national ID, the
-- hospital number here (exactly, ignoring case), or the start of the surname (surname_prefix is a
-- LIKE pattern, lower case and escaped); born on birth when it's set. max_rows is one more than
-- is shown, so the caller knows there are more.
SELECT p.id, p.first_name, p.last_name, p.date_of_birth, p.national_id,
       coalesce((SELECT e.hospital_number FROM episodes e
                 WHERE e.patient_id = p.id AND e.hospital_id = sqlc.arg(hospital_id) AND e.hospital_number <> ''
                 ORDER BY e.created_at DESC LIMIT 1), '')::text AS hospital_number
FROM patients p
WHERE p.org_id = sqlc.arg(org_id)
  AND EXISTS (SELECT 1 FROM episodes e WHERE e.patient_id = p.id AND e.hospital_id = sqlc.arg(hospital_id))
  AND ((sqlc.arg(national_id)::text <> '' AND p.national_id_type = sqlc.arg(national_id_type)::text
        AND p.national_id = sqlc.arg(national_id))
       OR EXISTS (SELECT 1 FROM episodes e WHERE e.patient_id = p.id AND e.hospital_id = sqlc.arg(hospital_id)
                  AND e.hospital_number <> '' AND lower(e.hospital_number) = lower(sqlc.arg(term)::text))
       OR lower(p.last_name) LIKE sqlc.arg(surname_prefix)::text)
  AND (sqlc.narg(birth)::date IS NULL OR p.date_of_birth = sqlc.narg(birth))
ORDER BY p.last_name, p.first_name, p.date_of_birth, p.id
LIMIT sqlc.arg(max_rows);

-- name: GetHospitalPatient :one
-- The trust's record of one of this hospital's patients; no row for anyone else.
SELECT p.id, p.first_name, p.last_name, p.date_of_birth, p.sex, p.national_id, p.national_id_type, p.mobile, p.email
FROM patients p
WHERE p.id = sqlc.arg(id) AND p.org_id = sqlc.arg(org_id)
  AND EXISTS (SELECT 1 FROM episodes e WHERE e.patient_id = p.id AND e.hospital_id = sqlc.arg(hospital_id));

-- name: ListPatientEpisodesHere :many
-- A patient's episodes at this hospital only, newest first: never another hospital's.
SELECT e.id, e.status, e.procedure, e.hospital_number, e.created_at, q.name AS questionnaire_name, v.version_no
FROM episodes e
JOIN questionnaire_versions v ON v.id = e.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
WHERE e.patient_id = sqlc.arg(patient_id) AND e.hospital_id = sqlc.arg(hospital_id)
ORDER BY e.created_at DESC;

-- name: PatientByNationalID :one
-- The trust's record with this national ID, if any, for matching typed details.
SELECT id, first_name, last_name, date_of_birth, national_id, mobile, email
FROM patients
WHERE org_id = $1 AND national_id_type = $2 AND national_id = $3;

-- name: PatientsBornOn :many
-- The trust's records born on one day, for the duplicate check (patient.Decide), with whether each
-- is this hospital's patient and its latest hospital number here: only those may be shown.
SELECT p.id, p.first_name, p.last_name, p.date_of_birth, p.national_id, p.mobile, p.email,
       EXISTS (SELECT 1 FROM episodes e WHERE e.patient_id = p.id AND e.hospital_id = sqlc.arg(hospital_id))::boolean AS here,
       coalesce((SELECT e.hospital_number FROM episodes e
                 WHERE e.patient_id = p.id AND e.hospital_id = sqlc.arg(hospital_id) AND e.hospital_number <> ''
                 ORDER BY e.created_at DESC LIMIT 1), '')::text AS hospital_number
FROM patients p
WHERE p.org_id = sqlc.arg(org_id) AND p.date_of_birth = sqlc.arg(date_of_birth)
ORDER BY p.created_at, p.id;

-- name: CreatePatient :one
INSERT INTO patients (org_id, first_name, last_name, date_of_birth, sex, national_id, national_id_type, mobile, email)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- The audit log (plan C1, session S11; STF-06). Rows are only ever inserted (00009_audit_log.sql).

-- name: InsertAudit :exec
-- at is the server's clock (Deps.Clock), so tests can place rows in time.
INSERT INTO audit_log (at, org_id, hospital_id, actor, user_id, action, route, status, patient_ids, episode_ids)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, sqlc.arg(patient_ids)::uuid[], sqlc.arg(episode_ids)::uuid[]);

-- name: ListAudit :many
-- One trust's log, newest first, with the filters that are set: someone (a staff member, or
-- patients_only for the patient's link), a time range [from_at, to_at), and patients (any of
-- patient_ids, when by_patient).
SELECT a.id, a.at, a.hospital_id, h.name AS hospital_name, a.actor, a.user_id,
       u.name AS user_name, u.email AS user_email, a.action, a.route, a.status, a.patient_ids, a.episode_ids
FROM audit_log a
LEFT JOIN hospitals h ON h.id = a.hospital_id
LEFT JOIN users u ON u.id = a.user_id
WHERE a.org_id = sqlc.arg(org_id)
  AND (sqlc.narg(user_id)::uuid IS NULL OR a.user_id = sqlc.narg(user_id))
  AND (NOT sqlc.arg(patients_only)::boolean OR a.actor = 'patient')
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR a.at >= sqlc.narg(from_at))
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR a.at < sqlc.narg(to_at))
  AND (NOT sqlc.arg(by_patient)::boolean OR a.patient_ids && sqlc.arg(patient_ids)::uuid[])
ORDER BY a.at DESC, a.id DESC
LIMIT sqlc.arg(max_rows);

-- name: ListAuditUsers :many
-- The staff who appear in a trust's log, for the "Who" filter. People removed from the trust
-- are still listed: their past actions stay (STF-04).
SELECT DISTINCT u.id, u.name, u.email
FROM audit_log a
JOIN users u ON u.id = a.user_id
WHERE a.org_id = $1
ORDER BY u.name, u.email;

-- name: MatchAuditPatients :many
-- The trust's patients that a "Patient" filter names: part of their name ("Jane Smith" or
-- "Smith Jane"), their hospital number, or their record's id.
SELECT id FROM patients
WHERE org_id = sqlc.arg(org_id) AND (
    id::text = lower(sqlc.arg(query)::text)
    OR strpos(lower(first_name || ' ' || last_name), lower(sqlc.arg(query))) > 0
    OR strpos(lower(last_name || ' ' || first_name), lower(sqlc.arg(query))) > 0
    OR (hospital_number <> '' AND strpos(lower(hospital_number), lower(sqlc.arg(query))) > 0));

-- name: AuditPatientNames :many
-- The names of the trust's patients in a page of the log, read now (the log holds only IDs).
SELECT id, first_name, last_name, hospital_number FROM patients
WHERE org_id = $1 AND id = ANY (sqlc.arg(ids)::uuid[]);

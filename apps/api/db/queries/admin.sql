-- The superadmin console and trust settings (plan C1, session S10).

-- name: ListOrgsForAdmin :many
-- Every trust, for the platform admin: no patient data (OPS-06), only how many hospitals.
SELECT o.id, o.name, o.code, o.status, o.created_at,
       (SELECT count(*) FROM hospitals h WHERE h.org_id = o.id)::int AS hospital_count
FROM orgs o
ORDER BY lower(o.name), o.id;

-- name: CreateOrg :many
-- Nothing is returned when the code is taken (any case).
INSERT INTO orgs (name, code, contact_name, contact_email, contact_phone, settings)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING
RETURNING id;

-- name: GetOrgForAdmin :one
SELECT id, name, code, status, contact_name, contact_email, contact_phone, settings, created_at
FROM orgs WHERE id = $1;

-- name: LockOrgSettings :one
-- The trust's stored settings, locked until the transaction ends so two changes can't lose one.
SELECT settings FROM orgs WHERE id = $1 FOR UPDATE;

-- name: SetOrgSettings :exec
UPDATE orgs SET settings = $2 WHERE id = $1;

-- name: SetOrgStatus :exec
UPDATE orgs SET status = $2 WHERE id = $1;

-- name: ListOrgHospitals :many
-- A trust's hospitals, suspended ones included, by name.
SELECT id, name, address, timezone, status, settings
FROM hospitals WHERE org_id = $1
ORDER BY lower(name), id;

-- name: CreateHospital :many
-- Nothing is returned when the trust already has a hospital of that name (any case).
INSERT INTO hospitals (org_id, name, address, timezone, settings)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT DO NOTHING
RETURNING id;

-- name: LockHospitalSettings :one
-- A hospital of the trust and its stored settings, locked until the transaction ends.
SELECT settings FROM hospitals WHERE org_id = $1 AND id = $2 FOR UPDATE;

-- name: SetHospitalSettings :exec
UPDATE hospitals SET settings = sqlc.arg(settings) WHERE org_id = sqlc.arg(org_id) AND id = sqlc.arg(id);

-- name: SetHospitalStatus :exec
UPDATE hospitals SET status = sqlc.arg(status) WHERE org_id = sqlc.arg(org_id) AND id = sqlc.arg(id);

-- name: ListTrustAdmins :many
-- The trust's whole-trust admins, for the platform admin, with each account's latest open
-- invite (set while it waits for its first sign-in).
SELECT u.id AS user_id, u.name, u.email, i.expires_at AS invite_expires_at
FROM memberships m
JOIN users u ON u.id = m.user_id
LEFT JOIN staff_invites i ON i.user_id = u.id AND i.accepted_at IS NULL
    AND NOT EXISTS (SELECT 1 FROM staff_invites later WHERE later.user_id = u.id
                    AND later.accepted_at IS NULL AND (later.expires_at, later.id) > (i.expires_at, i.id))
WHERE m.org_id = $1 AND m.hospital_id IS NULL AND m.role = 'admin'
ORDER BY lower(u.name), u.id;

-- name: PublishedQuestionnairesInOrg :many
-- The trust's questionnaires that have a published version: the choices for its default HQ.
SELECT q.id, q.name FROM questionnaires q
WHERE q.org_id = $1 AND q.hospital_id IS NULL
  AND EXISTS (SELECT 1 FROM questionnaire_versions v WHERE v.questionnaire_id = q.id AND v.status = 'published')
ORDER BY lower(q.name), q.id;

-- name: SignInState :one
-- At sign-in: how many memberships the user has, and how many are usable: in an active trust and,
-- for a hospital one, an active hospital; for a trust-wide one, a trust with an active hospital or
-- none yet (a new trust). None usable means sign-in is refused (ORG-03), as /api/me would list
-- nothing (ListMyHospitals).
SELECT count(*)::int AS total,
       count(*) FILTER (WHERE o.status = 'active' AND CASE
           WHEN m.hospital_id IS NULL THEN
               EXISTS (SELECT 1 FROM hospitals x WHERE x.org_id = m.org_id AND x.status = 'active')
               OR NOT EXISTS (SELECT 1 FROM hospitals x WHERE x.org_id = m.org_id)
           ELSE h.status = 'active' END)::int AS active
FROM memberships m
JOIN orgs o ON o.id = m.org_id
LEFT JOIN hospitals h ON h.id = m.hospital_id
WHERE m.user_id = $1;

-- name: UpdateOrgDetails :exec
-- A trust's details (S10 answers 3: the platform admin edits them at any time). A code another
-- trust has fails on orgs_code_upper_key.
UPDATE orgs SET name = $2, code = $3, contact_name = $4, contact_email = $5, contact_phone = $6 WHERE id = $1;

-- name: LockHospitalForAdmin :one
-- One hospital of the trust, locked until the transaction ends.
SELECT name, address, timezone, settings FROM hospitals WHERE org_id = $1 AND id = $2 FOR UPDATE;

-- name: UpdateHospitalDetails :exec
-- A name another hospital of the trust has fails on hospitals_org_name_key.
UPDATE hospitals SET name = sqlc.arg(name), address = sqlc.arg(address), timezone = sqlc.narg(timezone), settings = sqlc.arg(settings)
WHERE org_id = sqlc.arg(org_id) AND id = sqlc.arg(id);

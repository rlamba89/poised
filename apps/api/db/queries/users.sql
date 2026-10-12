-- name: ListUsers :many
SELECT id, name, email FROM users ORDER BY name;

-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = $1;

-- name: IsPlatformAdmin :one
SELECT EXISTS (SELECT 1 FROM platform_admins WHERE user_id = $1);

-- name: ListMyHospitals :many
-- Every hospital the user can work in, once per membership that reaches it: a trust-wide one
-- (hospital_id NULL) reaches every hospital of the trust, a hospital one only that hospital.
-- GET /api/me keeps the highest role per hospital.
SELECT o.id AS org_id, o.name AS org_name, o.code AS org_code,
       h.id AS hospital_id, h.name AS hospital_name, h.timezone,
       m.role, (m.hospital_id IS NULL)::boolean AS trust_wide
FROM memberships m
JOIN orgs o ON o.id = m.org_id
JOIN hospitals h ON h.org_id = m.org_id AND (m.hospital_id IS NULL OR m.hospital_id = h.id)
WHERE m.user_id = $1
ORDER BY o.name, o.id, h.name, h.id;

-- name: RolesAtHospital :many
-- The caller's roles at a hospital of a trust: trust-wide, at that hospital, or both. None when
-- they aren't a member, or the hospital isn't in that trust. Re-read on every request (STF-04).
SELECT m.hospital_id, m.role FROM memberships m
JOIN hospitals h ON h.org_id = m.org_id AND h.id = sqlc.arg(hospital_id)
WHERE m.user_id = sqlc.arg(user_id) AND m.org_id = sqlc.arg(org_id)
  AND (m.hospital_id IS NULL OR m.hospital_id = h.id);

-- name: ListOrgMemberships :many
-- The caller's memberships in one trust: hospital_id is NULL for a trust-wide one.
SELECT hospital_id, role FROM memberships WHERE user_id = $1 AND org_id = $2;

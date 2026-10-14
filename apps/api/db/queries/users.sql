-- name: ListUsers :many
SELECT id, name, email FROM users ORDER BY name;

-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = $1;

-- name: IsPlatformAdmin :one
SELECT EXISTS (SELECT 1 FROM platform_admins WHERE user_id = $1);

-- name: ListMyHospitals :many
-- Every hospital the user can work in, once per membership that reaches it: a trust-wide one
-- (hospital_id NULL) reaches every hospital of the trust, a hospital one only that hospital.
-- GET /api/me keeps the highest role per hospital. A trust-wide membership of a trust with no
-- hospitals yet gives one row with no hospital (S10: a new trust's first admin). Suspended trusts
-- and hospitals are left out (ORG-03), and so is a trust whose hospitals are all suspended: it
-- isn't new, and there's nowhere to work (S10 verification 1, finding 2).
SELECT o.id AS org_id, o.name AS org_name, o.code AS org_code,
       h.id AS hospital_id, h.name AS hospital_name, h.timezone,
       m.role, (m.hospital_id IS NULL)::boolean AS trust_wide
FROM memberships m
JOIN orgs o ON o.id = m.org_id AND o.status = 'active'
LEFT JOIN hospitals h ON h.org_id = m.org_id AND (m.hospital_id IS NULL OR m.hospital_id = h.id) AND h.status = 'active'
WHERE m.user_id = $1
  AND (h.id IS NOT NULL OR (m.hospital_id IS NULL AND NOT EXISTS (SELECT 1 FROM hospitals x WHERE x.org_id = m.org_id)))
ORDER BY o.name, o.id, h.name, h.id;

-- name: RolesAtHospital :many
-- The caller's roles at a hospital of a trust: trust-wide, at that hospital, or both. None when
-- they aren't a member, the hospital isn't in that trust, or either is suspended (ORG-03).
-- Re-read on every request (STF-04).
SELECT m.hospital_id, m.role FROM memberships m
JOIN orgs o ON o.id = m.org_id AND o.status = 'active'
JOIN hospitals h ON h.org_id = m.org_id AND h.id = sqlc.arg(hospital_id) AND h.status = 'active'
WHERE m.user_id = sqlc.arg(user_id) AND m.org_id = sqlc.arg(org_id)
  AND (m.hospital_id IS NULL OR m.hospital_id = h.id);

-- name: ListOrgMemberships :many
-- The caller's memberships in one trust: hospital_id is NULL for a trust-wide one. None when the
-- trust is suspended; a membership of a suspended hospital is left out (ORG-03).
SELECT m.hospital_id, m.role FROM memberships m
JOIN orgs o ON o.id = m.org_id AND o.status = 'active'
LEFT JOIN hospitals h ON h.id = m.hospital_id
WHERE m.user_id = $1 AND m.org_id = $2 AND (m.hospital_id IS NULL OR h.status = 'active');

-- name: AddPlatformAdmin :exec
-- Makes the user a superadmin (A-21); nothing changes if they already are.
INSERT INTO platform_admins (user_id) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: GetUserByCognitoSub :one
-- The staff member a Cognito sign-in is for (S12), once their first sign-in stored the sub.
SELECT id, name, email FROM users WHERE cognito_sub = $1;

-- name: SetCognitoSub :exec
-- Ties the account to its user in the staff pool at a sign-in found by the verified email (no
-- account holds that sub yet: the sub is looked up first).
UPDATE users SET cognito_sub = sqlc.arg(sub) WHERE id = sqlc.arg(id);

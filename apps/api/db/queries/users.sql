-- name: ListUsers :many
SELECT id, name, email FROM users ORDER BY name;

-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = $1;

-- name: ListMemberships :many
-- Every hospital the user can open, once per membership that covers it: a trust-wide
-- membership covers all the trust's hospitals. Suspended trusts are left out.
SELECT o.id AS org_id, o.name AS org_name, h.id AS hospital_id, h.name AS hospital_name, m.role
FROM memberships m
JOIN orgs o ON o.id = m.org_id AND o.status = 'active'
JOIN hospitals h ON h.org_id = m.org_id AND (m.hospital_id IS NULL OR h.id = m.hospital_id)
WHERE m.user_id = $1
ORDER BY o.name, h.name;

-- name: ListRolesAt :many
-- The user's roles that apply to one hospital reached through one trust: trust-wide ones and
-- ones for that hospital. None when the hospital isn't in that trust or the trust isn't active.
SELECT m.role
FROM memberships m
JOIN orgs o ON o.id = m.org_id AND o.status = 'active'
JOIN hospitals h ON h.id = @hospital_id AND h.org_id = m.org_id
WHERE m.user_id = @user_id
  AND m.org_id = @org_id
  AND (m.hospital_id IS NULL OR m.hospital_id = @hospital_id);

-- name: GetUserByCognitoSub :one
SELECT id, name, email FROM users WHERE cognito_sub = $1;

-- name: GetUserByEmail :one
-- Emails are compared in lower case; Cognito gives them in the case the person typed.
SELECT id, name, email, cognito_sub FROM users WHERE lower(email) = lower(@email);

-- name: SetCognitoSub :execrows
-- Saves the subject at a first sign-in, only if none is saved yet.
UPDATE users SET cognito_sub = @cognito_sub WHERE id = @id AND cognito_sub IS NULL;

-- name: CountActiveMemberships :one
SELECT count(*) FROM memberships m JOIN orgs o ON o.id = m.org_id AND o.status = 'active'
WHERE m.user_id = $1;

-- name: ListTrustRoles :many
-- The user's trust-wide roles in an active trust (memberships for the whole trust only).
SELECT m.role FROM memberships m JOIN orgs o ON o.id = m.org_id AND o.status = 'active'
WHERE m.user_id = $1 AND m.org_id = $2 AND m.hospital_id IS NULL;

-- name: CreateInvitedUser :one
INSERT INTO users (name, email, cognito_sub) VALUES ($1, $2, $3) RETURNING id;

-- name: CreateMembership :execrows
-- Adds a membership; none when the user already has one for that scope.
INSERT INTO memberships (user_id, org_id, hospital_id, role) VALUES ($1, $2, $3, $4)
ON CONFLICT ON CONSTRAINT memberships_scope_key DO NOTHING;

-- name: HospitalInOrg :one
SELECT EXISTS (SELECT 1 FROM hospitals WHERE id = $1 AND org_id = $2);

-- name: ListTrustStaff :many
-- Everyone with a membership in the trust, one row per membership.
SELECT u.id AS user_id, u.name, u.email, (u.cognito_sub IS NOT NULL)::boolean AS has_account,
       m.hospital_id, h.name AS hospital_name, m.role
FROM memberships m
JOIN users u ON u.id = m.user_id
LEFT JOIN hospitals h ON h.id = m.hospital_id
WHERE m.org_id = $1
ORDER BY u.name, h.name NULLS FIRST;

-- name: CountTrustMemberships :one
-- Any membership in the trust, trust-wide or for one of its hospitals.
SELECT count(*) FROM memberships WHERE user_id = $1 AND org_id = $2;

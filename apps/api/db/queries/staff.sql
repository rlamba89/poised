-- Staff, invites and memberships (plan C1, session S09).

-- name: LockOrgStaff :exec
-- Serialises staff changes in one trust, so two admins can't both remove the last admin, or
-- give one person two memberships.
SELECT pg_advisory_xact_lock(hashtext('staff:' || sqlc.arg(org_id)::uuid::text));

-- name: ListStaff :many
-- Everyone with access to the trust, one row per membership (a person may have several: the whole
-- trust and/or some hospitals), each person's whole-trust one first, then by hospital name. invite_expires_at is set while the
-- person's account is waiting for its first sign-in: the latest deadline of their open invites
-- (the join keeps only the latest of them).
SELECT u.id AS user_id, u.name, u.email, m.hospital_id, h.name AS hospital_name, m.role,
       i.expires_at AS invite_expires_at
FROM memberships m
JOIN users u ON u.id = m.user_id
LEFT JOIN hospitals h ON h.id = m.hospital_id
LEFT JOIN staff_invites i ON i.user_id = u.id AND i.accepted_at IS NULL
    AND NOT EXISTS (SELECT 1 FROM staff_invites later WHERE later.user_id = u.id
                    AND later.accepted_at IS NULL AND (later.expires_at, later.id) > (i.expires_at, i.id))
WHERE m.org_id = $1
ORDER BY lower(u.name), u.id, m.hospital_id IS NOT NULL, h.name, m.hospital_id;

-- name: GetStaffMember :many
-- One person's rows of ListStaff.
SELECT u.id AS user_id, u.name, u.email, m.hospital_id, h.name AS hospital_name, m.role,
       i.expires_at AS invite_expires_at
FROM memberships m
JOIN users u ON u.id = m.user_id
LEFT JOIN hospitals h ON h.id = m.hospital_id
LEFT JOIN staff_invites i ON i.user_id = u.id AND i.accepted_at IS NULL
    AND NOT EXISTS (SELECT 1 FROM staff_invites later WHERE later.user_id = u.id
                    AND later.accepted_at IS NULL AND (later.expires_at, later.id) > (i.expires_at, i.id))
WHERE m.org_id = $1 AND m.user_id = $2
ORDER BY m.hospital_id IS NOT NULL, h.name, m.hospital_id;

-- name: HospitalInOrg :one
SELECT EXISTS (SELECT 1 FROM hospitals WHERE org_id = $1 AND id = $2);

-- name: GetUserByEmail :one
SELECT id, name, email FROM users WHERE lower(email) = lower(sqlc.arg(email));

-- name: CreateUserIfNew :many
-- Nothing is returned when the email is already taken (by an invite running at the same time).
INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING id;

-- name: AwaitingFirstSignIn :one
-- Whether the account has an invite not yet accepted: it has never signed in.
SELECT EXISTS (SELECT 1 FROM staff_invites WHERE user_id = $1 AND accepted_at IS NULL);

-- name: UserOrgMemberships :many
SELECT hospital_id, role FROM memberships WHERE user_id = $1 AND org_id = $2;

-- name: MembershipRole :one
-- The user's role in one membership: the whole trust (hospital_id NULL) or one hospital.
SELECT role FROM memberships
WHERE user_id = sqlc.arg(user_id) AND org_id = sqlc.arg(org_id) AND hospital_id IS NOT DISTINCT FROM sqlc.narg(hospital_id);

-- name: UpdateMembership :exec
-- Moves one membership to another scope and/or role.
UPDATE memberships SET hospital_id = sqlc.narg(new_hospital_id), role = sqlc.arg(role)
WHERE user_id = sqlc.arg(user_id) AND org_id = sqlc.arg(org_id) AND hospital_id IS NOT DISTINCT FROM sqlc.narg(hospital_id);

-- name: DeleteMembership :exec
DELETE FROM memberships
WHERE user_id = sqlc.arg(user_id) AND org_id = sqlc.arg(org_id) AND hospital_id IS NOT DISTINCT FROM sqlc.narg(hospital_id);

-- name: DeleteUserOrgMemberships :exec
DELETE FROM memberships WHERE user_id = $1 AND org_id = $2;

-- name: AddMembership :exec
INSERT INTO memberships (user_id, org_id, hospital_id, role) VALUES ($1, $2, $3, $4);

-- name: CountOtherTrustAdmins :one
-- The trust's whole-trust admins other than this user.
SELECT count(*) FROM memberships
WHERE org_id = $1 AND hospital_id IS NULL AND role = 'admin' AND user_id <> sqlc.arg(user_id);

-- name: CreateStaffInvite :exec
INSERT INTO staff_invites (org_id, hospital_id, user_id, role, invited_by, created_at, expires_at, accepted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: InviteState :one
-- At sign-in: how many invites wait for the first sign-in, and how many of those are still open.
SELECT count(*) FILTER (WHERE accepted_at IS NULL) AS waiting,
       count(*) FILTER (WHERE accepted_at IS NULL AND expires_at > sqlc.arg(now)) AS open
FROM staff_invites WHERE user_id = sqlc.arg(user_id);

-- name: AcceptInvites :exec
UPDATE staff_invites SET accepted_at = sqlc.arg(now) WHERE user_id = sqlc.arg(user_id) AND accepted_at IS NULL;

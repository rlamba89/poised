-- Patient links (plan C2): invites (S14); sign-in and contact verification links (S15) are in
-- accounts.sql. Only a token's hash is stored (auth.HashLinkToken).

-- name: CreateInviteLink :one
-- A new invite link, stored before its messages go (so it works the moment they arrive). Once
-- they've gone it replaces the older ones (RevokeOlderInviteLinks); if none went, it's revoked
-- itself (RevokeLink) and the old link carries on.
INSERT INTO login_links (token_hash, kind, patient_id, episode_id, created_by)
VALUES ($1, 'invite', $2, $3, $4)
RETURNING id, created_at;

-- name: LockEpisode :one
-- Taken before an invite's links are replaced, so two resends at once are applied one after
-- the other.
SELECT id FROM episodes WHERE id = $1 FOR UPDATE;

-- name: ResetEpisodeDob :exec
-- A Resend: the new link starts with the episode's five tries again.
UPDATE episodes SET dob_failed_attempts = 0 WHERE id = $1;

-- name: RevokeOlderInviteLinks :exec
-- An invite went: the episode's links older than it stop working (only the newest works, even
-- when two resends cross: each revokes only what came before it).
UPDATE login_links SET revoked_at = now()
WHERE episode_id = sqlc.arg(episode_id) AND kind = 'invite' AND revoked_at IS NULL
  AND (created_at, id) < (sqlc.arg(created_at)::timestamptz, sqlc.arg(id)::uuid);

-- name: RevokeLink :exec
-- An invite whose messages all failed: its link never reached anyone.
UPDATE login_links SET revoked_at = now() WHERE id = $1;

-- name: GetLinkByHash :one
-- A link that works: not replaced, not expired, its episode's patient the link's. Locked links
-- are returned (locked_at), so the patient is told to ask for a new one.
SELECT l.id, e.patient_id, l.episode_id, l.locked_at,
       e.org_id, e.hospital_id, h.name AS hospital_name, p.date_of_birth,
       (o.status = 'active' AND h.status = 'active')::boolean AS open -- false while suspended (ORG-03)
FROM login_links l
JOIN episodes e ON e.id = l.episode_id AND e.patient_id = l.patient_id
JOIN hospitals h ON h.id = e.hospital_id
JOIN orgs o ON o.id = e.org_id
JOIN patients p ON p.id = l.patient_id
WHERE l.token_hash = $1 AND l.kind = 'invite' AND l.revoked_at IS NULL
  AND (l.expires_at IS NULL OR l.expires_at > sqlc.arg(now)::timestamptz);

-- name: FailLinkDob :one
-- Sign-in links (S15) only: an invite's tries are its episode's (FailInviteDob).
-- Counts a wrong date of birth, locking the link at max_attempts. No row: it was already locked.
UPDATE login_links
SET failed_dob_attempts = failed_dob_attempts + 1,
    locked_at = CASE WHEN failed_dob_attempts + 1 >= sqlc.arg(max_attempts)::int THEN now() END
WHERE id = sqlc.arg(id) AND locked_at IS NULL
RETURNING failed_dob_attempts;

-- name: FailInviteDob :one
-- Counts a wrong date of birth on an invite link against its episode (S16 test report 1): all the
-- episode's live invite links share max_attempts tries, and the last locks them all together.
-- No row: the link was already locked (or replaced).
WITH bumped AS (
    UPDATE episodes e SET dob_failed_attempts = e.dob_failed_attempts + 1
    FROM login_links l
    WHERE l.id = sqlc.arg(id) AND l.kind = 'invite' AND l.locked_at IS NULL AND l.revoked_at IS NULL
      AND e.id = l.episode_id
    RETURNING e.id, e.dob_failed_attempts
), locked AS (
    UPDATE login_links SET locked_at = now()
    FROM bumped
    WHERE login_links.episode_id = bumped.id AND login_links.kind = 'invite'
      AND login_links.revoked_at IS NULL AND login_links.locked_at IS NULL
      AND bumped.dob_failed_attempts >= sqlc.arg(max_attempts)::int
    RETURNING login_links.id
)
SELECT dob_failed_attempts FROM bumped;

-- name: PassInviteDob :execrows
-- The right date of birth on an invite link: the episode's count starts again. No row: the link
-- was locked meanwhile.
WITH passed AS (
    UPDATE login_links SET failed_dob_attempts = 0, used_at = now()
    WHERE login_links.id = $1 AND login_links.locked_at IS NULL
    RETURNING login_links.episode_id
)
UPDATE episodes SET dob_failed_attempts = 0 FROM passed WHERE episodes.id = passed.episode_id;

-- name: GetEpisodeInvite :one
-- The episode's working invite, for staff: when the newest link was sent, and whether five wrong
-- dates of birth locked the episode (any of its live links locked: they lock together). No row:
-- no invite has been sent.
SELECT l.created_at AS sent_at, coalesce(l.locked_at, (
           SELECT x.locked_at FROM login_links x
           WHERE x.episode_id = l.episode_id AND x.kind = 'invite' AND x.revoked_at IS NULL AND x.locked_at IS NOT NULL
           ORDER BY x.locked_at DESC LIMIT 1)) AS locked_at
FROM login_links l
WHERE l.episode_id = $1 AND l.kind = 'invite' AND l.revoked_at IS NULL
ORDER BY l.created_at DESC LIMIT 1;

-- name: GetInviteDetails :one
-- What an invite needs: who to send it to, the hospital's name and its trust's and own settings
-- (for the message templates).
SELECT e.id, e.patient_id, e.patient_submitted_at, p.first_name, p.mobile, p.email,
       h.name AS hospital_name, h.settings AS hospital_settings, o.settings AS org_settings
FROM episodes e
JOIN patients p ON p.id = e.patient_id
JOIN hospitals h ON h.id = e.hospital_id
JOIN orgs o ON o.id = e.org_id
WHERE e.id = $1;

-- name: InsertSentMessage :exec
-- MESSAGING=capture: a message stored instead of sent.
INSERT INTO sent_messages (channel, recipient, subject, body) VALUES ($1, $2, $3, $4);

-- name: ListSentMessages :many
-- The newest captured messages, to anyone whose address contains `recipient` ('' for everyone).
SELECT id, channel, recipient, subject, body, sent_at
FROM sent_messages
WHERE sqlc.arg(recipient)::text = '' OR recipient ILIKE '%' || sqlc.arg(recipient)::text || '%'
ORDER BY id DESC
LIMIT 100;

-- name: StartPatientSession :one
-- After the right date of birth: a session for the person's account (all their records), tied
-- to the link it was opened with.
INSERT INTO patient_sessions (link_id, account_id, created_at, last_seen_at)
VALUES (sqlc.arg(link_id), sqlc.arg(account_id), sqlc.arg(now), sqlc.arg(now))
RETURNING id;

-- name: TouchPatientSession :one
-- Every patient request: the session's account, if the session is live (not signed out, seen
-- since idle_since) and its link wasn't replaced by a resend or locked. It counts as activity.
-- A link's expiry governs only opening it: a session from a 15-minute sign-in link lasts like any
-- other. No row: the session has ended.
UPDATE patient_sessions s SET last_seen_at = sqlc.arg(now)
FROM login_links l
WHERE s.id = sqlc.arg(id) AND l.id = s.link_id
  AND s.ended_at IS NULL AND s.last_seen_at > sqlc.arg(idle_since)::timestamptz
  AND l.revoked_at IS NULL AND l.locked_at IS NULL
RETURNING s.account_id;

-- name: EndPatientSession :exec
UPDATE patient_sessions SET ended_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND ended_at IS NULL;

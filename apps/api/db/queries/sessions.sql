-- name: CreateSession :exec
INSERT INTO sessions (id_hash, user_id, expires_at)
VALUES (@id_hash, @user_id::uuid, now() + make_interval(secs => @max_age_seconds::float8));

-- name: GetSession :one
-- A live staff session: before its expiry, and used within the idle limit. (A patient
-- session has no user, so the join leaves it out.)
SELECT u.id AS user_id, u.name, s.last_seen_at
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.id_hash = @id_hash
  AND s.expires_at > now()
  AND s.last_seen_at > now() - make_interval(secs => @idle_seconds::float8);

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = now() WHERE id_hash = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id_hash = $1;

-- name: CreatePatientSession :exec
INSERT INTO sessions (id_hash, episode_id, expires_at)
VALUES (@id_hash, @episode_id::uuid, now() + make_interval(secs => @max_age_seconds::float8));

-- name: GetPatientSession :one
-- A live patient session: before its expiry, and used within the idle limit.
SELECT e.id AS episode_id, s.last_seen_at
FROM sessions s
JOIN episodes e ON e.id = s.episode_id
WHERE s.id_hash = @id_hash
  AND s.expires_at > now()
  AND s.last_seen_at > now() - make_interval(secs => @idle_seconds::float8);

-- name: DeleteEpisodeSessions :exec
-- Signs out every patient session of an episode, e.g. when its link is replaced.
DELETE FROM sessions WHERE episode_id = $1;

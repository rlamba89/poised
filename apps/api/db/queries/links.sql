-- name: CreateLoginLink :exec
INSERT INTO login_links (token_hash, token_sealed, episode_id) VALUES ($1, $2, $3);

-- name: GetLoginLink :one
-- A patient link by its hash, with the date of birth it must be confirmed with.
SELECT l.id, l.episode_id, l.locked_at, p.date_of_birth
FROM login_links l
JOIN episodes e ON e.id = l.episode_id
JOIN patients p ON p.id = e.patient_id
WHERE l.token_hash = $1;

-- name: RecordWrongDateOfBirth :one
-- Counts a wrong date of birth and locks the link at @max_attempts. Atomic, so tries sent in
-- parallel can't get past the limit.
UPDATE login_links
SET failed_dob_attempts = failed_dob_attempts + 1,
    locked_at = CASE WHEN failed_dob_attempts + 1 >= @max_attempts::int THEN now() END
WHERE id = @id AND locked_at IS NULL
RETURNING (locked_at IS NOT NULL)::boolean AS locked;

-- name: ResetDateOfBirthTries :exec
UPDATE login_links SET failed_dob_attempts = 0 WHERE id = $1;

-- name: GetEpisodeLinkSealed :one
-- The newest link of an episode, sealed (NULL when made before sealing).
SELECT token_sealed FROM login_links WHERE episode_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: DeleteEpisodeLinks :exec
DELETE FROM login_links WHERE episode_id = $1;

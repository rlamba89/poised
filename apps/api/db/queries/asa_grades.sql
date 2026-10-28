-- The anaesthetist's ASA grades of an episode (plan C6, session S44b; migration 00035): a history,
-- never edited. The handlers have checked the episode is the caller's hospital's, and hold
-- LockEpisodeStatus while they add or mark one.

-- name: ListAnaesthetistGrades :many
-- Every grade, oldest first, with who gave it (NULL: recorded before the history) and, for one
-- entered in error, who marked it, when and why.
SELECT g.id, g.grade, g.given_at, giver.name AS given_by_name, g.reason,
       g.error_at, marker.name AS error_by_name, g.error_reason
FROM anaesthetist_asa_grades g
LEFT JOIN users giver ON giver.id = g.given_by
LEFT JOIN users marker ON marker.id = g.error_by
WHERE g.episode_id = $1
ORDER BY g.given_at, g.id;

-- name: CountingAnaesthetistGrade :one
-- The grade that counts: the latest not entered in error. No row: none.
SELECT grade FROM anaesthetist_asa_grades WHERE episode_id = $1 AND error_at IS NULL
ORDER BY given_at DESC, id DESC LIMIT 1;

-- name: HasAnaesthetistGrade :one
-- Whether the episode has any grade, entered in error or not: every grade after its first needs a
-- reason (S44b answers 2).
SELECT EXISTS (SELECT 1 FROM anaesthetist_asa_grades WHERE episode_id = $1);

-- name: AddAnaesthetistGrade :one
INSERT INTO anaesthetist_asa_grades (episode_id, grade, given_by, reason)
VALUES (sqlc.arg(episode_id), sqlc.arg(grade), sqlc.arg(user_id), sqlc.narg(reason))
RETURNING id;

-- name: GetAnaesthetistGrade :one
SELECT id, grade, error_at FROM anaesthetist_asa_grades WHERE id = $1 AND episode_id = $2;

-- name: CountingAnaesthetistGradesBut :one
-- How many grades not entered in error the episode has besides this one.
SELECT count(*) FROM anaesthetist_asa_grades
WHERE episode_id = sqlc.arg(episode_id) AND id <> sqlc.arg(id) AND error_at IS NULL;

-- name: MarkAnaesthetistGradeInError :execrows
-- No row: it's already marked (it can't be unmarked).
UPDATE anaesthetist_asa_grades SET error_by = sqlc.arg(user_id), error_at = now(), error_reason = sqlc.arg(reason)::text
WHERE id = sqlc.arg(id) AND episode_id = sqlc.arg(episode_id) AND error_at IS NULL;

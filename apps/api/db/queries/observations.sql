-- An episode's observations (plan C7, session S46; migration 00037): a history, never edited. The
-- handlers have checked the episode is the caller's hospital's (GetEpisode), and hold
-- LockEpisodeStatus while they add, correct or mark one.

-- name: ListObservations :many
-- Every reading, oldest first, with who entered it and, for one entered in error, who marked it.
SELECT o.id, o.kind, o.value, o.value2, o.unit, o.note, o.taken_at, o.entered_at, enterer.name AS entered_by_name,
       o.corrects, o.reason, o.superseded_by, o.error_at, marker.name AS error_by_name, o.error_reason
FROM observations o
JOIN users enterer ON enterer.id = o.entered_by
LEFT JOIN users marker ON marker.id = o.error_by
WHERE o.episode_id = $1
ORDER BY o.taken_at, o.entered_at, o.id;

-- name: AddObservation :one
INSERT INTO observations (episode_id, kind, value, value2, unit, note, taken_at, entered_by, corrects, reason)
VALUES (sqlc.arg(episode_id), sqlc.arg(kind), sqlc.narg(value), sqlc.narg(value2), sqlc.narg(unit), sqlc.narg(note),
        sqlc.arg(taken_at), sqlc.arg(user_id), sqlc.narg(corrects), sqlc.narg(reason))
RETURNING id;

-- name: GetObservation :one
SELECT id, kind, value, value2, unit, note, taken_at, superseded_by, error_at
FROM observations WHERE id = $1 AND episode_id = $2;

-- name: SupersedeObservation :execrows
-- No row: it's already superseded or entered in error.
UPDATE observations SET superseded_by = sqlc.arg(by)
WHERE id = sqlc.arg(id) AND episode_id = sqlc.arg(episode_id) AND superseded_by IS NULL AND error_at IS NULL;

-- name: MarkObservationInError :execrows
-- No row: it's already superseded or entered in error.
UPDATE observations SET error_by = sqlc.arg(user_id), error_at = now(), error_reason = sqlc.arg(reason)::text
WHERE id = sqlc.arg(id) AND episode_id = sqlc.arg(episode_id) AND superseded_by IS NULL AND error_at IS NULL;

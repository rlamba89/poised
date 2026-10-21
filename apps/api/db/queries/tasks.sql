-- An episode's tasks (plan C3, session S21; TSK-01). The handlers have checked the episode is the
-- caller's hospital's, and hold LockEpisodeStatus while they change one.

-- name: ListTasks :many
-- Oldest first. The names are NULL where nobody is named: a rule's task, an open or never
-- re-opened one.
SELECT t.id, t.title, t.team, t.created_at, creator.name AS created_by_name, t.created_by_rule,
       t.done_at, doer.name AS done_by_name, t.reopened_at, reopener.name AS reopened_by_name
FROM tasks t
LEFT JOIN users creator ON creator.id = t.created_by
LEFT JOIN users doer ON doer.id = t.done_by
LEFT JOIN users reopener ON reopener.id = t.reopened_by
WHERE t.episode_id = $1
ORDER BY t.created_at, t.id;

-- name: CreateTask :one
INSERT INTO tasks (episode_id, title, team, created_by) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: GetTask :one
SELECT id, title, team, done_at FROM tasks WHERE id = $1 AND episode_id = $2;

-- name: CompleteTask :execrows
-- No row: it's already done.
UPDATE tasks SET done_by = $2, done_at = now() WHERE id = $1 AND done_at IS NULL;

-- name: ReopenTask :execrows
-- No row: it's already open.
UPDATE tasks SET done_by = NULL, done_at = NULL, reopened_by = $2, reopened_at = now() WHERE id = $1 AND done_at IS NOT NULL;

-- name: BumpEpisodeRevision :exec
-- A change to the episode's tasks is a change to the episode (S20's revision): another tab's
-- page is then stale, as after a status move.
UPDATE episodes SET revision = revision + 1 WHERE id = $1;

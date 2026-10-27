-- An episode's flags from the HQ's rules (plan C6, session S43; migration 00034). The handlers have
-- checked the episode is the caller's hospital's.

-- name: AddRuleFlag :execrows
-- A flag a rule adds on a send. No row: the same rule's flag is still standing (not cleared).
INSERT INTO episode_flags (episode_id, name, rule) VALUES ($1, $2, $3)
ON CONFLICT (episode_id, rule, name) WHERE cleared_at IS NULL DO NOTHING;

-- name: ListEpisodeFlags :many
-- Every flag, standing or cleared, oldest first, with who cleared it.
SELECT f.id, f.name, f.rule, f.created_at, f.cleared_at, u.name AS cleared_by_name, f.cleared_reason
FROM episode_flags f
LEFT JOIN users u ON u.id = f.cleared_by
WHERE f.episode_id = $1
ORDER BY f.created_at, f.id;

-- name: GetEpisodeFlag :one
SELECT id, name, cleared_at FROM episode_flags WHERE id = $1 AND episode_id = $2;

-- name: ClearEpisodeFlag :execrows
-- No row: it's already cleared. The caller holds LockEpisodeStatus.
UPDATE episode_flags SET cleared_by = sqlc.arg(user_id), cleared_at = now(), cleared_reason = sqlc.arg(reason)::text
WHERE id = sqlc.arg(id) AND episode_id = sqlc.arg(episode_id) AND cleared_at IS NULL;

-- name: RequireAnaesthetistASA :execrows
-- A rule requires the anaesthetist's ASA grade before Ready for admission (S43). No row: a rule
-- already does (the first one's name is kept).
UPDATE episodes SET anaesthetist_asa_required_by = sqlc.arg(rule)::text
WHERE id = sqlc.arg(id) AND anaesthetist_asa_required_by IS NULL;

-- name: SetRuleSuggestions :exec
-- The statuses the latest send's rules suggest ([{rule, status}]), shown on the episode.
UPDATE episodes SET rule_suggestions = sqlc.arg(suggestions)::jsonb WHERE id = sqlc.arg(id);

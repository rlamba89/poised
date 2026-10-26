-- The patient's Information for an episode (S36, INF-04): videos and leaflets from the content
-- library. A procedure's list (procedures.info_content, ids in order) is copied onto an episode when
-- it's made; a clinician adds to or removes from that one episode's list (episode_info_items).
-- Every read joins only Preopsis's items and the episode's own trust's.

-- name: TrustContentItems :many
-- Of these content items, those a trust may use: Preopsis's and its own, with their status.
SELECT id, title, status FROM content_items
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND (org_id IS NULL OR org_id = sqlc.arg(org_id));

-- name: CopyProcedureInfo :exec
-- The procedure's list onto the episode, in its order (retired items too: they show again if
-- brought back). One the episode already has (a clinician added it) keeps its row.
INSERT INTO episode_info_items (episode_id, content_item_id, from_procedure, position)
SELECT sqlc.arg(episode_id), c.id, true, x.ord::integer
FROM procedures p
CROSS JOIN LATERAL jsonb_array_elements_text(p.info_content) WITH ORDINALITY AS x(id, ord)
JOIN content_items c ON c.id::text = x.id AND (c.org_id IS NULL OR c.org_id = p.org_id)
WHERE p.id = sqlc.arg(procedure_id) AND p.org_id = sqlc.arg(org_id)
ON CONFLICT (episode_id, content_item_id) DO NOTHING;

-- name: ClearProcedureInfo :exec
-- The items copied from the episode's procedure, before its new procedure's are copied.
DELETE FROM episode_info_items WHERE episode_id = $1 AND from_procedure;

-- name: ListEpisodeInfo :many
-- The episode's list, in order: the procedure's, then a clinician's in the order added. Retired
-- items come too, for staff; the patient gets the active ones only.
SELECT c.id, c.kind, c.title, c.url, c.length_seconds, c.status, i.from_procedure, i.added_at,
       u.name AS added_by_name
FROM episode_info_items i
JOIN content_items c ON c.id = i.content_item_id AND (c.org_id IS NULL OR c.org_id = sqlc.arg(org_id))
LEFT JOIN users u ON u.id = i.added_by
WHERE i.episode_id = sqlc.arg(episode_id)
ORDER BY i.from_procedure DESC, i.position, i.added_at, c.id;

-- name: AddEpisodeInfo :execrows
-- A clinician's item, last in the list; 0 rows if the episode already has it.
INSERT INTO episode_info_items (episode_id, content_item_id, from_procedure, position, added_by)
VALUES (sqlc.arg(episode_id), sqlc.arg(content_item_id), false,
        (SELECT coalesce(max(position), 0) + 1 FROM episode_info_items WHERE episode_id = sqlc.arg(episode_id) AND NOT from_procedure),
        sqlc.arg(added_by))
ON CONFLICT (episode_id, content_item_id) DO NOTHING;

-- name: RemoveEpisodeInfo :one
-- Removes an item from the episode's list and returns its title, for the General notes; no row if
-- the episode hasn't it.
DELETE FROM episode_info_items i
WHERE i.episode_id = sqlc.arg(episode_id) AND i.content_item_id = sqlc.arg(content_item_id)
RETURNING (SELECT c.title FROM content_items c WHERE c.id = i.content_item_id)::text AS title;

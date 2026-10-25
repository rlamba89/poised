-- The content library (S33): Preopsis's items (org_id NULL) and each trust's own.

-- name: ListPlatformContent :many
-- Preopsis's items, retired ones included (last), for the platform admin's page.
SELECT c.id, c.kind, c.title, c.url, c.length_seconds, c.tags, c.status, c.updated_at,
       coalesce(u.name, '') AS updated_by_name
FROM content_items c
LEFT JOIN users u ON u.id = c.updated_by
WHERE c.org_id IS NULL
ORDER BY c.status, lower(c.title), c.id;

-- name: ListTrustContent :many
-- The trust's library page: its own items, retired ones included, then Preopsis's active ones.
SELECT c.id, (c.org_id IS NULL)::boolean AS platform, c.kind, c.title, c.url, c.length_seconds, c.tags, c.status,
       c.updated_at, coalesce(u.name, '') AS updated_by_name
FROM content_items c
LEFT JOIN users u ON u.id = c.updated_by
WHERE c.org_id = sqlc.arg(org_id) OR (c.org_id IS NULL AND c.status = 'active')
ORDER BY (c.org_id IS NULL), c.status, lower(c.title), c.id;

-- name: SearchContent :many
-- What an author may pick (S34's picker): active items, Preopsis's and this trust's, with the tag
-- (any case; '' any tag) and the search in the title or a tag (any case; '' everything), by title.
SELECT c.id, (c.org_id IS NULL)::boolean AS platform, c.kind, c.title, c.url, c.length_seconds, c.tags
FROM content_items c
WHERE (c.org_id IS NULL OR c.org_id = sqlc.arg(org_id)) AND c.status = 'active'
  AND (sqlc.arg(tag)::text = '' OR EXISTS (SELECT 1 FROM unnest(c.tags) t WHERE lower(t) = lower(sqlc.arg(tag)::text)))
  AND (sqlc.arg(search)::text = ''
       OR strpos(lower(c.title), lower(sqlc.arg(search)::text)) > 0
       OR EXISTS (SELECT 1 FROM unnest(c.tags) t WHERE strpos(lower(t), lower(sqlc.arg(search)::text)) > 0))
ORDER BY lower(c.title), c.id;

-- name: GetContentItem :one
-- One item, if it's Preopsis's (owner NULL) or this owner's; for a change. A platform admin passes
-- no owner, so only Preopsis's are found.
SELECT id, org_id, kind, title, url, length_seconds, tags, status
FROM content_items
WHERE id = sqlc.arg(id) AND (org_id IS NULL OR org_id = sqlc.narg(owner))
FOR UPDATE;

-- name: CreateContentItem :one
INSERT INTO content_items (org_id, kind, title, url, length_seconds, tags, created_by, updated_by)
VALUES (sqlc.narg(org_id), sqlc.arg(kind), sqlc.arg(title), sqlc.arg(url), sqlc.narg(length_seconds), sqlc.arg(tags),
        sqlc.arg(by), sqlc.arg(by))
RETURNING id;

-- name: UpdateContentItem :exec
UPDATE content_items
SET title = sqlc.arg(title), url = sqlc.arg(url), length_seconds = sqlc.narg(length_seconds), tags = sqlc.arg(tags),
    status = sqlc.arg(status), updated_by = sqlc.arg(by), updated_at = now()
WHERE id = sqlc.arg(id);

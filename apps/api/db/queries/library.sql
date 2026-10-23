-- The question library (S24): the platform's starter set (org_id NULL) and each trust's own.

-- name: ListLibraryQuestions :many
-- The platform's library and this trust's, matching the search (in the title or a tag, any case;
-- '' matches everything), by title.
SELECT l.id, (l.org_id IS NULL)::boolean AS platform, l.key, l.title, l.element, l.tags, l.created_at,
       coalesce(u.name, '') AS created_by_name
FROM library_questions l
LEFT JOIN users u ON u.id = l.created_by
WHERE (l.org_id IS NULL OR l.org_id = sqlc.arg(org_id))
  AND (sqlc.arg(search)::text = ''
       OR strpos(lower(l.title), lower(sqlc.arg(search)::text)) > 0
       OR EXISTS (SELECT 1 FROM unnest(l.tags) t WHERE strpos(lower(t), lower(sqlc.arg(search)::text)) > 0))
ORDER BY lower(l.title), l.key, l.id;

-- name: LibraryKeyTaken :one
-- Whether the platform's library or this trust's already has the key.
SELECT EXISTS (SELECT 1 FROM library_questions WHERE key = sqlc.arg(key) AND (org_id IS NULL OR org_id = sqlc.arg(org_id)));

-- name: CreateLibraryQuestion :one
INSERT INTO library_questions (org_id, key, title, element, tags, created_by)
VALUES (sqlc.arg(org_id), sqlc.arg(key), sqlc.arg(title), sqlc.arg(element), sqlc.arg(tags), sqlc.arg(created_by))
RETURNING id;

-- name: GetLibraryQuestionOwner :one
-- Whose the item is (NULL: the platform's), if this trust can see it; no row otherwise.
SELECT org_id FROM library_questions
WHERE id = sqlc.arg(id) AND (org_id IS NULL OR org_id = sqlc.arg(org_id));

-- name: DeleteLibraryQuestion :execrows
-- Only the trust's own: the platform's are the seed's.
DELETE FROM library_questions WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: ListPlatformLibraryElements :many
-- The platform library's questions (org_id NULL, the same for every trust), for pre-fill from
-- another trust (S26): only their keys may match another trust's question by key.
SELECT element FROM library_questions WHERE org_id IS NULL;

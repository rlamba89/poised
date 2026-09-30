-- name: SearchCodes :many
-- CLN-09: part of the code or description, within one code set, active only.
-- ILIKE '%…%' is served by the pg_trgm GIN indexes.
SELECT c.id, c.code_set, c.code, c.description, c.full_name, cat.name AS category_name
FROM codes c
LEFT JOIN categories cat ON cat.id = c.category_id
WHERE c.status = 'active'
  AND c.code_set = sqlc.arg(code_set)
  AND (c.code ILIKE sqlc.arg(query)::text || '%' OR c.description ILIKE '%' || sqlc.arg(query)::text || '%')
ORDER BY (lower(c.code) = lower(sqlc.arg(query)::text)) DESC,
         (c.description ILIKE sqlc.arg(query)::text || '%') DESC,
         similarity(c.description, sqlc.arg(query)::text) DESC,
         c.description
LIMIT 20;

-- name: ListCategories :many
SELECT id, name, note_only FROM categories ORDER BY note_only, name;

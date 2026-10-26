-- The trust's procedure list (PRC-01, S19). hospital_id NULL: offered at every hospital of the
-- trust; otherwise at that hospital only. Retired procedures stay listed for managing, and on the
-- episodes that have them, but aren't offered for new episodes.

-- name: ListProcedures :many
-- The trust's procedures that the caller sees: every one for a whole-trust member (all_hospitals),
-- else the trust-wide ones and those of the hospitals listed. Retired ones included.
SELECT p.id, p.name, p.code, p.specialty, p.hospital_id, h.name AS hospital_name,
       p.default_questionnaire_id, q.name AS default_questionnaire_name,
       p.info_content, p.proms_plan, p.retired, p.updated_at,
       -- The Information list's items (S36), in order: each one's kind, title and status, so the page
       -- can name a retired one. Only Preopsis's and this trust's items are ever found.
       (SELECT coalesce(jsonb_agg(jsonb_build_object('id', c.id, 'kind', c.kind, 'title', c.title, 'status', c.status) ORDER BY x.ord), '[]'::jsonb)
        FROM jsonb_array_elements_text(p.info_content) WITH ORDINALITY AS x(id, ord)
        JOIN content_items c ON c.id::text = x.id AND (c.org_id IS NULL OR c.org_id = p.org_id))::jsonb AS info_items
FROM procedures p
LEFT JOIN hospitals h ON h.id = p.hospital_id
LEFT JOIN questionnaires q ON q.id = p.default_questionnaire_id
WHERE p.org_id = sqlc.arg(org_id)
  AND (p.hospital_id IS NULL OR sqlc.arg(all_hospitals)::boolean OR p.hospital_id = ANY (sqlc.arg(hospital_ids)::uuid[]))
ORDER BY p.retired, lower(p.name), p.id;

-- name: ListProceduresOffered :many
-- The procedures a new episode at this hospital can be given: the trust's and this hospital's
-- own, not retired.
SELECT p.id, p.name, p.code, p.specialty, p.hospital_id, p.default_questionnaire_id
FROM procedures p
WHERE p.org_id = sqlc.arg(org_id) AND (p.hospital_id IS NULL OR p.hospital_id = sqlc.arg(hospital_id)::uuid)
  AND NOT p.retired
ORDER BY lower(p.name), p.id;

-- name: OfferedProcedureName :one
-- The procedure's name if an episode at this hospital can be given it now (ListProceduresOffered);
-- no row if it can't.
SELECT name FROM procedures
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
  AND (hospital_id IS NULL OR hospital_id = sqlc.arg(hospital_id)::uuid) AND NOT retired;

-- name: GetProcedure :one
-- One of the trust's procedures, as stored, for a change.
SELECT id, name, code, specialty, hospital_id, default_questionnaire_id, info_content, retired
FROM procedures WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
FOR UPDATE;

-- name: CreateProcedure :one
INSERT INTO procedures (org_id, hospital_id, name, code, specialty, default_questionnaire_id, info_content, updated_by)
VALUES (sqlc.arg(org_id), sqlc.arg(hospital_id), sqlc.arg(name), sqlc.arg(code), sqlc.arg(specialty),
        sqlc.arg(default_questionnaire_id), sqlc.arg(info_content), sqlc.arg(updated_by))
RETURNING id;

-- name: UpdateProcedure :exec
UPDATE procedures
SET hospital_id = sqlc.arg(hospital_id), name = sqlc.arg(name), code = sqlc.arg(code), specialty = sqlc.arg(specialty),
    default_questionnaire_id = sqlc.arg(default_questionnaire_id), info_content = sqlc.arg(info_content), retired = sqlc.arg(retired),
    updated_by = sqlc.arg(updated_by), updated_at = now()
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: GetHospitalAndOrgSettings :one
-- A hospital's stored settings and its trust's, for its effective values (the default HQ).
SELECT o.settings AS org_settings, h.settings AS hospital_settings
FROM hospitals h JOIN orgs o ON o.id = h.org_id
WHERE h.id = sqlc.arg(hospital_id) AND h.org_id = sqlc.arg(org_id);

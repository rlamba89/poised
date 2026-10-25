-- +goose Up
-- The content library (plan C5, session S33; INF-05, PX-13, A-6): videos and leaflets that
-- questionnaires (S34's Video element) and the patient's Information (S36) point to.
--   - org_id NULL: Preopsis's own, managed by platform admins (/api/admin/content) and seen by
--     every trust; org_id set: the trust's own, managed by its whole-trust admins.
--   - A video is a YouTube or Vimeo link, never a file of ours (A-6); its length is typed by hand.
--     A leaflet is an https link to a PDF on the web until plan C8 brings uploads (S33 answers 2).
--     Go checks the links (internal/content).
--   - Tags are free text; PX-13's checklist topics are offered as ready-made ones (answers 3).
--   - An item is never deleted, only retired: a retired item isn't offered any more, and anything
--     that already copied it keeps its copy (S34: a questionnaire keeps its own copy of a video).
-- No patient data.
CREATE TABLE content_items (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         uuid REFERENCES orgs ON DELETE CASCADE,
    kind           text NOT NULL CHECK (kind IN ('video', 'leaflet')),
    title          text NOT NULL CHECK (title <> ''),
    url            text NOT NULL CHECK (url LIKE 'https://%'),
    length_seconds integer CHECK (length_seconds > 0),
    tags           text[] NOT NULL DEFAULT '{}',
    status         text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'retired')),
    created_by     uuid REFERENCES users,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid REFERENCES users,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    -- A video has its length; a leaflet has none.
    CHECK ((kind = 'video') = (length_seconds IS NOT NULL))
);
CREATE INDEX content_items_org ON content_items (org_id);

-- The platform's starter questions can be retired by a platform admin (S33 answers 4): a retired
-- one isn't offered by Insert from library or "Same as…" any more. Questions already inserted keep
-- their copies and their key, and pre-fill across trusts still matches by it. The seed (which
-- adds new ones) leaves this alone. A trust's own library questions are removed, not retired (S24).
ALTER TABLE library_questions ADD COLUMN retired boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE library_questions DROP COLUMN retired;
DROP TABLE content_items;

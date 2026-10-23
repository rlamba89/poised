-- +goose Up
-- The question library (plan C4, session S24; §6 of saas-requirements.md, PNL-06 as changed).
-- A library question is one SurveyJS element, kept with its key (its `name`) and its options'
-- values, so every questionnaire it's inserted into shares the key and pre-fill (S25) can copy a
-- patient's earlier answer to it.
--   - org_id NULL: the platform's starter library, seeded into every deployment like the codes;
--     only the seed changes it (S24 answers 2; a superadmin screen comes with C5).
--   - org_id set: the trust's own, saved from the editor by whole-trust super clinicians and
--     admins, and removed by them. Removing one changes no questionnaire it was inserted into.
-- A key is unique per owner; a trust's item can't reuse a platform key (checked in Go, since the
-- platform's are only added by the seed).
-- title is the question text in the default language, read from the element by Go, for search.
CREATE TABLE library_questions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     uuid REFERENCES orgs ON DELETE CASCADE,
    key        text NOT NULL CHECK (key <> ''),
    title      text NOT NULL DEFAULT '',
    element    jsonb NOT NULL CHECK (jsonb_typeof(element) = 'object' AND element ->> 'name' = key),
    tags       text[] NOT NULL DEFAULT '{}',
    created_by uuid REFERENCES users,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX library_questions_platform_key ON library_questions (key) WHERE org_id IS NULL;
CREATE UNIQUE INDEX library_questions_trust_key ON library_questions (org_id, key) WHERE org_id IS NOT NULL;

-- +goose Down
DROP TABLE library_questions;

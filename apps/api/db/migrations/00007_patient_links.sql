-- +goose Up
-- Patient sign-in (auth-first Step 5, PAT-04/05): a link opens the HQ only after the patient
-- confirms their date of birth. The token is kept twice and never in plain text: its SHA-256,
-- to look it up, and sealed with AES-GCM (key LINK_KEY, outside the database) so clinicians
-- can see the link again.
CREATE TABLE login_links (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash          bytea NOT NULL UNIQUE,
    token_sealed        bytea,  -- NULL for links made before sealing: they work but can't be shown
    episode_id          uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    failed_dob_attempts int NOT NULL DEFAULT 0,
    locked_at           timestamptz,  -- set by the fifth wrong date of birth
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX login_links_episode_idx ON login_links (episode_id);

-- Existing links keep working: their hash moves here and the plain token goes.
INSERT INTO login_links (token_hash, episode_id)
SELECT sha256(convert_to(patient_token, 'UTF8')), id FROM episodes;
ALTER TABLE episodes DROP COLUMN patient_token;

-- Patient sessions share the sessions table: a session belongs to a user (staff) or to an
-- episode (a patient who signed in with its link).
ALTER TABLE sessions ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE sessions ADD COLUMN episode_id uuid REFERENCES episodes ON DELETE CASCADE;
ALTER TABLE sessions ADD CONSTRAINT sessions_owner_check CHECK ((user_id IS NULL) <> (episode_id IS NULL));
CREATE INDEX sessions_episode_idx ON sessions (episode_id);

-- +goose Down
DELETE FROM sessions WHERE episode_id IS NOT NULL;
DROP INDEX sessions_episode_idx;
ALTER TABLE sessions DROP CONSTRAINT sessions_owner_check;
ALTER TABLE sessions DROP COLUMN episode_id;
ALTER TABLE sessions ALTER COLUMN user_id SET NOT NULL;

-- The plain tokens are gone, so every episode gets a new one: old links stop working.
-- (md5 of random() is not secret-grade; this is only for rolling back a development database.)
ALTER TABLE episodes ADD COLUMN patient_token text UNIQUE;
UPDATE episodes SET patient_token = substr(md5(random()::text) || md5(random()::text), 1, 43);
ALTER TABLE episodes ALTER COLUMN patient_token SET NOT NULL;
DROP TABLE login_links;

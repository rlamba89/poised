-- +goose Up
-- Episode forms (plan C3, session S18). An episode's questionnaires become forms of its own: today
-- only the HQ, which every episode gets when it's created; C7 adds assessments and C9 PROMs, with
-- their due and closing dates. The answers move from the episode to the form, so one episode can
-- hold several forms' answers. The episode's HQ version and the patient's submit move with them.
--
-- A reshape, not a data migration (S18 answers 2, Rahul 20 Oct 2026): every episode before this
-- is made up, so they're removed, with everything of theirs (answers, General notes, messages,
-- follow-up jobs, and the invite links with the sessions opened from them). Patient records,
-- accounts, sign-in links and questionnaires are kept. `make seed` puts back the demo data.
DELETE FROM patient_sessions WHERE link_id IN (SELECT id FROM login_links WHERE episode_id IS NOT NULL);
DELETE FROM login_links WHERE episode_id IS NOT NULL;
DELETE FROM episodes; -- and, by ON DELETE CASCADE, their answers, events, messages and jobs

-- status: open until the patient (or staff for them, PAT-09) sends it, then submitted, with when
-- and, for staff, by whom (NULL: the patient). The answers on a submitted form are frozen.
CREATE TABLE episode_forms (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id   uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    version_id   uuid NOT NULL REFERENCES questionnaire_versions, -- a published version
    kind         text NOT NULL CHECK (kind IN ('hq', 'assessment', 'proms')),
    status       text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'submitted')),
    due_at       timestamptz, -- C7, C9
    closes_at    timestamptz, -- C9
    submitted_at timestamptz,
    submitted_by uuid REFERENCES users,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT episode_forms_submitted_check CHECK ((status = 'submitted') = (submitted_at IS NOT NULL))
);
-- One HQ per episode: the routes find "the episode's HQ" by this.
CREATE UNIQUE INDEX episode_forms_hq_key ON episode_forms (episode_id) WHERE kind = 'hq';
CREATE INDEX episode_forms_episode_idx ON episode_forms (episode_id, created_at);

-- One row per Question Set per actor, as before (00004, and the revision from 00005), now per form.
DROP TABLE episode_answers;
CREATE TABLE episode_answers (
    episode_form_id uuid NOT NULL REFERENCES episode_forms ON DELETE CASCADE,
    chapter_id      uuid NOT NULL REFERENCES chapters,
    actor           text NOT NULL CHECK (actor IN ('patient', 'clinician')),
    data            jsonb NOT NULL DEFAULT '{}',  -- SurveyJS survey.data
    updated_by      uuid REFERENCES users,        -- the patient's row: NULL, or staff entering it for them
    updated_at      timestamptz NOT NULL DEFAULT now(),
    validated_at    timestamptz,                  -- clinician rows: "Validated by … on …"
    revision        int NOT NULL DEFAULT 0,       -- counts the saves, for the stale-save 409 (S02a)
    PRIMARY KEY (episode_form_id, chapter_id, actor)
);

ALTER TABLE episodes DROP COLUMN version_id, DROP COLUMN patient_submitted_at, DROP COLUMN patient_submitted_by;

-- +goose Down
-- Back to one HQ per episode, with the answers on the episode. Episodes are removed again.
DELETE FROM patient_sessions WHERE link_id IN (SELECT id FROM login_links WHERE episode_id IS NOT NULL);
DELETE FROM login_links WHERE episode_id IS NOT NULL;
DELETE FROM episodes;
DROP TABLE episode_answers;
DROP TABLE episode_forms;
ALTER TABLE episodes
    ADD COLUMN version_id uuid NOT NULL REFERENCES questionnaire_versions,
    ADD COLUMN patient_submitted_at timestamptz,
    ADD COLUMN patient_submitted_by uuid REFERENCES users;
CREATE TABLE episode_answers (
    episode_id   uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    chapter_id   uuid NOT NULL REFERENCES chapters,
    actor        text NOT NULL CHECK (actor IN ('patient', 'clinician')),
    data         jsonb NOT NULL DEFAULT '{}',
    updated_by   uuid REFERENCES users,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    validated_at timestamptz,
    revision     int NOT NULL DEFAULT 0,
    PRIMARY KEY (episode_id, chapter_id, actor)
);

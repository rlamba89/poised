-- +goose Up
-- The episode workflow (docs/plan-workflow.md): patients, episodes, their answers and events.

ALTER TABLE memberships DROP CONSTRAINT memberships_role_check;
ALTER TABLE memberships ADD CONSTRAINT memberships_role_check
    CHECK (role IN ('viewer', 'author', 'reviewer', 'publisher', 'hospital_admin', 'clinician'));

-- Made-up patients only (NFR-03): this is still a local tool.
CREATE TABLE patients (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id     uuid NOT NULL REFERENCES hospitals ON DELETE CASCADE,
    first_name      text NOT NULL,
    last_name       text NOT NULL,
    date_of_birth   date NOT NULL,
    sex             text NOT NULL CHECK (sex IN ('female', 'male', 'other', 'unknown')),
    hospital_number text NOT NULL DEFAULT '',
    phone           text NOT NULL DEFAULT '',
    email           text NOT NULL DEFAULT ''
);
CREATE INDEX patients_hospital_idx ON patients (hospital_id, last_name, first_name);

-- Statuses are Lifebox's Full HQ states (statemodel.go) with readable keys.
CREATE TABLE episodes (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id          uuid NOT NULL REFERENCES hospitals ON DELETE CASCADE,
    patient_id           uuid NOT NULL REFERENCES patients,
    version_id           uuid NOT NULL REFERENCES questionnaire_versions,  -- a published version
    status               text NOT NULL DEFAULT 'hq_not_complete'
                         CHECK (status IN ('hq_not_complete', 'ready_for_review', 'ready_for_poa',
                                           'poa_complete', 'on_hold', 'not_ready', 'ready_for_admission')),
    procedure            text NOT NULL DEFAULT '',
    anaesthetic          text NOT NULL DEFAULT '',
    consultant           text NOT NULL DEFAULT '',
    nurse_asa            int CHECK (nurse_asa BETWEEN 1 AND 6),
    anaesthetist_asa     int CHECK (anaesthetist_asa BETWEEN 1 AND 6),
    patient_token        text NOT NULL UNIQUE,  -- the patient's link; 32 random bytes, base64url
    patient_submitted_at timestamptz,
    review_completed_by  uuid REFERENCES users,
    review_completed_at  timestamptz,
    created_by           uuid NOT NULL REFERENCES users,
    created_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX episodes_hospital_idx ON episodes (hospital_id, created_at DESC);

-- One row per Question Set per actor (Lifebox AnswerSubmission.actorType). The patient row is
-- frozen once the episode is submitted; the clinician row starts as a copy of it and is final.
CREATE TABLE episode_answers (
    episode_id   uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    chapter_id   uuid NOT NULL REFERENCES chapters,
    actor        text NOT NULL CHECK (actor IN ('patient', 'clinician')),
    data         jsonb NOT NULL DEFAULT '{}',  -- SurveyJS survey.data
    updated_by   uuid REFERENCES users,        -- null for the patient
    updated_at   timestamptz NOT NULL DEFAULT now(),
    validated_at timestamptz,                  -- clinician rows: "Validated by … on …"
    PRIMARY KEY (episode_id, chapter_id, actor)
);

-- "General notes" on the POA Summary: automatic events and clinicians' comments. Append-only.
CREATE TABLE episode_events (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('event', 'comment')),
    text       text NOT NULL,
    user_id    uuid REFERENCES users,  -- null for the patient
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX episode_events_episode_idx ON episode_events (episode_id, created_at);

-- +goose Down
DROP TABLE episode_events;
DROP TABLE episode_answers;
DROP TABLE episodes;
DROP TABLE patients;
DELETE FROM memberships WHERE role = 'clinician';
ALTER TABLE memberships DROP CONSTRAINT memberships_role_check;
ALTER TABLE memberships ADD CONSTRAINT memberships_role_check
    CHECK (role IN ('viewer', 'author', 'reviewer', 'publisher', 'hospital_admin'));

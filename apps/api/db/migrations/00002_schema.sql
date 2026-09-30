-- +goose Up
CREATE TABLE hospitals (
    id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL
);

CREATE TABLE users (
    id    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name  text NOT NULL,
    email text NOT NULL UNIQUE
);

CREATE TABLE memberships (
    user_id     uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    hospital_id uuid NOT NULL REFERENCES hospitals ON DELETE CASCADE,
    role        text NOT NULL CHECK (role IN ('viewer', 'author', 'reviewer', 'publisher', 'hospital_admin')),
    PRIMARY KEY (user_id, hospital_id, role)
);

CREATE TABLE questionnaires (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id uuid NOT NULL REFERENCES hospitals,
    name        text NOT NULL,
    description text NOT NULL,
    created_by  uuid NOT NULL REFERENCES users,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX questionnaires_hospital_idx ON questionnaires (hospital_id);

-- Only 'draft' is used in the slice; the other statuses are LCY-01's, for later.
CREATE TABLE questionnaire_versions (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    questionnaire_id uuid NOT NULL REFERENCES questionnaires ON DELETE CASCADE,
    version_no       int NOT NULL,
    status           text NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft', 'in_review', 'approved', 'published', 'retired')),
    updated_by       uuid NOT NULL REFERENCES users,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (questionnaire_id, version_no)
);

CREATE TABLE chapters (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id  uuid NOT NULL REFERENCES questionnaire_versions ON DELETE CASCADE,
    position    int NOT NULL,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    icon        text NOT NULL DEFAULT 'file-text',  -- Tabler icon name
    audience    text NOT NULL DEFAULT 'patient'
                CHECK (audience IN ('patient', 'clinician', 'clinician_document')),
    content     jsonb NOT NULL DEFAULT '{}',       -- the SurveyJS JSON
    revision    int NOT NULL DEFAULT 0,            -- +1 on every content save (LCY-04)
    updated_by  uuid NOT NULL REFERENCES users,
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX chapters_version_idx ON chapters (version_id, position);

-- Lifebox's 13 categories keep their Lifebox UUIDs, plus "Unassigned" (note_only).
CREATE TABLE categories (
    id        uuid PRIMARY KEY,
    name      text NOT NULL UNIQUE,
    note_only boolean NOT NULL DEFAULT false
);

CREATE TABLE codes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code_set    text NOT NULL CHECK (code_set IN ('SNOMED', 'ICD10')),
    code        text NOT NULL,
    description text NOT NULL,
    full_name   text NOT NULL,
    category_id uuid REFERENCES categories,  -- some Lifebox codes have none
    billable    boolean NOT NULL DEFAULT false,
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'retired')),
    UNIQUE (code_set, code)
);
CREATE INDEX codes_code_trgm_idx ON codes USING gin (code gin_trgm_ops);
CREATE INDEX codes_description_trgm_idx ON codes USING gin (description gin_trgm_ops);

-- +goose Down
DROP TABLE codes;
DROP TABLE categories;
DROP TABLE chapters;
DROP TABLE questionnaire_versions;
DROP TABLE questionnaires;
DROP TABLE memberships;
DROP TABLE users;
DROP TABLE hospitals;

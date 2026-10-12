-- +goose Up
-- Trusts (plan C1, session S08). A trust (org) is the tenant: it owns hospitals, patient records
-- and content. Memberships give a user a role in a whole trust (hospital_id NULL) or in one
-- hospital. Roles become one ladder, clinician < super_clinician < admin (A-5).
--
-- Rows from before this migration are carried over: each existing hospital becomes the only
-- hospital of a trust of its own, and its content, patients, episodes and memberships go with it.
-- `make seed` then shapes the demo trusts.

CREATE TABLE orgs (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    code       text NOT NULL UNIQUE,  -- a short code, e.g. for support
    status     text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    settings   jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- timezone: an IANA zone; NULL means the deployment's default (DEFAULT_TIME_ZONE, A-20).
ALTER TABLE hospitals
    ADD COLUMN org_id   uuid REFERENCES orgs ON DELETE CASCADE,
    ADD COLUMN status   text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    ADD COLUMN settings jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN timezone text;

INSERT INTO orgs (name, code)
SELECT name, 'migrated-' || id FROM hospitals;
UPDATE hospitals h SET org_id = o.id FROM orgs o WHERE o.code = 'migrated-' || h.id;
ALTER TABLE hospitals ALTER COLUMN org_id SET NOT NULL;
-- Lets other tables check that a hospital they name is in their trust.
ALTER TABLE hospitals ADD CONSTRAINT hospitals_org_id_id_key UNIQUE (org_id, id);

ALTER TABLE users ADD COLUMN cognito_sub text UNIQUE;  -- set at first Cognito sign-in (S12)

-- Superadmins (A-21): filled by hand, never from a screen. Locally the seed adds one.
CREATE TABLE platform_admins (
    user_id    uuid PRIMARY KEY REFERENCES users ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Memberships: one role per user per trust or hospital. Old roles map to the ladder, and where a
-- user had several roles in a hospital, the highest is kept.
ALTER TABLE memberships DROP CONSTRAINT memberships_pkey;
ALTER TABLE memberships DROP CONSTRAINT memberships_role_check;
ALTER TABLE memberships ADD COLUMN org_id uuid;
UPDATE memberships m SET org_id = h.org_id FROM hospitals h WHERE h.id = m.hospital_id;
UPDATE memberships SET role = CASE
    WHEN role = 'hospital_admin' THEN 'admin'
    WHEN role IN ('author', 'reviewer', 'publisher') THEN 'super_clinician'
    ELSE 'clinician' END;
DELETE FROM memberships m USING memberships other
WHERE other.user_id = m.user_id AND other.hospital_id = m.hospital_id
  AND (array_position(ARRAY['clinician', 'super_clinician', 'admin'], other.role),  m.ctid)
    > (array_position(ARRAY['clinician', 'super_clinician', 'admin'], m.role), other.ctid);
ALTER TABLE memberships
    ALTER COLUMN org_id SET NOT NULL,
    ALTER COLUMN hospital_id DROP NOT NULL,
    DROP CONSTRAINT memberships_hospital_id_fkey,
    ADD CONSTRAINT memberships_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs ON DELETE CASCADE,
    -- a hospital membership must be in the membership's trust
    ADD CONSTRAINT memberships_hospital_fkey FOREIGN KEY (org_id, hospital_id)
        REFERENCES hospitals (org_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT memberships_role_check CHECK (role IN ('clinician', 'super_clinician', 'admin')),
    ADD CONSTRAINT memberships_user_scope_key UNIQUE NULLS NOT DISTINCT (user_id, org_id, hospital_id);
CREATE INDEX memberships_org_idx ON memberships (org_id);

-- Content is the trust's (hospital_id NULL), or one hospital's. Everything so far becomes the
-- trust's, so all of a trust's hospitals use it.
ALTER TABLE questionnaires ADD COLUMN org_id uuid REFERENCES orgs ON DELETE CASCADE;
UPDATE questionnaires q SET org_id = h.org_id FROM hospitals h WHERE h.id = q.hospital_id;
DROP INDEX questionnaires_hospital_idx;
ALTER TABLE questionnaires
    ALTER COLUMN org_id SET NOT NULL,
    ALTER COLUMN hospital_id DROP NOT NULL,
    DROP CONSTRAINT questionnaires_hospital_id_fkey,
    ADD CONSTRAINT questionnaires_hospital_fkey FOREIGN KEY (org_id, hospital_id) REFERENCES hospitals (org_id, id);
UPDATE questionnaires SET hospital_id = NULL;
CREATE INDEX questionnaires_org_idx ON questionnaires (org_id);

ALTER TABLE option_lists ADD COLUMN org_id uuid REFERENCES orgs ON DELETE CASCADE;
UPDATE option_lists l SET org_id = h.org_id FROM hospitals h WHERE h.id = l.hospital_id;
ALTER TABLE option_lists
    ALTER COLUMN org_id SET NOT NULL,
    ALTER COLUMN hospital_id DROP NOT NULL,
    DROP CONSTRAINT option_lists_hospital_id_name_key,
    DROP CONSTRAINT option_lists_hospital_id_fkey,
    ADD CONSTRAINT option_lists_hospital_fkey FOREIGN KEY (org_id, hospital_id)
        REFERENCES hospitals (org_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT option_lists_org_id_name_key UNIQUE (org_id, name);
UPDATE option_lists SET hospital_id = NULL;

-- A patient record is the trust's: staff at any of its hospitals can open an episode for it,
-- but each hospital sees only its own episodes (S08 answers, 12 Oct). S13 reshapes it further.
ALTER TABLE patients ADD COLUMN org_id uuid REFERENCES orgs ON DELETE CASCADE;
UPDATE patients p SET org_id = h.org_id FROM hospitals h WHERE h.id = p.hospital_id;
DROP INDEX patients_hospital_idx;
ALTER TABLE patients
    ALTER COLUMN org_id SET NOT NULL,
    DROP COLUMN hospital_id,
    ADD CONSTRAINT patients_org_id_id_key UNIQUE (org_id, id);
CREATE INDEX patients_org_idx ON patients (org_id, last_name, first_name);

-- An episode is at one hospital, for a patient of the same trust.
ALTER TABLE episodes ADD COLUMN org_id uuid;
UPDATE episodes e SET org_id = h.org_id FROM hospitals h WHERE h.id = e.hospital_id;
ALTER TABLE episodes
    ALTER COLUMN org_id SET NOT NULL,
    DROP CONSTRAINT episodes_hospital_id_fkey,
    DROP CONSTRAINT episodes_patient_id_fkey,
    ADD CONSTRAINT episodes_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs ON DELETE CASCADE,
    ADD CONSTRAINT episodes_hospital_fkey FOREIGN KEY (org_id, hospital_id)
        REFERENCES hospitals (org_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT episodes_patient_fkey FOREIGN KEY (org_id, patient_id) REFERENCES patients (org_id, id);
CREATE INDEX episodes_org_idx ON episodes (org_id, created_at DESC);

-- +goose Down
-- Back to one hospital per row and the old roles. Trust-wide memberships and anything owned by a
-- trust with no hospital are lost; trust content goes to the trust's first hospital.
DROP INDEX episodes_org_idx;
ALTER TABLE episodes
    DROP CONSTRAINT episodes_patient_fkey,
    DROP CONSTRAINT episodes_hospital_fkey,
    DROP CONSTRAINT episodes_org_id_fkey,
    ADD CONSTRAINT episodes_hospital_id_fkey FOREIGN KEY (hospital_id) REFERENCES hospitals ON DELETE CASCADE,
    ADD CONSTRAINT episodes_patient_id_fkey FOREIGN KEY (patient_id) REFERENCES patients,
    DROP COLUMN org_id;

DROP INDEX patients_org_idx;
ALTER TABLE patients DROP CONSTRAINT patients_org_id_id_key, ADD COLUMN hospital_id uuid REFERENCES hospitals ON DELETE CASCADE;
UPDATE patients p SET hospital_id = coalesce(
    (SELECT e.hospital_id FROM episodes e WHERE e.patient_id = p.id LIMIT 1),
    (SELECT h.id FROM hospitals h WHERE h.org_id = p.org_id ORDER BY h.name LIMIT 1));
DELETE FROM patients WHERE hospital_id IS NULL;
ALTER TABLE patients ALTER COLUMN hospital_id SET NOT NULL, DROP COLUMN org_id;
CREATE INDEX patients_hospital_idx ON patients (hospital_id, last_name, first_name);

UPDATE option_lists l SET hospital_id = (SELECT h.id FROM hospitals h WHERE h.org_id = l.org_id ORDER BY h.name LIMIT 1)
WHERE hospital_id IS NULL;
DELETE FROM option_lists WHERE hospital_id IS NULL;
ALTER TABLE option_lists
    DROP CONSTRAINT option_lists_org_id_name_key,
    DROP CONSTRAINT option_lists_hospital_fkey,
    ALTER COLUMN hospital_id SET NOT NULL,
    ADD CONSTRAINT option_lists_hospital_id_fkey FOREIGN KEY (hospital_id) REFERENCES hospitals ON DELETE CASCADE,
    ADD CONSTRAINT option_lists_hospital_id_name_key UNIQUE (hospital_id, name),
    DROP COLUMN org_id;

DROP INDEX questionnaires_org_idx;
UPDATE questionnaires q SET hospital_id = (SELECT h.id FROM hospitals h WHERE h.org_id = q.org_id ORDER BY h.name LIMIT 1)
WHERE hospital_id IS NULL;
ALTER TABLE questionnaires
    DROP CONSTRAINT questionnaires_hospital_fkey,
    ALTER COLUMN hospital_id SET NOT NULL,
    ADD CONSTRAINT questionnaires_hospital_id_fkey FOREIGN KEY (hospital_id) REFERENCES hospitals,
    DROP COLUMN org_id;
CREATE INDEX questionnaires_hospital_idx ON questionnaires (hospital_id);

DROP INDEX memberships_org_idx;
DELETE FROM memberships WHERE hospital_id IS NULL;
ALTER TABLE memberships
    DROP CONSTRAINT memberships_user_scope_key,
    DROP CONSTRAINT memberships_role_check,
    DROP CONSTRAINT memberships_hospital_fkey,
    DROP CONSTRAINT memberships_org_id_fkey,
    ALTER COLUMN hospital_id SET NOT NULL,
    ADD CONSTRAINT memberships_hospital_id_fkey FOREIGN KEY (hospital_id) REFERENCES hospitals ON DELETE CASCADE,
    DROP COLUMN org_id;
UPDATE memberships SET role = CASE role WHEN 'admin' THEN 'hospital_admin' WHEN 'super_clinician' THEN 'author' ELSE 'clinician' END;
ALTER TABLE memberships
    ADD CONSTRAINT memberships_role_check
        CHECK (role IN ('viewer', 'author', 'reviewer', 'publisher', 'hospital_admin', 'clinician')),
    ADD PRIMARY KEY (user_id, hospital_id, role);

DROP TABLE platform_admins;
ALTER TABLE users DROP COLUMN cognito_sub;

ALTER TABLE hospitals
    DROP CONSTRAINT hospitals_org_id_id_key,
    DROP COLUMN timezone,
    DROP COLUMN settings,
    DROP COLUMN status,
    DROP COLUMN org_id;
DROP TABLE orgs;

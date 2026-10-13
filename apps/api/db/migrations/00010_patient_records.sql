-- +goose Up
-- Patient records (plan C2, session S13; PAT-01, PAT-01a, PAT-01b, PAT-02). A record is the
-- trust's, identified by its national ID when it has one (A-20: national_id + national_id_type,
-- validated in Go by internal/nationalid; only the NHS number is built). Episodes hang off it at
-- any of the trust's hospitals, and the hospital number moves to the episode: it's that site's
-- label for the patient, never their identity (PAT-01b).
--
-- Existing records are reshaped in place (a fresh app, A-1): each episode takes its patient's
-- hospital number, and a phone that isn't already in E.164 is dropped (made-up data only).

ALTER TABLE episodes ADD COLUMN hospital_number text NOT NULL DEFAULT '';
UPDATE episodes e SET hospital_number = p.hospital_number FROM patients p WHERE p.id = e.patient_id;
ALTER TABLE patients DROP COLUMN hospital_number;

ALTER TABLE patients RENAME COLUMN phone TO mobile;
UPDATE patients SET mobile = '' WHERE mobile !~ '^\+[1-9][0-9]{6,14}$';
UPDATE patients SET email = lower(trim(email));

ALTER TABLE patients
    ADD COLUMN national_id      text,
    ADD COLUMN national_id_type text,
    ADD COLUMN account_id       uuid,  -- the patient's sign-in account (S15 adds the table and its key)
    ADD COLUMN created_at       timestamptz NOT NULL DEFAULT now(),
    ADD CONSTRAINT patients_national_id_check CHECK ((national_id IS NULL) = (national_id_type IS NULL)),
    ADD CONSTRAINT patients_mobile_check CHECK (mobile = '' OR mobile ~ '^\+[1-9][0-9]{6,14}$');

-- One record per national ID in a trust: another trust may hold the same one.
CREATE UNIQUE INDEX patients_national_id_key ON patients (org_id, national_id_type, national_id)
    WHERE national_id IS NOT NULL;
-- The search ("surname", "surname + date of birth") and the duplicate check (same date of birth).
CREATE INDEX patients_surname_idx ON patients (org_id, lower(last_name) text_pattern_ops, date_of_birth);
CREATE INDEX patients_birth_idx ON patients (org_id, date_of_birth);
-- "This hospital's patients": those with an episode here; and the hospital number search.
CREATE INDEX episodes_hospital_patient_idx ON episodes (hospital_id, patient_id);
CREATE INDEX episodes_hospital_number_idx ON episodes (hospital_id, lower(hospital_number)) WHERE hospital_number <> '';

-- +goose Down
DROP INDEX episodes_hospital_number_idx;
DROP INDEX episodes_hospital_patient_idx;
DROP INDEX patients_birth_idx;
DROP INDEX patients_surname_idx;
DROP INDEX patients_national_id_key;
ALTER TABLE patients
    DROP CONSTRAINT patients_mobile_check,
    DROP CONSTRAINT patients_national_id_check,
    DROP COLUMN created_at,
    DROP COLUMN account_id,
    DROP COLUMN national_id_type,
    DROP COLUMN national_id;
ALTER TABLE patients RENAME COLUMN mobile TO phone;
ALTER TABLE patients ADD COLUMN hospital_number text NOT NULL DEFAULT '';
UPDATE patients p SET hospital_number = coalesce(
    (SELECT e.hospital_number FROM episodes e WHERE e.patient_id = p.id AND e.hospital_number <> ''
     ORDER BY e.created_at DESC LIMIT 1), '');
ALTER TABLE episodes DROP COLUMN hospital_number;

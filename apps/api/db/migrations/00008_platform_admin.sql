-- +goose Up
-- The superadmin console (plan C1, session S10; ORG-01…05). The platform admin creates trusts
-- (name, code, contact) and their hospitals (name, address, time zone). A trust's or hospital's
-- other details are settings (orgs.settings, hospitals.settings; internal/settings): the logo, the
-- patients' phone and email, the opt-ins and the rest of ORG-04.

-- Who the platform admin deals with at the trust: never shown to patients.
ALTER TABLE orgs
    ADD COLUMN contact_name  text NOT NULL DEFAULT '',
    ADD COLUMN contact_email text NOT NULL DEFAULT '',
    ADD COLUMN contact_phone text NOT NULL DEFAULT '';
-- Codes are kept in capitals; this also refuses "ta" beside "TA" if one is ever written otherwise.
CREATE UNIQUE INDEX orgs_code_upper_key ON orgs (upper(code));

ALTER TABLE hospitals ADD COLUMN address text NOT NULL DEFAULT '';
-- Two hospitals of one trust can't share a name, whatever its case: staff pick them by name.
CREATE UNIQUE INDEX hospitals_org_name_key ON hospitals (org_id, lower(name));

-- +goose Down
DROP INDEX hospitals_org_name_key;
ALTER TABLE hospitals DROP COLUMN address;
DROP INDEX orgs_code_upper_key;
ALTER TABLE orgs DROP COLUMN contact_phone, DROP COLUMN contact_email, DROP COLUMN contact_name;

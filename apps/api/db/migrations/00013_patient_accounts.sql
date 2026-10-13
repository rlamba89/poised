-- +goose Up
-- Patient accounts (plan C2, session S15; A-3, PAT-06, PAT-07, PAT-10, PAT-11). One account is one
-- person: their sign-in contacts (a mobile in E.164 and an email, either may be '') and the date
-- of birth they proved when it was made. Each trust's record of them (patients) joins it
-- (patients.account_id), so one sign-in shows every trust's episodes on their home.
--
-- A record joins an account when the person proves their date of birth through a link sent to a
-- contact (an invite, or a sign-in link): the account with that date of birth and that mobile or
-- email. The date of birth is part of the match, so two people sharing a phone or an email (a
-- couple, a parent and child) stay two accounts.
CREATE TABLE patient_accounts (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    mobile        text NOT NULL DEFAULT '' CHECK (mobile = '' OR mobile ~ '^\+[1-9][0-9]{6,14}$'),
    email         text NOT NULL DEFAULT '' CHECK (email = lower(email)),
    date_of_birth date NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX patient_accounts_mobile_idx ON patient_accounts (mobile) WHERE mobile <> '';
CREATE INDEX patient_accounts_email_idx ON patient_accounts (email) WHERE email <> '';

ALTER TABLE patients ADD CONSTRAINT patients_account_id_fkey FOREIGN KEY (account_id) REFERENCES patient_accounts;
CREATE INDEX patients_account_idx ON patients (account_id) WHERE account_id IS NOT NULL;
-- The person's records not yet joined, found by a contact they've just proved.
CREATE INDEX patients_unlinked_mobile_idx ON patients (mobile, date_of_birth) WHERE account_id IS NULL AND mobile <> '';
CREATE INDEX patients_unlinked_email_idx ON patients (email, date_of_birth) WHERE account_id IS NULL AND email <> '';
-- The home: a person's episodes in every trust (one query, through their records).
CREATE INDEX episodes_patient_idx ON episodes (patient_id, created_at DESC);

-- Sign-in links (kind 'signin', PAT-06) go to a contact, not a record: whoever's date of birth is
-- entered, among the accounts with that contact, is signed in. They work once, for 15 minutes.
-- Verification links (kind 'verify', PAT-10) confirm a new contact for an account before it's used.
ALTER TABLE login_links
    ALTER COLUMN patient_id DROP NOT NULL,
    ADD COLUMN contact    text,                                -- signin, verify: the mobile (E.164) or email it went to
    ADD COLUMN account_id uuid REFERENCES patient_accounts,    -- verify: whose contact changes
    DROP CONSTRAINT login_links_kind_check,
    DROP CONSTRAINT login_links_invite_check,
    ADD CONSTRAINT login_links_kind_check CHECK (
        (kind = 'invite' AND patient_id IS NOT NULL AND episode_id IS NOT NULL)
        OR (kind = 'signin' AND contact IS NOT NULL AND expires_at IS NOT NULL)
        OR (kind = 'verify' AND contact IS NOT NULL AND account_id IS NOT NULL AND expires_at IS NOT NULL));
CREATE INDEX login_links_account_idx ON login_links (account_id) WHERE account_id IS NOT NULL;

-- A session now belongs to an account (all of the person's records), not to one record. Sessions
-- last 30 minutes, so the open ones are simply ended.
DELETE FROM patient_sessions;
ALTER TABLE patient_sessions
    DROP COLUMN patient_id,
    ADD COLUMN account_id uuid NOT NULL REFERENCES patient_accounts;

-- Rate limits (PAT-11): one row per message asked for, by key ("contact:<mobile or email>",
-- "ip:<address>"). Counted over the last 15 minutes or hour; rows older than a day are dropped
-- as new ones come.
CREATE TABLE rate_limit_hits (
    key text NOT NULL,
    at  timestamptz NOT NULL
);
CREATE INDEX rate_limit_hits_key_idx ON rate_limit_hits (key, at);

-- +goose Down
DROP TABLE rate_limit_hits;
DELETE FROM patient_sessions;
ALTER TABLE patient_sessions
    DROP COLUMN account_id,
    ADD COLUMN patient_id uuid NOT NULL REFERENCES patients;
DELETE FROM login_links WHERE kind <> 'invite';
DROP INDEX login_links_account_idx;
ALTER TABLE login_links
    DROP CONSTRAINT login_links_kind_check,
    DROP COLUMN account_id,
    DROP COLUMN contact,
    ALTER COLUMN patient_id SET NOT NULL,
    ADD CONSTRAINT login_links_kind_check CHECK (kind IN ('invite', 'signin')),
    ADD CONSTRAINT login_links_invite_check CHECK (kind <> 'invite' OR episode_id IS NOT NULL);
DROP INDEX episodes_patient_idx;
DROP INDEX patients_unlinked_email_idx;
DROP INDEX patients_unlinked_mobile_idx;
DROP INDEX patients_account_idx;
UPDATE patients SET account_id = NULL;
ALTER TABLE patients DROP CONSTRAINT patients_account_id_fkey;
DROP TABLE patient_accounts;

-- +goose Up
-- An account is matched by surname too (S15 answers 8, Rahul 13 Oct 2026, after test report 2
-- found two people with the same first name, date of birth and contact but different surnames
-- merged): a record joins an account only with the same date of birth, the same mobile or email,
-- and the same first name and surname, each ignoring case and spaces (name_key, migration 00014).
-- A surname that changes, or is typed differently, keeps a separate account (it fails safe).
-- Supersedes 00014's first-name-only rule.
ALTER TABLE patient_accounts ADD COLUMN last_name_key text NOT NULL DEFAULT '';

-- Accounts made before this take their oldest record's surname; records joined with another
-- surname leave the account, and join the right one (or make their own) at their next right date
-- of birth on an invite.
UPDATE patient_accounts a SET last_name_key = coalesce(
    (SELECT name_key(p.last_name) FROM patients p WHERE p.account_id = a.id ORDER BY p.created_at, p.id LIMIT 1), '');
UPDATE patients p SET account_id = NULL
FROM patient_accounts a
WHERE p.account_id = a.id AND name_key(p.last_name) <> a.last_name_key;

ALTER TABLE patient_accounts ALTER COLUMN last_name_key DROP DEFAULT;

-- +goose Down
ALTER TABLE patient_accounts DROP COLUMN last_name_key;

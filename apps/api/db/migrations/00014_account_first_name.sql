-- +goose Up
-- An account is matched by first name too (S15 answers 6, Rahul 13 Oct 2026, after test report 1
-- found twins on one phone merged): a record joins an account only with the same date of birth,
-- the same mobile or email, and the same first name, ignoring case and spaces. Another spelling
-- simply stays unjoined and keeps working through its own invite links (it fails safe).
--
-- name_key is the one comparison, used by every query (Go's patient.NameKey is the same idea for
-- the duplicate check: lower case, no spaces).

-- +goose StatementBegin
CREATE FUNCTION name_key(name text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT lower(regexp_replace(name, '\s', '', 'g'))
$$;
-- +goose StatementEnd

ALTER TABLE patient_accounts ADD COLUMN first_name_key text NOT NULL DEFAULT '';

-- Accounts made before this: their first name is their oldest record's. Records already joined
-- with another first name (twins merged by the old rule) leave the account; their next right date
-- of birth on an invite joins them to the right account, or makes one.
UPDATE patient_accounts a SET first_name_key = coalesce(
    (SELECT name_key(p.first_name) FROM patients p WHERE p.account_id = a.id ORDER BY p.created_at, p.id LIMIT 1), '');
UPDATE patients p SET account_id = NULL
FROM patient_accounts a
WHERE p.account_id = a.id AND name_key(p.first_name) <> a.first_name_key;

ALTER TABLE patient_accounts ALTER COLUMN first_name_key DROP DEFAULT;

-- +goose Down
ALTER TABLE patient_accounts DROP COLUMN first_name_key;
DROP FUNCTION name_key(text);

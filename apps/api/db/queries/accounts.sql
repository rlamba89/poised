-- Patient accounts (plan C2, S15; A-3, PAT-06, PAT-07, PAT-10): one per person, joined by each
-- trust's record of them; sign-in links to a contact; the home; changing a contact.

-- name: LockPatientForAccount :one
-- The invited record, locked while it joins an account (two first sign-ins at once make one).
SELECT id, org_id, account_id, first_name, last_name, date_of_birth, mobile, email FROM patients WHERE id = $1 FOR UPDATE;

-- name: LockBirthDate :exec
-- Accounts are matched by date of birth: one person's records join one at a time, so two first
-- visits at once (from two trusts' invites) can't make two accounts.
SELECT pg_advisory_xact_lock(hashtextextended('patient_accounts:' || sqlc.arg(date_of_birth)::date::text, 0));

-- name: FindAccountForRecord :one
-- The account a record joins (PAT-07): the same date of birth, the same mobile or email, and the
-- same first name and surname, ignoring case and spaces (S15 answers 6 and 8: twins, or cousins
-- sharing a birthday and a first name, on one phone stay apart). The oldest, if (rarely) more
-- than one.
SELECT id FROM patient_accounts
WHERE date_of_birth = sqlc.arg(date_of_birth) AND first_name_key = name_key(sqlc.arg(first_name)::text)
  AND last_name_key = name_key(sqlc.arg(last_name)::text)
  AND ((mobile <> '' AND mobile = sqlc.arg(mobile)::text) OR (email <> '' AND email = sqlc.arg(email)::text))
ORDER BY created_at, id
LIMIT 1;

-- name: CreateAccount :one
INSERT INTO patient_accounts (mobile, email, date_of_birth, first_name_key, last_name_key)
VALUES (sqlc.arg(mobile), sqlc.arg(email), sqlc.arg(date_of_birth), name_key(sqlc.arg(first_name)::text), name_key(sqlc.arg(last_name)::text))
RETURNING id;

-- name: FillAccountContacts :exec
-- A record joining an account fills in a contact the account doesn't have yet (never replaces
-- one), unless another account of the same person's details (date of birth, first name) uses it.
UPDATE patient_accounts a SET
    mobile = CASE WHEN a.mobile = '' AND sqlc.arg(mobile)::text <> '' AND NOT EXISTS (
                 SELECT 1 FROM patient_accounts o WHERE o.id <> a.id AND o.date_of_birth = a.date_of_birth AND o.first_name_key = a.first_name_key AND o.last_name_key = a.last_name_key AND o.mobile = sqlc.arg(mobile)::text)
             THEN sqlc.arg(mobile)::text ELSE a.mobile END,
    email  = CASE WHEN a.email = '' AND sqlc.arg(email)::text <> '' AND NOT EXISTS (
                 SELECT 1 FROM patient_accounts o WHERE o.id <> a.id AND o.date_of_birth = a.date_of_birth AND o.first_name_key = a.first_name_key AND o.last_name_key = a.last_name_key AND o.email = sqlc.arg(email)::text)
             THEN sqlc.arg(email)::text ELSE a.email END
WHERE a.id = sqlc.arg(id);

-- name: JoinAccount :exec
UPDATE patients SET account_id = sqlc.arg(account_id) WHERE id = sqlc.arg(id) AND account_id IS NULL;

-- name: JoinMatchingRecords :many
-- The person's other records, in any trust, not yet joined: the same date of birth, the account's
-- first name and surname, and one of the contacts they've just proved (a link reached them there).
UPDATE patients SET account_id = sqlc.arg(account_id)
WHERE account_id IS NULL AND date_of_birth = sqlc.arg(date_of_birth)
  AND name_key(first_name) = name_key(sqlc.arg(first_name)::text) -- the account's (a key is its own key)
  AND name_key(last_name) = name_key(sqlc.arg(last_name)::text)
  AND ((mobile <> '' AND mobile = ANY(sqlc.arg(contacts)::text[])) OR (email <> '' AND email = ANY(sqlc.arg(contacts)::text[])))
RETURNING id, org_id;

-- name: FillRecordContacts :exec
-- Records that just joined take the account's mobile or email where they have none (S15 answers
-- 7), so every hospital can reach the patient at the contact they use. Never replaces one.
UPDATE patients p SET
    mobile = CASE WHEN p.mobile = '' THEN a.mobile ELSE p.mobile END,
    email  = CASE WHEN p.email = '' THEN a.email ELSE p.email END
FROM patient_accounts a
WHERE a.id = p.account_id AND p.id = ANY(sqlc.arg(ids)::uuid[]);

-- name: AccountHasContact :one
-- Whether any account signs in with this mobile or email (a sign-in link is sent only then).
SELECT EXISTS (SELECT 1 FROM patient_accounts WHERE mobile = sqlc.arg(contact)::text OR email = sqlc.arg(contact)::text)::boolean;

-- name: CreateContactLink :one
-- A sign-in link (kind signin) to a contact, or a verification link (kind verify) for an
-- account's new contact. Both expire.
INSERT INTO login_links (token_hash, kind, contact, account_id, expires_at, created_at)
VALUES (sqlc.arg(token_hash), sqlc.arg(kind), sqlc.arg(contact), sqlc.narg(account_id), sqlc.arg(expires_at), sqlc.arg(now))
RETURNING id;

-- name: GetContactLinkByHash :one
-- A sign-in or verification link, used or not, expired or not (the handler says which).
-- A replaced (revoked) verification link is returned too, and refused as expired.
SELECT id, contact, account_id, expires_at, used_at, locked_at, revoked_at
FROM login_links
WHERE token_hash = sqlc.arg(token_hash) AND kind = sqlc.arg(kind);

-- name: AccountsForSignin :many
-- Who the sign-in link's date of birth (and name, when given) names, among the accounts with its
-- contact. More than one without a name: people sharing the contact and the birthday, such as twins
-- (S15 answers 6 and 8).
SELECT id, first_name_key, last_name_key FROM patient_accounts
WHERE (mobile = sqlc.arg(contact)::text OR email = sqlc.arg(contact)::text) AND date_of_birth = sqlc.arg(date_of_birth)
  AND (NOT sqlc.arg(named)::boolean OR (first_name_key = name_key(sqlc.arg(first_name)::text)
                                        AND last_name_key = name_key(sqlc.arg(last_name)::text)))
ORDER BY created_at, id;

-- name: UseContactLink :execrows
-- A sign-in or verification link works once, until it expires. No row: used, locked or expired
-- meanwhile.
UPDATE login_links SET used_at = sqlc.arg(now), failed_dob_attempts = 0
WHERE id = sqlc.arg(id) AND used_at IS NULL AND locked_at IS NULL AND revoked_at IS NULL AND expires_at > sqlc.arg(now);

-- name: ListHome :many
-- The person's home (PX-08): every episode of every record joined to their account, in every
-- trust, newest first, with the hospital's name. Suspended trusts and hospitals are left out
-- (ORG-03). org, hospital and patient ids are for the audit log, never sent.
SELECT e.id, e.created_at, e.status, coalesce(pr.name, '')::text AS procedure, f.submitted_at AS patient_submitted_at, e.org_id, e.hospital_id, e.patient_id,
       p.first_name, p.last_name, h.name AS hospital_name, q.name AS questionnaire_name,
       EXISTS (SELECT 1 FROM episode_answers a WHERE a.episode_form_id = f.id AND a.actor = 'patient')::boolean AS started
FROM patients p
JOIN episodes e ON e.patient_id = p.id
JOIN hospitals h ON h.id = e.hospital_id
JOIN orgs o ON o.id = e.org_id
JOIN episode_forms f ON f.episode_id = e.id AND f.kind = 'hq'
JOIN questionnaire_versions v ON v.id = f.version_id
JOIN questionnaires q ON q.id = v.questionnaire_id
LEFT JOIN procedures pr ON pr.id = e.procedure_id
WHERE p.account_id = $1 AND o.status = 'active' AND h.status = 'active'
ORDER BY e.created_at DESC, e.id;

-- name: GetAccount :one
SELECT id, mobile, email, date_of_birth FROM patient_accounts WHERE id = $1;

-- name: LockAccount :one
SELECT id, mobile, email, date_of_birth, first_name_key, last_name_key FROM patient_accounts WHERE id = $1 FOR UPDATE;

-- name: RevokeVerifyLinks :exec
-- A newer request for the same kind of contact (mobile or email) replaces the older links.
UPDATE login_links SET revoked_at = sqlc.arg(now)
WHERE account_id = sqlc.arg(account_id) AND kind = 'verify' AND used_at IS NULL AND revoked_at IS NULL
  AND (contact LIKE '%@%') = sqlc.arg(email)::boolean;

-- name: ContactTaken :one
-- Another account with the same details (date of birth, first name, surname) already signs in with this
-- contact: two accounts would then answer the same sign-in, so the change is refused.
SELECT EXISTS (SELECT 1 FROM patient_accounts
               WHERE id <> sqlc.arg(id) AND date_of_birth = sqlc.arg(date_of_birth)
                 AND first_name_key = sqlc.arg(first_name_key) AND last_name_key = sqlc.arg(last_name_key)
                 AND (mobile = sqlc.arg(contact)::text OR email = sqlc.arg(contact)::text))::boolean;

-- name: SetAccountContact :exec
UPDATE patient_accounts SET
    mobile = CASE WHEN sqlc.arg(email)::boolean THEN mobile ELSE sqlc.arg(contact)::text END,
    email  = CASE WHEN sqlc.arg(email)::boolean THEN sqlc.arg(contact)::text ELSE email END
WHERE id = sqlc.arg(id);

-- name: SetRecordsContact :many
-- The patient's own change (PAT-10) is made to every trust's record of them, so invites and
-- reminders go to the new contact. Returns the records changed, for each trust's audit log.
UPDATE patients SET
    mobile = CASE WHEN sqlc.arg(email)::boolean THEN mobile ELSE sqlc.arg(contact)::text END,
    email  = CASE WHEN sqlc.arg(email)::boolean THEN sqlc.arg(contact)::text ELSE email END
WHERE account_id = sqlc.arg(account_id)
RETURNING id, org_id;

-- Rate limits (PAT-11): rate_limit_hits.

-- name: LockRateKey :exec
-- Hits on one key are counted and added one request at a time.
SELECT pg_advisory_xact_lock(hashtextextended('rate:' || sqlc.arg(key)::text, 0));

-- name: CountHits :one
SELECT count(*)::int FROM rate_limit_hits WHERE key = sqlc.arg(key) AND at > sqlc.arg(since);

-- name: AddHit :exec
INSERT INTO rate_limit_hits (key, at) VALUES (sqlc.arg(key), sqlc.arg(at));

-- name: DropOldHits :exec
DELETE FROM rate_limit_hits WHERE key = sqlc.arg(key) AND at < sqlc.arg(before);

-- name: ResetRateLimits :exec
-- MESSAGING=capture only (local, QA): a tester's fresh start.
DELETE FROM rate_limit_hits;

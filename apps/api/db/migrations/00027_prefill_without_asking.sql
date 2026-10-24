-- +goose Up
-- Pre-fill from other trusts without asking (session S27; Rahul, 24 Oct, decisions.md §2).
-- Another trust's answers are always copied, so the trusts' crossTrustPrefill setting (S10, used
-- by S26) and the patients' answers to "Use the answers you gave …?" (S26) go. Answers already
-- copied stay. Instead, a Question Set holding another trust's answers opens on a card that names
-- the hospital, with a third button, Start fresh, which clears that set's pre-filled answers and is
-- recorded like the other two (prefill_choice start_fresh).
DROP TABLE patient_prefill_consents;

UPDATE orgs SET settings = settings - 'crossTrustPrefill' WHERE settings ? 'crossTrustPrefill';
UPDATE hospitals SET settings = settings - 'crossTrustPrefill' WHERE settings ? 'crossTrustPrefill';

ALTER TABLE episode_answers DROP CONSTRAINT episode_answers_prefill_choice_check,
    ADD CONSTRAINT episode_answers_prefill_choice_check CHECK (prefill_choice IN ('check_each', 'still_correct', 'start_fresh'));

-- +goose Down
-- The settings' old values can't be put back (they were off by default). A Start fresh becomes
-- Check each answer, the nearest of the old choices: the patient answered the set themselves.
UPDATE episode_answers SET prefill_choice = 'check_each' WHERE prefill_choice = 'start_fresh';
ALTER TABLE episode_answers DROP CONSTRAINT episode_answers_prefill_choice_check,
    ADD CONSTRAINT episode_answers_prefill_choice_check CHECK (prefill_choice IN ('check_each', 'still_correct'));

CREATE TABLE patient_prefill_consents (
    account_id uuid NOT NULL REFERENCES patient_accounts ON DELETE CASCADE,
    org_id     uuid NOT NULL REFERENCES orgs,
    agreed     boolean NOT NULL,
    decided_at timestamptz NOT NULL DEFAULT now(),
    episode_id uuid REFERENCES episodes ON DELETE SET NULL,
    PRIMARY KEY (account_id, org_id)
);

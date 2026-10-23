-- +goose Up
-- Pre-fill from other trusts (plan C4, session S26; §6 point 7, ORG-04's crossTrustPrefill).
--
-- patient_prefill_consents: the patient's answer to "Use the answers you gave <hospital> in
-- <month>?", once per account and source trust (S26 answers 3). Asked only when both trusts have
-- crossTrustPrefill on; the answers of a trust with agreed = true are then used to pre-fill the
-- person's HQs at any trust that allows it, never on their behalf (S26 answers 4). The choice
-- can't be changed yet. episode_id is where it was asked. Closing the account deletes it.
CREATE TABLE patient_prefill_consents (
    account_id uuid NOT NULL REFERENCES patient_accounts ON DELETE CASCADE,
    org_id     uuid NOT NULL REFERENCES orgs, -- the trust whose answers it's about
    agreed     boolean NOT NULL,
    decided_at timestamptz NOT NULL DEFAULT now(),
    episode_id uuid REFERENCES episodes ON DELETE SET NULL,
    PRIMARY KEY (account_id, org_id)
);

-- A pre-filled Question Set starts at revision 1, not 0 (S25 verify, for S26): 0 is what a
-- browser sends for "no row yet", so a tab that loaded the set before it was pre-filled could
-- overwrite it without the stale 409, and the prefill map would then describe answers no longer
-- in the row. Rows already pre-filled move up one (not an episode's activity: no trigger).
ALTER TABLE episode_answers DISABLE TRIGGER episode_answers_touch;
UPDATE episode_answers SET revision = revision + 1 WHERE prefill IS NOT NULL;
ALTER TABLE episode_answers ENABLE TRIGGER episode_answers_touch;

-- started: the patient has acted on this Question Set (reminders, the home's "In progress"): saved
-- it (a revision past the one it was created with: 0 for a row they made, 1 for a pre-filled one),
-- or answered its pre-fill card (S25). One definition, for accounts.sql and jobs.sql.
ALTER TABLE episode_answers ADD COLUMN started boolean NOT NULL GENERATED ALWAYS AS (
    revision > CASE WHEN prefill IS NULL THEN 0 ELSE 1 END OR prefill_choice IS NOT NULL
) STORED;

-- +goose Down
ALTER TABLE episode_answers DROP COLUMN started;
ALTER TABLE episode_answers DISABLE TRIGGER episode_answers_touch;
UPDATE episode_answers SET revision = revision - 1 WHERE prefill IS NOT NULL;
ALTER TABLE episode_answers ENABLE TRIGGER episode_answers_touch;
DROP TABLE patient_prefill_consents;

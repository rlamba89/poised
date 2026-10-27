-- +goose Up
-- The HQ rules' other actions (plan C6, session S43; ASA-02, ASA-03, TSK-01, EPI-02). When a sent HQ's
-- result (form_outputs, S41) has a rule that fires, Go makes its writes in the send's transaction
-- (internal/triage.ActionsOf): a task for a team (tasks.created_by_rule, S21), a flag on the episode,
-- the anaesthetist's ASA grade required before Ready for admission, and statuses it suggests.

-- A rule's flag on an episode (e.g. "Anaesthetic review"), shown as a badge on the episode and in
-- the lists until a clinician at the hospital clears it with a short reason (S43 answers 1: who,
-- when and why are recorded). A cleared flag is kept; a later send firing the rule again adds it
-- again. rule: the rule's name, as the status history names it.
CREATE TABLE episode_flags (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id     uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    name           text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    rule           text NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    cleared_by     uuid REFERENCES users,
    cleared_at     timestamptz,
    cleared_reason text,
    CONSTRAINT episode_flags_cleared_check CHECK ((cleared_by IS NULL) = (cleared_at IS NULL)
        AND (cleared_at IS NULL) = (cleared_reason IS NULL))
);
-- One standing flag per rule and name: the same rule firing again adds none.
CREATE UNIQUE INDEX episode_flags_standing_idx ON episode_flags (episode_id, rule, name) WHERE cleared_at IS NULL;

-- anaesthetist_asa_required_by: the rule that requires the anaesthetist's ASA grade before Ready
-- for admission (NULL: none). It's met while episodes.anaesthetist_asa is set.
-- rule_suggestions: the statuses the latest send's rules suggest, [{rule, status}], shown on the
-- episode; nothing moves by itself.
ALTER TABLE episodes
    ADD COLUMN anaesthetist_asa_required_by text,
    ADD COLUMN rule_suggestions jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE episodes DROP COLUMN anaesthetist_asa_required_by, DROP COLUMN rule_suggestions;
DROP TABLE episode_flags;

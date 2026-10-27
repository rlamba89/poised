-- +goose Up
-- Auto-triage (plan C6, session S42; ASA-02, A-8, EPI-02, EPI-03, EPI-07). When a patient (or staff
-- for them) sends the HQ and the evaluate service's result (form_outputs, S41) has a rule that
-- auto-triages, and every guard holds (internal/triage), the episode moves straight on from Ready for
-- review to the rule's target. The move is in the status history as auto_triage, with the rule's
-- name and no user.
ALTER TABLE episode_status_events DROP CONSTRAINT episode_status_events_move_check;
ALTER TABLE episode_status_events ADD CONSTRAINT episode_status_events_move_check
    CHECK (move IN ('create', 'submit', 'complete_review', 'set', 'admit', 'discharge', 'cancel', 'reopen', 'undo',
                    'archive', 'unarchive', 'auto_triage'));

-- The episode's auto-triage, while it stands (a re-open of the HQ review clears it: a person is
-- reviewing it now):
--   triage_rule, triage_summary, triaged_at: the rule that moved it, the stamp's words ("ASA II, no
--     red or amber flags") and when;
--   triage_message_pending: the patient's status message waits for Checked (the HQ's
--     messageAfterCheck setting, S42 answers 2);
--   triage_checked_by, triage_checked_at: the clinician who clicked Checked (S42 answers 1), which
--     takes it off the Auto-triaged worklist.
ALTER TABLE episodes
    ADD COLUMN triage_rule            text,
    ADD COLUMN triage_summary         text,
    ADD COLUMN triaged_at             timestamptz,
    ADD COLUMN triage_message_pending boolean NOT NULL DEFAULT false,
    ADD COLUMN triage_checked_by      uuid REFERENCES users,
    ADD COLUMN triage_checked_at      timestamptz,
    ADD CONSTRAINT episodes_triage_check CHECK ((triage_rule IS NULL) = (triaged_at IS NULL)
        AND (triage_rule IS NULL) = (triage_summary IS NULL)
        AND (triage_checked_at IS NULL OR triaged_at IS NOT NULL)
        AND (triage_checked_at IS NULL) = (triage_checked_by IS NULL)
        AND NOT (triage_message_pending AND (triaged_at IS NULL OR triage_checked_at IS NOT NULL)));

-- in_worklist (S21, S22) with Auto-triaged (EPI-02): auto-triaged and not yet checked. Like the
-- others, never a cancelled or archived episode.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION in_worklist(worklist text, episode_id uuid, status text, not_responding_at timestamptz,
                                       planned_date date, today date, soon_days int) RETURNS boolean
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT CASE worklist
        WHEN 'ready_for_review' THEN status = 'ready_for_review'
        WHEN 'auto_triaged' THEN status NOT IN ('cancelled', 'archived') AND EXISTS (
            SELECT 1 FROM episodes x WHERE x.id = in_worklist.episode_id AND x.triaged_at IS NOT NULL AND x.triage_checked_at IS NULL)
        WHEN 'not_responding' THEN status = 'hq_not_complete' AND not_responding_at IS NOT NULL
        WHEN 'anaesthetic_review' THEN status NOT IN ('cancelled', 'archived') AND EXISTS (
            SELECT 1 FROM tasks t WHERE t.episode_id = in_worklist.episode_id AND t.done_at IS NULL AND t.team = 'anaesthetist')
        WHEN 'procedure_soon' THEN status NOT IN ('ready_for_admission', 'admitted', 'discharged', 'cancelled', 'archived')
            AND planned_date BETWEEN today AND today + soon_days
        WHEN 'on_hold' THEN status = 'on_hold'
        WHEN 'open_tasks' THEN status NOT IN ('cancelled', 'archived') AND EXISTS (
            SELECT 1 FROM tasks t WHERE t.episode_id = in_worklist.episode_id AND t.done_at IS NULL)
        ELSE false
    END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION in_worklist(worklist text, episode_id uuid, status text, not_responding_at timestamptz,
                                       planned_date date, today date, soon_days int) RETURNS boolean
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT CASE worklist
        WHEN 'ready_for_review' THEN status = 'ready_for_review'
        WHEN 'not_responding' THEN status = 'hq_not_complete' AND not_responding_at IS NOT NULL
        WHEN 'anaesthetic_review' THEN status NOT IN ('cancelled', 'archived') AND EXISTS (
            SELECT 1 FROM tasks t WHERE t.episode_id = in_worklist.episode_id AND t.done_at IS NULL AND t.team = 'anaesthetist')
        WHEN 'procedure_soon' THEN status NOT IN ('ready_for_admission', 'admitted', 'discharged', 'cancelled', 'archived')
            AND planned_date BETWEEN today AND today + soon_days
        WHEN 'on_hold' THEN status = 'on_hold'
        WHEN 'open_tasks' THEN status NOT IN ('cancelled', 'archived') AND EXISTS (
            SELECT 1 FROM tasks t WHERE t.episode_id = in_worklist.episode_id AND t.done_at IS NULL)
        ELSE false
    END
$$;
-- +goose StatementEnd
ALTER TABLE episodes DROP CONSTRAINT episodes_triage_check,
    DROP COLUMN triage_rule, DROP COLUMN triage_summary, DROP COLUMN triaged_at, DROP COLUMN triage_message_pending,
    DROP COLUMN triage_checked_by, DROP COLUMN triage_checked_at;
DELETE FROM episode_status_events WHERE move = 'auto_triage';
ALTER TABLE episode_status_events DROP CONSTRAINT episode_status_events_move_check;
ALTER TABLE episode_status_events ADD CONSTRAINT episode_status_events_move_check
    CHECK (move IN ('create', 'submit', 'complete_review', 'set', 'admit', 'discharge', 'cancel', 'reopen', 'undo',
                    'archive', 'unarchive'));

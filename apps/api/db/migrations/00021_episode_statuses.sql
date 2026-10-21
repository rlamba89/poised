-- +goose Up
-- Every status, cancel and re-open (plan C3, session S20; EPI-01, EPI-03, EPI-07). The moves
-- between statuses, and who may make them, are internal/episode's transition table.
--
-- New statuses: admitted (with the admission date), discharged (with the discharge date) and
-- cancelled (with a reason from a fixed list; final). `recovery` (C9) and `archived` (S22) come
-- with their plans.
ALTER TABLE episodes DROP CONSTRAINT episodes_status_check;
ALTER TABLE episodes ADD CONSTRAINT episodes_status_check
    CHECK (status IN ('hq_not_complete', 'ready_for_review', 'ready_for_poa', 'poa_complete', 'on_hold', 'not_ready',
                      'ready_for_admission', 'admitted', 'discharged', 'cancelled'));

-- revision counts the changes staff make to the episode (its status and its details). A write
-- sends the revision its page loaded, and a stale one gets 409, so two tabs can't overwrite each
-- other (pending.md §1a). The jobs' bookkeeping (reminders, "not responding") and a resent
-- invite don't count: they change nothing a page edits.
--
-- admitted_at and discharged_at are dates, as the clinician enters them (EPI-01). cancel_reason
-- is the list's key; the reason's words, with any details, are in the status history.
ALTER TABLE episodes
    ADD COLUMN revision      int NOT NULL DEFAULT 0,
    ADD COLUMN admitted_at   date,
    ADD COLUMN discharged_at date,
    ADD COLUMN cancel_reason text CHECK (cancel_reason IN ('procedure_cancelled', 'patient_withdrew', 'moved_hospital', 'other'));

-- The status history (EPI-03): every change, with who, when, from, to and why. Append-only,
-- except that an undo stamps the change it steps back (undone_at), so the next undo goes one
-- further back. seq orders the changes exactly, even two in the same instant.
--   move: internal/episode's Move (create, submit, complete_review, set, admit, discharge,
--         cancel, reopen, undo);
--   from_status: NULL for the first (create);
--   user_id: NULL for the patient's submit and for automatic changes;
--   rule: an automatic change's rule (auto-triage, C6), else NULL.
-- General notes still get a readable line for each change.
CREATE TABLE episode_status_events (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    seq         bigint GENERATED ALWAYS AS IDENTITY,
    episode_id  uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    move        text NOT NULL CHECK (move IN ('create', 'submit', 'complete_review', 'set', 'admit', 'discharge',
                                              'cancel', 'reopen', 'undo')),
    from_status text,
    to_status   text NOT NULL,
    reason      text NOT NULL DEFAULT '',
    user_id     uuid REFERENCES users,
    rule        text,
    undone_at   timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT episode_status_events_from_check CHECK ((move = 'create') = (from_status IS NULL))
);
CREATE INDEX episode_status_events_episode_idx ON episode_status_events (episode_id, seq);

-- +goose Down
DROP TABLE episode_status_events;
UPDATE episodes SET status = 'on_hold' WHERE status IN ('admitted', 'discharged', 'cancelled'); -- the nearest older status
ALTER TABLE episodes DROP COLUMN revision, DROP COLUMN admitted_at, DROP COLUMN discharged_at, DROP COLUMN cancel_reason;
ALTER TABLE episodes DROP CONSTRAINT episodes_status_check;
ALTER TABLE episodes ADD CONSTRAINT episodes_status_check
    CHECK (status IN ('hq_not_complete', 'ready_for_review', 'ready_for_poa', 'poa_complete', 'on_hold', 'not_ready',
                      'ready_for_admission'));

-- +goose Up
-- Episode tasks, worklists and status messages (plan C3, session S21; TSK-01, EPI-02, EPI-08).

-- An episode's tasks (TSK-01). Each is for a team from a fixed list (S21 answers 1: nurse,
-- anaesthetist, admin); anyone at the hospital may complete any task, and re-open it.
--   created_by: the staff member who added it; NULL for a task a rule made (C6), which names its
--               rule in created_by_rule instead;
--   done_by, done_at: set together when it's completed, cleared when it's re-opened;
--   reopened_by, reopened_at: the latest re-open, so the panel can say who re-opened it.
CREATE TABLE tasks (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id      uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    title           text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    team            text NOT NULL CHECK (team IN ('nurse', 'anaesthetist', 'admin')),
    created_by      uuid REFERENCES users,
    created_by_rule text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    done_by         uuid REFERENCES users,
    done_at         timestamptz,
    reopened_by     uuid REFERENCES users,
    reopened_at     timestamptz,
    CONSTRAINT tasks_creator_check CHECK ((created_by IS NULL) <> (created_by_rule IS NULL)),
    CONSTRAINT tasks_done_check CHECK ((done_by IS NULL) = (done_at IS NULL))
);
CREATE INDEX tasks_episode_idx ON tasks (episode_id, created_at);
-- The worklists look for an episode's open tasks (any team, or the anaesthetist's).
CREATE INDEX tasks_open_idx ON tasks (episode_id, team) WHERE done_at IS NULL;

-- The worklists filter a hospital's (or the trust's) episodes by status.
CREATE INDEX episodes_hospital_status_idx ON episodes (hospital_id, status);

-- in_worklist is each worklist's rule (EPI-02, S21 answers 2), in one place: the episode list's
-- ?worklist= filter and the tabs' counts both use it, so a tab's count is always its list's length.
-- today is the date at the episode's hospital; soon_days the hospital's "procedure within N days".
--   ready_for_review:   the patient has sent the HQ and nobody has completed the review;
--   not_responding:     S16's flag, while the HQ is still waiting for the patient;
--   anaesthetic_review: an open task for the anaesthetist team;
--   procedure_soon:     a planned procedure date from today to today + N days, and not yet Ready
--                       for admission (nor admitted, discharged or cancelled: past that point);
--   on_hold:            On hold;
--   open_tasks:         any open task.
-- A cancelled episode is in none of them: its tasks can't be changed (it's read-only, S20).
-- +goose StatementBegin
CREATE FUNCTION in_worklist(worklist text, episode_id uuid, status text, not_responding_at timestamptz,
                            planned_date date, today date, soon_days int) RETURNS boolean
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT CASE worklist
        WHEN 'ready_for_review' THEN status = 'ready_for_review'
        WHEN 'not_responding' THEN status = 'hq_not_complete' AND not_responding_at IS NOT NULL
        WHEN 'anaesthetic_review' THEN status <> 'cancelled' AND EXISTS (
            SELECT 1 FROM tasks t WHERE t.episode_id = in_worklist.episode_id AND t.done_at IS NULL AND t.team = 'anaesthetist')
        WHEN 'procedure_soon' THEN status NOT IN ('ready_for_admission', 'admitted', 'discharged', 'cancelled')
            AND planned_date BETWEEN today AND today + soon_days
        WHEN 'on_hold' THEN status = 'on_hold'
        WHEN 'open_tasks' THEN status <> 'cancelled' AND EXISTS (
            SELECT 1 FROM tasks t WHERE t.episode_id = in_worklist.episode_id AND t.done_at IS NULL)
        ELSE false
    END
$$;
-- +goose StatementEnd

-- Status messages to the patient (EPI-08) are recorded with the invites and reminders (S17), so
-- one that isn't delivered is shown to staff the same way. status: the status it was sent for.
ALTER TABLE episode_messages DROP CONSTRAINT episode_messages_kind_check;
ALTER TABLE episode_messages ADD CONSTRAINT episode_messages_kind_check CHECK (kind IN ('invite', 'reminder', 'status'));
ALTER TABLE episode_messages ADD COLUMN status text;
ALTER TABLE episode_messages ADD CONSTRAINT episode_messages_status_check CHECK ((kind = 'status') = (status IS NOT NULL));

-- +goose Down
DELETE FROM episode_messages WHERE kind = 'status';
ALTER TABLE episode_messages DROP CONSTRAINT episode_messages_status_check;
ALTER TABLE episode_messages DROP COLUMN status;
ALTER TABLE episode_messages DROP CONSTRAINT episode_messages_kind_check;
ALTER TABLE episode_messages ADD CONSTRAINT episode_messages_kind_check CHECK (kind IN ('invite', 'reminder'));
DROP FUNCTION in_worklist(text, uuid, text, timestamptz, date, date, int);
DROP INDEX episodes_hospital_status_idx;
DROP TABLE tasks;

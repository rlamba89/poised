-- +goose Up
-- Archive, auto-archive and closing a patient account (plan C3, session S22; EPI-04, EPI-05,
-- EPI-09). Closing an account needs no new column: it deletes the account's rows.

-- Archived (EPI-04): read-only, in no worklist, still found by search and on the patient's page.
-- archived_from is the status it was archived from, which un-archive restores exactly; it's set
-- exactly while the episode is archived.
ALTER TABLE episodes DROP CONSTRAINT episodes_status_check;
ALTER TABLE episodes ADD CONSTRAINT episodes_status_check
    CHECK (status IN ('hq_not_complete', 'ready_for_review', 'ready_for_poa', 'poa_complete', 'on_hold', 'not_ready',
                      'ready_for_admission', 'admitted', 'discharged', 'cancelled', 'archived'));
ALTER TABLE episodes
    ADD COLUMN archived_from text CHECK (archived_from IN ('hq_not_complete', 'ready_for_review', 'ready_for_poa', 'poa_complete',
                                                           'on_hold', 'not_ready', 'ready_for_admission', 'admitted', 'discharged',
                                                           'cancelled')),
    ADD CONSTRAINT episodes_archived_check CHECK ((status = 'archived') = (archived_from IS NOT NULL));

ALTER TABLE episode_status_events DROP CONSTRAINT episode_status_events_move_check;
ALTER TABLE episode_status_events ADD CONSTRAINT episode_status_events_move_check
    CHECK (move IN ('create', 'submit', 'complete_review', 'set', 'admit', 'discharge', 'cancel', 'reopen', 'undo',
                    'archive', 'unarchive'));

-- last_activity_at (S22 answers 3): the newest write to the episode, for "no activity for M
-- months" (EPI-05). Every write moves it forward, by the triggers below rather than in each
-- handler, so a write path added later can't forget it: the General notes (every status change,
-- note, task change, invite, reminder and message writes a line), the answers (the patient's
-- autosaves and the clinician's), the status history, the tasks and the messages sent.
-- Carried over from the newest such row each episode already has. A row with its own time (a
-- note, a status change, a message) moves it to that time, so a job run "as of" a later time
-- (the testers' /qa/jobs) dates its activity as of then (S22 test report 1, F-2); it never moves
-- it back.
ALTER TABLE episodes ADD COLUMN last_activity_at timestamptz NOT NULL DEFAULT now();
UPDATE episodes e SET last_activity_at = GREATEST(e.created_at,
    (SELECT max(x.created_at) FROM episode_events x WHERE x.episode_id = e.id),
    (SELECT max(x.created_at) FROM episode_status_events x WHERE x.episode_id = e.id),
    (SELECT max(GREATEST(x.created_at, x.done_at, x.reopened_at)) FROM tasks x WHERE x.episode_id = e.id),
    (SELECT max(x.created_at) FROM episode_messages x WHERE x.episode_id = e.id),
    (SELECT max(a.updated_at) FROM episode_answers a JOIN episode_forms f ON f.id = a.episode_form_id WHERE f.episode_id = e.id));

-- seq orders General notes written at the same time: a job run writes its notes as of its own
-- time (a reminder, then "not responding"), so created_at alone can tie.
ALTER TABLE episode_events ADD COLUMN seq bigint GENERATED ALWAYS AS IDENTITY;

-- +goose StatementBegin
CREATE FUNCTION touch_episode() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE episodes SET last_activity_at = NEW.created_at WHERE id = NEW.episode_id AND last_activity_at < NEW.created_at;
    RETURN NULL;
END;
$$;
CREATE FUNCTION touch_episode_now() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE episodes SET last_activity_at = now() WHERE id = NEW.episode_id AND last_activity_at < now();
    RETURN NULL;
END;
$$;
CREATE FUNCTION touch_episode_of_form() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE episodes e SET last_activity_at = now()
    FROM episode_forms f
    WHERE f.id = NEW.episode_form_id AND e.id = f.episode_id AND e.last_activity_at < now();
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER episode_events_touch AFTER INSERT ON episode_events FOR EACH ROW EXECUTE FUNCTION touch_episode();
CREATE TRIGGER episode_status_events_touch AFTER INSERT ON episode_status_events FOR EACH ROW EXECUTE FUNCTION touch_episode();
CREATE TRIGGER tasks_touch AFTER INSERT OR UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION touch_episode_now(); -- a completion has no row time of its own
CREATE TRIGGER episode_messages_touch AFTER INSERT ON episode_messages FOR EACH ROW EXECUTE FUNCTION touch_episode();
CREATE TRIGGER episode_answers_touch AFTER INSERT OR UPDATE ON episode_answers FOR EACH ROW EXECUTE FUNCTION touch_episode_of_form();

-- The daily auto-archive (EPI-05, S22 answers 4): one job per hospital, at 02:00 there, which
-- schedules its own next run. The runner adds a hospital's job when it has none (internal/jobs).
ALTER TABLE jobs DROP CONSTRAINT jobs_kind_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_kind_check CHECK (kind IN ('followup', 'auto_archive'));
ALTER TABLE jobs
    ADD COLUMN hospital_id uuid REFERENCES hospitals ON DELETE CASCADE, -- an auto-archive's hospital
    ADD CONSTRAINT jobs_auto_archive_check CHECK (kind <> 'auto_archive' OR hospital_id IS NOT NULL);
CREATE UNIQUE INDEX jobs_auto_archive_key ON jobs (hospital_id) WHERE kind = 'auto_archive' AND done_at IS NULL;

-- in_worklist (S21) as before, but an archived episode is in none of the worklists (EPI-04),
-- as a cancelled one isn't.
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

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION in_worklist(worklist text, episode_id uuid, status text, not_responding_at timestamptz,
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
DELETE FROM jobs WHERE kind = 'auto_archive';
DROP INDEX jobs_auto_archive_key;
ALTER TABLE jobs DROP CONSTRAINT jobs_auto_archive_check, DROP COLUMN hospital_id;
ALTER TABLE jobs DROP CONSTRAINT jobs_kind_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_kind_check CHECK (kind IN ('followup'));
DROP TRIGGER episode_answers_touch ON episode_answers;
DROP TRIGGER episode_messages_touch ON episode_messages;
DROP TRIGGER tasks_touch ON tasks;
DROP TRIGGER episode_status_events_touch ON episode_status_events;
DROP TRIGGER episode_events_touch ON episode_events;
DROP FUNCTION touch_episode_of_form();
DROP FUNCTION IF EXISTS touch_episode_now(); -- IF EXISTS: a database migrated before test report 1 has none
DROP FUNCTION touch_episode();
ALTER TABLE episodes DROP COLUMN last_activity_at;
ALTER TABLE episode_events DROP COLUMN IF EXISTS seq;
DELETE FROM episode_status_events WHERE move IN ('archive', 'unarchive');
ALTER TABLE episode_status_events DROP CONSTRAINT episode_status_events_move_check;
ALTER TABLE episode_status_events ADD CONSTRAINT episode_status_events_move_check
    CHECK (move IN ('create', 'submit', 'complete_review', 'set', 'admit', 'discharge', 'cancel', 'reopen', 'undo'));
ALTER TABLE episodes DROP CONSTRAINT episodes_archived_check;
UPDATE episodes SET status = archived_from WHERE status = 'archived'; -- back where each was archived from
ALTER TABLE episodes DROP COLUMN archived_from;
ALTER TABLE episodes DROP CONSTRAINT episodes_status_check;
ALTER TABLE episodes ADD CONSTRAINT episodes_status_check
    CHECK (status IN ('hq_not_complete', 'ready_for_review', 'ready_for_poa', 'poa_complete', 'on_hold', 'not_ready',
                      'ready_for_admission', 'admitted', 'discharged', 'cancelled'));

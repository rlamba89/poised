-- +goose Up
-- Scheduled jobs (plan C2, session S16): reminders on the days after the invite, and "not
-- responding" (EPI-02) when the HQ hasn't been sent by the hospital's day (ORG-04's
-- reminders.days and reminders.notRespondingDay).
--
-- A job is a row; jobs.RunDue(now) claims the due ones with FOR UPDATE SKIP LOCKED, so two
-- runners never take the same row. Locally a timer in cmd/api runs it every minute; on AWS (S17)
-- the jobs Lambda does, on a schedule. Nothing depends on a server staying up.
CREATE TABLE jobs (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind       text NOT NULL CHECK (kind IN ('followup')),
    episode_id uuid REFERENCES episodes ON DELETE CASCADE, -- a follow-up's episode
    payload    jsonb NOT NULL DEFAULT '{}',
    run_at     timestamptz NOT NULL,       -- due from then on
    done_at    timestamptz,                -- finished: nothing more to do
    attempts   int NOT NULL DEFAULT 0,     -- failed runs in a row
    last_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT jobs_followup_check CHECK (kind <> 'followup' OR episode_id IS NOT NULL)
);
CREATE INDEX jobs_due_idx ON jobs (run_at) WHERE done_at IS NULL;
-- One open follow-up per episode.
CREATE UNIQUE INDEX jobs_followup_key ON jobs (episode_id) WHERE kind = 'followup' AND done_at IS NULL;

-- The follow-up's state on the episode:
--   invited_at: the first invite that reached the patient (the reminder days count from it);
--   reminders_sent: how many of the reminder days have passed and been dealt with;
--   not_responding_at: when it was marked "not responding" (shown until the HQ is sent).
ALTER TABLE episodes
    ADD COLUMN invited_at        timestamptz,
    ADD COLUMN reminders_sent    int NOT NULL DEFAULT 0,
    ADD COLUMN not_responding_at timestamptz;

-- A General notes entry written by the jobs, not by staff or the patient ("Automatic").
ALTER TABLE episode_events ADD COLUMN automatic boolean NOT NULL DEFAULT false;

-- MESSAGING=capture only (local and QA): the jobs timer's clock can be moved ahead, so a tester
-- sees a reminder that's due in days arrive within a minute. One row; make seed sets it back.
CREATE TABLE qa_jobs_clock (
    id             boolean PRIMARY KEY DEFAULT true CHECK (id),
    offset_minutes int NOT NULL DEFAULT 0,
    last_tick_at   timestamptz, -- the timer's last run, in real time
    last_tick_as_of timestamptz -- the time it ran the jobs as of (real time + the offset)
);
INSERT INTO qa_jobs_clock DEFAULT VALUES;

-- +goose Down
DROP TABLE qa_jobs_clock;
ALTER TABLE episode_events DROP COLUMN automatic;
ALTER TABLE episodes DROP COLUMN invited_at, DROP COLUMN reminders_sent, DROP COLUMN not_responding_at;
DROP TABLE jobs;

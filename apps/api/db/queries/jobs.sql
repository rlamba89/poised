-- Scheduled jobs (S16): internal/jobs. Only follow-ups exist so far: an episode's reminders and
-- "not responding".

-- name: StartFollowup :exec
-- The first invite that reaches the patient starts the follow-up: the reminder days count from
-- it. Later invites (Resend) change nothing.
WITH invited AS (
    UPDATE episodes SET invited_at = sqlc.arg(now)
    WHERE episodes.id = sqlc.arg(episode_id) AND invited_at IS NULL
    RETURNING episodes.id
)
INSERT INTO jobs (kind, episode_id, run_at)
SELECT 'followup', invited.id, sqlc.arg(now) FROM invited
ON CONFLICT DO NOTHING;

-- name: ClaimDueJob :one
-- Takes one due job: its run_at moves to lease_until, so no other runner sees it as due while
-- this one runs it (FOR UPDATE SKIP LOCKED: a row another runner is taking at the same moment is
-- skipped). If this runner dies, the job is due again at lease_until. Nothing stays locked while
-- the job runs, so a runner holds one database connection at a time.
UPDATE jobs SET run_at = sqlc.arg(lease_until)
WHERE id = (SELECT j.id FROM jobs j WHERE j.done_at IS NULL AND j.run_at <= sqlc.arg(now)
            ORDER BY j.run_at LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING id, kind, episode_id, attempts;

-- name: RescheduleJob :exec
-- The job ran: look again at run_at.
UPDATE jobs SET run_at = sqlc.arg(run_at), attempts = 0, last_error = '' WHERE id = sqlc.arg(id);

-- name: FinishJob :exec
UPDATE jobs SET done_at = sqlc.arg(now)::timestamptz, attempts = 0, last_error = '' WHERE id = sqlc.arg(id);

-- name: FailJob :exec
-- The job failed: try again at run_at.
UPDATE jobs SET run_at = sqlc.arg(run_at), attempts = attempts + 1, last_error = sqlc.arg(last_error)
WHERE id = sqlc.arg(id);

-- name: GetFollowup :one
-- What the follow-up decides on: the episode's state, whether the patient has started (any
-- answers of theirs on the HQ form, or entered for them), and the hospital's zone and settings.
SELECT e.id, e.invited_at, e.reminders_sent, e.not_responding_at, f.submitted_at AS patient_submitted_at,
       EXISTS (SELECT 1 FROM episode_answers a WHERE a.episode_form_id = f.id AND a.actor = 'patient')::boolean AS started,
       EXISTS (SELECT 1 FROM login_links l WHERE l.episode_id = e.id AND l.kind = 'invite'
               AND l.revoked_at IS NULL AND l.locked_at IS NOT NULL)::boolean AS locked,
       (o.status = 'active' AND h.status = 'active')::boolean AS open,
       h.timezone, h.settings AS hospital_settings, o.settings AS org_settings
FROM episodes e
JOIN episode_forms f ON f.episode_id = e.id AND f.kind = 'hq'
JOIN hospitals h ON h.id = e.hospital_id
JOIN orgs o ON o.id = e.org_id
WHERE e.id = $1;

-- name: CountReminder :execrows
-- Takes reminder number `count` for the episode before it's sent, so it's sent at most once even
-- if two runners decide at once. No row: it was already taken, or the HQ has been sent.
UPDATE episodes e SET reminders_sent = sqlc.arg(count)
WHERE e.id = sqlc.arg(id) AND e.reminders_sent < sqlc.arg(count)
  AND EXISTS (SELECT 1 FROM episode_forms f WHERE f.episode_id = e.id AND f.kind = 'hq' AND f.status = 'open');

-- name: MarkNotResponding :execrows
-- No row: already marked, or the HQ has been sent.
UPDATE episodes e SET not_responding_at = sqlc.arg(now)::timestamptz
WHERE e.id = sqlc.arg(id) AND e.not_responding_at IS NULL
  AND EXISTS (SELECT 1 FROM episode_forms f WHERE f.episode_id = e.id AND f.kind = 'hq' AND f.status = 'open');

-- name: GetQAJobsClock :one
-- MESSAGING=capture only: how far ahead of real time the jobs timer runs, and its last run.
SELECT offset_minutes, last_tick_at, last_tick_as_of FROM qa_jobs_clock;

-- name: SetQAJobsClock :exec
UPDATE qa_jobs_clock SET offset_minutes = $1;

-- name: NoteQAJobsTick :exec
UPDATE qa_jobs_clock SET last_tick_at = sqlc.arg(at)::timestamptz, last_tick_as_of = sqlc.arg(as_of)::timestamptz;

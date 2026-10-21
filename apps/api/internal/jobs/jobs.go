// Package jobs runs the scheduled work (plan C2, session S16): each episode's follow-up, which
// sends reminders on the hospital's days after the invite and marks it "not responding" on its
// day (EPI-02; ORG-04's reminders.days and reminders.notRespondingDay); and each hospital's daily
// auto-archive (S22, EPI-05), at 02:00 there, which archives what episode.ArchiveDue says is due.
//
// Everything goes through one function, Runner.RunDue(ctx, now), which keeps nothing in memory
// between calls and touches no local disk, so it runs the same anywhere:
//   - locally, a timer in cmd/api calls it every minute (Tick; JOBS_TIMER=false turns it off);
//   - on AWS (S17), the jobs Lambda calls it on an EventBridge schedule;
//   - with MESSAGING=capture, testers call it "as of" a later time (POST /api/qa/jobs/run).
//
// A job is a row of the jobs table. Due rows are claimed one at a time with FOR UPDATE SKIP
// LOCKED, which moves the row's run_at on by a lease, so two runners never run the same job; each
// job's decision is the pure function Due.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/invite"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// Runner runs the due jobs.
type Runner struct {
	Pool        *pgxpool.Pool
	Inviter     invite.Sender // sends the reminders, as invites are sent
	DefaultZone string        // the deployment's time zone, for hospitals without their own (A-20)
}

// Summary is what one RunDue did.
type Summary struct {
	AsOf          time.Time `json:"asOf"`
	Jobs          int       `json:"jobs"`          // jobs run
	Reminders     int       `json:"reminders"`     // reminders that reached the patient by at least one message
	NotResponding int       `json:"notResponding"` // episodes marked "not responding"
	AutoArchives  int       `json:"autoArchives"`  // hospitals' auto-archive jobs run (S22)
	Archived      int       `json:"archived"`      // episodes they archived
	Failed        int       `json:"failed"`        // jobs that failed; each is tried again later
}

// maxPerRun bounds one RunDue, so a backlog can't run past a Lambda's time limit; the rest wait
// for the next run.
const maxPerRun = 500

// RunDue runs every job due at now, one at a time. It returns an error only when
// the jobs table itself can't be read or written; a job that fails is logged, counted in Failed
// and tried again later.
func (r Runner) RunDue(ctx context.Context, now time.Time) (Summary, error) {
	sum := Summary{AsOf: now}
	if err := r.addAutoArchives(ctx, now); err != nil {
		return sum, err
	}
	for sum.Jobs < maxPerRun {
		ran, err := r.runOne(ctx, now, &sum)
		if err != nil || !ran {
			return sum, err
		}
	}
	return sum, nil
}

// lease is how long a claimed job is hidden from other runners while this one runs it.
const lease = 10 * time.Minute

// runOne claims one due job and runs it, then reschedules, finishes or fails it.
func (r Runner) runOne(ctx context.Context, now time.Time, sum *Summary) (bool, error) {
	q := db.New(r.Pool)
	job, err := q.ClaimDueJob(ctx, db.ClaimDueJobParams{Now: now, LeaseUntil: now.Add(lease)})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	sum.Jobs++
	var next time.Time
	switch job.Kind {
	case "followup":
		next, err = r.followup(ctx, job.EpisodeID.UUID, now, sum)
	case "auto_archive":
		sum.AutoArchives++
		next, err = r.autoArchive(ctx, job.HospitalID.UUID, now, sum)
	default:
		err = fmt.Errorf("unknown kind %q", job.Kind)
	}
	switch {
	case err != nil:
		sum.Failed++
		log.Printf("jobs: %s job %s (attempt %d): %v", job.Kind, job.ID, job.Attempts+1, err)
		err = q.FailJob(ctx, db.FailJobParams{ID: job.ID, RunAt: now.Add(backoff(job.Attempts)), LastError: err.Error()})
	case next.IsZero():
		err = q.FinishJob(ctx, db.FinishJobParams{ID: job.ID, Now: now})
	default:
		err = q.RescheduleJob(ctx, db.RescheduleJobParams{ID: job.ID, RunAt: next})
	}
	return err == nil, err
}

// backoff is how long a failed job waits: 5 minutes, doubling with each failure in a row, at
// most 6 hours.
func backoff(attempts int32) time.Duration {
	d := 5 * time.Minute
	for i := int32(0); i < attempts && d < 6*time.Hour; i++ {
		d *= 2
	}
	return min(d, 6*time.Hour)
}

// followup does what Due decides for one episode and returns when to look again (zero: never).
//
// A reminder is counted on the episode (CountReminder) before it's sent, so it goes at most once
// even if two runners decide together; if sending then fails, the General notes say so and it
// isn't retried (as with an invite, staff can resend).
func (r Runner) followup(ctx context.Context, episodeID uuid.UUID, now time.Time, sum *Summary) (time.Time, error) {
	q := db.New(r.Pool)
	e, err := q.GetFollowup(ctx, episodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil // nothing to follow up
	}
	if err != nil {
		return time.Time{}, err
	}
	sched, err := r.schedule(e)
	if err != nil {
		return time.Time{}, err
	}
	f := Followup{Submitted: e.PatientSubmittedAt.Valid, Started: e.Started, Open: e.Open, Locked: e.Locked,
		RemindersSent: int(e.RemindersSent), NotResponding: e.NotRespondingAt.Valid}
	if e.InvitedAt.Valid {
		f.InvitedAt = e.InvitedAt.Time
	}
	a := Due(f, sched, now)

	if a.Reminder > 0 {
		n, err := q.CountReminder(ctx, db.CountReminderParams{ID: e.ID, Count: int32(a.ReminderCount)})
		if err != nil {
			return time.Time{}, err
		}
		if n == 1 && a.Withheld {
			// A reminder's new link would undo the lock, which only staff's Resend may (S14).
			note := fmt.Sprintf("Day %d reminder not sent: the patient's link is locked after 5 wrong dates of birth. "+
				"Resend the invite to send them a new link.", a.Reminder)
			if err := q.AddAutomaticEpisodeEvent(ctx, db.AddAutomaticEpisodeEventParams{EpisodeID: e.ID, Text: note, At: now}); err != nil {
				log.Printf("jobs: episode %s: General notes: %v", e.ID, err)
			}
		} else if n == 1 {
			what := fmt.Sprintf("Day %d reminder", a.Reminder)
			note := what + " not sent: something went wrong. Use Resend invite to send the patient a new link."
			res, err := r.Inviter.Send(ctx, e.ID, uuid.Nil, invite.Reminder)
			if err != nil {
				log.Printf("jobs: reminder for episode %s: %v", e.ID, err)
			} else {
				note = res.Note(what+" sent", what+" not sent")
				if res.Sent {
					sum.Reminders++
				}
			}
			if err := q.AddAutomaticEpisodeEvent(ctx, db.AddAutomaticEpisodeEventParams{EpisodeID: e.ID, Text: note, At: now}); err != nil {
				log.Printf("jobs: episode %s: General notes: %v", e.ID, err) // the reminder itself is done
			}
		}
	}
	if a.NotResponding {
		n, err := q.MarkNotResponding(ctx, db.MarkNotRespondingParams{ID: e.ID, Now: now})
		if err != nil {
			return time.Time{}, err
		}
		if n == 1 {
			sum.NotResponding++
			note := fmt.Sprintf("Not responding: the HQ hasn't been sent %d days after the invite.", sched.NotRespondingDay)
			if err := q.AddAutomaticEpisodeEvent(ctx, db.AddAutomaticEpisodeEventParams{EpisodeID: e.ID, Text: note, At: now}); err != nil {
				log.Printf("jobs: episode %s: General notes: %v", e.ID, err)
			}
		}
	}
	return a.Next, nil
}

// addAutoArchives gives each hospital that has none its auto-archive job, first due at its next
// 02:00 (S22 answers 4): a new hospital, or a fresh database, needs nothing else.
func (r Runner) addAutoArchives(ctx context.Context, now time.Time) error {
	q := db.New(r.Pool)
	missing, err := q.HospitalsWithoutAutoArchive(ctx)
	if err != nil {
		return err
	}
	for _, h := range missing {
		zone, err := r.zone(h.Timezone)
		if err != nil {
			return err
		}
		if err := q.AddAutoArchiveJob(ctx, db.AddAutoArchiveJobParams{HospitalID: uuid.NullUUID{UUID: h.ID, Valid: true},
			RunAt: episode.NextArchiveRun(now, zone)}); err != nil {
			return err
		}
	}
	return nil
}

// RunAllAsOf is the testers' "run the jobs as of" (MESSAGING=capture only, S16): every job due at
// asOf, and every hospital's auto-archive whatever its next 02:00 (S22), so a tester checking
// "31 days after discharge" sees it whatever they ran before.
func (r Runner) RunAllAsOf(ctx context.Context, asOf time.Time) (Summary, error) {
	if err := r.addAutoArchives(ctx, asOf); err != nil {
		return Summary{AsOf: asOf}, err
	}
	if err := db.New(r.Pool).MakeAutoArchivesDue(ctx, asOf); err != nil {
		return Summary{AsOf: asOf}, err
	}
	return r.RunDue(ctx, asOf)
}

// autoArchive archives the hospital's episodes that are due (episode.ArchiveDue: N days after
// discharge, M months without activity before admission, from the hospital's effective settings),
// and returns its next run: 02:00 tomorrow at the hospital. While its trust or the hospital is
// suspended nothing is archived (ORG-03).
func (r Runner) autoArchive(ctx context.Context, hospitalID uuid.UUID, now time.Time, sum *Summary) (time.Time, error) {
	q := db.New(r.Pool)
	h, err := q.GetArchivePolicy(ctx, hospitalID)
	if err != nil {
		return time.Time{}, err
	}
	zone, err := r.zone(h.Timezone)
	if err != nil {
		return time.Time{}, err
	}
	next := episode.NextArchiveRun(now, zone)
	if !h.Open {
		return next, nil
	}
	p := episode.ArchivePolicy{Zone: zone}
	v, err := effective(h.OrgSettings, h.HospitalSettings)
	if err != nil {
		return time.Time{}, err
	}
	if err := settings.Get(v, "autoArchive.afterDischargeDays", &p.DaysAfterDischarge); err != nil {
		return time.Time{}, err
	}
	if err := settings.Get(v, "autoArchive.inactiveMonths", &p.InactiveMonths); err != nil {
		return time.Time{}, err
	}
	episodes, err := q.ListArchiveCandidates(ctx, hospitalID)
	if err != nil {
		return time.Time{}, err
	}
	for _, e := range episodes {
		f := episode.ArchiveFacts{Status: e.Status, LastActivity: e.LastActivityAt}
		if e.DischargedAt.Valid {
			f.DischargedAt = e.DischargedAt.Time
		}
		if e.UnarchivedAt.Valid {
			f.UnarchivedAt = e.UnarchivedAt.Time
		}
		rule, due := episode.ArchiveDue(f, p, now)
		if !due {
			continue
		}
		archived, err := r.archive(ctx, e, rule, now)
		if err != nil {
			return time.Time{}, fmt.Errorf("episode %s: %w", e.ID, err)
		}
		if archived {
			sum.Archived++
		}
	}
	return next, nil
}

// archive archives one episode by rule, in one transaction, as a super clinician's archive is
// (statuses.go): the status and archived_from, the status history naming the rule (no user), an
// Automatic General notes line, and its follow-up stopped, all dated now: the run's time, which is
// a later one for a run "as of" then (S22 test report 1, F-2). false: the episode changed since it
// was read, so it's left for the next run to decide again.
func (r Runner) archive(ctx context.Context, e db.ListArchiveCandidatesRow, rule string, now time.Time) (bool, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	n, err := q.AutoArchiveEpisode(ctx, db.AutoArchiveEpisodeParams{ID: e.ID, Status: e.Status, LastActivityAt: e.LastActivityAt})
	if err != nil || n == 0 {
		return false, err
	}
	from := e.Status
	if err := q.AddStatusEvent(ctx, db.AddStatusEventParams{EpisodeID: e.ID, Move: string(episode.Archive), FromStatus: &from,
		ToStatus: episode.Archived, Rule: &rule, At: pgtype.Timestamptz{Time: now, Valid: true}}); err != nil {
		return false, err
	}
	if err := q.AddAutomaticEpisodeEvent(ctx, db.AddAutomaticEpisodeEventParams{EpisodeID: e.ID, Text: "Episode archived (" + rule + ")", At: now}); err != nil {
		return false, err
	}
	if err := q.FinishEpisodeJobs(ctx, uuid.NullUUID{UUID: e.ID, Valid: true}); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// effective is a hospital's settings: its own, else its trust's, else the platform's.
func effective(orgSettings, hospitalSettings []byte) (map[string]json.RawMessage, error) {
	org, err := settings.Decode(orgSettings)
	if err != nil {
		return nil, fmt.Errorf("trust settings: %w", err)
	}
	own, err := settings.Decode(hospitalSettings)
	if err != nil {
		return nil, fmt.Errorf("hospital settings: %w", err)
	}
	return settings.Effective(settings.Platform(), org, own), nil
}

// zone is a hospital's time zone, or the deployment's when it has none (A-20).
func (r Runner) zone(tz *string) (*time.Location, error) {
	name := r.DefaultZone
	if tz != nil && *tz != "" {
		name = *tz
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("time zone %q: %w", name, err)
	}
	return loc, nil
}

// schedule is the episode's hospital's follow-up settings and time zone.
func (r Runner) schedule(e db.GetFollowupRow) (Schedule, error) {
	org, err := settings.Decode(e.OrgSettings)
	if err != nil {
		return Schedule{}, fmt.Errorf("trust settings: %w", err)
	}
	own, err := settings.Decode(e.HospitalSettings)
	if err != nil {
		return Schedule{}, fmt.Errorf("hospital settings: %w", err)
	}
	v := settings.Effective(settings.Platform(), org, own)
	var s Schedule
	if err := settings.Get(v, "reminders.days", &s.ReminderDays); err != nil {
		return Schedule{}, err
	}
	if err := settings.Get(v, "reminders.notRespondingDay", &s.NotRespondingDay); err != nil {
		return Schedule{}, err
	}
	slices.Sort(s.ReminderDays) // Validate stores them in order; Due relies on it
	name := r.DefaultZone
	if e.Timezone != nil && *e.Timezone != "" {
		name = *e.Timezone
	}
	if s.Zone, err = time.LoadLocation(name); err != nil {
		return Schedule{}, fmt.Errorf("time zone %q: %w", name, err)
	}
	return s, nil
}

// TimerInterval is how often the local timer runs the jobs (S16). The jobs Lambda's schedule is
// S17's (every 5 minutes).
const TimerInterval = time.Minute

// Tick is one scheduled run: RunDue at the real time. With qaClock (MESSAGING=capture: local and
// QA only) the time is moved ahead by the testers' offset (PUT /api/qa/jobs/clock), and the run is
// noted for the jobs page.
func (r Runner) Tick(ctx context.Context, real time.Time, qaClock bool) (Summary, error) {
	now := real
	q := db.New(r.Pool)
	if qaClock {
		c, err := q.GetQAJobsClock(ctx)
		if err != nil {
			return Summary{}, err
		}
		now = real.Add(time.Duration(c.OffsetMinutes) * time.Minute)
	}
	sum, err := r.RunDue(ctx, now)
	if err != nil {
		return sum, err
	}
	if qaClock {
		err = q.NoteQAJobsTick(ctx, db.NoteQAJobsTickParams{At: real, AsOf: now})
	}
	return sum, err
}

// Every runs Tick every interval, starting now, until ctx ends: the local timer (cmd/api). Each
// run is logged when it did something or failed.
func (r Runner) Every(ctx context.Context, interval time.Duration, qaClock bool) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		sum, err := r.Tick(ctx, time.Now(), qaClock)
		switch {
		case err != nil:
			log.Printf("jobs: %v", err)
		case sum.Jobs > 0:
			log.Printf("jobs: ran %d as of %s: %d reminders, %d not responding, %d archived, %d failed",
				sum.Jobs, sum.AsOf.Format(time.RFC3339), sum.Reminders, sum.NotResponding, sum.Archived, sum.Failed)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

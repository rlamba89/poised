package httpapi

import (
	"net/http"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/jobs"
)

// The scheduled jobs' test controls (S16), served only with MESSAGING=capture (local and QA,
// like the captured messages): nothing here exists where real messages are sent. Testers can't
// wait days for a reminder, so they run the jobs "as of" a later time, or move the local timer's
// clock ahead. No sign-in, as for /api/qa/messages; the responses hold counts, never patient data.

// maxQAOffset is how far ahead the timer's clock can be moved: 90 days, past every schedule a
// hospital can set (reminders up to day 60, "not responding" up to day 90).
const maxQAOffset = 90 * 24 * 60

// NewJobsRunner is the jobs runner for these dependencies, as the server's own routes send
// invites: the local timer (cmd/api) and, in S17, the jobs Lambda use it.
func NewJobsRunner(d Deps) jobs.Runner { return newServer(d).jobs }

// qaRunJobs is POST /api/qa/jobs/run {asOf}: runs every job due at asOf (an RFC 3339 time, e.g.
// two days on) now, and says what it did.
func (s *server) qaRunJobs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AsOf string `json:"asOf"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	asOf, err := time.Parse(time.RFC3339Nano, in.AsOf)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Give the date and time to run the jobs as of, e.g. 2026-10-15T09:30:00Z.")
		return
	}
	sum, err := s.jobs.RunDue(r.Context(), asOf)
	if err != nil {
		serverError(w, "run jobs", err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// qaJobs is GET /api/qa/jobs: whether this server's timer runs the jobs, how far ahead its clock
// is, and when it last ran.
func (s *server) qaJobs(w http.ResponseWriter, r *http.Request) {
	c, err := s.q.GetQAJobsClock(r.Context())
	if err != nil {
		serverError(w, "jobs clock", err)
		return
	}
	var last, lastAsOf *time.Time
	if c.LastTickAt.Valid {
		last, lastAsOf = &c.LastTickAt.Time, &c.LastTickAsOf.Time
	}
	// Who runs the jobs by themselves: locally the timer in cmd/api (S16); on AWS the jobs
	// Lambda's schedule (S17).
	timer, runner, interval := s.cfg.JobsTimer, "timer", int(jobs.TimerInterval/time.Second)
	if m := s.cfg.JobsScheduleMinutes; m > 0 {
		timer, runner, interval = true, "schedule", m*60
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timer": timer, "runner": runner, "intervalSeconds": interval,
		"offsetMinutes": c.OffsetMinutes, "lastTickAt": last, "lastTickAsOf": lastAsOf,
	})
}

// qaSetJobsClock is PUT /api/qa/jobs/clock {offsetMinutes}: the timer runs the jobs as of real
// time plus this many minutes, from its next run. 0 sets it back; make seed does too.
func (s *server) qaSetJobsClock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OffsetMinutes *int `json:"offsetMinutes"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	if in.OffsetMinutes == nil || *in.OffsetMinutes < 0 || *in.OffsetMinutes > maxQAOffset {
		writeError(w, http.StatusBadRequest, "Move the clock ahead by 0 to 90 days.")
		return
	}
	if err := s.q.SetQAJobsClock(r.Context(), int32(*in.OffsetMinutes)); err != nil {
		serverError(w, "set jobs clock", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

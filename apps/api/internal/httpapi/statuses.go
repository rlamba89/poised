package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// Every status, cancel, re-open and undo (plan C3, session S20; EPI-01, EPI-03, EPI-07). Which
// moves are allowed, and to whom, is internal/episode's transition table; these routes apply it:
//
//   - PATCH …/episodes/{eid}/status {to, reason, date, revision}: a clinician's pick among the
//     reviewed statuses, Admit (with the admission date) and Discharge (with the discharge date);
//   - POST …/cancel {reason, details, revision}: final; the patient's links, sessions and
//     follow-up end with it (S20 answers 5);
//   - POST …/reopen {reason, revision}: a super clinician re-opens the HQ review (answers 6);
//   - POST …/undo {reason, revision}: any clinician steps back the latest move made by hand
//     (answers 2), again to go further back.
//
// Each sends the episode's revision its page loaded: a stale one gets 409, so a move made in an
// old tab is never applied to an episode that has changed. Each change is recorded in the status
// history (who, when, from, to, why) and as a readable line in the General notes.

const (
	staleEpisodeMsg = "This episode changed since you opened it. Reload to see the latest; your change was not saved."
	cancelledMsg    = "This episode is cancelled, so it can't be changed."
	maxReasonLen    = 1000
)

// writeStale answers a write sent with a stale revision: 409, with code "stale" so the page can
// offer Reload and keep what was typed.
func writeStale(w http.ResponseWriter) {
	writeJSON(w, http.StatusConflict, map[string]string{"error": staleEpisodeMsg, "code": "stale"})
}

// statusRequest is the body of every status route; each reads the fields it needs.
type statusRequest struct {
	To       string `json:"to"`
	Reason   string `json:"reason"`  // the reason's words; for cancel, the list's key
	Details  string `json:"details"` // cancel only: the sentence ("Other" needs one)
	Date     string `json:"date"`    // YYYY-MM-DD: the admission or discharge date
	Revision *int32 `json:"revision"`
}

// statusChange is one move to apply, worked out from the request before the episode is locked.
type statusChange struct {
	move     episode.Move
	to       string // "" for an undo: it depends on the history
	reason   string // as recorded
	date     pgtype.Date
	cancel   *string // the cancel reason's key
	revision int32
}

// changeStatus is PATCH …/episodes/{eid}/status.
func (s *server) changeStatus(w http.ResponseWriter, r *http.Request) {
	e, in, ok := s.readStatusRequest(w, r)
	if !ok {
		return
	}
	c := statusChange{move: episode.Set, to: in.To, revision: *in.Revision}
	switch in.To {
	case episode.Admitted:
		c.move = episode.Admit
	case episode.Discharged:
		c.move = episode.Discharge
	}
	msg := ""
	c.reason, msg = optionalReason(in.Reason)
	if msg == "" && (c.move == episode.Admit || c.move == episode.Discharge) {
		c.date, msg = statusDate(in.Date, c.move)
	}
	if _, known := episode.Labels[in.To]; msg == "" && !known { // not a status at all: bad input, not a conflict
		msg = "Choose a status."
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	s.applyStatus(w, r, e, c)
}

// cancelEpisode is POST …/episodes/{eid}/cancel.
func (s *server) cancelEpisode(w http.ResponseWriter, r *http.Request) {
	e, in, ok := s.readStatusRequest(w, r)
	if !ok {
		return
	}
	reason, msg := episode.CancelReason(in.Reason, in.Details)
	if msg == "" && len(reason) > maxReasonLen {
		msg = "The reason is too long (1,000 characters at most)."
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	key := in.Reason
	s.applyStatus(w, r, e, statusChange{move: episode.Cancel, to: episode.Cancelled, reason: reason, cancel: &key, revision: *in.Revision})
}

// reopenReview is POST …/episodes/{eid}/reopen.
func (s *server) reopenReview(w http.ResponseWriter, r *http.Request) {
	e, in, ok := s.readStatusRequest(w, r)
	if !ok {
		return
	}
	reason, msg := requiredReason(in.Reason, "Say why the HQ review is being re-opened.")
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	s.applyStatus(w, r, e, statusChange{move: episode.Reopen, to: episode.ReadyForReview, reason: reason, revision: *in.Revision})
}

// undoStatus is POST …/episodes/{eid}/undo.
func (s *server) undoStatus(w http.ResponseWriter, r *http.Request) {
	e, in, ok := s.readStatusRequest(w, r)
	if !ok {
		return
	}
	reason, msg := requiredReason(in.Reason, "Say why the status change is being undone.")
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	s.applyStatus(w, r, e, statusChange{move: episode.Undo, reason: reason, revision: *in.Revision})
}

// readStatusRequest loads the episode (404 if it isn't this hospital's), then reads the body.
func (s *server) readStatusRequest(w http.ResponseWriter, r *http.Request) (db.GetEpisodeRow, statusRequest, bool) {
	var in statusRequest
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return e, in, false
	}
	if !readJSON(w, r, 8<<10, &in) {
		return e, in, false
	}
	if in.Revision == nil || *in.Revision < 0 {
		writeError(w, http.StatusBadRequest, "The request could not be read.")
		return e, in, false
	}
	in.To = strings.TrimSpace(in.To)
	return e, in, true
}

// optionalReason trims a reason that may be left out (S20 answers 4: for the other moves by hand).
func optionalReason(s string) (string, string) {
	s = strings.TrimSpace(s)
	if len(s) > maxReasonLen {
		return "", "The reason is too long (1,000 characters at most)."
	}
	return s, ""
}

// requiredReason is a reason that must be given: for cancel, re-open and undo.
func requiredReason(s, missing string) (string, string) {
	s, msg := optionalReason(s)
	if msg == "" && s == "" {
		msg = missing
	}
	return s, msg
}

// statusDate reads the admission or discharge date: required, a real date.
func statusDate(s string, move episode.Move) (pgtype.Date, string) {
	what := "admission"
	if move == episode.Discharge {
		what = "discharge"
	}
	d, ok := optionalDate(s)
	if !ok || !d.Valid {
		return d, "Enter the " + what + " date as a date."
	}
	return d, ""
}

// applyStatus makes the change in one transaction: the episode is locked, its revision and the
// transition table checked against it as it is now, then the status, the history and the General
// notes are written, with what goes with the move (the dates, the patient's side of a cancel).
func (s *server) applyStatus(w http.ResponseWriter, r *http.Request, e db.GetEpisodeRow, c statusChange) {
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	now, err := q.LockEpisodeStatus(ctx, e.ID)
	if err != nil {
		serverError(w, "lock episode", err)
		return
	}
	if now.Revision != c.revision {
		writeStale(w)
		return
	}
	from := now.Status
	var undo *db.LatestUndoableRow
	if c.move == episode.Undo {
		latest, err := q.LatestUndoable(ctx, e.ID)
		switch {
		case err == nil:
			undo = &latest
			c.to, _ = episode.UndoTarget(from, &episode.Event{Move: episode.Move(latest.Move), From: latest.FromStatus, To: latest.ToStatus})
		case !errors.Is(err, pgx.ErrNoRows):
			serverError(w, "latest status change", err)
			return
		}
		if c.to == "" {
			writeError(w, http.StatusConflict, "There's no status change to undo.")
			return
		}
	} else if _, err := episode.Check(from, c.to, c.move, here(r).Role); err != nil {
		writeStatusRefusal(w, err, from, c)
		return
	}

	set := db.SetEpisodeStatusParams{ID: e.ID, Status: c.to, AdmittedAt: now.AdmittedAt, DischargedAt: now.DischargedAt,
		CancelReason: c.cancel, Reopen: c.move == episode.Reopen}
	note := ""
	switch c.move {
	case episode.Admit:
		if c.date.Time.After(dateToday(s.today(now.Timezone))) {
			writeError(w, http.StatusBadRequest, "The admission date can't be in the future.")
			return
		}
		set.AdmittedAt = c.date
		note = "Status changed to Admitted (admission date " + dateString(c.date) + ")"
	case episode.Discharge:
		switch {
		case c.date.Time.After(dateToday(s.today(now.Timezone))):
			writeError(w, http.StatusBadRequest, "The discharge date can't be in the future.")
			return
		case now.AdmittedAt.Valid && c.date.Time.Before(now.AdmittedAt.Time):
			writeError(w, http.StatusBadRequest, "The discharge date can't be before the admission date ("+dateString(now.AdmittedAt)+").")
			return
		}
		set.DischargedAt = c.date
		note = "Status changed to Discharged (discharge date " + dateString(c.date) + ")"
	case episode.Cancel:
		note = "Episode cancelled: " + c.reason
	case episode.Reopen:
		note = "HQ review re-opened: " + c.reason
	case episode.Undo:
		// Stepping back an admission or a discharge takes its date away with it.
		if from == episode.Admitted {
			set.AdmittedAt = pgtype.Date{}
		}
		if from == episode.Discharged {
			set.DischargedAt = pgtype.Date{}
		}
		note = "Status change undone, back to " + episode.Labels[c.to] + ": " + c.reason
	default:
		note = "Status changed to " + episode.Labels[c.to]
		if c.reason != "" {
			note += ": " + c.reason
		}
	}
	uid := currentUser(r).ID
	if err := q.SetEpisodeStatus(ctx, set); err != nil {
		serverError(w, "set status", err)
		return
	}
	if undo != nil {
		if err := q.MarkUndone(ctx, undo.ID); err != nil {
			serverError(w, "mark undone", err)
			return
		}
	}
	if err := recordStatus(ctx, q, e.ID, c.move, from, c.to, c.reason, uid, note); err != nil {
		serverError(w, "status history", err)
		return
	}
	if c.move == episode.Cancel {
		if err := q.RevokeEpisodeInvites(ctx, uuid.NullUUID{UUID: e.ID, Valid: true}); err != nil {
			serverError(w, "revoke invites", err)
			return
		}
		if err := q.FinishEpisodeJobs(ctx, uuid.NullUUID{UUID: e.ID, Valid: true}); err != nil {
			serverError(w, "stop follow-up", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return
	}
	if stem := episode.StatusMessage(c.move, c.to); stem != "" {
		s.statusMessage(r, e.ID, stem, c.to) // EPI-08 (S21): a failure is noted, never refused
	}
	w.WriteHeader(http.StatusNoContent)
}

// statusMessage sends the patient the hospital's message for the new status, if it's switched on
// there (S21 answers 4), and notes in the General notes how it went. The move is already saved, so
// a failure is noted and logged, never refused.
func (s *server) statusMessage(r *http.Request, eid uuid.UUID, stem, status string) {
	by, what := currentUser(r).ID, episode.Labels[status]+" message"
	res, on, err := s.inviter.SendStatus(context.WithoutCancel(r.Context()), eid, stem, status) // the move is saved: finish what's started
	if err != nil {
		log.Printf("status message for episode %s: %v", eid, err)
		s.event(r, eid, what+" not sent: something went wrong.", by)
		return
	}
	if on {
		s.event(r, eid, res.Note(what+" sent", what+" not sent"), by)
	}
}

// writeStatusRefusal answers a move the table refuses: 403 for a role too low, else 409.
func writeStatusRefusal(w http.ResponseWriter, err error, from string, c statusChange) {
	switch {
	case errors.Is(err, episode.ErrRole) && c.move == episode.Reopen:
		writeError(w, http.StatusForbidden, "Only a super clinician or an admin can re-open the HQ review.")
	case errors.Is(err, episode.ErrRole):
		writeError(w, http.StatusForbidden, "Your role can't make this change.")
	case from == episode.Cancelled:
		writeError(w, http.StatusConflict, cancelledMsg)
	case c.move == episode.Reopen:
		writeError(w, http.StatusConflict, "The HQ review can be re-opened only after it's complete and before admission.")
	case c.move == episode.Cancel:
		writeError(w, http.StatusConflict, "A discharged episode can't be cancelled.")
	default:
		writeError(w, http.StatusConflict, "This episode is "+labelOf(from)+", so its status can't be changed to "+labelOf(c.to)+".")
	}
}

func labelOf(status string) string {
	if l, ok := episode.Labels[status]; ok {
		return l
	}
	return "“" + status + "”"
}

// recordStatus writes one change to the status history (EPI-03) and its General notes line.
// user is uuid.Nil for the patient. from is "" for the first status (create).
func recordStatus(ctx context.Context, q *db.Queries, eid uuid.UUID, move episode.Move, from, to, reason string, user uuid.UUID, note string) error {
	var fromStatus *string
	if from != "" {
		fromStatus = &from
	}
	who := uuid.NullUUID{UUID: user, Valid: user != uuid.Nil}
	if err := q.AddStatusEvent(ctx, db.AddStatusEventParams{EpisodeID: eid, Move: string(move), FromStatus: fromStatus, ToStatus: to,
		Reason: reason, UserID: who}); err != nil {
		return err
	}
	if note == "" {
		return nil
	}
	return q.AddEpisodeEvent(ctx, db.AddEpisodeEventParams{EpisodeID: eid, Kind: "event", Text: note, UserID: who})
}

// statusView is what the episode page shows about the status: the moves this user may make
// from here (the status control offers only these), the undo if there's one, and the history.
type statusView struct {
	Moves   []episode.Option         `json:"moves"`
	Undo    *undoOption              `json:"undo"`
	History []db.ListStatusEventsRow `json:"statusHistory"`
}

// undoOption is the undo the control offers: back to `to`, from the current status.
type undoOption struct {
	To    string `json:"to"`
	Label string `json:"label"`
}

func (s *server) statusView(ctx context.Context, e db.GetEpisodeRow, who role.Role) (statusView, error) {
	v := statusView{Moves: episode.Options(e.Status, who)}
	if v.Moves == nil {
		v.Moves = []episode.Option{}
	}
	latest, err := s.q.LatestUndoable(ctx, e.ID)
	switch {
	case err == nil:
		if to, ok := episode.UndoTarget(e.Status, &episode.Event{Move: episode.Move(latest.Move), From: latest.FromStatus, To: latest.ToStatus}); ok && who.AtLeast(role.Clinician) {
			v.Undo = &undoOption{To: to, Label: episode.Labels[to]}
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return v, err
	}
	v.History, err = s.q.ListStatusEvents(ctx, e.ID)
	if v.History == nil {
		v.History = []db.ListStatusEventsRow{}
	}
	return v, err
}

// today's date at a hospital, as a date (midnight UTC), to compare entered dates with.
func dateToday(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

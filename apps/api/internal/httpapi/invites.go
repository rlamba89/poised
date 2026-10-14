package httpapi

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/invite"
)

// Patient invites (plan C2, session S14; PAT-03): creating an episode texts and emails the
// patient a link to their HQ, by whichever the record has; staff can send it again. The wording
// is the hospital's message templates (settings, S10), with the hospital's name. A new invite
// replaces the episode's earlier links, so only the newest works. The sending itself is
// internal/invite's, shared with the reminders (S16).
//
// A message that can't be sent never stops the episode being created: what was sent, and what
// wasn't, is written to the episode's General notes, where staff see it and can resend.

// inviteAfterCreate sends a new episode's invite. A failure is logged and noted on the episode,
// never refused: the episode is saved, and staff can resend.
func (s *server) inviteAfterCreate(r *http.Request, episodeID uuid.UUID) {
	by := currentUser(r).ID
	ctx := context.WithoutCancel(r.Context()) // the episode exists: finish what's started
	note := "No invite sent: something went wrong. Use Resend invite to try again."
	if res, err := s.inviter.Send(ctx, episodeID, by, invite.First); err != nil {
		log.Printf("invite for episode %s: %v", episodeID, err)
	} else {
		note = res.Note("Invite sent", "No invite sent")
	}
	s.event(r, episodeID, note, by)
}

// resendInvite is POST …/episodes/{eid}/resend-invite: a new link, by SMS and email, replacing the
// old one (which also undoes a lock: PAT-05's "ask for a new one").
func (s *server) resendInvite(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	if e.PatientSubmittedAt.Valid {
		writeError(w, http.StatusConflict, "The patient's answers have already been sent, so there's nothing to invite them to.")
		return
	}
	by := currentUser(r).ID
	res, err := s.inviter.Send(context.WithoutCancel(r.Context()), e.ID, by, invite.Again)
	if err != nil {
		serverError(w, "resend invite", err)
		return
	}
	note := res.Note("Invite sent again", "No invite sent")
	if !res.Tried { // no mobile that can be texted, and no email: nothing written
		writeError(w, http.StatusConflict, note+" Fill in the HQ on their behalf, or ask them to contact the hospital.")
		return
	}
	s.event(r, e.ID, note, by)
	if !res.Sent {
		writeError(w, http.StatusBadGateway, note)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": note})
}

// qaMessages is GET /api/qa/messages?to=: the newest captured messages (MESSAGING=capture only),
// for the captured-messages page and the end-to-end journeys. No sign-in: there is nothing to
// sign in to locally, and capture mode is refused outside QA (S17's CDK check).
func (s *server) qaMessages(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListSentMessages(r.Context(), strings.TrimSpace(r.URL.Query().Get("to")))
	if err != nil {
		serverError(w, "list messages", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// qaResetRateLimits is DELETE /api/qa/rate-limits (MESSAGING=capture only): forgets every sign-in
// and verification request counted (S15), so a tester or a journey run from one machine, which all
// counts as one IP address, can start again without waiting an hour. Like the captured messages,
// it never exists where real messages are sent.
func (s *server) qaResetRateLimits(w http.ResponseWriter, r *http.Request) {
	if err := s.q.ResetRateLimits(r.Context()); err != nil {
		serverError(w, "reset rate limits", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// qaClient answers with the address the rate limits count this caller by (clientIP) and the
// X-Forwarded-For the API received, so a tester can check on QA that behind Amplify's rewrite
// and API Gateway it's the browser's own address (S06, config FORWARDED_HOPS). Capture mode only.
func (s *server) qaClient(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"address":      clientIP(r),
		"forwardedFor": r.Header.Get("X-Forwarded-For"),
	})
}

package httpapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/patient"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// Patient invites (plan C2, session S14; PAT-03): creating an episode texts and emails the
// patient a link to their HQ, by whichever the record has; staff can send it again. The wording
// is the hospital's message templates (settings, S10), with the hospital's name. A new invite
// replaces the episode's earlier links, so only the newest works.
//
// A message that can't be sent never stops the episode being created: what was sent, and what
// wasn't, is written to the episode's General notes, where staff see it and can resend.

// inviteResult is what an invite did, in words for the General notes.
type inviteResult struct {
	tried bool   // the patient can be reached, so a link was made and sent
	sent  bool   // at least one message went
	note  string // for the General notes
}

// invite sends the episode's patient a new link by SMS and email. by is the staff member; again
// marks a resend. It returns an error only when nothing could be tried.
func (s *server) invite(r *http.Request, episodeID, by uuid.UUID, again bool) (inviteResult, error) {
	ctx := context.WithoutCancel(r.Context()) // the episode exists: finish what's started
	d, err := s.q.GetInviteDetails(ctx, episodeID)
	if err != nil {
		return inviteResult{}, err
	}
	useSMS := d.Mobile != "" && patient.SMSAllowed(d.Mobile, s.cfg.Country.PhoneCountry)
	var notes []string
	if d.Mobile != "" && !useSMS { // PAT-11: texts only to the deployment's country (SMS pumping)
		notes = append(notes, "No text message: texts go only to mobile numbers in this country.")
	}
	if !useSMS && d.Email == "" {
		if d.Mobile == "" {
			notes = append(notes, "The patient has no mobile or email.")
		}
		return inviteResult{note: "No invite sent. " + strings.Join(notes, " ")}, nil // tried: false
	}

	// The order matters (S14 verification 1): the messages are built first, then the new link is
	// stored (so it works the moment a message arrives), then they're sent, and only then does
	// the new link replace the old ones. If nothing went, the new link is dropped and the old one,
	// with any session opened from it, carries on.
	token, err := auth.NewLinkToken()
	if err != nil {
		return inviteResult{}, err
	}
	msgs, err := inviteMessages(d, s.cfg.Country.AppURL+"/i/"+token)
	if err != nil {
		return inviteResult{}, err
	}
	link, err := s.q.CreateInviteLink(ctx, db.CreateInviteLinkParams{
		TokenHash: auth.HashLinkToken(token), PatientID: uuid.NullUUID{UUID: d.PatientID, Valid: true},
		EpisodeID: uuid.NullUUID{UUID: d.ID, Valid: true}, CreatedBy: uuid.NullUUID{UUID: by, Valid: by != uuid.Nil},
	})
	if err != nil {
		return inviteResult{}, err
	}

	var sentBy []string
	if useSMS {
		if err := s.sms.SendSMS(ctx, d.Mobile, msgs.sms); err != nil {
			log.Printf("invite for episode %s: text message: %v", episodeID, err)
			notes = append(notes, "The text message couldn't be sent.")
		} else {
			sentBy = append(sentBy, "text message")
		}
	}
	if d.Email != "" {
		if err := s.email.SendEmail(ctx, d.Email, msgs.subject, msgs.email); err != nil {
			log.Printf("invite for episode %s: email: %v", episodeID, err)
			notes = append(notes, "The email couldn't be sent.")
		} else {
			sentBy = append(sentBy, "email")
		}
	}
	res := inviteResult{tried: true, sent: len(sentBy) > 0}
	if err := s.settleInviteLink(ctx, d.ID, link, res.sent); err != nil {
		return inviteResult{}, err
	}
	switch {
	case res.sent && again:
		res.note = "Invite sent again by " + strings.Join(sentBy, " and ") + "."
	case res.sent:
		res.note = "Invite sent by " + strings.Join(sentBy, " and ") + "."
	default:
		res.note = "No invite sent."
	}
	if len(notes) > 0 {
		res.note += " " + strings.Join(notes, " ")
	}
	return res, nil
}

// settleInviteLink, once the messages have been tried, makes the new link the episode's only
// working one if a message went (revoking only the links older than it, so of two resends that
// cross, the newer wins), or drops it if none did. The episode's row is locked meanwhile, so two
// resends are settled one after the other.
func (s *server) settleInviteLink(ctx context.Context, episodeID uuid.UUID, link db.CreateInviteLinkRow, sent bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	if _, err := q.LockEpisode(ctx, episodeID); err != nil {
		return err
	}
	if sent {
		err = q.RevokeOlderInviteLinks(ctx, db.RevokeOlderInviteLinksParams{
			EpisodeID: uuid.NullUUID{UUID: episodeID, Valid: true}, CreatedAt: link.CreatedAt, ID: link.ID})
	} else {
		err = q.RevokeLink(ctx, link.ID)
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type invitation struct{ sms, subject, email string }

// inviteMessages fills in the hospital's invitation templates (its trust's, or Poised's defaults).
func inviteMessages(d db.GetInviteDetailsRow, link string) (invitation, error) {
	org, err := settings.Decode(d.OrgSettings)
	if err != nil {
		return invitation{}, fmt.Errorf("trust settings: %w", err)
	}
	own, err := settings.Decode(d.HospitalSettings)
	if err != nil {
		return invitation{}, fmt.Errorf("hospital settings: %w", err)
	}
	v := settings.Effective(settings.Platform(), org, hospitalOnly(own))
	vars := map[string]string{"firstName": d.FirstName, "hospital": d.HospitalName, "link": link}
	var m invitation
	if m.sms, err = settings.Fill(v, "templates.inviteSms", vars); err != nil {
		return m, err
	}
	if m.email, err = settings.Fill(v, "templates.inviteEmail", vars); err != nil {
		return m, err
	}
	m.subject = d.HospitalName + ": your health questionnaire"
	return m, nil
}

// inviteAfterCreate sends a new episode's invite. A failure is logged and noted on the episode,
// never refused: the episode is saved, and staff can resend.
func (s *server) inviteAfterCreate(r *http.Request, episodeID uuid.UUID) {
	by := currentUser(r).ID
	res, err := s.invite(r, episodeID, by, false)
	if err != nil {
		log.Printf("invite for episode %s: %v", episodeID, err)
		res.note = "No invite sent: something went wrong. Use Resend invite to try again."
	}
	s.event(r, episodeID, res.note, by)
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
	res, err := s.invite(r, e.ID, by, true)
	if err != nil {
		serverError(w, "resend invite", err)
		return
	}
	if !res.tried { // no mobile that can be texted, and no email: nothing written
		writeError(w, http.StatusConflict, res.note+" Fill in the HQ on their behalf, or ask them to contact the hospital.")
		return
	}
	s.event(r, e.ID, res.note, by)
	if !res.sent {
		writeError(w, http.StatusBadGateway, res.note)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": res.note})
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

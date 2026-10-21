// Package invite texts and emails a patient a new link to their episode's HQ (plan C2; PAT-03):
// the invite when the episode is created and staff's Resend (S14), and the reminders the jobs
// send (S16). The wording is the hospital's message templates (settings, S10). It moved here from
// internal/httpapi in S16 so the jobs (internal/jobs), which have no HTTP request, can use it too.
//
// A message that can't be sent is reported, never retried: the caller writes what went, and what
// didn't, to the episode's General notes, where staff see it and can resend.
package invite

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/clock"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
	"github.com/rlamba89/poised/apps/api/internal/patient"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// Kind is why a link is sent.
type Kind int

const (
	First    Kind = iota // creating the episode
	Again                // staff's Resend: the new link replaces the old ones (S14)
	Reminder             // a reminder (S16): the patient's earlier links keep working
)

// Sender sends the links. Its zero value isn't usable: every field is needed.
type Sender struct {
	Pool         *pgxpool.Pool
	SMS          messaging.SMS
	Email        messaging.Email
	Clock        clock.Clock
	AppURL       string // the deployment's web address (config.Country.AppURL)
	PhoneCountry string // texts go only to this country's mobiles (PAT-11)
}

// Result is what a send did.
type Result struct {
	Tried    bool     // the patient can be reached, so a link was made and sent
	Sent     bool     // at least one message went
	SentBy   []string // "text message", "email"
	Problems []string // sentences for the General notes: what didn't go, and why
}

// Note describes the result for the General notes: "<sent> by text message and email." or
// "<notSent>.", then any problems.
func (r Result) Note(sent, notSent string) string {
	note := notSent + "."
	if r.Sent {
		note = sent + " by " + strings.Join(r.SentBy, " and ") + "."
	}
	if len(r.Problems) > 0 {
		note += " " + strings.Join(r.Problems, " ")
	}
	return note
}

// Send sends the episode's patient a new link by SMS and email, by whichever their record has
// now. by is the staff member (uuid.Nil for the jobs). It returns an error only when something
// other than a message failed (the database); then nothing may have been sent.
//
// The first invite that goes starts the episode's follow-up (reminders and "not responding",
// S16), counted from now.
func (s Sender) Send(ctx context.Context, episodeID, by uuid.UUID, kind Kind) (Result, error) {
	q := db.New(s.Pool)
	d, err := q.GetInviteDetails(ctx, episodeID)
	if err != nil {
		return Result{}, err
	}
	var res Result
	useSMS := d.Mobile != "" && patient.SMSAllowed(d.Mobile, s.PhoneCountry)
	if d.Mobile != "" && !useSMS { // PAT-11: texts only to the deployment's country (SMS pumping)
		res.Problems = append(res.Problems, "No text message: texts go only to mobile numbers in this country.")
	}
	if !useSMS && d.Email == "" {
		if d.Mobile == "" {
			res.Problems = append(res.Problems, "The patient has no mobile or email.")
		}
		return res, nil // Tried: false
	}
	res.Tried = true

	// The order matters (S14 verification 1): the messages are built first, then the new link is
	// stored (so it works the moment a message arrives), then they're sent, and only then does
	// the new link replace the old ones. If nothing went, the new link is dropped and the old one,
	// with any session opened from it, carries on.
	token, err := auth.NewLinkToken()
	if err != nil {
		return Result{}, err
	}
	msgs, err := messages(d, s.AppURL+"/i/"+token, kind)
	if err != nil {
		return Result{}, err
	}
	link, err := q.CreateInviteLink(ctx, db.CreateInviteLinkParams{
		TokenHash: auth.HashLinkToken(token), PatientID: uuid.NullUUID{UUID: d.PatientID, Valid: true},
		EpisodeID: uuid.NullUUID{UUID: d.ID, Valid: true}, CreatedBy: uuid.NullUUID{UUID: by, Valid: by != uuid.Nil},
	})
	if err != nil {
		return Result{}, err
	}

	now := s.now() // one time for the send's text and email: the episode's undelivered list groups by it
	if useSMS {
		id, err := s.SMS.SendSMS(ctx, d.Mobile, msgs.sms)
		if err != nil {
			log.Printf("invite for episode %s: text message: %v", episodeID, err)
			res.Problems = append(res.Problems, "The text message couldn't be sent.")
		} else {
			res.SentBy = append(res.SentBy, "text message")
		}
		s.record(ctx, q, d.ID, kind.name(), nil, "sms", id, err, now)
	}
	if d.Email != "" {
		id, err := s.Email.SendEmail(ctx, d.Email, msgs.subject, msgs.email)
		if err != nil {
			log.Printf("invite for episode %s: email: %v", episodeID, err)
			res.Problems = append(res.Problems, "The email couldn't be sent.")
		} else {
			res.SentBy = append(res.SentBy, "email")
		}
		s.record(ctx, q, d.ID, kind.name(), nil, "email", id, err, now)
	}
	res.Sent = len(res.SentBy) > 0
	if err := s.settle(ctx, d.ID, link, res.Sent, kind); err != nil {
		return Result{}, err
	}
	return res, nil
}

// settle, once the messages have been tried, drops the new link if none went. If one did:
//   - an invite or Resend makes it the episode's only working link (revoking only the links older
//     than it, so of two resends that cross, the newer wins), with the episode's five
//     date-of-birth tries again (clearing a lock), and the first starts the follow-up;
//   - a reminder's link is added beside the patient's earlier ones, which keep working, so the
//     invite they already have still opens their HQ (S16).
//
// The episode's row is locked meanwhile, so two sends are settled one after the other.
func (s Sender) settle(ctx context.Context, episodeID uuid.UUID, link db.CreateInviteLinkRow, sent bool, kind Kind) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	if _, err := q.LockEpisode(ctx, episodeID); err != nil {
		return err
	}
	switch {
	case !sent:
		err = q.RevokeLink(ctx, link.ID)
	case kind == Reminder:
	default:
		err = q.RevokeOlderInviteLinks(ctx, db.RevokeOlderInviteLinksParams{
			EpisodeID: uuid.NullUUID{UUID: episodeID, Valid: true}, CreatedAt: link.CreatedAt, ID: link.ID})
		if err == nil { // the new link has the episode's five tries again: this is what clears a lock
			err = q.ResetEpisodeDob(ctx, episodeID)
		}
		if err == nil {
			err = q.StartFollowup(ctx, db.StartFollowupParams{EpisodeID: episodeID, Now: s.now()})
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// name is the kind as episode_messages records it.
func (k Kind) name() string {
	if k == Reminder {
		return "reminder"
	}
	return "invite"
}

// record notes one message on the episode (S17), with its ID, which AWS's delivery events name
// (internal/delivery), or as not sent. kind is "invite", "reminder" or "status" (S21, with the
// status it was sent for). The message is already sent, so a failure here is only logged: it
// costs the message's place in the undelivered list, not the message.
func (s Sender) record(ctx context.Context, q *db.Queries, episodeID uuid.UUID, kind string, status *string, channel, id string, sendErr error, now time.Time) {
	row := db.RecordEpisodeMessageParams{EpisodeID: episodeID, Kind: kind, Status: status, Channel: channel, Now: now}
	if sendErr != nil {
		row.FailedAt = pgtype.Timestamptz{Time: now, Valid: true}
		row.Failure = "it couldn't be sent"
	} else {
		row.ProviderID = &id
	}
	if err := q.RecordEpisodeMessage(ctx, row); err != nil {
		log.Printf("invite for episode %s: record the %s: %v", episodeID, channel, err)
	}
}

func (s Sender) now() time.Time {
	if s.Clock == nil {
		return time.Now()
	}
	return s.Clock.Now()
}

type texts struct{ sms, subject, email string }

// messages fills in the hospital's templates (its trust's, or Preopsis's defaults): the invitation
// for an invite or Resend, the reminder for a reminder.
func messages(d db.GetInviteDetailsRow, link string, kind Kind) (texts, error) {
	v, err := effective(d) // a hospital's value counts only where it may override
	if err != nil {
		return texts{}, err
	}
	vars := map[string]string{"firstName": d.FirstName, "hospital": d.HospitalName, "link": link}
	smsKey, emailKey, subject := "templates.inviteSms", "templates.inviteEmail", d.HospitalName+": your health questionnaire"
	if kind == Reminder {
		smsKey, emailKey, subject = "templates.reminderSms", "templates.reminderEmail", d.HospitalName+": a reminder about your health questionnaire"
	}
	var m texts
	if m.sms, err = settings.Fill(v, smsKey, vars); err != nil {
		return m, err
	}
	if m.email, err = settings.Fill(v, emailKey, vars); err != nil {
		return m, err
	}
	m.subject = subject
	return m, nil
}

// SendStatus sends the patient the hospital's message about their episode's new status (EPI-08,
// S21 answers 4): stem is episode.StatusMessage's ("readyForAdmission", "cancelled"), status the
// status it's for. The text and the email each go only if the hospital's switch for it is on
// (statusMessages.<stem>Sms / Email, off by default), with templates.<stem>Sms / Email. There's
// no new link: {{link}}, if the template has it, is the patient's sign-in page (S15).
//
// on is false when both switches are off: then nothing was tried, and there's nothing to note.
// It returns an error only when something other than a message failed (the database).
func (s Sender) SendStatus(ctx context.Context, episodeID uuid.UUID, stem, status string) (res Result, on bool, err error) {
	q := db.New(s.Pool)
	d, err := q.GetInviteDetails(ctx, episodeID)
	if err != nil {
		return Result{}, false, err
	}
	v, err := effective(d)
	if err != nil {
		return Result{}, false, err
	}
	var smsOn, emailOn bool
	if err := settings.Get(v, "statusMessages."+stem+"Sms", &smsOn); err != nil {
		return Result{}, false, err
	}
	if err := settings.Get(v, "statusMessages."+stem+"Email", &emailOn); err != nil {
		return Result{}, false, err
	}
	if !smsOn && !emailOn {
		return Result{}, false, nil
	}
	useSMS := smsOn && d.Mobile != "" && patient.SMSAllowed(d.Mobile, s.PhoneCountry)
	useEmail := emailOn && d.Email != ""
	if smsOn && d.Mobile != "" && !useSMS { // PAT-11: texts only to the deployment's country
		res.Problems = append(res.Problems, "No text message: texts go only to mobile numbers in this country.")
	}
	switch noMobile, noEmail := smsOn && d.Mobile == "", emailOn && d.Email == ""; {
	case noMobile && noEmail:
		res.Problems = append(res.Problems, "The patient has no mobile or email.")
	case noMobile:
		res.Problems = append(res.Problems, "The patient has no mobile.")
	case noEmail:
		res.Problems = append(res.Problems, "The patient has no email address.")
	}
	if !useSMS && !useEmail {
		return res, true, nil
	}
	res.Tried = true
	vars := map[string]string{"firstName": d.FirstName, "hospital": d.HospitalName, "link": s.AppURL + "/signin"}
	now := s.now()
	if useSMS {
		body, err := settings.Fill(v, "templates."+stem+"Sms", vars)
		if err != nil {
			return Result{}, true, err
		}
		id, err := s.SMS.SendSMS(ctx, d.Mobile, body)
		if err != nil {
			log.Printf("status message for episode %s: text message: %v", episodeID, err)
			res.Problems = append(res.Problems, "The text message couldn't be sent.")
		} else {
			res.SentBy = append(res.SentBy, "text message")
		}
		s.record(ctx, q, d.ID, "status", &status, "sms", id, err, now)
	}
	if useEmail {
		body, err := settings.Fill(v, "templates."+stem+"Email", vars)
		if err != nil {
			return Result{}, true, err
		}
		subject := d.HospitalName + ": ready for your operation"
		if status != "ready_for_admission" {
			subject = d.HospitalName + ": your health questionnaire was cancelled"
		}
		id, err := s.Email.SendEmail(ctx, d.Email, subject, body)
		if err != nil {
			log.Printf("status message for episode %s: email: %v", episodeID, err)
			res.Problems = append(res.Problems, "The email couldn't be sent.")
		} else {
			res.SentBy = append(res.SentBy, "email")
		}
		s.record(ctx, q, d.ID, "status", &status, "email", id, err, now)
	}
	res.Sent = len(res.SentBy) > 0
	return res, true, nil
}

// effective is the hospital's settings: its own where it may override, else its trust's, else
// Preopsis's defaults.
func effective(d db.GetInviteDetailsRow) (settings.Values, error) {
	org, err := settings.Decode(d.OrgSettings)
	if err != nil {
		return nil, fmt.Errorf("trust settings: %w", err)
	}
	own, err := settings.Decode(d.HospitalSettings)
	if err != nil {
		return nil, fmt.Errorf("hospital settings: %w", err)
	}
	return settings.Effective(settings.Platform(), org, own), nil
}

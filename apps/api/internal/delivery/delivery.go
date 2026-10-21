// Package delivery records the texts and emails AWS reports as undelivered (plan C2, session S17;
// OPS-05). SES (a bounce, a reject) and End User Messaging SMS (an invalid number, an unreachable
// phone…) publish their delivery events to the deployment's SNS topic, which runs the jobs Lambda
// (cmd/jobs) with them. Each failure marks the episode's message (episode_messages), adds a
// General notes entry for staff, and logs a DELIVERY_FAILED line, which the alarm counts.
//
// A message that isn't an episode's (a sign-in link, a contact change) is only logged and counted.
package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
)

// Failure is one message that didn't reach the patient.
type Failure struct {
	ProviderID string // SES's or End User Messaging SMS's message ID
	Channel    string // "sms" or "email"
	Reason     string // why, in words for staff: "the mobile number isn't valid"
}

// event holds the fields used from either service's event.
type event struct {
	EventType string `json:"eventType"`
	MessageID string `json:"messageId"` // SMS
	Mail      struct {
		MessageID string `json:"messageId"`
	} `json:"mail"` // SES
	Bounce struct {
		BounceType string `json:"bounceType"`
	} `json:"bounce"` // SES
}

// smsFailures are End User Messaging SMS's final failure events (its TEXT_* event types; the
// configuration set publishes only these), with the reason staff see.
var smsFailures = map[string]string{
	"TEXT_INVALID":             "the mobile number isn't valid",
	"TEXT_UNREACHABLE":         "the phone couldn't be reached (it may be switched off)",
	"TEXT_CARRIER_UNREACHABLE": "the mobile network couldn't deliver it",
	"TEXT_TTL_EXPIRED":         "the mobile network couldn't deliver it",
	"TEXT_UNKNOWN":             "the mobile network couldn't deliver it", // no final word within 72 hours
	"TEXT_BLOCKED":             "the phone blocks these texts",
	"TEXT_CARRIER_BLOCKED":     "the mobile network blocked it",
	"TEXT_SPAM":                "the mobile network blocked it",
	"TEXT_INVALID_MESSAGE":     "the text itself couldn't be delivered",
	"TEXT_PROTECT_BLOCKED":     "texts go only to mobile numbers in this country",
}

// SMSFailureEvents are the event types infra's SMS configuration set publishes (keep the two the same).
func SMSFailureEvents() []string {
	out := make([]string, 0, len(smsFailures))
	for k := range smsFailures {
		out = append(out, k)
	}
	return out
}

var bounceReasons = map[string]string{
	"Permanent": "the email address doesn't exist or doesn't accept email",
	"Transient": "the patient's mailbox didn't accept it (it may be full)",
}

// Parse reads one event (an SNS message's body). failed is false for an event that isn't a
// failure (a delivery, a delay, SNS's own messages); an error means a failure event without the
// message's ID, or one that isn't JSON.
func Parse(message string) (f Failure, failed bool, err error) {
	var e event
	if err := json.Unmarshal([]byte(message), &e); err != nil {
		return Failure{}, false, fmt.Errorf("delivery event: %w", err)
	}
	switch e.EventType {
	case "Bounce", "Reject", "Rendering Failure":
		f = Failure{ProviderID: e.Mail.MessageID, Channel: "email", Reason: "the email service refused it"}
		if e.EventType == "Bounce" {
			f.Reason = "it bounced"
			if r, ok := bounceReasons[e.Bounce.BounceType]; ok {
				f.Reason = r
			}
		}
	default:
		reason, ok := smsFailures[e.EventType]
		if !ok {
			return Failure{}, false, nil
		}
		f = Failure{ProviderID: e.MessageID, Channel: "sms", Reason: reason}
	}
	if f.ProviderID == "" {
		return Failure{}, false, fmt.Errorf("delivery event %s has no message ID", e.EventType)
	}
	return f, true, nil
}

// Note is the General notes entry for a failure: kind is the message's ("invite", "reminder" or
// "status"), and status, for a status message (S21), the status it was sent for.
func Note(kind, status, channel, reason string) string {
	what, contact := "email", "email address"
	if channel == "sms" {
		what, contact = "text message", "mobile number"
	}
	if kind == "status" {
		return fmt.Sprintf("The %s %s wasn't delivered: %s. Check the patient's %s, and tell them another way.",
			episode.Labels[status], what, reason, contact)
	}
	which := "The invite's"
	if kind == "reminder" {
		which = "A reminder's"
	}
	return fmt.Sprintf("%s %s wasn't delivered: %s. Check the patient's %s, then resend the invite.", which, what, reason, contact)
}

// Record marks the message undelivered and tells staff. A second report of the same message
// changes nothing. It returns an error only when the database fails (then SNS tries again).
func Record(ctx context.Context, pool *pgxpool.Pool, f Failure, now time.Time) error {
	tx, err := pool.Begin(ctx) // the mark and the note together, so a retry after a failure adds the note
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	row, err := q.FailEpisodeMessage(ctx, db.FailEpisodeMessageParams{ProviderID: f.ProviderID, Failure: f.Reason, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		messaging.LogDeliveryFailed(fmt.Sprintf("%s %s not delivered (%s): not an episode's, or already noted", f.Channel, f.ProviderID, f.Reason))
		return nil
	}
	if err != nil {
		return err
	}
	if err := q.AddAutomaticEpisodeEvent(ctx, db.AddAutomaticEpisodeEventParams{EpisodeID: row.EpisodeID, Text: Note(row.Kind, row.Status, row.Channel, f.Reason)}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	messaging.LogDeliveryFailed(fmt.Sprintf("%s %s for episode %s not delivered (%s)", f.Channel, f.ProviderID, row.EpisodeID, f.Reason))
	return nil
}

// HandleSNS records every failure in the SNS notification the jobs Lambda was given. An event it
// can't read is logged and skipped (a retry wouldn't help); a database error is returned, so the
// Lambda fails and SNS tries again.
func HandleSNS(ctx context.Context, pool *pgxpool.Pool, ev events.SNSEvent, now time.Time) error {
	var errs []error
	for _, r := range ev.Records {
		f, failed, err := Parse(r.SNS.Message)
		switch {
		case err != nil:
			log.Printf("delivery: %v", err)
		case failed:
			if err := Record(ctx, pool, f, now); err != nil {
				errs = append(errs, fmt.Errorf("delivery %s: %w", f.ProviderID, err))
			}
		}
	}
	return errors.Join(errs...)
}

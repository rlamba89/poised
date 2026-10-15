// Package messaging sends patients their texts and emails (plan C2): invites, reminders and
// sign-in links. The senders:
//   - Capture (MESSAGING=capture: local and QA) stores every message in sent_messages and sends
//     nothing; the captured-messages page and GET /api/qa/messages read them back;
//   - SES and SMSAWS (aws.go; MESSAGING=send, S17) really send, by SES v2 and AWS End User
//     Messaging SMS;
//   - TestRecipients (QA, S17) captures every message and also really sends the ones addressed to
//     the named test recipients, so QA can send a real text and email to Rahul only;
//   - Recorder is the fake for tests: it records each message.
//
// Each returns the message's ID (the provider's, for a real send), which AWS's later delivery
// events name (internal/delivery).
package messaging

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

// SMS sends a text message to a mobile in E.164 and returns the message's ID. The caller has
// checked the number may be texted (patient.SMSAllowed, PAT-11). An error never holds the number.
type SMS interface {
	SendSMS(ctx context.Context, to, body string) (id string, err error)
}

// Email sends a plain-text email and returns the message's ID. An error never holds the address.
type Email interface {
	SendEmail(ctx context.Context, to, subject, body string) (id string, err error)
}

// Sender sends both.
type Sender interface {
	SMS
	Email
}

// Message is one text or email.
type Message struct {
	ID      string
	Channel string // "sms" or "email"
	To      string
	Subject string // "" for a text
	Body    string
}

// Capture stores messages in sent_messages instead of sending them (MESSAGING=capture: local
// and QA only). A message's ID is "capture-<its row>".
type Capture struct{ DB db.Querier }

func (c Capture) SendSMS(ctx context.Context, to, body string) (string, error) {
	n, err := c.DB.InsertSentMessage(ctx, db.InsertSentMessageParams{Channel: "sms", Recipient: to, Body: body})
	return captureID(n), err
}

func (c Capture) SendEmail(ctx context.Context, to, subject, body string) (string, error) {
	n, err := c.DB.InsertSentMessage(ctx, db.InsertSentMessageParams{Channel: "email", Recipient: to, Subject: subject, Body: body})
	return captureID(n), err
}

func captureID(n int64) string { return fmt.Sprintf("capture-%d", n) }

// TestRecipients is QA's way to send a real message (S17): every message is captured, as in
// capture mode, and one addressed to a test recipient is also really sent, by SMS or Email. So
// the journeys and testers keep reading the captured messages, and nobody but the named
// recipients (and SES's mailbox simulator, which is AWS's own) ever receives anything.
type TestRecipients struct {
	Capture Sender // Capture, in a real server
	SMS     SMS    // the real senders
	Email   Email  //
	// Allowed are the test recipients: E.164 mobiles and email addresses (any case).
	Allowed []string
}

// mailboxSimulator is SES's test domain: its addresses bounce, complain or deliver on purpose
// and never reach a person, and they work in SES's sandbox.
const mailboxSimulator = "@simulator.amazonses.com"

func (t TestRecipients) allowed(to string) bool {
	if strings.HasSuffix(strings.ToLower(to), mailboxSimulator) {
		return true
	}
	for _, a := range t.Allowed {
		if strings.EqualFold(strings.TrimSpace(a), to) {
			return true
		}
	}
	return false
}

// SendSMS captures the text, and really sends it to a test recipient: then its ID is the real one.
func (t TestRecipients) SendSMS(ctx context.Context, to, body string) (string, error) {
	id, err := t.Capture.SendSMS(ctx, to, body)
	if err != nil || !t.allowed(to) {
		return id, err
	}
	return t.SMS.SendSMS(ctx, to, body)
}

// SendEmail captures the email, and really sends it to a test recipient.
func (t TestRecipients) SendEmail(ctx context.Context, to, subject, body string) (string, error) {
	id, err := t.Capture.SendEmail(ctx, to, subject, body)
	if err != nil || !t.allowed(to) {
		return id, err
	}
	return t.Email.SendEmail(ctx, to, subject, body)
}

// Recorder is the fake for tests: it records every message that succeeds.
type Recorder struct {
	mu   sync.Mutex
	sent []Message
	err  error
}

// FailWith makes every later message fail with err (nil: succeed again).
func (r *Recorder) FailWith(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

func (r *Recorder) SendSMS(_ context.Context, to, body string) (string, error) {
	return r.record(Message{Channel: "sms", To: to, Body: body})
}

func (r *Recorder) SendEmail(_ context.Context, to, subject, body string) (string, error) {
	return r.record(Message{Channel: "email", To: to, Subject: subject, Body: body})
}

// record keeps m with the next ID, "rec-1", "rec-2"…
func (r *Recorder) record(m Message) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return "", r.err
	}
	m.ID = fmt.Sprintf("rec-%d", len(r.sent)+1)
	r.sent = append(r.sent, m)
	return m.ID, nil
}

// Sent is the messages so far, oldest first.
func (r *Recorder) Sent() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.sent...)
}

// To is the messages sent to one address, oldest first.
func (r *Recorder) To(to string) []Message {
	var out []Message
	for _, m := range r.Sent() {
		if m.To == to {
			out = append(out, m)
		}
	}
	return out
}

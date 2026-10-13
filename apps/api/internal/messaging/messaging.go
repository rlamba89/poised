// Package messaging sends patients their texts and emails (plan C2: invites now, reminders and
// sign-in links later). The real senders, AWS End User Messaging SMS and SES v2, come with S17.
// Until then:
//   - Capture (MESSAGING=capture) stores every message in sent_messages and sends nothing; the
//     captured-messages page and GET /api/qa/messages read them back;
//   - Recorder is the fake for tests: it records each message.
package messaging

import (
	"context"
	"sync"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

// SMS sends a text message to a mobile in E.164. The caller has checked the number may be
// texted (patient.SMSAllowed, PAT-11).
type SMS interface {
	SendSMS(ctx context.Context, to, body string) error
}

// Email sends a plain-text email.
type Email interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// Message is one text or email.
type Message struct {
	Channel string // "sms" or "email"
	To      string
	Subject string // "" for a text
	Body    string
}

// Capture stores messages in sent_messages instead of sending them (MESSAGING=capture: local
// and QA only).
type Capture struct{ DB db.Querier }

func (c Capture) SendSMS(ctx context.Context, to, body string) error {
	return c.DB.InsertSentMessage(ctx, db.InsertSentMessageParams{Channel: "sms", Recipient: to, Body: body})
}

func (c Capture) SendEmail(ctx context.Context, to, subject, body string) error {
	return c.DB.InsertSentMessage(ctx, db.InsertSentMessageParams{Channel: "email", Recipient: to, Subject: subject, Body: body})
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

func (r *Recorder) SendSMS(_ context.Context, to, body string) error {
	return r.record(Message{Channel: "sms", To: to, Body: body})
}

func (r *Recorder) SendEmail(_ context.Context, to, subject, body string) error {
	return r.record(Message{Channel: "email", To: to, Subject: subject, Body: body})
}

func (r *Recorder) record(m Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.sent = append(r.sent, m)
	return nil
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

// Package cognito is the staff user pool (plan C1). Staff accounts are created here when an
// admin invites someone (STF-01); Cognito emails them a temporary password, valid for 7 days.
//
// The real client (AdminCreateUser) comes with S12, on AWS. Until then:
//   - Local is used by the local servers: it only logs, and the invited person signs in with the
//     dev login, where they appear at once;
//   - Recorder is the fake for tests: it records each call.
package cognito

import (
	"context"
	"log"
	"sync"
)

// Client creates staff accounts and sends their invitations.
type Client interface {
	// CreateUser creates the account for email and sends the invitation (AdminCreateUser).
	CreateUser(ctx context.Context, email string) error
	// ResendInvite sends a fresh invitation to an account that hasn't signed in yet
	// (AdminCreateUser with MessageAction RESEND).
	ResendInvite(ctx context.Context, email string) error
}

// Local stands in for Cognito on a developer's machine: no account, no email.
type Local struct{}

func (Local) CreateUser(_ context.Context, email string) error {
	log.Printf("cognito (local): new staff account for %s; no email is sent, so sign in with the dev login", email)
	return nil
}

func (Local) ResendInvite(_ context.Context, email string) error {
	log.Printf("cognito (local): invitation sent again to %s; no email is sent", email)
	return nil
}

// Call is one request to the Recorder.
type Call struct {
	Op    string // "CreateUser" or "ResendInvite"
	Email string
}

// Recorder is the fake for tests: it records every call that succeeds.
type Recorder struct {
	mu    sync.Mutex
	calls []Call
	err   error
}

// FailWith makes every later call fail with err (nil: succeed again).
func (r *Recorder) FailWith(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

func (r *Recorder) CreateUser(_ context.Context, email string) error {
	return r.record("CreateUser", email)
}

func (r *Recorder) ResendInvite(_ context.Context, email string) error {
	return r.record("ResendInvite", email)
}

func (r *Recorder) record(op, email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.calls = append(r.calls, Call{Op: op, Email: email})
	return nil
}

// Calls are the calls so far, oldest first.
func (r *Recorder) Calls() []Call {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Call(nil), r.calls...)
}

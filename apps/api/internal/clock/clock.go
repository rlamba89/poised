// Package clock is the time, behind an interface so tests can move it (plan C1: invites expire
// after 7 days; S16's reminders will use it too).
package clock

import (
	"sync"
	"time"
)

// Clock tells the time.
type Clock interface {
	Now() time.Time
}

// Real is the system clock.
type Real struct{}

func (Real) Now() time.Time { return time.Now() }

// Fake is a clock a test sets and moves. Its zero value starts at the real time of its first use.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.now.IsZero() {
		f.now = time.Now()
	}
	return f.now
}

// Add moves the clock on by d.
func (f *Fake) Add(d time.Duration) {
	now := f.Now()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = now.Add(d)
}

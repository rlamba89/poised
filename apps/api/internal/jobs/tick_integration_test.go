//go:build integration

package jobs_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/invite"
	"github.com/rlamba89/poised/apps/api/internal/jobs"
)

// The local timer's run (Tick), which has no API route: it's what cmd/api calls every minute and
// S17's jobs Lambda will call on a schedule. Like cmd/platform-admin's test, this uses the apitest
// database harness directly, a stated exception to "integration tests are API → database only";
// everything else about the jobs is tested through POST /api/qa/jobs/run (internal/httpapi).

func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }

func TestTick(t *testing.T) {
	c, fx := apitest.New(t)
	h := fx.Hospital(t)
	hq := fx.PublishedHQ(t, h, fx.TrustUser(t, h.Org, "super_clinician"), "workflow.json")
	staff := c.LoginAs(t, fx.User(t, h, "clinician"))
	staff.Do("POST", h.Path("/episodes"), map[string]any{
		"versionId": hq.VersionID,
		"patient": map[string]string{"firstName": "Sam", "lastName": "Ticked", "dateOfBirth": "1980-05-17", "sex": "female",
			"mobile": "07700 900401"},
	}).Want(t, http.StatusCreated)
	sent := func() int { return len(c.Messages.To("+447700900401")) }
	if sent() != 1 {
		t.Fatalf("invite: %+v", c.Messages.Sent())
	}
	r := jobs.Runner{Pool: c.DB, DefaultZone: "Europe/London", Inviter: invite.Sender{Pool: c.DB, SMS: c.Messages, Email: c.Messages,
		AppURL: "http://localhost:3000", PhoneCountry: "GB"}}
	ctx := t.Context()

	t.Run("on real time nothing is due yet", func(t *testing.T) {
		if s, err := r.Tick(ctx, time.Now(), true); err != nil || s.Jobs != 1 || sent() != 1 {
			t.Errorf("summary %+v, err %v, sent %d", s, err, sent())
		}
	})
	t.Run("with the testers' clock 2 days ahead, the day 2 reminder goes, and the run is noted", func(t *testing.T) {
		c.Do("PUT", "/api/qa/jobs/clock", map[string]int{"offsetMinutes": 2*24*60 + 60}).Want(t, http.StatusNoContent)
		real := time.Now()
		s, err := r.Tick(ctx, real, true)
		if err != nil || s.Reminders != 1 || sent() != 2 {
			t.Errorf("summary %+v, err %v, sent %d", s, err, sent())
		}
		if want := real.Add(49 * time.Hour); !s.AsOf.Equal(want) {
			t.Errorf("as of %v, want %v", s.AsOf, want)
		}
		got := c.Do("GET", "/api/qa/jobs", nil)
		var clock struct{ LastTickAt, LastTickAsOf *time.Time }
		got.Decode(t, &clock)
		if clock.LastTickAt == nil || !clock.LastTickAt.Equal(real.Truncate(time.Microsecond)) ||
			!clock.LastTickAsOf.Equal(real.Add(49*time.Hour).Truncate(time.Microsecond)) {
			t.Errorf("clock %+v (real %v)", clock, real)
		}
	})
	t.Run("without the QA clock (not capture mode) the offset is ignored", func(t *testing.T) {
		c.Do("PUT", "/api/qa/jobs/clock", map[string]int{"offsetMinutes": 6 * 24 * 60}).Want(t, http.StatusNoContent)
		if s, err := r.Tick(ctx, time.Now(), false); err != nil || s.Jobs != 0 || sent() != 2 {
			t.Errorf("summary %+v, err %v, sent %d", s, err, sent())
		}
	})
}

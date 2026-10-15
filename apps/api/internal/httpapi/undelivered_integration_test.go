//go:build integration

package httpapi_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/delivery"
)

// Undelivered invites (S17; OPS-05). Every message about an episode is recorded with the
// sender's ID; a failed send, or AWS's later report that a message wasn't delivered
// (delivery.Record, which the jobs Lambda calls with SES's and End User Messaging SMS's events;
// like the jobs' Tick it has no route, so it's called directly here), shows on the episode
// (GET …/episodes/{eid} "undelivered") and in its General notes.

type undelivered struct {
	Kind, Channel, Failure string
	SentAt                 time.Time
	FailedAt               *time.Time
}

func readUndelivered(t *testing.T, c *apitest.Client, path string) []undelivered {
	t.Helper()
	var got struct{ Undelivered []undelivered }
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	r.Decode(t, &got)
	return got.Undelivered
}

// providerID is the ID the sender gave the episode's newest message on channel.
func providerID(t *testing.T, w *clinic, episode uuid.UUID, channel string) string {
	t.Helper()
	var id string
	if err := w.db.QueryRow(t.Context(), `SELECT provider_id FROM episode_messages WHERE episode_id = $1 AND channel = $2
		ORDER BY created_at DESC LIMIT 1`, episode, channel).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestUndeliveredMessages(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	id := newPatientEpisode(t, w, "Bounced", "07700 900321", "sam.bounced@example.org")
	path := at(w.a, "/episodes/"+id.String())
	ctx := t.Context()

	t.Run("each message is recorded with the sender's ID, and none is undelivered", func(t *testing.T) {
		sms, email := w.anon.Messages.To("+447700900321"), w.anon.Messages.To("sam.bounced@example.org")
		if providerID(t, w, id, "sms") != sms[0].ID || providerID(t, w, id, "email") != email[0].ID {
			t.Errorf("recorded IDs %s %s, sent %s %s", providerID(t, w, id, "sms"), providerID(t, w, id, "email"), sms[0].ID, email[0].ID)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_messages WHERE episode_id = $1 AND kind = 'invite' AND failed_at IS NULL`, id); n != 2 {
			t.Errorf("%d invite rows", n)
		}
		if got := readUndelivered(t, w.clinician, path); len(got) != 0 {
			t.Errorf("undelivered %+v", got)
		}
	})

	t.Run("AWS reports the text undelivered: the episode shows it, with a General notes entry, once", func(t *testing.T) {
		f := delivery.Failure{ProviderID: providerID(t, w, id, "sms"), Channel: "sms", Reason: "the mobile number isn't valid"}
		for range 2 { // SNS can deliver an event twice
			if err := delivery.Record(ctx, w.db, f, time.Now()); err != nil {
				t.Fatal(err)
			}
		}
		got := readUndelivered(t, w.clinician, path)
		if len(got) != 1 || got[0].Channel != "sms" || got[0].Kind != "invite" || got[0].Failure != "the mobile number isn't valid" || got[0].FailedAt == nil {
			t.Errorf("undelivered %+v", got)
		}
		want := "The invite's text message wasn't delivered: the mobile number isn't valid. Check the patient's mobile number, then resend the invite."
		if n := count(t, w.db, `SELECT count(*) FROM episode_events WHERE episode_id = $1 AND text = $2 AND automatic AND user_id IS NULL`, id, want); n != 1 {
			t.Errorf("%d notes saying %q", n, want)
		}
	})

	t.Run("not shown to another hospital or trust", func(t *testing.T) {
		refused(t, w.site2.Do("GET", at(w.a2, "/episodes/"+id.String()), nil), http.StatusNotFound)
		refused(t, w.otherClin.Do("GET", at(w.b, "/episodes/"+id.String()), nil), http.StatusNotFound)
	})

	t.Run("a message that isn't an episode's (a sign-in link) changes nothing", func(t *testing.T) {
		before := count(t, w.db, `SELECT count(*) FROM episode_events`)
		if err := delivery.Record(ctx, w.db, delivery.Failure{ProviderID: "not-ours", Channel: "email", Reason: "it bounced"}, time.Now()); err != nil {
			t.Fatal(err)
		}
		if after := count(t, w.db, `SELECT count(*) FROM episode_events`); after != before {
			t.Errorf("events %d → %d", before, after)
		}
	})

	t.Run("a Resend whose messages can't be sent shows those, not the earlier round's", func(t *testing.T) {
		w.anon.Clock.Add(time.Minute)
		w.anon.Messages.FailWith(errors.New("refused"))
		w.clinician.Do("POST", path+"/resend-invite", nil)
		w.anon.Messages.FailWith(nil)
		got := readUndelivered(t, w.clinician, path)
		if len(got) != 2 || got[0].Failure != "it couldn't be sent" || got[1].Failure != "it couldn't be sent" || got[0].Channel == got[1].Channel {
			t.Errorf("undelivered %+v", got)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_messages WHERE episode_id = $1 AND provider_id IS NULL AND failed_at IS NOT NULL`, id); n != 2 {
			t.Errorf("%d unsent rows", n)
		}
	})

	t.Run("a Resend that goes clears the list", func(t *testing.T) {
		w.anon.Clock.Add(time.Minute)
		w.clinician.Do("POST", path+"/resend-invite", nil).Want(t, http.StatusOK)
		if got := readUndelivered(t, w.clinician, path); len(got) != 0 {
			t.Errorf("undelivered %+v", got)
		}
	})

	t.Run("an undelivered reminder is shown as the reminder's", func(t *testing.T) {
		s := runJobs(t, w.anon, w.anon.Clock.Now().Add(49*time.Hour))
		if s.Reminders != 1 {
			t.Fatalf("summary %+v", s)
		}
		var rid string
		if err := w.db.QueryRow(ctx, `SELECT provider_id FROM episode_messages WHERE episode_id = $1 AND kind = 'reminder' AND channel = 'email'`, id).Scan(&rid); err != nil {
			t.Fatal(err)
		}
		if err := delivery.Record(ctx, w.db, delivery.Failure{ProviderID: rid, Channel: "email", Reason: "it bounced"}, time.Now()); err != nil {
			t.Fatal(err)
		}
		got := readUndelivered(t, w.clinician, path)
		if len(got) != 1 || got[0].Kind != "reminder" || got[0].Channel != "email" {
			t.Errorf("undelivered %+v", got)
		}
		want := "A reminder's email wasn't delivered: it bounced. Check the patient's email address, then resend the invite."
		if n := count(t, w.db, `SELECT count(*) FROM episode_events WHERE episode_id = $1 AND text = $2`, id, want); n != 1 {
			t.Errorf("%d notes saying %q", n, want)
		}
	})
}

// In capture mode (local and QA) a message's ID is its captured row's.
func TestCapturedMessagesAreRecordedWithTheirRow(t *testing.T) {
	t.Parallel()
	c, fx := apitest.NewWith(t, apitest.Options{Capture: true})
	h := fx.Hospital(t)
	hq := fx.PublishedHQ(t, h, fx.TrustUser(t, h.Org, "super_clinician"), "workflow.json")
	c.LoginAs(t, fx.User(t, h, "clinician")).Do("POST", h.Path("/episodes"), map[string]any{
		"versionId": hq.VersionID,
		"patient":   map[string]string{"firstName": "Sam", "lastName": "Captured", "dateOfBirth": "1980-05-17", "sex": "female", "mobile": "07700 900402"},
	}).Want(t, http.StatusCreated)
	var n int
	if err := c.DB.QueryRow(t.Context(), `SELECT count(*) FROM episode_messages m JOIN sent_messages s ON m.provider_id = 'capture-' || s.id
		WHERE s.recipient = '+447700900402' AND m.channel = 'sms'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("%d rows, %v", n, err)
	}
}

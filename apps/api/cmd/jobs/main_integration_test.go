//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

// The jobs Lambda's handler (S17): the EventBridge schedule's call runs the due jobs (as the
// local timer does), and the SNS topic's call records undelivered messages. Like the jobs' own
// Tick test, it uses the apitest database harness directly: the Lambda has no HTTP route.

func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }

func TestHandle(t *testing.T) {
	c, fx := apitest.New(t)
	h := fx.Hospital(t)
	hq := fx.PublishedHQ(t, h, fx.TrustUser(t, h.Org, "super_clinician"), "workflow.json")
	staff := c.LoginAs(t, fx.User(t, h, "clinician"))
	r := staff.Do("POST", h.Path("/episodes"), map[string]any{
		"versionId": hq.VersionID,
		"patient": map[string]string{"firstName": "Sam", "lastName": "Scheduled", "dateOfBirth": "1980-05-17", "sex": "female",
			"mobile": "07700 900411"},
	})
	r.Want(t, http.StatusCreated)
	var ep struct{ ID string }
	r.Decode(t, &ep)
	deps := httpapi.Deps{Pool: c.DB, Config: c.Config(), Cognito: cognito.Local{}, SMS: c.Messages, Email: c.Messages}
	handle := handler(deps)
	ctx := t.Context()

	t.Run("the schedule's call runs the jobs, with the testers' clock in capture mode", func(t *testing.T) {
		c.Do("PUT", "/api/qa/jobs/clock", map[string]int{"offsetMinutes": 2*24*60 + 60}).Want(t, http.StatusNoContent)
		out, err := handle(ctx, json.RawMessage(`{"run":"jobs"}`))
		if err != nil {
			t.Fatal(err)
		}
		if s, ok := out.(summary); !ok || s.Jobs.Reminders != 1 || len(c.Messages.To("+447700900411")) != 2 {
			t.Errorf("out %+v, sent %d", out, len(c.Messages.To("+447700900411")))
		}
	})

	t.Run("an SNS call records the undelivered message", func(t *testing.T) {
		id := c.Messages.To("+447700900411")[1].ID // the reminder
		event := map[string]any{"Records": []map[string]any{{
			"EventSource": "aws:sns",
			"Sns":         map[string]any{"Message": `{"eventType":"TEXT_UNREACHABLE","isFinal":true,"messageId":"` + id + `"}`},
		}}}
		raw, _ := json.Marshal(event)
		out, err := handle(ctx, raw)
		if err != nil {
			t.Fatal(err)
		}
		if s, ok := out.(summary); !ok || s.Deliveries != 1 {
			t.Errorf("out %+v", out)
		}
		var n int
		if err := c.DB.QueryRow(ctx, `SELECT count(*) FROM episode_messages WHERE provider_id = $1 AND failed_at IS NOT NULL
			AND failure = 'the phone couldn''t be reached (it may be switched off)'`, id).Scan(&n); err != nil || n != 1 {
			t.Errorf("%d rows, %v", n, err)
		}
	})

	t.Run("a run without the QA clock outside capture mode", func(t *testing.T) {
		send := httpapi.Deps{Pool: c.DB, Config: c.Config(), Cognito: cognito.Local{}, SMS: c.Messages, Email: c.Messages}
		send.Config.Messaging = "send"
		before := time.Now()
		out, err := handler(send)(ctx, json.RawMessage(`{}`))
		s, ok := out.(summary)
		if err != nil || !ok || s.Jobs.AsOf.Before(before) || s.Jobs.AsOf.After(time.Now()) {
			t.Errorf("out %+v, err %v: want a run at the real time", out, err)
		}
	})
}

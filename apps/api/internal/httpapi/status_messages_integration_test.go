//go:build integration

package httpapi_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/delivery"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// Status messages to the patient (S21, EPI-08): when a clinician moves an episode to Ready for
// admission, or cancels it, the patient is texted and emailed the hospital's wording, if the
// hospital has switched that message on (each is off by default). They go through the same
// messaging as the invites, are recorded in episode_messages (kind "status"), and one that can't
// be sent is shown as for an invite (S17). There's no new route: these are the status routes' (S20).

// statusMsgEpisode is an episode at hospital h made through the API (so the patient has a mobile
// and an email and got their invite), moved on to Ready for POA, with the invite's messages
// forgotten.
func statusMsgEpisode(t *testing.T, w *clinic, last, mobile, email string) apitest.Episode {
	t.Helper()
	id := newPatientEpisode(t, w, last, mobile, email)
	var e apitest.Episode
	e.ID = id
	if err := w.db.QueryRow(t.Context(), `SELECT id FROM episode_forms WHERE episode_id = $1`, id).Scan(&e.FormID); err != nil {
		t.Fatal(err)
	}
	setStatus(t, w, e, episode.ReadyForPOA)
	return e
}

func setHospitalSettings(t *testing.T, w *clinic, h apitest.Hospital, settings string) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET settings = $2 WHERE id = $1`, h.ID, settings); err != nil {
		t.Fatal(err)
	}
}

func move(t *testing.T, w *clinic, e apitest.Episode, to string) {
	t.Helper()
	w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": to, "revision": revisionOf(t, w, e.ID)}).
		Want(t, http.StatusNoContent)
}

func statusMessageRows(t *testing.T, w *clinic, e apitest.Episode) []string {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT status || ':' || channel || ':' || (failed_at IS NOT NULL)::text
		FROM episode_messages WHERE episode_id = $1 AND kind = 'status' ORDER BY created_at, channel`, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func TestStatusMessages(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	msgs := w.anon.Messages
	name := hospitalName(t, w, w.a)

	t.Run("off by default: moving to Ready for admission and cancelling send nothing", func(t *testing.T) {
		e := statusMsgEpisode(t, w, "Quiet", "07700 900401", "quiet@example.org")
		before := len(msgs.Sent())
		move(t, w, e, episode.ReadyForAdmission)
		w.clinician.Do("POST", epPath(w.a, e, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		if len(msgs.Sent()) != before || len(statusMessageRows(t, w, e)) != 0 {
			t.Errorf("sent %+v", msgs.Sent()[before:])
		}
		if n := lastNote(t, w, e.ID); strings.Contains(n, "message") {
			t.Errorf("note %q: nothing to say when the message is off", n)
		}
	})

	t.Run("switched on at the hospital: its own wording, by text and email, without a link", func(t *testing.T) {
		setHospitalSettings(t, w, w.a, `{"statusMessages.readyForAdmissionSms": true, "statusMessages.readyForAdmissionEmail": true,
			"templates.readyForAdmissionSms": "{{hospital}}: {{firstName}}, you are ready for your operation.",
			"templates.readyForAdmissionEmail": "Dear {{firstName}}, {{hospital}} is ready for you. Sign in: {{link}}"}`)
		e := statusMsgEpisode(t, w, "Ready", "07700 900402", "ready@example.org")
		move(t, w, e, episode.ReadyForAdmission)
		sms, email := msgs.To("+447700900402"), msgs.To("ready@example.org")
		if len(sms) != 2 || sms[1].Body != name+": Sam, you are ready for your operation." {
			t.Errorf("texts %+v", sms)
		}
		if len(email) != 2 || email[1].Body != "Dear Sam, "+name+" is ready for you. Sign in: http://localhost:3000/signin" ||
			!strings.Contains(email[1].Subject, name) {
			t.Errorf("emails %+v", email)
		}
		if got := statusMessageRows(t, w, e); strings.Join(got, ",") != "ready_for_admission:email:false,ready_for_admission:sms:false" {
			t.Errorf("rows %v", got)
		}
		if n := lastNote(t, w, e.ID); n != "Ready for admission message sent by text message and email." {
			t.Errorf("note %q", n)
		}
	})

	t.Run("only on a move by hand to it: undoing an admission sends nothing", func(t *testing.T) {
		e := statusMsgEpisode(t, w, "Undo", "07700 900403", "")
		move(t, w, e, episode.ReadyForAdmission)
		w.clinician.Do("PATCH", epPath(w.a, e, "/status"), map[string]any{"to": episode.Admitted, "date": yesterday(), "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		w.clinician.Do("POST", epPath(w.a, e, "/undo"), map[string]any{"reason": "Wrong patient", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		if got := statusMessageRows(t, w, e); len(got) != 1 || len(msgs.To("+447700900403")) != 2 { // the invite, then one status text
			t.Errorf("rows %v, texts %+v", got, msgs.To("+447700900403"))
		}
	})

	t.Run("each channel has its own switch, and the other hospital follows its own (off)", func(t *testing.T) {
		setHospitalSettings(t, w, w.a, `{"statusMessages.cancelledSms": true}`)
		e := statusMsgEpisode(t, w, "Cancel", "07700 900404", "cancel@example.org")
		w.clinician.Do("POST", epPath(w.a, e, "/cancel"), map[string]any{"reason": "other", "details": "Moved away", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		sms := msgs.To("+447700900404")
		if len(sms) != 2 || !strings.Contains(sms[1].Body, "has cancelled") || len(msgs.To("cancel@example.org")) != 1 {
			t.Errorf("texts %+v, emails %+v", sms, msgs.To("cancel@example.org"))
		}
		if got := statusMessageRows(t, w, e); strings.Join(got, ",") != "cancelled:sms:false" {
			t.Errorf("rows %v", got)
		}
		if n := lastNote(t, w, e.ID); n != "Cancelled message sent by text message." {
			t.Errorf("note %q", n)
		}

		// A2 sets nothing and the trust nothing: off.
		at2 := w.fx.EpisodeAt(t, w.hq, w.a2, w.cara)
		setStatus(t, w, at2, episode.ReadyForPOA)
		before := len(msgs.Sent())
		site2 := w.site2
		site2.Do("POST", epPath(w.a2, at2, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, at2.ID)}).
			Want(t, http.StatusNoContent)
		if len(msgs.Sent()) != before {
			t.Errorf("A2 sent %+v", msgs.Sent()[before:])
		}
	})

	t.Run("the cancelled email's subject is neutral too, from any status (S27 test report 1, F2)", func(t *testing.T) {
		setHospitalSettings(t, w, w.a, `{"statusMessages.cancelledEmail": true}`)
		defer setHospitalSettings(t, w, w.a, `{}`)
		e := statusMsgEpisode(t, w, "CancelSubject", "", "cancel.subject@example.org")
		move(t, w, e, episode.ReadyForAdmission)
		w.clinician.Do("POST", epPath(w.a, e, "/cancel"), map[string]any{"reason": "patient_withdrew", "revision": revisionOf(t, w, e.ID)}).
			Want(t, http.StatusNoContent)
		em := msgs.To("cancel.subject@example.org")
		if len(em) != 2 || !strings.HasSuffix(em[1].Subject, ": your pre-operative assessment was cancelled") || strings.Contains(em[1].Subject, "questionnaire") {
			t.Errorf("emails %+v", em)
		}
	})

	t.Run("a trust's switch applies at every hospital that sets none", func(t *testing.T) {
		setHospitalSettings(t, w, w.a, `{}`)
		if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET settings = '{"statusMessages.readyForAdmissionEmail": true}' WHERE id = $1`, w.a.Org); err != nil {
			t.Fatal(err)
		}
		e := statusMsgEpisode(t, w, "Trust", "", "trust@example.org")
		move(t, w, e, episode.ReadyForAdmission)
		if em := msgs.To("trust@example.org"); len(em) != 2 || !strings.Contains(em[1].Body, "ready for your operation") {
			t.Errorf("emails %+v", em)
		}
	})

	t.Run("a message that can't be sent is recorded and shown; the move itself stands", func(t *testing.T) {
		e := statusMsgEpisode(t, w, "Fails", "", "fails@example.org")
		msgs.FailWith(errors.New("SES is down"))
		defer msgs.FailWith(nil)
		move(t, w, e, episode.ReadyForAdmission)
		if s := readStatus(t, w, e.ID); s.Status != episode.ReadyForAdmission {
			t.Errorf("status %+v", s)
		}
		if got := statusMessageRows(t, w, e); strings.Join(got, ",") != "ready_for_admission:email:true" {
			t.Errorf("rows %v", got)
		}
		if n := lastNote(t, w, e.ID); n != "Ready for admission message not sent. The email couldn't be sent." {
			t.Errorf("note %q", n)
		}
		var got struct {
			Undelivered []struct{ Kind, Channel, Status, Failure string }
		}
		r := w.clinician.Do("GET", epPath(w.a, e, ""), nil)
		r.Want(t, http.StatusOK)
		r.Decode(t, &got)
		if len(got.Undelivered) != 1 || got.Undelivered[0].Kind != "status" || got.Undelivered[0].Status != episode.ReadyForAdmission ||
			got.Undelivered[0].Channel != "email" || got.Undelivered[0].Failure == "" {
			t.Errorf("undelivered %+v", got.Undelivered)
		}
	})

	t.Run("only the email switched on, for a patient with no email: noted, nothing sent", func(t *testing.T) {
		e := statusMsgEpisode(t, w, "NoEmail", "07700 900405", "") // the trust's email switch is still on
		before := len(msgs.Sent())
		move(t, w, e, episode.ReadyForAdmission)
		if len(msgs.Sent()) != before {
			t.Errorf("sent %+v", msgs.Sent()[before:])
		}
		if n := lastNote(t, w, e.ID); n != "Ready for admission message not sent. The patient has no email address." {
			t.Errorf("note %q", n)
		}
	})
	t.Run("AWS reports a status message undelivered: the episode shows it, with a General notes entry", func(t *testing.T) {
		e := statusMsgEpisode(t, w, "Bounce", "", "bounce@example.org")
		move(t, w, e, episode.ReadyForAdmission)
		var id string
		if err := w.db.QueryRow(t.Context(), `SELECT provider_id FROM episode_messages WHERE episode_id = $1 AND kind = 'status'`, e.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if err := delivery.Record(t.Context(), w.db, delivery.Failure{ProviderID: id, Channel: "email", Reason: "it bounced"}, time.Now()); err != nil {
			t.Fatal(err)
		}
		want := "The Ready for admission email wasn't delivered: it bounced. Check the patient's email address, and tell them another way."
		if n := lastNote(t, w, e.ID); n != want {
			t.Errorf("note %q, want %q", n, want)
		}
		if got := statusMessageRows(t, w, e); strings.Join(got, ",") != "ready_for_admission:email:true" {
			t.Errorf("rows %v", got)
		}
	})
	t.Run("a patient with no mobile or email: noted, nothing sent", func(t *testing.T) {
		setHospitalSettings(t, w, w.a, `{"statusMessages.readyForAdmissionSms": true, "statusMessages.readyForAdmissionEmail": true}`)
		e := w.fx.Episode(t, w.hq, w.cara) // a made-up patient with no contact
		setStatus(t, w, e, episode.ReadyForPOA)
		before := len(msgs.Sent())
		move(t, w, e, episode.ReadyForAdmission)
		if len(msgs.Sent()) != before {
			t.Errorf("sent %+v", msgs.Sent()[before:])
		}
		if n := lastNote(t, w, e.ID); n != "Ready for admission message not sent. The patient has no mobile or email." {
			t.Errorf("note %q", n)
		}
	})
}

//go:build integration

package httpapi_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Everyone in a hospital is at least a clinician and sees all its data (A-5), so the episode
// routes have no role below them. Their "role" case is instead a clinician at the trust's other
// hospital (A2), who gets 404 at A: a hospital-scoped user sees only their hospital. "Another
// trust" is Bob, a clinician in trust B: at A's URL (not a member: 404) and at B's URL with A's
// ids (scoped to B: 404).

// clinic adds the workflow's people and data to the usual cast.
type clinic struct {
	*cast
	a2        apitest.Hospital // trust A's second hospital
	cara, bob apitest.User
	clinician *apitest.Client // Cara, a clinician at A
	site2     *apitest.Client // a clinician at A2 only
	otherClin *apitest.Client // Bob, a clinician in B
	hq        apitest.HQ      // trust A's published Workflow HQ: 2 patient Question Sets, then 1 clinician set
	ep        apitest.Episode // on hq at A, waiting for the patient
}

func newClinic(t *testing.T) *clinic {
	t.Helper()
	w := &clinic{cast: newCast(t)}
	w.a2 = w.fx.HospitalIn(t, w.a.Org)
	w.cara = w.fx.User(t, w.a, "clinician")
	w.bob = w.fx.User(t, w.b, "clinician")
	w.clinician = w.anon.LoginAs(t, w.cara)
	w.site2 = w.anon.LoginAs(t, w.fx.User(t, w.a2, "clinician"))
	w.otherClin = w.anon.LoginAs(t, w.bob)
	w.hq = w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
	w.ep = w.fx.Episode(t, w.hq, w.cara)
	return w
}

// episodeRow is the part of an episodes row the tests check.
type episodeRow struct {
	Status, Procedure, Anaesthetic, Consultant string
	NurseAsa, AnaesthetistAsa                  *int
	Submitted, ReviewCompleted                 *time.Time
	ReviewCompletedBy                          *uuid.UUID
}

func readEpisode(t *testing.T, w *clinic, id uuid.UUID) episodeRow {
	t.Helper()
	var e episodeRow
	err := w.db.QueryRow(t.Context(), `SELECT status, procedure, anaesthetic, consultant, nurse_asa, anaesthetist_asa,
		patient_submitted_at, review_completed_at, review_completed_by FROM episodes WHERE id = $1`, id).Scan(
		&e.Status, &e.Procedure, &e.Anaesthetic, &e.Consultant, &e.NurseAsa, &e.AnaesthetistAsa,
		&e.Submitted, &e.ReviewCompleted, &e.ReviewCompletedBy)
	if err != nil {
		t.Fatalf("episode %s: %v", id, err)
	}
	return e
}

// eventRow is one General notes row.
type eventRow struct {
	Kind, Text string
	User       *uuid.UUID
	At         time.Time
}

func readEvents(t *testing.T, w *clinic, episode uuid.UUID) []eventRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT kind, text, user_id, created_at FROM episode_events
		WHERE episode_id = $1 ORDER BY created_at, id`, episode)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var events []eventRow
	for rows.Next() {
		var e eventRow
		if err := rows.Scan(&e.Kind, &e.Text, &e.User, &e.At); err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return events
}

// setStatus moves an episode on directly, as the patient's submit and the review would.
func setStatus(t *testing.T, w *clinic, e apitest.Episode, status string) {
	t.Helper()
	_, err := w.db.Exec(t.Context(), `UPDATE episodes SET status = $2,
		patient_submitted_at = CASE WHEN $2 = 'hq_not_complete' THEN NULL ELSE now() END,
		review_completed_at = CASE WHEN $2 IN ('hq_not_complete', 'ready_for_review') THEN NULL ELSE now() END
		WHERE id = $1`, e.ID, status)
	if err != nil {
		t.Fatal(err)
	}
}

// saveAnswers writes an answers row directly: the patient's (by is nil) or the clinician's.
func saveAnswers(t *testing.T, w *clinic, e apitest.Episode, chapter uuid.UUID, by *apitest.User, data string) {
	t.Helper()
	actor, user := "patient", (*uuid.UUID)(nil)
	if by != nil {
		actor, user = "clinician", &by.ID
	}
	_, err := w.db.Exec(t.Context(), `INSERT INTO episode_answers (episode_id, chapter_id, actor, data, updated_by)
		VALUES ($1, $2, $3, $4, $5)`, e.ID, chapter, actor, data, user)
	if err != nil {
		t.Fatal(err)
	}
}

// epPath is a path under /api/o/{trust}/h/{hospital}/episodes/{episode}.
func epPath(hospital apitest.Hospital, e apitest.Episode, rest string) string {
	return at(hospital, "/episodes/"+e.ID.String()+rest)
}

func TestListPublishedHQs(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	draft := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	theirs := w.fx.PublishedHQ(t, w.b, w.bea, "basic.json")
	versions := func(t *testing.T, r *apitest.Resp) []uuid.UUID {
		r.Want(t, http.StatusOK)
		var got []struct{ VersionID uuid.UUID }
		r.Decode(t, &got)
		var out []uuid.UUID
		for _, v := range got {
			out = append(out, v.VersionID)
		}
		return out
	}

	t.Run("happy path: published versions only, not drafts", func(t *testing.T) {
		got := versions(t, w.clinician.Do("GET", at(w.a, "/published-hqs"), nil))
		if len(got) != 1 || got[0] != w.hq.VersionID {
			t.Errorf("versions = %v, want only %s (not the draft %s)", got, w.hq.VersionID, draft.VersionID)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", at(w.a, "/published-hqs"), nil), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, w.site2.Do("GET", at(w.a, "/published-hqs"), nil), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.otherClin.Do("GET", at(w.a, "/published-hqs"), nil), http.StatusNotFound)
		if got := versions(t, w.otherClin.Do("GET", at(w.b, "/published-hqs"), nil)); len(got) != 1 || got[0] != theirs.VersionID {
			t.Errorf("B's versions = %v, want only %s", got, theirs.VersionID)
		}
	})
	t.Run("invalid input: not a hospital id", func(t *testing.T) {
		for _, p := range badIDs(w.a, "/published-hqs") {
			refused(t, w.clinician.Do("GET", p, nil), http.StatusNotFound)
		}
	})
}

func TestListEpisodes(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	submitted := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, submitted, "ready_for_review")
	theirs := w.fx.Episode(t, w.fx.PublishedHQ(t, w.b, w.bea, "basic.json"), w.bob)
	list := func(t *testing.T, c *apitest.Client, path string) []uuid.UUID {
		r := c.Do("GET", path, nil)
		r.Want(t, http.StatusOK)
		var got []struct{ ID uuid.UUID }
		r.Decode(t, &got)
		var out []uuid.UUID
		for _, e := range got {
			out = append(out, e.ID)
		}
		return out
	}

	t.Run("happy path: the hospital's episodes, newest first", func(t *testing.T) {
		if got := list(t, w.clinician, at(w.a, "/episodes")); len(got) != 2 || got[0] != submitted.ID || got[1] != w.ep.ID {
			t.Errorf("episodes = %v, want [%s %s]", got, submitted.ID, w.ep.ID)
		}
	})
	t.Run("happy path: filtered by status", func(t *testing.T) {
		if got := list(t, w.clinician, at(w.a, "/episodes?status=ready_for_review")); len(got) != 1 || got[0] != submitted.ID {
			t.Errorf("episodes = %v, want only %s", got, submitted.ID)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", at(w.a, "/episodes"), nil), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, w.site2.Do("GET", at(w.a, "/episodes"), nil), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.otherClin.Do("GET", at(w.a, "/episodes"), nil), http.StatusNotFound)
		if got := list(t, w.otherClin, at(w.b, "/episodes")); len(got) != 1 || got[0] != theirs.ID {
			t.Errorf("B's episodes = %v, want only %s", got, theirs.ID)
		}
	})
	t.Run("invalid input: an unknown status matches nothing", func(t *testing.T) {
		if got := list(t, w.clinician, at(w.a, "/episodes?status=nonsense")); len(got) != 0 {
			t.Errorf("episodes = %v, want none", got)
		}
	})
}

func TestCreateEpisode(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	create := func(c *apitest.Client, h apitest.Hospital, body any) *apitest.Resp {
		return c.Do("POST", at(h, "/episodes"), body)
	}
	body := func(patient, version uuid.UUID) map[string]any {
		return map[string]any{"patientId": patient, "versionId": version,
			"procedure": " Hip replacement ", "anaesthetic": " General ", "consultant": " Dr Grey "}
	}
	good := body(w.ep.PatientID, w.hq.VersionID)
	episodes := func(t *testing.T) int { return count(t, w.db, "SELECT count(*) FROM episodes") }
	events := func(t *testing.T) int { return count(t, w.db, "SELECT count(*) FROM episode_events") }
	beforeEpisodes, beforeEvents := episodes(t), events(t)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, create(w.anon, w.a, good), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, create(w.site2, w.a, good), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		theirs := w.fx.Episode(t, w.fx.PublishedHQ(t, w.b, w.bea, "basic.json"), w.bob)
		beforeEpisodes++ // that fixture
		refused(t, create(w.otherClin, w.a, good), http.StatusNotFound)
		// Bob in B can't give B's patient A's HQ, or A's patient B's.
		refused(t, create(w.otherClin, w.b, body(theirs.PatientID, w.hq.VersionID)), http.StatusBadRequest)
		refused(t, create(w.otherClin, w.b, body(w.ep.PatientID, w.hq.VersionID)), http.StatusBadRequest)
	})
	t.Run("needs a published version", func(t *testing.T) {
		draft := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		retired := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		if _, err := w.db.Exec(t.Context(), "UPDATE questionnaire_versions SET status = 'retired' WHERE id = $1", retired.VersionID); err != nil {
			t.Fatal(err)
		}
		refused(t, create(w.clinician, w.a, body(w.ep.PatientID, draft.VersionID)), http.StatusBadRequest)
		refused(t, create(w.clinician, w.a, body(w.ep.PatientID, retired.VersionID)), http.StatusBadRequest)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, create(w.clinician, w.a, body(uuid.New(), w.hq.VersionID)), http.StatusBadRequest)
		refused(t, create(w.clinician, w.a, map[string]any{"versionId": w.hq.VersionID}), http.StatusBadRequest)
		refused(t, create(w.clinician, w.a, `{"patientId": "abc"}`), http.StatusBadRequest)
		refused(t, create(w.clinician, w.a, `{"patientId":`), http.StatusBadRequest)
	})
	if n, m := episodes(t), events(t); n != beforeEpisodes || m != beforeEvents {
		t.Fatalf("refusals wrote %d episodes and %d events, want %d and %d", n, m, beforeEpisodes, beforeEvents)
	}
	t.Run("happy path: waiting for the patient, with a link and a created event", func(t *testing.T) {
		r := create(w.clinician, w.a, good)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		e := readEpisode(t, w, got.ID)
		if e.Status != "hq_not_complete" || e.Procedure != "Hip replacement" || e.Anaesthetic != "General" || e.Consultant != "Dr Grey" {
			t.Errorf("episode = %+v", e)
		}
		var token string
		if err := w.db.QueryRow(t.Context(), "SELECT patient_token FROM episodes WHERE id = $1", got.ID).Scan(&token); err != nil {
			t.Fatal(err)
		}
		if len(token) != 43 {
			t.Errorf("token %q: want 43 characters (32 random bytes)", token)
		}
		ev := readEvents(t, w, got.ID)
		if len(ev) != 1 || ev[0].Kind != "event" || ev[0].Text != "Episode created" || ev[0].User == nil || *ev[0].User != w.cara.ID {
			t.Errorf("events = %+v, want one 'Episode created' by Cara", ev)
		}
	})
}

func TestGetEpisode(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	if _, err := w.db.Exec(t.Context(), `INSERT INTO episode_events (episode_id, kind, text, user_id)
		VALUES ($1, 'comment', 'Called the patient', $2)`, w.ep.ID, w.cara.ID); err != nil {
		t.Fatal(err)
	}
	get := func(c *apitest.Client, h apitest.Hospital) *apitest.Resp {
		return c.Do("GET", epPath(h, w.ep, ""), nil)
	}

	t.Run("happy path: the episode, its patient and its General notes", func(t *testing.T) {
		r := get(w.clinician, w.a)
		r.Want(t, http.StatusOK)
		var got struct {
			Episode struct {
				ID, PatientID     uuid.UUID
				Status, LastName  string
				QuestionnaireName string
			}
			Events []struct{ Kind, Text, UserName string }
		}
		r.Decode(t, &got)
		if got.Episode.ID != w.ep.ID || got.Episode.PatientID != w.ep.PatientID || got.Episode.Status != "hq_not_complete" ||
			got.Episode.QuestionnaireName != w.hq.Name || !strings.HasPrefix(got.Episode.LastName, "Patient ") {
			t.Errorf("episode = %+v", got.Episode)
		}
		if len(got.Events) != 1 || got.Events[0].Text != "Called the patient" || got.Events[0].UserName != w.cara.Name {
			t.Errorf("events = %+v, want Cara's comment", got.Events)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, get(w.anon, w.a), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, get(w.site2, w.a), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, get(w.otherClin, w.a), http.StatusNotFound)
		refused(t, get(w.otherClin, w.b), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/abc"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/"+uuid.NewString()), nil), http.StatusNotFound)
	})
}

func TestUpdateEpisode(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	patch := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body any) *apitest.Resp {
		return c.Do("PATCH", epPath(h, e, ""), body)
	}
	reviewed := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, reviewed, "ready_for_poa")
	change := map[string]any{"status": "poa_complete"}
	before := readEpisode(t, w, reviewed.ID)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, patch(w.anon, w.a, reviewed, change), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, patch(w.site2, w.a, reviewed, change), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, patch(w.otherClin, w.a, reviewed, change), http.StatusNotFound)
		refused(t, patch(w.otherClin, w.b, reviewed, change), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{
			// A null text field crashed the handler (S01 code review): now a 400.
			`{"status": null}`,
			`{"procedure": null}`,
			`{"anaesthetic": null}`,
			`{"consultant": null}`,
			map[string]int{"nurseAsa": 7},
			map[string]int{"anaesthetistAsa": -1},
			`{"status":`,
		} {
			refused(t, patch(w.clinician, w.a, reviewed, bad), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("PATCH", at(w.a, "/episodes/abc"), change), http.StatusNotFound)
	})
	t.Run("a status the workflow sets is refused", func(t *testing.T) {
		for _, status := range []string{"ready_for_review", "hq_not_complete", "nonsense"} {
			refused(t, patch(w.clinician, w.a, reviewed, map[string]string{"status": status}), http.StatusConflict)
		}
		// Before the HQ review is complete, no status can be picked by hand.
		refused(t, patch(w.clinician, w.a, w.ep, change), http.StatusConflict)
	})
	if after := readEpisode(t, w, reviewed.ID); !reflect.DeepEqual(after, before) {
		t.Fatalf("refusals changed the episode: %+v, was %+v", after, before)
	}
	if ev := readEvents(t, w, reviewed.ID); len(ev) != 0 {
		t.Fatalf("refusals wrote events: %+v", ev)
	}

	t.Run("happy path: status, details and ASA grades, each change with an event", func(t *testing.T) {
		patch(w.clinician, w.a, reviewed, map[string]any{"status": "poa_complete", "procedure": " Knee scope ",
			"consultant": " Dr Who ", "nurseAsa": 2, "anaesthetistAsa": 3}).Want(t, http.StatusNoContent)
		e := readEpisode(t, w, reviewed.ID)
		if e.Status != "poa_complete" || e.Procedure != "Knee scope" || e.Consultant != "Dr Who" || e.Anaesthetic != before.Anaesthetic ||
			e.NurseAsa == nil || *e.NurseAsa != 2 || e.AnaesthetistAsa == nil || *e.AnaesthetistAsa != 3 {
			t.Errorf("episode = %+v", e)
		}
		want := []string{"Status changed to POA complete", "Nurse ASA grade set to 2", "Anaesthetist ASA grade set to 3"}
		ev := readEvents(t, w, reviewed.ID)
		if len(ev) != len(want) {
			t.Fatalf("events = %+v, want %q", ev, want)
		}
		for i, text := range want {
			if ev[i].Kind != "event" || ev[i].Text != text || ev[i].User == nil || *ev[i].User != w.cara.ID {
				t.Errorf("event %d = %+v, want %q by Cara", i, ev[i], text)
			}
		}
	})
	t.Run("happy path: details only, before the review", func(t *testing.T) {
		patch(w.clinician, w.a, w.ep, map[string]string{"procedure": "Hip"}).Want(t, http.StatusNoContent)
		if e := readEpisode(t, w, w.ep.ID); e.Status != "hq_not_complete" || e.Procedure != "Hip" {
			t.Errorf("episode = %+v, want the procedure changed and the status kept", e)
		}
		if ev := readEvents(t, w, w.ep.ID); len(ev) != 0 {
			t.Errorf("events = %+v, want none", ev)
		}
	})
	t.Run("happy path: no change, no event; 0 clears a grade", func(t *testing.T) {
		patch(w.clinician, w.a, reviewed, map[string]any{"status": "poa_complete", "nurseAsa": 0, "anaesthetistAsa": 3}).Want(t, http.StatusNoContent)
		e := readEpisode(t, w, reviewed.ID)
		if e.NurseAsa != nil || e.AnaesthetistAsa == nil || *e.AnaesthetistAsa != 3 {
			t.Errorf("episode = %+v, want the nurse grade cleared and the anaesthetist's kept", e)
		}
		ev := readEvents(t, w, reviewed.ID)
		if len(ev) != 4 || ev[3].Text != "Nurse ASA grade cleared" {
			t.Errorf("events = %+v, want one more: 'Nurse ASA grade cleared'", ev)
		}
	})
}

// A PATCH that changes only details sends no status, so the server keeps the one it loaded. If
// the workflow moves the status on between that load and the write, the PATCH must not put the
// old status back (S02's verification, item 2): it gets a 409 and writes nothing.
func TestUpdateEpisodeKeepsAStatusThatJustChanged(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	for _, race := range []struct{ name, from, sql, want string }{
		{"the review is completed", "ready_for_review",
			`UPDATE episodes SET status = 'ready_for_poa', review_completed_at = now() WHERE id = $1`, "ready_for_poa"},
		{"the patient submits", "hq_not_complete",
			`UPDATE episodes SET status = 'ready_for_review', patient_submitted_at = now() WHERE id = $1`, "ready_for_review"},
	} {
		t.Run(race.name, func(t *testing.T) {
			e := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, e, race.from)
			r := meanwhile(t, w.db, race.sql, []any{e.ID}, func() *apitest.Resp {
				return w.clinician.Do("PATCH", epPath(w.a, e, ""), map[string]any{"procedure": "Knee scope", "nurseAsa": 2})
			})
			refused(t, r, http.StatusConflict)
			if !strings.Contains(r.Message(), "changed since you opened it") {
				t.Errorf("message = %q", r.Message())
			}
			if got := readEpisode(t, w, e.ID); got.Status != race.want || got.Procedure != "" || got.NurseAsa != nil {
				t.Errorf("episode = %+v, want status %s and nothing else changed", got, race.want)
			}
			if ev := readEvents(t, w, e.ID); len(ev) != 0 {
				t.Errorf("events = %+v, want none", ev)
			}
		})
	}
}

func TestAddEpisodeNote(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	add := func(c *apitest.Client, h apitest.Hospital, body any) *apitest.Resp {
		return c.Do("POST", epPath(h, w.ep, "/notes"), body)
	}
	note := map[string]string{"text": " Called the patient "}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, add(w.anon, w.a, note), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, add(w.site2, w.a, note), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, add(w.otherClin, w.a, note), http.StatusNotFound)
		refused(t, add(w.otherClin, w.b, note), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, add(w.clinician, w.a, map[string]string{"text": "   "}), http.StatusBadRequest)
		refused(t, add(w.clinician, w.a, `{"text":`), http.StatusBadRequest)
		refused(t, add(w.clinician, w.a, map[string]string{"text": strings.Repeat("x", 20<<10)}), http.StatusBadRequest)
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/abc/notes"), note), http.StatusNotFound)
	})
	if ev := readEvents(t, w, w.ep.ID); len(ev) != 0 {
		t.Fatalf("refusals wrote notes: %+v", ev)
	}

	t.Run("happy path: appended, with the user and the time", func(t *testing.T) {
		start := time.Now().Add(-time.Minute)
		add(w.clinician, w.a, note).Want(t, http.StatusCreated)
		add(w.clinician, w.a, map[string]string{"text": "Second note"}).Want(t, http.StatusCreated)
		ev := readEvents(t, w, w.ep.ID)
		if len(ev) != 2 || ev[0].Text != "Called the patient" || ev[1].Text != "Second note" {
			t.Fatalf("notes = %+v, want both, in order, trimmed", ev)
		}
		for _, e := range ev {
			if e.Kind != "comment" || e.User == nil || *e.User != w.cara.ID || e.At.Before(start) {
				t.Errorf("note = %+v, want a comment by Cara with the time", e)
			}
		}
	})
	t.Run("append-only: notes can't be changed or deleted", func(t *testing.T) {
		for _, method := range []string{"PUT", "PATCH", "DELETE"} {
			w.clinician.Do(method, epPath(w.a, w.ep, "/notes"), note).Want(t, http.StatusMethodNotAllowed)
		}
		if ev := readEvents(t, w, w.ep.ID); len(ev) != 2 {
			t.Errorf("notes = %+v, want the 2 from the happy path", ev)
		}
	})
}

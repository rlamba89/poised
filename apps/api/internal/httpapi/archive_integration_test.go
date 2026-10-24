//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// Archive and un-archive (S22, EPI-04): POST …/archive and …/unarchive, a super clinician's for
// both (S22 answers 1), un-archive with a reason. Each gets the five cases; on top, every write
// to an archived episode → 409 (answers 2), and un-archive restores the exact previous status.

// archivedFrom is the episode's archived_from ("" when it isn't archived).
func archivedFrom(t *testing.T, w *clinic, id uuid.UUID) string {
	t.Helper()
	var from *string
	if err := w.db.QueryRow(t.Context(), `SELECT archived_from FROM episodes WHERE id = $1`, id).Scan(&from); err != nil {
		t.Fatal(err)
	}
	if from == nil {
		return ""
	}
	return *from
}

// withRevision is a status route's body with the episode's current revision.
func withRevision(t *testing.T, w *clinic, id uuid.UUID, body map[string]any) map[string]any {
	t.Helper()
	out := map[string]any{"revision": revisionOf(t, w, id)}
	for k, v := range body {
		out[k] = v
	}
	return out
}

// archive archives e as Alex (a whole-trust super clinician), failing the test if it's refused.
func archive(t *testing.T, w *clinic, e apitest.Episode) {
	t.Helper()
	w.author.Do("POST", epPath(w.a, e, "/archive"), withRevision(t, w, e.ID, nil)).Want(t, http.StatusNoContent)
}

func auditCount(t *testing.T, w *clinic, action string, episodeID uuid.UUID) int {
	t.Helper()
	return count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = $1 AND $2 = ANY(episode_ids) AND status = 204`, action, episodeID)
}

func TestArchive(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	sueUser := w.fx.User(t, w.a, "super_clinician") // a super clinician of hospital A only
	sue := w.anon.LoginAs(t, sueUser)
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.OnHold)
	post := func(c *apitest.Client, h apitest.Hospital, body map[string]any) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/archive"), body)
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, post(w.anon, w.a, withRevision(t, w, e.ID, nil)), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician can't archive (S22 answers 1)", func(t *testing.T) {
		r := post(w.clinician, w.a, withRevision(t, w, e.ID, nil))
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(r.Message(), "super clinician") {
			t.Errorf("message %q", r.Message())
		}
		refused(t, post(w.site2, w.a, withRevision(t, w, e.ID, nil)), http.StatusNotFound) // the trust's other hospital
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, post(w.other, w.a, withRevision(t, w, e.ID, nil)), http.StatusNotFound) // Bea, a super clinician of trust B
		refused(t, post(w.other, w.b, withRevision(t, w, e.ID, nil)), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{
			map[string]any{},               // no revision
			map[string]any{"revision": -1}, // not a revision
			withRevision(t, w, e.ID, map[string]any{"reason": strings.Repeat("x", 1001)}),
			`{"revision":`,
		} {
			refused(t, sue.Do("POST", epPath(w.a, e, "/archive"), bad), http.StatusBadRequest)
		}
		refused(t, sue.Do("POST", at(w.a, "/episodes/abc/archive"), withRevision(t, w, e.ID, nil)), http.StatusNotFound)
		refused(t, sue.Do("POST", at(w.a, "/episodes/"+uuid.NewString()+"/archive"), withRevision(t, w, e.ID, nil)), http.StatusNotFound)
	})
	t.Run("a stale revision: 409 stale", func(t *testing.T) {
		r := post(sue, w.a, map[string]any{"revision": revisionOf(t, w, e.ID) + 1})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(string(r.Body), `"code":"stale"`) {
			t.Errorf("response %s", r)
		}
	})
	if s := readStatus(t, w, e.ID); s.Status != episode.OnHold || s.Revision != 0 || len(readHistory(t, w, e.ID)) != 0 {
		t.Fatalf("refusals changed the episode: %+v", s)
	}

	t.Run("a hospital's super clinician archives, with a reason; the follow-up stops", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `INSERT INTO jobs (kind, episode_id, run_at) VALUES ('followup', $1, now() + interval '1 day')`, e.ID); err != nil {
			t.Fatal(err)
		}
		post(sue, w.a, withRevision(t, w, e.ID, map[string]any{"reason": " Made-up: a duplicate "})).Want(t, http.StatusNoContent)
		if s := readStatus(t, w, e.ID); s.Status != episode.Archived || s.Revision != 1 || archivedFrom(t, w, e.ID) != episode.OnHold {
			t.Errorf("episode %+v, from %q", s, archivedFrom(t, w, e.ID))
		}
		h := readHistory(t, w, e.ID)
		if last := h[len(h)-1]; last.Move != "archive" || last.From != episode.OnHold || last.To != episode.Archived ||
			last.Reason != "Made-up: a duplicate" || last.User == nil || *last.User != sueUser.ID {
			t.Errorf("history %+v", last)
		}
		if got := lastNote(t, w, e.ID); got != "Episode archived: Made-up: a duplicate" {
			t.Errorf("General notes %q", got)
		}
		if n := count(t, w.db, `SELECT count(*) FROM jobs WHERE episode_id = $1 AND done_at IS NULL`, e.ID); n != 0 {
			t.Errorf("%d follow-up jobs still open", n)
		}
		if n := auditCount(t, w, "episode.archive", e.ID); n != 1 {
			t.Errorf("%d audit rows", n)
		}
	})
	t.Run("archiving an archived episode: 409", func(t *testing.T) {
		r := post(sue, w.a, withRevision(t, w, e.ID, nil))
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "archived") {
			t.Errorf("message %q", r.Message())
		}
	})
	t.Run("a whole-trust super clinician archives from any status, without a reason", func(t *testing.T) {
		for _, from := range episode.Statuses {
			if from == episode.Archived {
				continue
			}
			f := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, f, from)
			archive(t, w, f)
			if got := readStatus(t, w, f.ID).Status; got != episode.Archived || archivedFrom(t, w, f.ID) != from {
				t.Errorf("from %s: %s, archived from %q", from, got, archivedFrom(t, w, f.ID))
			}
			if got := lastNote(t, w, f.ID); got != "Episode archived" {
				t.Errorf("from %s: General notes %q", from, got)
			}
		}
	})
}

func TestUnarchive(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	sue := w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician"))
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.NotReady)
	archive(t, w, e)
	post := func(c *apitest.Client, h apitest.Hospital, body map[string]any) *apitest.Resp {
		return c.Do("POST", epPath(h, e, "/unarchive"), body)
	}
	why := func() map[string]any {
		return withRevision(t, w, e.ID, map[string]any{"reason": "Made-up: archived by mistake"})
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, post(w.anon, w.a, why()), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician can't un-archive", func(t *testing.T) {
		r := post(w.clinician, w.a, why())
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(r.Message(), "super clinician") {
			t.Errorf("message %q", r.Message())
		}
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, post(w.other, w.a, why()), http.StatusNotFound)
		refused(t, post(w.other, w.b, why()), http.StatusNotFound)
	})
	t.Run("invalid input: a reason is required", func(t *testing.T) {
		for _, bad := range []any{
			withRevision(t, w, e.ID, nil),
			withRevision(t, w, e.ID, map[string]any{"reason": "   "}),
			withRevision(t, w, e.ID, map[string]any{"reason": strings.Repeat("x", 1001)}),
			map[string]any{"reason": "x"}, // no revision
			`{"reason":`,
		} {
			refused(t, sue.Do("POST", epPath(w.a, e, "/unarchive"), bad), http.StatusBadRequest)
		}
		refused(t, sue.Do("POST", at(w.a, "/episodes/abc/unarchive"), why()), http.StatusNotFound)
	})
	t.Run("a stale revision: 409 stale", func(t *testing.T) {
		r := post(sue, w.a, map[string]any{"reason": "x", "revision": revisionOf(t, w, e.ID) - 1})
		refused(t, r, http.StatusConflict)
	})
	if s := readStatus(t, w, e.ID); s.Status != episode.Archived {
		t.Fatalf("refusals changed the episode: %+v", s)
	}
	t.Run("the clinician's page offers nothing; the super clinician's offers the un-archive", func(t *testing.T) {
		type view struct {
			Moves     []struct{ To, Move string }
			Unarchive *struct{ To, Label string }
		}
		read := func(c *apitest.Client) view {
			r := c.Do("GET", epPath(w.a, e, ""), nil)
			r.Want(t, http.StatusOK)
			var v view
			r.Decode(t, &v)
			return v
		}
		if v := read(w.clinician); len(v.Moves) != 0 || v.Unarchive != nil {
			t.Errorf("clinician: %+v", v)
		}
		if v := read(sue); len(v.Moves) != 0 || v.Unarchive == nil || v.Unarchive.To != episode.NotReady || v.Unarchive.Label != "Not ready for admission" {
			t.Errorf("super clinician: %+v", v)
		}
	})
	t.Run("a super clinician un-archives: back to the exact status, with the reason", func(t *testing.T) {
		post(sue, w.a, why()).Want(t, http.StatusNoContent)
		if s := readStatus(t, w, e.ID); s.Status != episode.NotReady || archivedFrom(t, w, e.ID) != "" {
			t.Errorf("episode %+v", s)
		}
		h := readHistory(t, w, e.ID)
		if last := h[len(h)-1]; last.Move != "unarchive" || last.From != episode.Archived || last.To != episode.NotReady || last.Reason != "Made-up: archived by mistake" {
			t.Errorf("history %+v", last)
		}
		if got := lastNote(t, w, e.ID); got != "Episode un-archived, back to Not ready for admission: Made-up: archived by mistake" {
			t.Errorf("General notes %q", got)
		}
		if n := auditCount(t, w, "episode.unarchive", e.ID); n != 1 {
			t.Errorf("%d audit rows", n)
		}
	})
	t.Run("not archived: 409", func(t *testing.T) {
		r := post(sue, w.a, why())
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "isn't archived") {
			t.Errorf("message %q", r.Message())
		}
	})
	t.Run("un-archive restores the exact previous status, with what went with it", func(t *testing.T) {
		for _, from := range episode.Statuses {
			if from == episode.Archived {
				continue
			}
			f := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, f, from)
			if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET admitted_at = '2026-01-10', discharged_at = '2026-01-12',
				cancel_reason = CASE WHEN status = 'cancelled' THEN 'patient_withdrew' END WHERE id = $1`, f.ID); err != nil {
				t.Fatal(err)
			}
			before := readStatus(t, w, f.ID)
			archive(t, w, f)
			w.author.Do("POST", epPath(w.a, f, "/unarchive"), withRevision(t, w, f.ID, map[string]any{"reason": "x"})).Want(t, http.StatusNoContent)
			after := readStatus(t, w, f.ID)
			before.Revision += 2 // archive and un-archive
			if after != before {
				t.Errorf("from %s: %+v, want %+v", from, after, before)
			}
		}
	})
}

// Every write to an archived episode is refused with 409, except un-archive (S22 answers 2); it
// can still be read, found by the search and on the patient's page; the patient's form is closed
// to changes (a sent one can still be viewed, S27).
func TestArchivedEpisodeIsReadOnly(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.ReadyForReview) // where answers and completing the review would otherwise work
	task := addTask(t, w, e, "Book anaesthetic review", "anaesthetist", false)
	archive(t, w, e)
	set := w.hq.Chapters[0].String()
	rev := revisionOf(t, w, e.ID)
	notes, history := len(readEvents(t, w, e.ID)), len(readHistory(t, w, e.ID))

	for _, write := range []struct {
		method, path string
		body         any
	}{
		{"PATCH", "", map[string]any{"revision": rev, "anaesthetic": "GA", "consultant": "Mr Made-up"}},
		{"POST", "/notes", map[string]any{"text": "A comment"}},
		{"PATCH", "/status", map[string]any{"to": episode.OnHold, "revision": rev}},
		{"POST", "/cancel", map[string]any{"reason": "patient_withdrew", "revision": rev}},
		{"POST", "/reopen", map[string]any{"reason": "x", "revision": rev}},
		{"POST", "/undo", map[string]any{"reason": "x", "revision": rev}},
		{"POST", "/archive", map[string]any{"revision": rev}},
		{"POST", "/tasks", map[string]any{"title": "Call the patient", "team": "nurse", "revision": rev}},
		{"PATCH", "/tasks/" + task.String(), map[string]any{"done": true, "revision": rev}},
		{"PUT", "/answers/" + set, map[string]any{"data": map[string]any{"q": "x"}, "validated": true, "revision": 0}},
		{"POST", "/complete-review", nil},
		{"POST", "/resend-invite", nil},
		{"PUT", "/patient-form/answers/" + set, map[string]any{"data": map[string]any{"q": "x"}, "revision": 0}},
		{"POST", "/patient-form/submit", nil},
	} {
		for _, c := range []*apitest.Client{w.clinician, w.author} { // a clinician, and a super clinician
			r := c.Do(write.method, epPath(w.a, e, write.path), write.body)
			refused(t, r, http.StatusConflict)
			if !strings.Contains(r.Message(), "archived") {
				t.Errorf("%s %s: %q", write.method, write.path, r.Message())
			}
		}
	}
	if s := readStatus(t, w, e.ID); s.Status != episode.Archived || s.Revision != rev {
		t.Errorf("the writes changed the episode: %+v", s)
	}
	if n, h := len(readEvents(t, w, e.ID)), len(readHistory(t, w, e.ID)); n != notes || h != history {
		t.Errorf("General notes %d (was %d), history %d (was %d)", n, notes, h, history)
	}
	if n := count(t, w.db, `SELECT count(*) FROM episode_answers`) + count(t, w.db, `SELECT count(*) FROM tasks WHERE done_at IS NOT NULL`); n != 0 {
		t.Errorf("%d answers or completed tasks written", n)
	}

	t.Run("the patient's form is closed to changes", func(t *testing.T) {
		p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
		// It was sent (Ready for review) before it was archived, so it can still be viewed (S27).
		p.Do("GET", patientPath(e, ""), nil).Want(t, http.StatusOK)
		for _, r := range []*apitest.Resp{
			p.Do("PUT", patientPath(e, "/answers/"+set), map[string]any{"data": map[string]any{}, "revision": 0}),
			p.Do("POST", patientPath(e, "/submit"), nil),
		} {
			refused(t, r, http.StatusForbidden)
			if !strings.Contains(string(r.Body), `"code":"archived"`) || !strings.Contains(r.Message(), "closed") {
				t.Errorf("patient: %s", r)
			}
		}
	})
	t.Run("still read: the episode, the patient's page, the search", func(t *testing.T) {
		w.clinician.Do("GET", epPath(w.a, e, ""), nil).Want(t, http.StatusOK)
		w.clinician.Do("GET", epPath(w.a, e, "/hq"), nil).Want(t, http.StatusOK)
		w.clinician.Do("GET", epPath(w.a, e, "/tasks"), nil).Want(t, http.StatusOK)
		r := w.clinician.Do("GET", at(w.a, "/patients/"+e.PatientID.String()), nil)
		r.Want(t, http.StatusOK)
		var page struct{ Episodes []struct{ ID uuid.UUID } }
		r.Decode(t, &page)
		if len(page.Episodes) != 1 || page.Episodes[0].ID != e.ID {
			t.Errorf("patient page %s", r)
		}
		var last string
		if err := w.db.QueryRow(t.Context(), `SELECT last_name FROM patients WHERE id = $1`, e.PatientID).Scan(&last); err != nil {
			t.Fatal(err)
		}
		r = w.clinician.Do("GET", at(w.a, "/patients?q="+strings.ReplaceAll(last, " ", "+")), nil)
		r.Want(t, http.StatusOK)
		if !strings.Contains(string(r.Body), e.PatientID.String()) {
			t.Errorf("the search didn't find the patient: %s", r)
		}
	})
	t.Run("un-archived, it can be changed again", func(t *testing.T) {
		w.author.Do("POST", epPath(w.a, e, "/unarchive"), withRevision(t, w, e.ID, map[string]any{"reason": "x"})).Want(t, http.StatusNoContent)
		w.clinician.Do("POST", epPath(w.a, e, "/notes"), map[string]any{"text": "A comment"}).Want(t, http.StatusCreated)
		w.clinician.Do("PATCH", epPath(w.a, e, "/tasks/"+task.String()), map[string]any{"done": true, "revision": revisionOf(t, w, e.ID)}).Want(t, http.StatusNoContent)
	})
}

// Archived episodes leave every worklist, and the All list unless "Show archived" (S22 answers 2).
func TestArchivedLeavesWorklists(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, e, episode.OnHold)
	if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET planned_date = current_date + 1 WHERE id = $1`, e.ID); err != nil {
		t.Fatal(err)
	}
	addTask(t, w, e, "Book anaesthetic review", "anaesthetist", false)
	other := w.fx.Episode(t, w.hq, w.cara) // stays put, in All
	in := func(path string) bool { return slices.Contains(listIDs(t, w.clinician, path), e.ID) }
	countOf := func(path, key string) int {
		for _, c := range counts(t, w.clinician, path) {
			if c.Key == key {
				return c.Count
			}
		}
		return -1
	}
	lists := []string{"on_hold", "anaesthetic_review", "open_tasks", "procedure_soon"}
	for _, key := range lists {
		if !in(at(w.a, "/episodes?worklist="+key)) || countOf(at(w.a, "/worklists"), key) != 1 {
			t.Fatalf("before archiving, not in %s", key)
		}
	}
	archive(t, w, e)
	for _, key := range worklistKeys {
		for _, path := range []string{at(w.a, "/episodes?worklist="+key), "/api/o/" + w.a.Org.String() + "/episodes?worklist=" + key} {
			if in(path) {
				t.Errorf("archived, still in %s", path)
			}
		}
		if n := countOf(at(w.a, "/worklists"), key); n != 0 {
			t.Errorf("archived, %s counts %d", key, n)
		}
	}
	if in(at(w.a, "/episodes")) || in(at(w.a, "/episodes?status=")) {
		t.Error("archived, still in All")
	}
	if !in(at(w.a, "/episodes?archived=true")) || !slices.Contains(listIDs(t, w.clinician, at(w.a, "/episodes?archived=true")), other.ID) {
		t.Error("Show archived lists neither the archived nor the others")
	}
	if got := listIDs(t, w.clinician, at(w.a, "/episodes?status=archived")); !slices.Equal(got, []uuid.UUID{e.ID}) {
		t.Errorf("status archived: %v", got)
	}
	trustAll := "/api/o/" + w.a.Org.String() + "/episodes"
	if in(trustAll) || !slices.Contains(listIDs(t, w.author, trustAll+"?archived=true"), e.ID) {
		t.Error("All hospitals: archived should be shown only with Show archived")
	}
	w.author.Do("POST", epPath(w.a, e, "/unarchive"), withRevision(t, w, e.ID, map[string]any{"reason": "x"})).Want(t, http.StatusNoContent)
	for _, key := range lists {
		if !in(at(w.a, "/episodes?worklist="+key)) || countOf(at(w.a, "/worklists"), key) != 1 {
			t.Errorf("un-archived, not back in %s", key)
		}
	}
}

// On the patient's home an archived questionnaire is "closed" (no form), and one cancelled before
// it was archived is still "cancelled"; un-archived, it's as before.
func TestPatientHomeShowsArchived(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	sam := newPerson("1980-05-17")
	open, cancelled := w.recordIn(t, sam, w.a, w.hq, w.cara), w.recordIn(t, sam, w.a, w.hq, w.cara)
	b := w.anon.AsPatient(t, open.Link, sam.dob)
	setStatus(t, w.clinic, cancelled, "cancelled")
	archive(t, w.clinic, open)
	archive(t, w.clinic, cancelled)
	states := func() map[uuid.UUID]string {
		out := map[uuid.UUID]string{}
		for _, e := range readHome(t, b).Episodes {
			out[e.ID] = e.State
		}
		return out
	}
	if s := states(); s[open.ID] != "closed" || s[cancelled.ID] != "cancelled" {
		t.Errorf("home states %v", s)
	}
	w.author.Do("POST", epPath(w.a, open, "/unarchive"), withRevision(t, w.clinic, open.ID, map[string]any{"reason": "x"})).Want(t, http.StatusNoContent)
	if s := states(); s[open.ID] != "to_do" {
		t.Errorf("un-archived: %v", s)
	}
	b.Do("GET", patientPath(open, ""), nil).Want(t, http.StatusOK) // and the form opens again
}

// A questionnaire the patient sent, then archived (S27; pending.md, S22; Rahul, 24 Oct): the home
// still says Sent with View, and the form opens read-only, as a sent one does; every write is
// refused as before. One archived before it was sent stays "closed".
func TestArchivedAfterSentKeepsItsView(t *testing.T) {
	t.Parallel()
	w := newPeople(t)
	sam := newPerson("1981-06-18")
	sent, unsent := w.recordIn(t, sam, w.a, w.hq, w.cara), w.recordIn(t, sam, w.a, w.hq, w.cara)
	b := w.anon.AsPatient(t, sent.Link, sam.dob)
	set := w.hq.Chapters[0].String()
	savedRevision(t, b.Do("PUT", patientPath(sent, "/answers/"+set), map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 0}))
	b.Do("POST", patientPath(sent, "/submit"), nil).Want(t, http.StatusNoContent)
	archive(t, w.clinic, sent)
	archive(t, w.clinic, unsent)

	states := map[uuid.UUID]string{}
	for _, e := range readHome(t, b).Episodes {
		states[e.ID] = e.State
	}
	if states[sent.ID] != "sent" || states[unsent.ID] != "closed" {
		t.Errorf("home states %v, want sent and closed", states)
	}
	r := b.Do("GET", patientPath(sent, ""), nil)
	r.Want(t, http.StatusOK)
	var f struct {
		Submitted bool
		Answers   map[string]map[string]json.RawMessage
	}
	r.Decode(t, &f)
	if !f.Submitted || string(f.Answers[set]["q_smokes"]) != `"o_no"` {
		t.Errorf("the View: %s", r)
	}
	for _, r := range []*apitest.Resp{
		b.Do("PUT", patientPath(sent, "/answers/"+set), map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 1}),
		b.Do("POST", patientPath(sent, "/submit"), nil),
		b.Do("POST", patientPath(sent, "/prefill/"+set), map[string]string{"choice": "check_each"}),
		b.Do("GET", patientPath(unsent, ""), nil),
	} {
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(string(r.Body), `"code":"archived"`) {
			t.Errorf("archived write: %s", r)
		}
	}
	if n := count(t, w.db, `SELECT count(*) FROM episode_answers a JOIN episode_forms f ON f.id = a.episode_form_id WHERE f.episode_id = $1 AND a.data->>'q_smokes' = 'o_yes'`, sent.ID); n != 0 {
		t.Error("an archived answer changed")
	}
	// Staff have the episode's own pages; the on-behalf form stays refused once archived.
	refused(t, w.clinician.Do("GET", epPath(w.a, sent, "/patient-form"), nil), http.StatusConflict)
}

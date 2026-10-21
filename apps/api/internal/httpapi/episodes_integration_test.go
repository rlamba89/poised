//go:build integration

package httpapi_test

import (
	"maps"
	"net/http"
	"reflect"
	"slices"
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
	// Trust A's procedure list (S19): hip and knee for every hospital, cataract for A only,
	// appendix retired; B's own.
	hip, knee, cataract, appendix, theirProc uuid.UUID
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
	proc := func(org uuid.UUID, name string, hospital *uuid.UUID, retired bool) uuid.UUID {
		var id uuid.UUID
		if err := w.db.QueryRow(t.Context(), `INSERT INTO procedures (org_id, hospital_id, name, retired) VALUES ($1, $2, $3, $4) RETURNING id`,
			org, hospital, name, retired).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	w.hip, w.knee = proc(w.a.Org, "Hip replacement", nil, false), proc(w.a.Org, "Knee scope", nil, false)
	w.cataract, w.appendix = proc(w.a.Org, "Cataract", &w.a.ID, false), proc(w.a.Org, "Appendix", nil, true)
	w.theirProc = proc(w.b.Org, "Theirs", nil, false)
	return w
}

// episodeRow is the part of an episodes row the tests check.
type episodeRow struct {
	Status, Procedure, Anaesthetic, Consultant string // Procedure: its name in the trust's list, or ""
	Planned, Poa                               string // YYYY-MM-DD, or ""
	NurseAsa, AnaesthetistAsa                  *int
	Submitted, ReviewCompleted                 *time.Time
	ReviewCompletedBy                          *uuid.UUID
}

func readEpisode(t *testing.T, w *clinic, id uuid.UUID) episodeRow {
	t.Helper()
	var e episodeRow
	err := w.db.QueryRow(t.Context(), `SELECT e.status, coalesce(p.name, ''), coalesce(e.planned_date::text, ''), coalesce(e.poa_date::text, ''),
		e.anaesthetic, e.consultant, e.nurse_asa, e.anaesthetist_asa, f.submitted_at, e.review_completed_at, e.review_completed_by
		FROM episodes e JOIN episode_forms f ON f.episode_id = e.id AND f.kind = 'hq' LEFT JOIN procedures p ON p.id = e.procedure_id
		WHERE e.id = $1`, id).Scan(
		&e.Status, &e.Procedure, &e.Planned, &e.Poa, &e.Anaesthetic, &e.Consultant, &e.NurseAsa, &e.AnaesthetistAsa,
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
		WHERE episode_id = $1 ORDER BY created_at, seq`, episode)
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
	// An archived one is archived from On hold (S22: archived_from is set exactly while archived).
	_, err := w.db.Exec(t.Context(), `UPDATE episodes SET status = $2,
		review_completed_at = CASE WHEN $2 IN ('hq_not_complete', 'ready_for_review') THEN NULL ELSE now() END,
		archived_from = CASE WHEN $2 = 'archived' THEN 'on_hold' END
		WHERE id = $1`, e.ID, status)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.db.Exec(t.Context(), `UPDATE episode_forms SET
		status = CASE WHEN $2 = 'hq_not_complete' THEN 'open' ELSE 'submitted' END,
		submitted_at = CASE WHEN $2 = 'hq_not_complete' THEN NULL ELSE now() END
		WHERE id = $1`, e.FormID, status)
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
	_, err := w.db.Exec(t.Context(), `INSERT INTO episode_answers (episode_form_id, chapter_id, actor, data, updated_by)
		VALUES ($1, $2, $3, $4, $5)`, e.FormID, chapter, actor, data, user)
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

	t.Run("happy path: published versions only, not drafts", func(t *testing.T) { // before the next case adds one
		got := versions(t, w.clinician.Do("GET", at(w.a, "/published-hqs"), nil))
		if len(got) != 1 || got[0] != w.hq.VersionID {
			t.Errorf("versions = %v, want only %s (not the draft %s)", got, w.hq.VersionID, draft.VersionID)
		}
	})
	t.Run("happy path: the hospital's default HQ is marked, its own value over the trust's (S19)", func(t *testing.T) {
		defaults := func(t *testing.T) []uuid.UUID {
			r := w.clinician.Do("GET", at(w.a, "/published-hqs"), nil)
			r.Want(t, http.StatusOK)
			var got []struct {
				VersionID, QuestionnaireID uuid.UUID
				IsDefault                  bool
			}
			r.Decode(t, &got)
			var out []uuid.UUID
			for _, v := range got {
				if v.IsDefault {
					out = append(out, v.QuestionnaireID)
				}
			}
			return out
		}
		if got := defaults(t); len(got) != 0 {
			t.Errorf("defaults = %v, want none", got)
		}
		other := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		if _, err := w.db.Exec(t.Context(), `UPDATE orgs SET settings = jsonb_build_object('defaultHq', $2::text) WHERE id = $1`, w.a.Org, w.hq.ID.String()); err != nil {
			t.Fatal(err)
		}
		if got := defaults(t); len(got) != 1 || got[0] != w.hq.ID {
			t.Errorf("defaults = %v, want the trust's %s", got, w.hq.ID)
		}
		if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET settings = jsonb_build_object('defaultHq', $2::text) WHERE id = $1`, w.a.ID, other.ID.String()); err != nil {
			t.Fatal(err)
		}
		if got := defaults(t); len(got) != 1 || got[0] != other.ID {
			t.Errorf("defaults = %v, want the hospital's own %s", got, other.ID)
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
	t.Run("happy path: each with its procedure's name and dates (S19)", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET procedure_id = $2, planned_date = '2026-11-03', poa_date = '2026-10-28'
			WHERE id = $1`, submitted.ID, w.appendix); err != nil {
			t.Fatal(err)
		}
		r := w.clinician.Do("GET", at(w.a, "/episodes"), nil)
		r.Want(t, http.StatusOK)
		var got []struct {
			ID                   uuid.UUID
			Procedure            string
			PlannedDate, PoaDate *string
		}
		r.Decode(t, &got)
		if len(got) != 2 || got[0].Procedure != "Appendix" || got[0].PlannedDate == nil || *got[0].PlannedDate != "2026-11-03" ||
			got[0].PoaDate == nil || *got[0].PoaDate != "2026-10-28" || got[1].Procedure != "" || got[1].PlannedDate != nil {
			t.Errorf("episodes = %+v, want the retired procedure's name and the dates on the first, none on the second", got)
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
		return map[string]any{"patientId": patient, "versionId": version, "procedureId": w.hip,
			"plannedDate": "2026-11-03", "poaDate": " 2026-10-28 ", "anaesthetic": " General ", "consultant": " Dr Grey "}
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
	t.Run("a procedure the list doesn't offer here is refused (S19)", func(t *testing.T) {
		for name, id := range map[string]any{
			"retired": w.appendix, "another hospital's own": w.cataract, "another trust's": w.theirProc,
			"unknown": uuid.New(), "not an id": "hip",
		} {
			b := body(w.ep.PatientID, w.hq.VersionID)
			b["procedureId"] = id
			h, c := w.a, w.clinician
			if name == "another hospital's own" {
				h, c = w.a2, w.site2
				b["patientId"] = nil
				b["patient"] = map[string]any{"firstName": "Al", "lastName": "Other", "dateOfBirth": "1980-01-01", "sex": "male", "mobile": "07700 900111"}
			}
			r := create(c, h, b)
			refused(t, r, http.StatusBadRequest)
			if !strings.Contains(r.Message(), "procedure") {
				t.Errorf("%s: message = %q", name, r.Message())
			}
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, dates := range []map[string]string{{"plannedDate": "03/11/2026"}, {"poaDate": "2026-13-01"}} {
			b := body(w.ep.PatientID, w.hq.VersionID)
			for k, v := range dates {
				b[k] = v
			}
			refused(t, create(w.clinician, w.a, b), http.StatusBadRequest)
		}
		refused(t, create(w.clinician, w.a, body(uuid.New(), w.hq.VersionID)), http.StatusBadRequest)
		refused(t, create(w.clinician, w.a, map[string]any{"versionId": w.hq.VersionID}), http.StatusBadRequest)
		refused(t, create(w.clinician, w.a, `{"patientId": "abc"}`), http.StatusBadRequest)
		refused(t, create(w.clinician, w.a, `{"patientId":`), http.StatusBadRequest)
	})
	if n, m := episodes(t), events(t); n != beforeEpisodes || m != beforeEvents {
		t.Fatalf("refusals wrote %d episodes and %d events, want %d and %d", n, m, beforeEpisodes, beforeEvents)
	}
	t.Run("happy path: waiting for the patient, with a created event (the invite: invites_integration_test.go)", func(t *testing.T) {
		r := create(w.clinician, w.a, good)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		e := readEpisode(t, w, got.ID)
		if e.Status != "hq_not_complete" || e.Procedure != "Hip replacement" || e.Planned != "2026-11-03" || e.Poa != "2026-10-28" ||
			e.Anaesthetic != "General" || e.Consultant != "Dr Grey" {
			t.Errorf("episode = %+v", e)
		}
		ev := readEvents(t, w, got.ID)
		if len(ev) != 2 || ev[0].Kind != "event" || ev[0].Text != "Episode created" || ev[0].User == nil || *ev[0].User != w.cara.ID {
			t.Errorf("events = %+v, want 'Episode created' by Cara, then the invite's note", ev)
		}
		// Its one form: the HQ on the chosen version, open, with no dates yet (S18).
		if n := count(t, w.db, `SELECT count(*) FROM episode_forms WHERE episode_id = $1`, got.ID); n != 1 {
			t.Errorf("forms = %d, want 1", n)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_forms WHERE episode_id = $1 AND kind = 'hq' AND version_id = $2
			AND status = 'open' AND submitted_at IS NULL AND due_at IS NULL AND closes_at IS NULL`, got.ID, w.hq.VersionID); n != 1 {
			t.Error("the episode's form isn't its open HQ on the chosen version")
		}
	})
	t.Run("happy path: no procedure or dates; one of this hospital's own", func(t *testing.T) {
		b := body(w.ep.PatientID, w.hq.VersionID)
		delete(b, "procedureId")
		b["plannedDate"], b["poaDate"] = "", nil
		r := create(w.clinician, w.a, b)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		if e := readEpisode(t, w, got.ID); e.Procedure != "" || e.Planned != "" || e.Poa != "" {
			t.Errorf("episode = %+v, want no procedure or dates", e)
		}
		b["procedureId"] = w.cataract
		r = create(w.clinician, w.a, b)
		r.Want(t, http.StatusCreated)
		r.Decode(t, &got)
		if e := readEpisode(t, w, got.ID); e.Procedure != "Cataract" {
			t.Errorf("episode = %+v, want A's own Cataract", e)
		}
	})
	t.Run("every episode has its one HQ form, and the refusals made none", func(t *testing.T) {
		if n, m := count(t, w.db, "SELECT count(*) FROM episode_forms"), episodes(t); n != m {
			t.Errorf("%d forms for %d episodes, want one each", n, m)
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
	t.Run("happy path: the procedure, its id and the dates (S19)", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET procedure_id = $2, poa_date = '2026-10-28' WHERE id = $1`, w.ep.ID, w.hip); err != nil {
			t.Fatal(err)
		}
		var got struct {
			Episode struct {
				ProcedureID          *uuid.UUID
				Procedure            string
				PlannedDate, PoaDate *string
			}
		}
		get(w.clinician, w.a).Decode(t, &got)
		e := got.Episode
		if e.ProcedureID == nil || *e.ProcedureID != w.hip || e.Procedure != "Hip replacement" || e.PlannedDate != nil || e.PoaDate == nil || *e.PoaDate != "2026-10-28" {
			t.Errorf("episode = %+v", e)
		}
	})
	t.Run("the HQ's version and submit time come from its form, under the names they had before forms (S18)", func(t *testing.T) {
		type hq struct {
			Episode struct {
				VersionID          uuid.UUID
				VersionNo          int
				PatientSubmittedAt *time.Time
			}
		}
		var before, after hq
		get(w.clinician, w.a).Decode(t, &before)
		setStatus(t, w, w.ep, "ready_for_review")
		get(w.clinician, w.a).Decode(t, &after)
		setStatus(t, w, w.ep, "hq_not_complete")
		if before.Episode.VersionID != w.hq.VersionID || before.Episode.VersionNo != 1 || before.Episode.PatientSubmittedAt != nil {
			t.Errorf("before the submit: %+v", before.Episode)
		}
		if after.Episode.VersionID != w.hq.VersionID || after.Episode.PatientSubmittedAt == nil {
			t.Errorf("after the submit: %+v", after.Episode)
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
	// patch sends the episode's current revision, as the page would (S20), unless the body has one.
	patch := func(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body any) *apitest.Resp {
		if m, ok := body.(map[string]any); ok {
			if _, has := m["revision"]; !has {
				m = maps.Clone(m)
				m["revision"] = revisionOf(t, w, e.ID)
				body = m
			}
		}
		return c.Do("PATCH", epPath(h, e, ""), body)
	}
	reviewed := w.fx.Episode(t, w.hq, w.cara)
	setStatus(t, w, reviewed, "ready_for_poa")
	change := map[string]any{"consultant": "Dr Who"}
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
			`{"anaesthetic": null, "revision": 0}`,
			`{"consultant": null, "revision": 0}`,
			map[string]any{"nurseAsa": 7},
			map[string]any{"anaesthetistAsa": -1},
			map[string]any{"procedureId": "hip"},
			map[string]any{"plannedDate": "tomorrow"},
			map[string]any{"poaDate": "2026-02-30"},
			map[string]any{"consultant": "Dr Who", "revision": nil}, // S20: the revision the page loaded is required
			map[string]any{"status": "poa_complete"},                // S20: the status has its own route
			`{"consultant":`,
		} {
			refused(t, patch(w.clinician, w.a, reviewed, bad), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("PATCH", at(w.a, "/episodes/abc"), change), http.StatusNotFound)
	})
	t.Run("a stale revision is refused (S20)", func(t *testing.T) {
		refused(t, patch(w.clinician, w.a, reviewed, map[string]any{"consultant": "Dr Who", "revision": 7}), http.StatusConflict)
	})
	if after := readEpisode(t, w, reviewed.ID); !reflect.DeepEqual(after, before) {
		t.Fatalf("refusals changed the episode: %+v, was %+v", after, before)
	}
	if ev := readEvents(t, w, reviewed.ID); len(ev) != 0 {
		t.Fatalf("refusals wrote events: %+v", ev)
	}

	t.Run("happy path: details and ASA grades, each change with an event", func(t *testing.T) {
		patch(w.clinician, w.a, reviewed, map[string]any{
			"consultant": " Dr Who ", "nurseAsa": 2, "anaesthetistAsa": 3}).Want(t, http.StatusNoContent)
		e := readEpisode(t, w, reviewed.ID)
		if e.Status != "ready_for_poa" || e.Consultant != "Dr Who" || e.Anaesthetic != before.Anaesthetic ||
			e.NurseAsa == nil || *e.NurseAsa != 2 || e.AnaesthetistAsa == nil || *e.AnaesthetistAsa != 3 {
			t.Errorf("episode = %+v", e)
		}
		if rev := revisionOf(t, w, reviewed.ID); rev != 1 {
			t.Errorf("revision %d, want 1", rev)
		}
		want := []string{"Nurse ASA grade set to 2", "Anaesthetist ASA grade set to 3"}
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
	t.Run("happy path: the procedure and its dates, before the review, each change with an event (S19)", func(t *testing.T) {
		patch(w.clinician, w.a, w.ep, map[string]any{"procedureId": w.hip.String(), "plannedDate": "2026-11-03",
			"poaDate": "2026-10-28"}).Want(t, http.StatusNoContent)
		if e := readEpisode(t, w, w.ep.ID); e.Status != "hq_not_complete" || e.Procedure != "Hip replacement" ||
			e.Planned != "2026-11-03" || e.Poa != "2026-10-28" {
			t.Errorf("episode = %+v, want the procedure and dates set and the status kept", e)
		}
		setStatus(t, w, w.ep, "ready_for_review") // the patient has sent it: still before the review is complete
		patch(w.clinician, w.a, w.ep, map[string]any{"procedureId": w.knee, "plannedDate": nil, "poaDate": "2026-10-29"}).Want(t, http.StatusNoContent)
		if e := readEpisode(t, w, w.ep.ID); e.Procedure != "Knee scope" || e.Planned != "" || e.Poa != "2026-10-29" {
			t.Errorf("episode = %+v", e)
		}
		patch(w.clinician, w.a, w.ep, map[string]any{"procedureId": ""}).Want(t, http.StatusNoContent)
		want := []string{"Procedure set to Hip replacement", "Planned procedure date set to 2026-11-03", "POA appointment date set to 2026-10-28",
			"Procedure set to Knee scope", "Planned procedure date cleared", "POA appointment date set to 2026-10-29", "Procedure cleared"}
		var got []string
		for _, ev := range readEvents(t, w, w.ep.ID) {
			got = append(got, ev.Text)
		}
		if !slices.Equal(got, want) {
			t.Errorf("events = %q, want %q", got, want)
		}
		setStatus(t, w, w.ep, "hq_not_complete")
	})
	t.Run("a retired procedure stays on the episode, but can't be newly chosen; nor another hospital's", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET procedure_id = $2 WHERE id = $1`, e.ID, w.appendix); err != nil {
			t.Fatal(err)
		}
		patch(w.clinician, w.a, e, map[string]any{"consultant": "Dr Grey", "procedureId": w.appendix, "poaDate": "2026-10-30"}).Want(t, http.StatusNoContent)
		if got := readEpisode(t, w, e.ID); got.Procedure != "Appendix" || got.Poa != "2026-10-30" {
			t.Errorf("episode = %+v, want the retired procedure kept", got)
		}
		other := w.fx.Episode(t, w.hq, w.cara)
		refused(t, patch(w.clinician, w.a, other, map[string]any{"procedureId": w.appendix}), http.StatusBadRequest)
		refused(t, patch(w.clinician, w.a, other, map[string]any{"procedureId": w.theirProc}), http.StatusBadRequest)
		atA2 := w.fx.EpisodeAt(t, w.hq, w.a2, w.cara)
		refused(t, patch(w.site2, w.a2, atA2, map[string]any{"procedureId": w.cataract}), http.StatusBadRequest)
		if got := readEpisode(t, w, other.ID); got.Procedure != "" {
			t.Errorf("episode = %+v, want no procedure", got)
		}
	})
	t.Run("once the review is complete, the procedure and its dates can't change (409); the rest can", func(t *testing.T) {
		for _, change := range []map[string]any{{"procedureId": w.hip}, {"plannedDate": "2026-12-01"}, {"poaDate": "2026-12-01"}} {
			r := patch(w.clinician, w.a, reviewed, change)
			refused(t, r, http.StatusConflict)
			if !strings.Contains(r.Message(), "review is complete") {
				t.Errorf("message = %q", r.Message())
			}
		}
		if e := readEpisode(t, w, reviewed.ID); e.Procedure != "" || e.Planned != "" || e.Poa != "" {
			t.Errorf("episode = %+v, want no procedure or dates", e)
		}
		// Sending them unchanged is no change.
		patch(w.clinician, w.a, reviewed, map[string]any{"procedureId": "", "plannedDate": "", "anaesthetic": "Spinal"}).Want(t, http.StatusNoContent)
	})
	t.Run("happy path: no change, no event; 0 clears a grade", func(t *testing.T) {
		patch(w.clinician, w.a, reviewed, map[string]any{"nurseAsa": 0, "anaesthetistAsa": 3}).Want(t, http.StatusNoContent)
		e := readEpisode(t, w, reviewed.ID)
		if e.NurseAsa != nil || e.AnaesthetistAsa == nil || *e.AnaesthetistAsa != 3 {
			t.Errorf("episode = %+v, want the nurse grade cleared and the anaesthetist's kept", e)
		}
		ev := readEvents(t, w, reviewed.ID)
		if len(ev) != 3 || ev[2].Text != "Nurse ASA grade cleared" {
			t.Errorf("events = %+v, want one more: 'Nurse ASA grade cleared'", ev)
		}
	})
}

// A PATCH sends the revision its page loaded (S20). If the workflow moves the episode on between
// that load and the write (the patient submits, the review is completed), the PATCH is refused
// with 409 and writes nothing: the page didn't see the change. (Before S20 this compared the
// status the request loaded, not the page's: pending.md §1a.)
func TestUpdateEpisodeKeepsAStatusThatJustChanged(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	for _, race := range []struct{ name, from, want string }{
		{"the review is completed", "ready_for_review", "ready_for_poa"},
		{"the patient submits", "hq_not_complete", "ready_for_review"},
	} {
		t.Run(race.name, func(t *testing.T) {
			e := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, e, race.from)
			loaded := revisionOf(t, w, e.ID) // the page loads
			if race.from == "ready_for_review" {
				w.clinician.Do("POST", epPath(w.a, e, "/complete-review"), nil).Want(t, http.StatusNoContent)
			} else {
				w.clinician.Do("POST", epPath(w.a, e, "/patient-form/submit"), nil).Want(t, http.StatusNoContent)
			}
			r := w.clinician.Do("PATCH", epPath(w.a, e, ""), map[string]any{"procedureId": w.knee, "nurseAsa": 2, "revision": loaded})
			refused(t, r, http.StatusConflict)
			if !strings.Contains(r.Message(), "changed since you opened it") {
				t.Errorf("message = %q", r.Message())
			}
			if got := readEpisode(t, w, e.ID); got.Status != race.want || got.Procedure != "" || got.NurseAsa != nil {
				t.Errorf("episode = %+v, want status %s and nothing else changed", got, race.want)
			}
		})
	}
	t.Run("two PATCHes at once from the same revision: the second waits, then is refused", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		r := meanwhile(t, w.db, `UPDATE episodes SET nurse_asa = 2, revision = revision + 1 WHERE id = $1`, []any{e.ID}, func() *apitest.Resp {
			return w.clinician.Do("PATCH", epPath(w.a, e, ""), map[string]any{"anaesthetistAsa": 3, "revision": 0})
		})
		refused(t, r, http.StatusConflict)
		if got := readEpisode(t, w, e.ID); got.NurseAsa == nil || *got.NurseAsa != 2 || got.AnaesthetistAsa != nil {
			t.Errorf("episode = %+v", got)
		}
	})
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

//go:build integration

package httpapi_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Trusts, hospitals and the role ladder (plan C1, session S08): the cases C1 adds to the
// standard five. Trust A has hospitals A1 and A2; trust B has B1.

type trusts struct {
	fx         *apitest.Fixtures
	anon       *apitest.Client
	a1, a2, b1 apitest.Hospital
	hq         apitest.HQ      // trust A's published HQ, used by both its hospitals
	ep1, ep2   apitest.Episode // at A1 and at A2
	epB        apitest.Episode // at B1
	adminA     *apitest.Client // trust A's trust-wide admin
	cara       *apitest.Client // a clinician at A1 only
	superA     *apitest.Client // trust A's trust-wide super clinician
}

func newTrusts(t *testing.T) *trusts {
	t.Helper()
	c, fx := apitest.New(t)
	w := &trusts{fx: fx, anon: c, a1: fx.Hospital(t), b1: fx.Hospital(t)}
	w.a2 = fx.HospitalIn(t, w.a1.Org)
	admin := fx.TrustUser(t, w.a1.Org, "admin")
	cara := fx.User(t, w.a1, "clinician")
	sup := fx.TrustUser(t, w.a1.Org, "super_clinician")
	bea := fx.TrustUser(t, w.b1.Org, "super_clinician")
	w.adminA, w.cara, w.superA = c.LoginAs(t, admin), c.LoginAs(t, cara), c.LoginAs(t, sup)
	w.hq = fx.PublishedHQ(t, w.a1, admin, "workflow.json")
	w.ep1 = fx.EpisodeAt(t, w.hq, w.a1, cara)
	w.ep2 = fx.EpisodeAt(t, w.hq, w.a2, admin)
	w.epB = fx.Episode(t, fx.PublishedHQ(t, w.b1, bea, "basic.json"), bea)
	return w
}

// orgPath is a trust route: /api/o/{org} + p.
func orgPath(org uuid.UUID, p string) string { return "/api/o/" + org.String() + p }

func episodeIDs(t *testing.T, r *apitest.Resp) []uuid.UUID {
	t.Helper()
	r.Want(t, http.StatusOK)
	var got []struct{ ID uuid.UUID }
	r.Decode(t, &got)
	ids := []uuid.UUID{}
	for _, e := range got {
		ids = append(ids, e.ID)
	}
	return ids
}

// C1: an admin of trust A gets 404 for anything of trust B, however the URL is put together.
func TestTrustAAdminGets404ForTrustB(t *testing.T) {
	t.Parallel()
	w := newTrusts(t)
	mixed := apitest.Hospital{Org: w.a1.Org, ID: w.b1.ID} // trust A's id with trust B's hospital
	flipped := apitest.Hospital{Org: w.b1.Org, ID: w.a1.ID}
	for _, h := range []apitest.Hospital{w.b1, mixed, flipped} {
		for _, p := range []string{
			"/questionnaires", "/option-lists", "/patients", "/published-hqs", "/episodes",
			"/episodes/" + w.epB.ID.String(), "/episodes/" + w.epB.ID.String() + "/hq",
		} {
			refused(t, w.adminA.Do("GET", h.Path(p), nil), http.StatusNotFound)
		}
		refused(t, w.adminA.Do("POST", h.Path("/questionnaires"), map[string]string{"name": "x", "description": "x"}), http.StatusNotFound)
	}
	refused(t, w.adminA.Do("GET", orgPath(w.b1.Org, "/episodes"), nil), http.StatusNotFound)
	if n := count(t, w.anon.DB, "SELECT count(*) FROM questionnaires WHERE name = 'x'"); n != 0 {
		t.Errorf("%d questionnaires written through a refused URL", n)
	}
}

// C1: a hospital-scoped clinician can't see another hospital's episodes, by URL or by list.
func TestHospitalScopedClinicianSeesOnlyTheirHospital(t *testing.T) {
	t.Parallel()
	w := newTrusts(t)

	t.Run("the other hospital's URLs are 404", func(t *testing.T) {
		for _, p := range []string{"/episodes", "/patients", "/questionnaires", "/episodes/" + w.ep2.ID.String()} {
			refused(t, w.cara.Do("GET", w.a2.Path(p), nil), http.StatusNotFound)
		}
	})
	t.Run("the other hospital's episode at their own hospital's URL is 404", func(t *testing.T) {
		for _, p := range []string{"", "/hq"} {
			refused(t, w.cara.Do("GET", w.a1.Path("/episodes/"+w.ep2.ID.String()+p), nil), http.StatusNotFound)
		}
		refused(t, w.cara.Do("POST", w.a1.Path("/episodes/"+w.ep2.ID.String()+"/notes"), map[string]string{"text": "hi"}), http.StatusNotFound)
		if n := count(t, w.anon.DB, "SELECT count(*) FROM episode_events WHERE episode_id = $1 AND kind = 'comment'", w.ep2.ID); n != 0 {
			t.Errorf("a refused note was written: %d", n)
		}
	})
	t.Run("their hospital's list has only its episodes", func(t *testing.T) {
		if got := episodeIDs(t, w.cara.Do("GET", w.a1.Path("/episodes"), nil)); !slices.Equal(got, []uuid.UUID{w.ep1.ID}) {
			t.Errorf("episodes = %v, want only A1's %s", got, w.ep1.ID)
		}
	})
	t.Run("but they can open an episode at their hospital for a patient first seen at the other", func(t *testing.T) {
		// Not by the record's id: A2's patient isn't one of A1's (S13)…
		refused(t, w.cara.Do("POST", w.a1.Path("/episodes"), map[string]any{"patientId": w.ep2.PatientID, "versionId": w.hq.VersionID}),
			http.StatusBadRequest)
		// …but by typing their NHS number, date of birth and surname, which links silently.
		var last string
		if err := w.anon.DB.QueryRow(t.Context(), `UPDATE patients SET national_id = '9434765919', national_id_type = 'nhs'
			WHERE id = $1 RETURNING last_name`, w.ep2.PatientID).Scan(&last); err != nil {
			t.Fatal(err)
		}
		r := w.cara.Do("POST", w.a1.Path("/episodes"), map[string]any{"versionId": w.hq.VersionID, "patient": map[string]string{
			"firstName": "T", "lastName": last, "dateOfBirth": "1970-01-01", "sex": "unknown", "nationalId": "9434765919"}})
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		if n := count(t, w.anon.DB, "SELECT count(*) FROM episodes WHERE id = $1 AND hospital_id = $2 AND patient_id = $3",
			got.ID, w.a1.ID, w.ep2.PatientID); n != 1 {
			t.Error("the episode isn't at A1 for that patient")
		}
	})
	t.Run("a trust-wide user works at both hospitals", func(t *testing.T) {
		w.superA.Do("GET", w.a1.Path("/episodes/"+w.ep1.ID.String()), nil).Want(t, http.StatusOK)
		w.superA.Do("GET", w.a2.Path("/episodes/"+w.ep2.ID.String()), nil).Want(t, http.StatusOK)
		// …but each hospital's URL still shows only that hospital's episode.
		refused(t, w.superA.Do("GET", w.a1.Path("/episodes/"+w.ep2.ID.String()), nil), http.StatusNotFound)
	})
}

// Content is the trust's for every hospital, unless it's one hospital's own.
func TestHospitalOwnedContentStaysAtItsHospital(t *testing.T) {
	t.Parallel()
	w := newTrusts(t)
	var mine uuid.UUID
	err := w.anon.DB.QueryRow(t.Context(), `INSERT INTO option_lists (org_id, hospital_id, name, options, updated_by)
		SELECT $1, $2, 'A2 only', '[{"text": "Yes"}]', created_by FROM questionnaires WHERE id = $3 RETURNING id`,
		w.a2.Org, w.a2.ID, w.hq.ID).Scan(&mine)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.anon.DB.Exec(t.Context(), "UPDATE questionnaires SET hospital_id = $1 WHERE id = $2", w.a2.ID, w.hq.ID); err != nil {
		t.Fatal(err)
	}
	names := func(t *testing.T, h apitest.Hospital) int {
		r := w.superA.Do("GET", h.Path("/option-lists"), nil)
		r.Want(t, http.StatusOK)
		var got []struct{ ID uuid.UUID }
		r.Decode(t, &got)
		return len(got)
	}
	if names(t, w.a1) != 0 || names(t, w.a2) != 1 {
		t.Error("A2's option list should show at A2 only")
	}
	refused(t, w.superA.Do("GET", w.a1.Path("/questionnaires/"+w.hq.ID.String()), nil), http.StatusNotFound)
	w.superA.Do("GET", w.a2.Path("/questionnaires/"+w.hq.ID.String()), nil).Want(t, http.StatusOK)
	refused(t, w.superA.Do("POST", w.a1.Path("/episodes"), map[string]any{"patientId": w.ep1.PatientID, "versionId": w.hq.VersionID}), http.StatusBadRequest)
	refused(t, w.superA.Do("DELETE", w.a1.Path("/option-lists/"+mine.String()), nil), http.StatusNotFound)
}

// A-5: clinician < super clinician < admin, at the hospital where the request is made: the
// higher of a trust-wide role and a role at that hospital.
func TestRoleLadder(t *testing.T) {
	t.Parallel()
	w := newTrusts(t)
	draft := func(t *testing.T) apitest.HQ {
		return w.fx.DraftHQ(t, w.a1, w.fx.User(t, w.a1, "super_clinician"), "workflow.json") // the trust publishes workflow.json too; basic.json's q_smokes is another kind of question (S23 lock)
	}
	publish := func(c *apitest.Client, h apitest.Hospital, hq apitest.HQ) *apitest.Resp {
		return c.Do("POST", h.Path("/questionnaires/"+hq.ID.String()+"/publish"), map[string]any{"versionId": hq.VersionID})
	}
	del := func(c *apitest.Client, h apitest.Hospital, hq apitest.HQ) *apitest.Resp {
		return c.Do("DELETE", h.Path("/questionnaires/"+hq.ID.String()), nil)
	}

	t.Run("a clinician can't publish; a super clinician can", func(t *testing.T) {
		hq := draft(t)
		refused(t, publish(w.cara, w.a1, hq), http.StatusForbidden)
		publish(w.superA, w.a1, hq).Want(t, http.StatusNoContent)
	})
	t.Run("a super clinician can't delete someone else's draft; an admin can", func(t *testing.T) {
		hq := draft(t)
		refused(t, del(w.superA, w.a1, hq), http.StatusForbidden)
		del(w.adminA, w.a1, hq).Want(t, http.StatusNoContent)
	})
	t.Run("a trust-wide clinician who is super clinician at one hospital publishes only that hospital's own HQ", func(t *testing.T) {
		u := w.fx.TrustUser(t, w.a1.Org, "clinician")
		w.fx.Member(t, u, w.a2, "super_clinician")
		c := w.anon.LoginAs(t, u)
		trusts := draft(t) // the trust's: a whole-trust role is needed (S08 answers, 5)
		refused(t, publish(c, w.a2, trusts), http.StatusForbidden)
		own := ownedBy(t, w, draft(t), w.a2)
		refused(t, publish(c, w.a1, own), http.StatusNotFound) // A1 doesn't see A2's own HQ
		publish(c, w.a2, own).Want(t, http.StatusNoContent)
	})
	t.Run("everyone in a hospital sees its data, whatever their role", func(t *testing.T) {
		w.cara.Do("GET", w.a1.Path("/questionnaires"), nil).Want(t, http.StatusOK)
		w.adminA.Do("GET", w.a1.Path("/episodes/"+w.ep1.ID.String()), nil).Want(t, http.StatusOK)
	})
}

// STF-04: memberships are re-read on every request, so removing one ends access at once.
func TestRemovedMembershipEndsAccessOnTheNextRequest(t *testing.T) {
	t.Parallel()
	w := newTrusts(t)
	u := w.fx.User(t, w.a1, "clinician")
	c := w.anon.LoginAs(t, u)
	c.Do("GET", w.a1.Path("/episodes"), nil).Want(t, http.StatusOK)
	if _, err := w.anon.DB.Exec(t.Context(), "DELETE FROM memberships WHERE user_id = $1", u.ID); err != nil {
		t.Fatal(err)
	}
	refused(t, c.Do("GET", w.a1.Path("/episodes"), nil), http.StatusNotFound)
	refused(t, c.Do("GET", orgPath(w.a1.Org, "/episodes"), nil), http.StatusNotFound)
}

// GET /api/o/{oid}/episodes: "All hospitals" (STF-03). Everyone in a trust is at least a
// clinician, so the role case is what a hospital-scoped user gets: only their own hospitals.
func TestListTrustEpisodes(t *testing.T) {
	t.Parallel()
	w := newTrusts(t)
	setStatus := func(e apitest.Episode, status string) {
		if _, err := w.anon.DB.Exec(t.Context(), "UPDATE episodes SET status = $2 WHERE id = $1", e.ID, status); err != nil {
			t.Fatal(err)
		}
	}
	setStatus(w.ep2, "ready_for_review")

	t.Run("happy path: a trust-wide user sees every hospital's episodes, newest first, with the hospital", func(t *testing.T) {
		r := w.superA.Do("GET", orgPath(w.a1.Org, "/episodes"), nil)
		if got := episodeIDs(t, r); !slices.Equal(got, []uuid.UUID{w.ep2.ID, w.ep1.ID}) {
			t.Errorf("episodes = %v, want A2's then A1's", got)
		}
		var rows []struct {
			HospitalID   uuid.UUID
			HospitalName string
		}
		r.Decode(t, &rows)
		if rows[0].HospitalID != w.a2.ID || rows[0].HospitalName == "" {
			t.Errorf("rows = %+v, want each with its hospital", rows)
		}
	})
	t.Run("happy path: filtered by status", func(t *testing.T) {
		got := episodeIDs(t, w.superA.Do("GET", orgPath(w.a1.Org, "/episodes?status=ready_for_review"), nil))
		if !slices.Equal(got, []uuid.UUID{w.ep2.ID}) {
			t.Errorf("episodes = %v, want only %s", got, w.ep2.ID)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", orgPath(w.a1.Org, "/episodes"), nil), http.StatusUnauthorized)
	})
	t.Run("role: a hospital-scoped user gets only their hospitals' episodes", func(t *testing.T) {
		if got := episodeIDs(t, w.cara.Do("GET", orgPath(w.a1.Org, "/episodes"), nil)); !slices.Equal(got, []uuid.UUID{w.ep1.ID}) {
			t.Errorf("Cara's episodes = %v, want only A1's %s", got, w.ep1.ID)
		}
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.adminA.Do("GET", orgPath(w.b1.Org, "/episodes"), nil), http.StatusNotFound)
		refused(t, w.adminA.Do("GET", orgPath(uuid.New(), "/episodes"), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.adminA.Do("GET", "/api/o/abc/episodes", nil), http.StatusNotFound)
		if got := episodeIDs(t, w.adminA.Do("GET", orgPath(w.a1.Org, "/episodes?status=nonsense"), nil)); len(got) != 0 {
			t.Errorf("episodes = %v, want none", got)
		}
	})
}

// ownedBy makes hq one hospital's own instead of the trust's.
func ownedBy(t *testing.T, w *trusts, hq apitest.HQ, h apitest.Hospital) apitest.HQ {
	t.Helper()
	if _, err := w.anon.DB.Exec(t.Context(), "UPDATE questionnaires SET hospital_id = $1 WHERE id = $2", h.ID, hq.ID); err != nil {
		t.Fatal(err)
	}
	hq.Hospital = h
	return hq
}

// S08 answers, 5 (Rahul, 12 Oct): only super clinicians and admins of the whole trust change the
// trust's questionnaires and option lists. A super clinician of one hospital can see them, and
// can change only that hospital's own content.
func TestTrustContentNeedsAWholeTrustRole(t *testing.T) {
	t.Parallel()
	w := newTrusts(t)
	sue := w.fx.User(t, w.a1, "super_clinician") // A1 only
	hospAdmin := w.fx.User(t, w.a1, "admin")     // A1 only
	suec, hadmin := w.anon.LoginAs(t, sue), w.anon.LoginAs(t, hospAdmin)
	author := w.fx.TrustUser(t, w.a1.Org, "super_clinician")
	draft := w.fx.DraftHQ(t, w.a1, author, "basic.json")
	published := w.fx.PublishedHQ(t, w.a1, author, "basic.json")
	var list uuid.UUID
	if err := w.anon.DB.QueryRow(t.Context(), `INSERT INTO option_lists (org_id, name, options, updated_by)
		VALUES ($1, 'Yes or no', '[{"text": "Yes"}]', $2) RETURNING id`, w.a1.Org, author.ID).Scan(&list); err != nil {
		t.Fatal(err)
	}
	q := func(hq apitest.HQ, p string) string { return w.a1.Path("/questionnaires/" + hq.ID.String() + p) }
	ch := func(p string) string { return w.a1.Path("/chapters/" + draft.Chapters[0].String() + p) }
	content := map[string]any{"content": map[string]any{"pages": []any{}}, "revision": 0}
	options := map[string]any{"name": "Yes, no", "options": []map[string]string{{"text": "Yes"}, {"text": "No"}}}
	// Every change to the trust's content, and its status when allowed.
	changes := []struct {
		method, path string
		body         any
		ok           int
	}{
		{"POST", w.a1.Path("/questionnaires"), map[string]string{"name": "New", "description": "x"}, http.StatusCreated},
		{"PUT", q(draft, "/chapter-order"), []uuid.UUID{draft.Chapters[1], draft.Chapters[0]}, http.StatusNoContent},
		{"POST", q(draft, "/chapters"), map[string]string{"name": "More"}, http.StatusCreated},
		{"PATCH", ch(""), map[string]string{"name": "Renamed"}, http.StatusNoContent},
		{"PUT", ch("/content"), content, http.StatusOK},
		{"POST", q(published, "/versions"), nil, http.StatusCreated},
		{"POST", w.a1.Path("/option-lists"), options, http.StatusCreated},
		{"PUT", w.a1.Path("/option-lists/" + list.String()), map[string]any{"name": "Yes", "options": options["options"]}, http.StatusNoContent},
		{"POST", q(draft, "/publish"), map[string]any{"versionId": draft.VersionID}, http.StatusNoContent},
	}
	snapshot := func(t *testing.T) string {
		var s string
		err := w.anon.DB.QueryRow(t.Context(), `SELECT concat_ws('|',
			(SELECT count(*) FROM questionnaires), (SELECT count(*) FROM questionnaire_versions),
			(SELECT string_agg(status, ',' ORDER BY id) FROM questionnaire_versions),
			(SELECT string_agg(name || revision || position, ',' ORDER BY id) FROM chapters),
			(SELECT string_agg(name, ',' ORDER BY id) FROM option_lists))`).Scan(&s)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := snapshot(t)

	t.Run("no sign-in", func(t *testing.T) {
		for _, c := range changes {
			refused(t, w.anon.Do(c.method, c.path, c.body), http.StatusUnauthorized)
		}
	})
	t.Run("wrong role: a super clinician, or an admin, of one hospital only", func(t *testing.T) {
		for _, who := range []*apitest.Client{suec, hadmin} {
			for _, c := range changes {
				if r := who.Do(c.method, c.path, c.body); r.Code != http.StatusForbidden || r.Message() == "" {
					t.Errorf("%s %s: got %s, want 403", c.method, c.path, r)
				}
			}
			refused(t, who.Do("DELETE", w.a1.Path("/option-lists/"+list.String()), nil), http.StatusForbidden)
			refused(t, who.Do("DELETE", q(draft, ""), nil), http.StatusForbidden)
		}
	})
	t.Run("another trust", func(t *testing.T) {
		bAdmin := w.anon.LoginAs(t, w.fx.TrustUser(t, w.b1.Org, "admin"))
		for _, c := range changes[1:] {
			refused(t, bAdmin.Do(c.method, c.path, c.body), http.StatusNotFound)
		}
	})
	t.Run("invalid input: refused for the role before the body is read", func(t *testing.T) {
		refused(t, suec.Do("POST", w.a1.Path("/questionnaires"), `{"name":`), http.StatusForbidden)
		refused(t, suec.Do("PUT", ch("/content"), `{"content":`), http.StatusForbidden)
	})
	if after := snapshot(t); after != before {
		t.Fatalf("refusals changed the trust's content:\n%s\n%s", before, after)
	}
	t.Run("but they can see it", func(t *testing.T) {
		suec.Do("GET", q(draft, ""), nil).Want(t, http.StatusOK)
		suec.Do("GET", ch(""), nil).Want(t, http.StatusOK)
		suec.Do("GET", w.a1.Path("/option-lists"), nil).Want(t, http.StatusOK)
	})
	t.Run("and they change their hospital's own content", func(t *testing.T) {
		own := ownedBy(t, w, w.fx.DraftHQ(t, w.a1, author, "basic.json"), w.a1)
		// The trust has published both basic.json and workflow.json, whose q_smokes differ: a draft
		// holding q_smokes can't satisfy both option locks (S23), so this one asks nothing yet.
		if _, err := w.anon.DB.Exec(t.Context(), "UPDATE chapters SET content = '{}' WHERE version_id = $1", own.VersionID); err != nil {
			t.Fatal(err)
		}
		suec.Do("POST", q(own, "/chapters"), map[string]string{"name": "Ours"}).Want(t, http.StatusCreated)
		suec.Do("POST", q(own, "/publish"), map[string]any{"versionId": own.VersionID}).Want(t, http.StatusNoContent)
	})
	t.Run("happy path: a whole-trust super clinician changes the trust's content", func(t *testing.T) {
		for _, c := range changes {
			if r := w.superA.Do(c.method, c.path, c.body); r.Code != c.ok {
				t.Errorf("%s %s: got %s, want %d", c.method, c.path, r, c.ok)
			}
		}
		if n := count(t, w.anon.DB, "SELECT count(*) FROM questionnaires WHERE name = 'New' AND org_id = $1 AND hospital_id IS NULL", w.a1.Org); n != 1 {
			t.Errorf("the new questionnaire isn't the trust's: %d", n)
		}
	})
	t.Run("happy path: deleting a draft takes a whole-trust admin, or its creator", func(t *testing.T) {
		mine := w.fx.DraftHQ(t, w.a1, author, "basic.json")
		refused(t, w.superA.Do("DELETE", q(mine, ""), nil), http.StatusForbidden)
		w.adminA.Do("DELETE", q(mine, ""), nil).Want(t, http.StatusNoContent)
		w.adminA.Do("DELETE", w.a1.Path("/option-lists/"+list.String()), nil).Want(t, http.StatusNoContent)
	})
}

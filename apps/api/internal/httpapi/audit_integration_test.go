//go:build integration

package httpapi_test

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The audit log (plan C1, session S11; STF-06). One row per request that reads or changes
// patient data, list screens included (S11 answers 1): who, when, trust, hospital, the action and
// route, and the IDs of the records touched, never patient values. Whole-trust admins read it at
// GET /api/o/{oid}/audit, as JSON or CSV. It's append-only (NFR-06), and nothing is deleted.

// auditRow is an audit_log row as the database holds it.
type auditRow struct {
	Org                  uuid.UUID
	Hospital             *uuid.UUID
	Actor, Action, Route string
	User                 *uuid.UUID
	Status               int
	Patients, Episodes   []uuid.UUID
	At                   time.Time
}

// auditRows reads the rows written after id `after`, oldest first.
func auditRows(t *testing.T, w *clinic, after int64) []auditRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT org_id, hospital_id, actor, action, route, user_id, status,
		patient_ids, episode_ids, at FROM audit_log WHERE id > $1 ORDER BY id`, after)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []auditRow
	for rows.Next() {
		var a auditRow
		if err := rows.Scan(&a.Org, &a.Hospital, &a.Actor, &a.Action, &a.Route, &a.User, &a.Status,
			&a.Patients, &a.Episodes, &a.At); err != nil {
			t.Fatal(err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// lastAuditID is the newest row's id, or 0.
func lastAuditID(t *testing.T, w *clinic) int64 {
	t.Helper()
	var id int64
	if err := w.db.QueryRow(t.Context(), `SELECT coalesce(max(id), 0) FROM audit_log`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// audits sends a request and returns the audit rows it wrote.
func audits(t *testing.T, w *clinic, request func() *apitest.Resp, want int) []auditRow {
	t.Helper()
	before := lastAuditID(t, w)
	request().Want(t, want)
	return auditRows(t, w, before)
}

// oneAudit checks a request wrote exactly one row, with this action, and returns it.
func oneAudit(t *testing.T, w *clinic, action string, request func() *apitest.Resp, want int) auditRow {
	t.Helper()
	rows := audits(t, w, request, want)
	if len(rows) != 1 {
		t.Fatalf("%s: %d audit rows, want 1: %+v", action, len(rows), rows)
	}
	if rows[0].Action != action {
		t.Errorf("action = %q, want %q", rows[0].Action, action)
	}
	return rows[0]
}

func (a auditRow) at(t *testing.T, org uuid.UUID, hospital *uuid.UUID, user *apitest.User) {
	t.Helper()
	if a.Org != org {
		t.Errorf("%s: org = %s, want %s", a.Action, a.Org, org)
	}
	if (a.Hospital == nil) != (hospital == nil) || (hospital != nil && *a.Hospital != *hospital) {
		t.Errorf("%s: hospital = %v, want %v", a.Action, a.Hospital, hospital)
	}
	switch {
	case user == nil && (a.Actor != "patient" || a.User != nil):
		t.Errorf("%s: actor %s user %v, want the patient", a.Action, a.Actor, a.User)
	case user != nil && (a.Actor != "staff" || a.User == nil || *a.User != user.ID):
		t.Errorf("%s: actor %s user %v, want %s", a.Action, a.Actor, a.User, user.Name)
	}
}

func (a auditRow) touched(t *testing.T, patients, episodes []uuid.UUID) {
	t.Helper()
	sortIDs := func(ids []uuid.UUID) []uuid.UUID {
		s := slices.Clone(ids)
		slices.SortFunc(s, func(x, y uuid.UUID) int { return strings.Compare(x.String(), y.String()) })
		return s
	}
	if !slices.Equal(sortIDs(a.Patients), sortIDs(patients)) || !slices.Equal(sortIDs(a.Episodes), sortIDs(episodes)) {
		t.Errorf("%s: patients %v episodes %v, want %v %v", a.Action, a.Patients, a.Episodes, patients, episodes)
	}
}

func ids(xs ...uuid.UUID) []uuid.UUID { return xs }

// auditWorld is the clinic plus trust A's admins and trust B's admin.
type auditWorld struct {
	*clinic
	ada, hosAdmin, beaAdmin apitest.User
	adaC, supC, hosC, beaC  *apitest.Client
}

func newAuditWorld(t *testing.T) *auditWorld {
	t.Helper()
	w := &auditWorld{clinic: newClinic(t)}
	w.ada = w.fx.TrustUser(t, w.a.Org, "admin")
	w.hosAdmin = w.fx.User(t, w.a, "admin")
	w.beaAdmin = w.fx.TrustUser(t, w.b.Org, "admin")
	w.adaC, w.hosC, w.beaC = w.anon.LoginAs(t, w.ada), w.anon.LoginAs(t, w.hosAdmin), w.anon.LoginAs(t, w.beaAdmin)
	w.supC = w.author // Alex: a whole-trust super clinician of trust A
	return w
}

// C1: audit rows are written for patient-data reads and changes, one per request.
func TestAuditRowsWritten(t *testing.T) {
	t.Parallel()
	w := newAuditWorld(t)
	about, nurse := w.hq.Chapters[0], w.hq.Chapters[2]
	a, a2 := w.a.ID, w.a2.ID
	ep, pat := w.ep.ID, w.ep.PatientID
	atA2 := w.fx.EpisodeAt(t, w.hq, w.a2, w.ada) // at the trust's other hospital

	t.Run("listing the trust's patients records every patient listed", func(t *testing.T) {
		row := oneAudit(t, w.clinic, "patient.list", func() *apitest.Resp { return w.clinician.Do("GET", at(w.a, "/patients"), nil) }, 200)
		row.at(t, w.a.Org, &a, &w.cara)
		row.touched(t, ids(pat, atA2.PatientID), nil)
		if row.Route != "GET /api/o/{oid}/h/{hid}/patients" || row.Status != 200 {
			t.Errorf("route %q status %d", row.Route, row.Status)
		}
	})
	t.Run("adding a patient records the new record", func(t *testing.T) {
		body := map[string]string{"firstName": "Audit", "lastName": "Added", "dateOfBirth": "1980-02-03", "sex": "female"}
		row := oneAudit(t, w.clinic, "patient.create", func() *apitest.Resp { return w.clinician.Do("POST", at(w.a, "/patients"), body) }, 201)
		var id uuid.UUID
		if err := w.db.QueryRow(t.Context(), `SELECT id FROM patients WHERE last_name = 'Added'`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		row.at(t, w.a.Org, &a, &w.cara)
		row.touched(t, ids(id), nil)
	})
	t.Run("a hospital's episode list records its episodes and their patients", func(t *testing.T) {
		row := oneAudit(t, w.clinic, "episode.list", func() *apitest.Resp { return w.clinician.Do("GET", at(w.a, "/episodes"), nil) }, 200)
		row.at(t, w.a.Org, &a, &w.cara)
		row.touched(t, ids(pat), ids(ep))
	})
	t.Run("All hospitals is a trust route: no hospital", func(t *testing.T) {
		row := oneAudit(t, w.clinic, "episode.list_trust", func() *apitest.Resp { return w.adaC.Do("GET", orgPath(w.a.Org, "/episodes"), nil) }, 200)
		row.at(t, w.a.Org, nil, &w.ada)
		row.touched(t, ids(pat, atA2.PatientID), ids(ep, atA2.ID))
	})
	t.Run("creating an episode records it and its patient", func(t *testing.T) {
		body := map[string]any{"patientId": pat, "versionId": w.hq.VersionID, "procedure": "Knee"}
		row := oneAudit(t, w.clinic, "episode.create", func() *apitest.Resp { return w.clinician.Do("POST", at(w.a, "/episodes"), body) }, 201)
		var id uuid.UUID
		if err := w.db.QueryRow(t.Context(), `SELECT id FROM episodes WHERE procedure = 'Knee'`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		row.touched(t, ids(pat), ids(id))
	})
	for _, c := range []struct {
		action, method, path string
		body                 any
		want                 int
	}{
		{"episode.view", "GET", "", nil, 200},
		{"answers.view", "GET", "/hq", nil, 200},
		{"episode.update", "PATCH", "", map[string]any{"procedure": "Hip"}, 204},
		{"episode.note", "POST", "/notes", map[string]string{"text": "Rang the patient"}, 201},
	} {
		t.Run("one episode: "+c.action, func(t *testing.T) {
			row := oneAudit(t, w.clinic, c.action, func() *apitest.Resp { return w.clinician.Do(c.method, epPath(w.a, w.ep, c.path), c.body) }, c.want)
			row.at(t, w.a.Org, &a, &w.cara)
			row.touched(t, ids(pat), ids(ep))
		})
	}
	t.Run("the patient's link: the patient is the actor, with the episode's trust and hospital", func(t *testing.T) {
		row := oneAudit(t, w.clinic, "link.view", func() *apitest.Resp { return w.anon.Do("GET", patientPath(w.ep, ""), nil) }, 200)
		row.at(t, w.a.Org, &a, nil)
		row.touched(t, ids(pat), ids(ep))
		if strings.Contains(row.Route, w.ep.Token) || row.Route != "GET /api/p/{token}" {
			t.Errorf("route %q: the link's token must never be stored", row.Route)
		}
		save := map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 0}
		oneAudit(t, w.clinic, "link.save", func() *apitest.Resp { return w.anon.Do("PUT", patientPath(w.ep, "/answers/"+about.String()), save) }, 200).
			touched(t, ids(pat), ids(ep))
		oneAudit(t, w.clinic, "link.submit", func() *apitest.Resp { return w.anon.Do("POST", patientPath(w.ep, "/submit"), nil) }, 204).
			at(t, w.a.Org, &a, nil)
	})
	t.Run("validating: saving answers and completing the review", func(t *testing.T) {
		save := map[string]any{"data": map[string]any{"q_nurse": "fine"}, "validated": true, "revision": 0}
		oneAudit(t, w.clinic, "answers.save", func() *apitest.Resp {
			return w.clinician.Do("PUT", epPath(w.a, w.ep, "/answers/"+nurse.String()), save)
		}, 200).
			touched(t, ids(pat), ids(ep))
		oneAudit(t, w.clinic, "review.complete", func() *apitest.Resp { return w.clinician.Do("POST", epPath(w.a, w.ep, "/complete-review"), nil) }, 204).
			at(t, w.a.Org, &a, &w.cara)
	})
	t.Run("a refusal after the membership check is recorded with its status: probing for episodes shows", func(t *testing.T) {
		row := oneAudit(t, w.clinic, "episode.view", func() *apitest.Resp {
			return w.clinician.Do("GET", at(w.a, "/episodes/"+atA2.ID.String()), nil) // A2's episode at A's URL
		}, 404)
		row.at(t, w.a.Org, &a, &w.cara)
		row.touched(t, nil, nil)
		if row.Status != 404 {
			t.Errorf("status = %d", row.Status)
		}
		oneAudit(t, w.clinic, "episode.update", func() *apitest.Resp {
			return w.clinician.Do("PATCH", epPath(w.a, w.ep, ""), map[string]any{"nurseAsa": 9})
		}, 400).touched(t, ids(pat), ids(ep))
	})
	t.Run("requests refused before the membership check, or by an unknown link, write nothing", func(t *testing.T) {
		for _, r := range []func() *apitest.Resp{
			func() *apitest.Resp { return w.anon.Do("GET", epPath(w.a, w.ep, ""), nil) },         // no sign-in
			func() *apitest.Resp { return w.site2.Do("GET", epPath(w.a, w.ep, ""), nil) },        // another hospital's clinician
			func() *apitest.Resp { return w.otherClin.Do("GET", epPath(w.a, w.ep, ""), nil) },    // another trust
			func() *apitest.Resp { return w.beaC.Do("GET", orgPath(w.a.Org, "/episodes"), nil) }, // another trust's admin
			func() *apitest.Resp { return w.anon.Do("GET", "/api/p/"+strings.Repeat("A", 43), nil) },
		} {
			before := lastAuditID(t, w.clinic)
			if resp := r(); resp.Code < 400 {
				t.Errorf("not refused: %s", resp)
			}
			if rows := auditRows(t, w.clinic, before); len(rows) != 0 {
				t.Errorf("refused request wrote %+v", rows)
			}
		}
	})
	t.Run("routes without patient data write nothing", func(t *testing.T) {
		for _, p := range []string{"/api/me", at(w.a, "/questionnaires"), at(w.a, "/published-hqs"), orgPath(w.a.Org, "/staff"), orgPath(w.a.Org, "/settings")} {
			if rows := audits(t, w.clinic, func() *apitest.Resp { return w.adaC.Do("GET", p, nil) }, 200); len(rows) != 0 {
				t.Errorf("GET %s wrote %+v", p, rows)
			}
		}
	})
	t.Run("a hospital-scoped clinician's rows are at their hospital", func(t *testing.T) {
		row := oneAudit(t, w.clinic, "episode.list", func() *apitest.Resp { return w.site2.Do("GET", at(w.a2, "/episodes"), nil) }, 200)
		if row.Org != w.a.Org || row.Hospital == nil || *row.Hospital != a2 || row.Actor != "staff" || row.User == nil {
			t.Errorf("row = %+v", row)
		}
		row.touched(t, ids(atA2.PatientID), ids(atA2.ID))
	})
}

// No audit row, no data: if the row can't be written, the request fails and nothing is sent.
func TestAuditFailsClosed(t *testing.T) {
	t.Parallel()
	w := newAuditWorld(t)
	var last string
	if err := w.db.QueryRow(t.Context(), `SELECT last_name FROM patients WHERE id = $1`, w.ep.PatientID).Scan(&last); err != nil {
		t.Fatal(err)
	}
	_, err := w.db.Exec(t.Context(), `
		CREATE FUNCTION refuse_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'disk full'; END; $$;
		CREATE TRIGGER refuse_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION refuse_audit();`)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []*apitest.Resp{
		w.clinician.Do("GET", epPath(w.a, w.ep, ""), nil),
		w.clinician.Do("GET", at(w.a, "/patients"), nil),
		w.anon.Do("GET", patientPath(w.ep, ""), nil),
	} {
		refused(t, r, http.StatusInternalServerError)
		if strings.Contains(string(r.Body), last) {
			t.Errorf("patient data sent without an audit row: %s", r)
		}
	}
}

// NFR-06: nobody can change or delete an audit row, not even through the database.
func TestAuditLogIsAppendOnly(t *testing.T) {
	t.Parallel()
	w := newAuditWorld(t)
	w.clinician.Do("GET", epPath(w.a, w.ep, ""), nil).Want(t, 200)
	for _, sql := range []string{
		`UPDATE audit_log SET action = 'nothing'`,
		`UPDATE audit_log SET user_id = NULL, actor = 'patient'`,
		`DELETE FROM audit_log`,
		`TRUNCATE audit_log`,
	} {
		if _, err := w.db.Exec(t.Context(), sql); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s: err = %v, want the append-only refusal", sql, err)
		}
	}
	if n := count(t, w.db, `SELECT count(*) FROM audit_log WHERE action = 'episode.view'`); n != 1 {
		t.Errorf("%d rows left", n)
	}
}

// auditList is GET /api/o/{oid}/audit's JSON.
type auditList struct {
	Rows []struct {
		ID           int64
		At           time.Time
		Actor        string
		UserID       *uuid.UUID
		UserName     string
		UserEmail    string
		HospitalID   *uuid.UUID
		HospitalName string
		Action       string
		ActionLabel  string
		Route        string
		Status       int
		Patients     []struct {
			ID             uuid.UUID
			Name           string
			HospitalNumber string
		}
		PatientCount int
		EpisodeIDs   []uuid.UUID
		EpisodeCount int
	}
	More  bool
	Limit int
	Users []struct {
		ID          uuid.UUID
		Name, Email string
	}
}

func (l auditList) actions() []string {
	var out []string
	for _, r := range l.Rows {
		out = append(out, r.Action)
	}
	return out
}

func auditPath(org uuid.UUID, query url.Values) string {
	p := orgPath(org, "/audit")
	if len(query) > 0 {
		p += "?" + query.Encode()
	}
	return p
}

func getAudit(t *testing.T, c *apitest.Client, org uuid.UUID, query url.Values) auditList {
	t.Helper()
	r := c.Do("GET", auditPath(org, query), nil)
	r.Want(t, http.StatusOK)
	var l auditList
	r.Decode(t, &l)
	return l
}

func TestListAudit(t *testing.T) {
	t.Parallel()
	w := newAuditWorld(t)
	var hospitalName, first, last string
	if err := w.db.QueryRow(t.Context(), `SELECT h.name, p.first_name, p.last_name FROM hospitals h, patients p
		WHERE h.id = $1 AND p.id = $2`, w.a.ID, w.ep.PatientID).Scan(&hospitalName, &first, &last); err != nil {
		t.Fatal(err)
	}
	if _, err := w.db.Exec(t.Context(), `UPDATE patients SET hospital_number = 'HN-AUD-1' WHERE id = $1`, w.ep.PatientID); err != nil {
		t.Fatal(err)
	}
	// Two days ago: Cara opens the episode list. Now: Cara opens the episode; the patient opens their link.
	w.anon.Clock.Add(-48 * time.Hour)
	w.clinician.Do("GET", at(w.a, "/episodes"), nil).Want(t, 200)
	w.anon.Clock.Add(48 * time.Hour)
	w.clinician.Do("GET", epPath(w.a, w.ep, ""), nil).Want(t, 200)
	w.anon.Do("GET", patientPath(w.ep, ""), nil).Want(t, 200)
	now := w.anon.Clock.Now()
	// Trust B's own row.
	theirs := w.fx.Episode(t, w.fx.PublishedHQ(t, w.b, w.bea, "basic.json"), w.bea)
	w.otherClin.Do("GET", epPath(w.b, theirs, ""), nil).Want(t, 200)

	t.Run("happy path: newest first, with names read now", func(t *testing.T) {
		l := getAudit(t, w.adaC, w.a.Org, nil)
		if got := l.actions(); !slices.Equal(got, []string{"link.view", "episode.view", "episode.list"}) {
			t.Fatalf("actions = %v", got)
		}
		view := l.Rows[1]
		if view.UserName != w.cara.Name || view.UserID == nil || *view.UserID != w.cara.ID || view.Actor != "staff" ||
			view.HospitalName != hospitalName || view.ActionLabel == "" || view.Status != 200 ||
			view.Route != "GET /api/o/{oid}/h/{hid}/episodes/{eid}" {
			t.Errorf("row = %+v", view)
		}
		if len(view.Patients) != 1 || view.Patients[0].ID != w.ep.PatientID || view.Patients[0].Name != first+" "+last ||
			view.Patients[0].HospitalNumber != "HN-AUD-1" || !slices.Equal(view.EpisodeIDs, ids(w.ep.ID)) {
			t.Errorf("records = %+v %v", view.Patients, view.EpisodeIDs)
		}
		if l.Rows[0].Actor != "patient" || l.Rows[0].UserID != nil {
			t.Errorf("link row = %+v", l.Rows[0])
		}
		if len(l.Users) != 1 || l.Users[0].ID != w.cara.ID || l.More || l.Limit != 500 {
			t.Errorf("users %+v more %v limit %d", l.Users, l.More, l.Limit)
		}
	})
	t.Run("reading the log is itself recorded, and shows next time", func(t *testing.T) {
		l := getAudit(t, w.adaC, w.a.Org, url.Values{"user": {w.ada.ID.String()}})
		if len(l.Rows) == 0 || l.Rows[0].Action != "audit.view" || l.Rows[0].HospitalID != nil || len(l.Rows[0].Patients) != 0 {
			t.Errorf("rows = %+v", l.Rows)
		}
	})
	t.Run("filter by user, and the patient's link", func(t *testing.T) {
		l := getAudit(t, w.adaC, w.a.Org, url.Values{"user": {w.cara.ID.String()}})
		if got := l.actions(); !slices.Equal(got, []string{"episode.view", "episode.list"}) {
			t.Errorf("Cara's = %v", got)
		}
		l = getAudit(t, w.adaC, w.a.Org, url.Values{"user": {"patient"}})
		if got := l.actions(); !slices.Equal(got, []string{"link.view"}) {
			t.Errorf("the patient's = %v", got)
		}
		if l := getAudit(t, w.adaC, w.a.Org, url.Values{"user": {w.bob.ID.String()}}); len(l.Rows) != 0 {
			t.Errorf("Bob (trust B) = %v", l.actions())
		}
	})
	t.Run("filter by time: from is inclusive, to is not", func(t *testing.T) {
		q := url.Values{"user": {w.cara.ID.String()}, "from": {now.Add(-time.Hour).Format(time.RFC3339)}}
		if got := getAudit(t, w.adaC, w.a.Org, q).actions(); !slices.Equal(got, []string{"episode.view"}) {
			t.Errorf("since an hour ago = %v", got)
		}
		q = url.Values{"user": {w.cara.ID.String()}, "to": {now.Add(-time.Hour).Format(time.RFC3339)}}
		if got := getAudit(t, w.adaC, w.a.Org, q).actions(); !slices.Equal(got, []string{"episode.list"}) {
			t.Errorf("before an hour ago = %v", got)
		}
		q = url.Values{"from": {now.Format(time.RFC3339Nano)}, "to": {now.Add(time.Second).Format(time.RFC3339Nano)}, "user": {"patient"}}
		if got := getAudit(t, w.adaC, w.a.Org, q).actions(); !slices.Equal(got, []string{"link.view"}) {
			t.Errorf("exactly now = %v", got)
		}
		q = url.Values{"from": {now.Add(-72 * time.Hour).Format(time.RFC3339)}, "to": {now.Add(-24 * time.Hour).Format(time.RFC3339)}}
		if got := getAudit(t, w.adaC, w.a.Org, q).actions(); !slices.Equal(got, []string{"episode.list"}) {
			t.Errorf("two days ago = %v", got)
		}
	})
	t.Run("filter by patient: name, either order, hospital number or record id", func(t *testing.T) {
		for _, q := range []string{last, strings.ToUpper(first + " " + last), last + " " + first, "hn-aud", w.ep.PatientID.String()} {
			got := getAudit(t, w.adaC, w.a.Org, url.Values{"patient": {q}, "user": {w.cara.ID.String()}}).actions()
			if !slices.Equal(got, []string{"episode.view", "episode.list"}) {
				t.Errorf("patient %q = %v", q, got)
			}
		}
		if l := getAudit(t, w.adaC, w.a.Org, url.Values{"patient": {"Nobody-Here"}}); len(l.Rows) != 0 {
			t.Errorf("unknown patient = %v", l.actions())
		}
		// Trust B's patient isn't trust A's, even by id.
		if l := getAudit(t, w.adaC, w.a.Org, url.Values{"patient": {theirs.PatientID.String()}}); len(l.Rows) != 0 {
			t.Errorf("trust B's patient = %v", l.actions())
		}
	})
	t.Run("searching the log for a patient is a read of their data: the row names the patients found", func(t *testing.T) {
		before := lastAuditID(t, w.clinic)
		getAudit(t, w.adaC, w.a.Org, url.Values{"patient": {last}})
		w.adaC.Do("GET", auditPath(w.a.Org, url.Values{"patient": {last}, "format": {"csv"}}), nil).Want(t, http.StatusOK)
		getAudit(t, w.adaC, w.a.Org, url.Values{"patient": {"Nobody-Here"}})
		rows := auditRows(t, w.clinic, before)
		if len(rows) != 3 {
			t.Fatalf("rows = %+v", rows)
		}
		rows[0].touched(t, ids(w.ep.PatientID), nil)
		rows[1].touched(t, ids(w.ep.PatientID), nil)
		if rows[1].Action != "audit.export" {
			t.Errorf("second = %s", rows[1].Action)
		}
		rows[2].touched(t, nil, nil) // nobody matched
		// A later search for the patient shows that their history was looked up.
		l := getAudit(t, w.adaC, w.a.Org, url.Values{"patient": {last}, "user": {w.ada.ID.String()}})
		// (The earlier patient searches above are there too, after these two.)
		if got := l.actions(); len(got) < 2 || !slices.Equal(got[:2], []string{"audit.export", "audit.view"}) {
			t.Errorf("Ada's lookups of the patient = %v", got)
		}
		// An unfiltered view names no patients (it would copy every name shown into the log).
		before = lastAuditID(t, w.clinic)
		getAudit(t, w.adaC, w.a.Org, nil)
		auditRows(t, w.clinic, before)[0].touched(t, nil, nil)
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", auditPath(w.a.Org, nil), nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician, a whole-trust super clinician and an admin of one hospital get 403", func(t *testing.T) {
		before := lastAuditID(t, w.clinic)
		for _, c := range []*apitest.Client{w.clinician, w.supC, w.hosC} {
			refused(t, c.Do("GET", auditPath(w.a.Org, nil), nil), http.StatusForbidden)
			refused(t, c.Do("GET", auditPath(w.a.Org, url.Values{"format": {"csv"}}), nil), http.StatusForbidden)
		}
		if rows := auditRows(t, w.clinic, before); len(rows) != 0 {
			t.Errorf("a refused read of the log wrote %+v", rows)
		}
	})
	t.Run("another trust: B's admin gets 404 for A, and sees none of A's rows in B's log", func(t *testing.T) {
		refused(t, w.beaC.Do("GET", auditPath(w.a.Org, nil), nil), http.StatusNotFound)
		l := getAudit(t, w.beaC, w.b.Org, nil)
		if got := l.actions(); !slices.Equal(got, []string{"episode.view"}) || l.Rows[0].UserID == nil || *l.Rows[0].UserID != w.bob.ID {
			t.Errorf("trust B's log = %+v", l.Rows)
		}
		for _, u := range l.Users {
			if u.ID == w.cara.ID {
				t.Errorf("trust B's users list Cara")
			}
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, q := range []url.Values{
			{"user": {"nope"}},
			{"from": {"yesterday"}},
			{"to": {"2026-13-01T00:00:00Z"}},
			{"from": {now.Format(time.RFC3339)}, "to": {now.Add(-time.Hour).Format(time.RFC3339)}},
			{"format": {"xml"}},
			{"format": {"csv"}, "tz": {"Mars/Olympus"}},
			{"format": {"csv"}, "tz": {"Local"}},
			{"patient": {strings.Repeat("x", 101)}},
		} {
			refused(t, w.adaC.Do("GET", auditPath(w.a.Org, q), nil), http.StatusBadRequest)
		}
		refused(t, w.adaC.Do("GET", "/api/o/abc/audit", nil), http.StatusNotFound)
	})
	t.Run("a refused export is logged as an export, a refused view as a view (S11 test report 2)", func(t *testing.T) {
		for _, c := range []struct {
			query  url.Values
			action string
		}{
			{url.Values{"format": {"csv"}, "patient": {strings.Repeat("x", 101)}}, "audit.export"},
			{url.Values{"format": {"csv"}, "user": {"nope"}}, "audit.export"},
			{url.Values{"patient": {strings.Repeat("x", 101)}}, "audit.view"},
		} {
			row := oneAudit(t, w.clinic, c.action, func() *apitest.Resp { return w.adaC.Do("GET", auditPath(w.a.Org, c.query), nil) }, 400)
			if row.Status != 400 {
				t.Errorf("%v: status %d", c.query, row.Status)
			}
		}
	})
}

// The list stops at its limit and says there's more; the CSV has every row.
// A list screen's row can touch every patient in the trust. The screen gets the count and the
// first ten names per row, so the page's size doesn't grow with the trust; the CSV has them all.
func TestAuditListCapsPatientsPerRow(t *testing.T) {
	t.Parallel()
	w := newAuditWorld(t)
	_, err := w.db.Exec(t.Context(), `INSERT INTO patients (org_id, first_name, last_name, date_of_birth, sex, hospital_number)
		SELECT $1, 'Many', 'Patient ' || n, '1970-01-01', 'unknown', 'HN-' || n FROM generate_series(1, 40) AS n`, w.a.Org)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 { // two lists of every patient: 41 each
		w.clinician.Do("GET", at(w.a, "/patients"), nil).Want(t, http.StatusOK)
	}
	r := w.adaC.Do("GET", auditPath(w.a.Org, nil), nil)
	r.Want(t, http.StatusOK)
	var l auditList
	r.Decode(t, &l)
	if len(l.Rows) != 2 {
		t.Fatalf("rows %d", len(l.Rows))
	}
	for _, row := range l.Rows {
		if row.PatientCount != 41 || len(row.Patients) != 10 || row.Patients[0].Name == "" {
			t.Errorf("count %d patients %d", row.PatientCount, len(row.Patients))
		}
	}
	// Episodes too: a list of 30 episodes gives 10 ids and the count.
	for range 29 {
		w.fx.Episode(t, w.hq, w.cara)
	}
	w.clinician.Do("GET", at(w.a, "/episodes"), nil).Want(t, http.StatusOK)
	var eps auditList
	w.adaC.Do("GET", auditPath(w.a.Org, url.Values{"user": {w.cara.ID.String()}}), nil).Decode(t, &eps)
	if e := eps.Rows[0]; e.Action != "episode.list" || e.EpisodeCount != 30 || len(e.EpisodeIDs) != 10 || e.PatientCount != 30 {
		t.Errorf("episode list row: %s episodes %d/%d patients %d", e.Action, len(e.EpisodeIDs), e.EpisodeCount, e.PatientCount)
	}
	if len(r.Body) > 12_000 {
		t.Errorf("the page is %d bytes for two rows: patients aren't capped", len(r.Body))
	}
	one := w.adaC.Do("GET", auditPath(w.a.Org, url.Values{"patient": {w.ep.PatientID.String()}, "user": {w.cara.ID.String()}}), nil)
	one.Want(t, http.StatusOK)
	var single auditList
	one.Decode(t, &single)
	if len(single.Rows) != 3 || single.Rows[1].PatientCount != 41 { // the episode list, then the two patient lists
		t.Errorf("filtered: %d rows", len(single.Rows))
	}

	c := w.adaC.Do("GET", auditPath(w.a.Org, url.Values{"format": {"csv"}, "user": {w.cara.ID.String()}}), nil)
	c.Want(t, http.StatusOK)
	recs, err := csv.NewReader(strings.NewReader(string(c.Body))).ReadAll()
	if err != nil || len(recs) != 4 {
		t.Fatalf("records %d err %v", len(recs), err)
	}
	if ids := strings.Split(recs[1][7], "; "); len(ids) != 30 {
		t.Errorf("the CSV's episode list row has %d episode ids, want 30", len(ids))
	}
	if names, pids := strings.Split(recs[2][5], "; "), strings.Split(recs[2][6], "; "); len(names) != 41 || len(pids) != 41 {
		t.Errorf("the CSV row has %d names and %d ids, want 41", len(names), len(pids))
	}
}

func TestAuditLimitAndCSV(t *testing.T) {
	t.Parallel()
	w := newAuditWorld(t)
	_, err := w.db.Exec(t.Context(), `INSERT INTO audit_log (at, org_id, hospital_id, actor, user_id, action, route, status, patient_ids, episode_ids)
		SELECT now() - interval '1 hour' - make_interval(secs => n), $1, $2, 'staff', $3, 'episode.view', 'GET /api/o/{oid}/h/{hid}/episodes/{eid}', 200, ARRAY[$4::uuid], ARRAY[$5::uuid]
		FROM generate_series(1, 510) AS n`, w.a.Org, w.a.ID, w.cara.ID, w.ep.PatientID, w.ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Someone whose name a spreadsheet would run as a formula (CSV injection).
	if _, err := w.db.Exec(t.Context(), `UPDATE users SET name = '=HYPERLINK("http://x")' WHERE id = $1`, w.cara.ID); err != nil {
		t.Fatal(err)
	}
	l := getAudit(t, w.adaC, w.a.Org, nil)
	if len(l.Rows) != 500 || !l.More {
		t.Errorf("rows %d more %v", len(l.Rows), l.More)
	}

	t.Run("happy path: every row, times in the browser's zone, formulas defused, and the export recorded", func(t *testing.T) {
		before := lastAuditID(t, w.clinic)
		r := w.adaC.Do("GET", auditPath(w.a.Org, url.Values{"format": {"csv"}, "tz": {"America/New_York"}}), nil)
		r.Want(t, http.StatusOK)
		if !strings.HasPrefix(r.Header("Content-Type"), "text/csv") || !strings.Contains(r.Header("Content-Disposition"), "attachment") {
			t.Errorf("headers %q %q", r.Header("Content-Type"), r.Header("Content-Disposition"))
		}
		recs, err := csv.NewReader(strings.NewReader(string(r.Body))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		// The header, the 510 rows and the earlier read of the log (written after its query).
		if len(recs) != 1+510+1 {
			t.Fatalf("%d records", len(recs))
		}
		head := strings.Join(recs[0], ",")
		if !strings.HasPrefix(head, "Time,Who,Email,Action,Hospital,Patients,Patient record IDs,Episode IDs,Route,Result") {
			t.Errorf("header %q", head)
		}
		if recs[1][3] != "Viewed the audit log" {
			t.Errorf("newest = %v", recs[1])
		}
		row := recs[2]
		if row[1] != `'=HYPERLINK("http://x")` || row[3] != "Opened an episode" || row[6] != w.ep.PatientID.String() ||
			row[7] != w.ep.ID.String() || row[9] != "200" {
			t.Errorf("row = %q", row)
		}
		at, err := time.Parse(time.RFC3339, row[0])
		if err != nil {
			t.Fatalf("time %q: %v", row[0], err)
		}
		if _, off := at.Zone(); off != -4*3600 && off != -5*3600 {
			t.Errorf("time %q is not New York's", row[0])
		}
		rows := auditRows(t, w.clinic, before)
		if len(rows) != 1 || rows[0].Action != "audit.export" || rows[0].Hospital != nil || *rows[0].User != w.ada.ID {
			t.Errorf("export rows = %+v", rows)
		}
	})
	t.Run("the CSV takes the list's filters", func(t *testing.T) {
		r := w.adaC.Do("GET", auditPath(w.a.Org, url.Values{"format": {"csv"}, "user": {"patient"}}), nil)
		r.Want(t, http.StatusOK)
		recs, err := csv.NewReader(strings.NewReader(string(r.Body))).ReadAll()
		if err != nil || len(recs) != 1 {
			t.Errorf("records %d err %v", len(recs), err)
		}
	})
	t.Run("no time zone: the deployment's", func(t *testing.T) {
		r := w.adaC.Do("GET", auditPath(w.a.Org, url.Values{"format": {"csv"}}), nil)
		r.Want(t, http.StatusOK)
		if !strings.Contains(string(r.Body), "Opened an episode") {
			t.Errorf("csv = %.200s", r.Body)
		}
	})
}

//go:build integration

package httpapi_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The patient's side (S14; A-4, PAT-04, PAT-05). Their link opens nothing by itself: Continue
// returns only the hospital's name, and their date of birth starts a session (a cookie). The
// session then opens their own episodes' HQs. So the five cases become:
//   - happy path: the link, then the date of birth, then the HQ;
//   - no sign-in: no session cookie → 401, and the link alone returns no data;
//   - wrong role: a signed-in clinician without a patient session gets nothing (401);
//   - another patient or trust: a session opens only its own patient's episodes (404);
//   - invalid input: a bad, unknown, replaced or locked link, and a date that isn't one.

func linkPath(e apitest.Episode, rest string) string { return "/api/p/links/" + e.Link + rest }

func patientPath(e apitest.Episode, rest string) string {
	return "/api/p/episodes/" + e.ID.String() + rest
}

func dob(d string) map[string]string { return map[string]string{"dateOfBirth": d} }

// patientAnswers is the patient's answers row for a Question Set, or "" if there isn't one.
func patientAnswers(t *testing.T, w *clinic, e apitest.Episode, chapter uuid.UUID) string {
	t.Helper()
	var data string
	err := w.db.QueryRow(t.Context(), `SELECT data::text FROM episode_answers
		WHERE episode_id = $1 AND chapter_id = $2 AND actor = 'patient'`, e.ID, chapter).Scan(&data)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return data
}

// linkRow is the part of a login_links row the tests check.
type linkRow struct {
	Attempts int
	Locked   *time.Time
	Used     *time.Time
}

func readLink(t *testing.T, w *clinic, token string) linkRow {
	t.Helper()
	var l linkRow
	hash := sha256.Sum256([]byte(token))
	if err := w.db.QueryRow(t.Context(), `SELECT failed_dob_attempts, locked_at, used_at FROM login_links WHERE token_hash = $1`,
		hash[:]).Scan(&l.Attempts, &l.Locked, &l.Used); err != nil {
		t.Fatal(err)
	}
	return l
}

func TestPatientLinkContinue(t *testing.T) {
	t.Parallel()
	w := newClinic(t)

	t.Run("happy path: only the hospital's name, and nothing is used up", func(t *testing.T) {
		r := w.anon.Do("POST", linkPath(w.ep, "/continue"), nil)
		r.Want(t, http.StatusOK)
		var got map[string]any
		r.Decode(t, &got)
		var hospital string
		if err := w.db.QueryRow(t.Context(), `SELECT name FROM hospitals WHERE id = $1`, w.a.ID).Scan(&hospital); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got["hospital"] != hospital {
			t.Errorf("got %s, want only the hospital's name", r.Body)
		}
		if r.Header("Set-Cookie") != "" {
			t.Errorf("Continue started a session: %s", r.Header("Set-Cookie"))
		}
		w.anon.Do("POST", linkPath(w.ep, "/continue"), nil).Want(t, http.StatusOK) // again: still works
		if l := readLink(t, w, w.ep.Link); l.Used != nil || l.Attempts != 0 {
			t.Errorf("link = %+v", l)
		}
	})
	t.Run("no sign-in: the link alone returns no data", func(t *testing.T) {
		for _, method := range []string{"GET", "HEAD"} {
			r := w.anon.Do(method, "/api/p/links/"+w.ep.Link, nil)
			if r.Code < 400 {
				t.Errorf("%s the link: %s", method, r)
			}
		}
		r := w.anon.Do("GET", "/api/p/links/"+w.ep.Link+"/continue", nil) // a scanner following the link
		if r.Code < 400 || strings.Contains(string(r.Body), "Test") {
			t.Errorf("GET continue: %s", r)
		}
	})
	t.Run("invalid input: a bad, unknown or replaced link is 404", func(t *testing.T) {
		refused(t, w.anon.Do("POST", "/api/p/links/short/continue", nil), http.StatusNotFound)
		refused(t, w.anon.Do("POST", "/api/p/links/"+strings.Repeat("A", 43)+"/continue", nil), http.StatusNotFound)
		refused(t, w.anon.Do("POST", "/api/p/links/"+strings.Repeat("!", 43)+"/continue", nil), http.StatusNotFound)
		replaced := w.fx.Episode(t, w.hq, w.cara)
		if _, err := w.db.Exec(t.Context(), `UPDATE login_links SET revoked_at = now() WHERE episode_id = $1`, replaced.ID); err != nil {
			t.Fatal(err)
		}
		refused(t, w.anon.Do("POST", linkPath(replaced, "/continue"), nil), http.StatusNotFound)
	})
	t.Run("a locked link says so", func(t *testing.T) {
		locked := w.fx.Episode(t, w.hq, w.cara)
		if _, err := w.db.Exec(t.Context(), `UPDATE login_links SET locked_at = now(), failed_dob_attempts = 5 WHERE episode_id = $1`, locked.ID); err != nil {
			t.Fatal(err)
		}
		r := w.anon.Do("POST", linkPath(locked, "/continue"), nil)
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(string(r.Body), `"code":"locked"`) || !strings.Contains(r.Message(), "contact the hospital") {
			t.Errorf("got %s", r)
		}
	})
}

func TestPatientLinkDob(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	const wrong = "1971-02-03"

	t.Run("invalid input isn't a try", func(t *testing.T) {
		for _, body := range []any{dob(""), dob("1970-13-01"), dob("01/01/1970"), dob("1970-02-30"), `{"dateOfBirth":`, map[string]int{"dateOfBirth": 1}} {
			refused(t, w.anon.Do("POST", linkPath(w.ep, "/dob"), body), http.StatusBadRequest)
		}
		refused(t, w.anon.Do("POST", "/api/p/links/"+strings.Repeat("A", 43)+"/dob", dob(w.ep.DateOfBirth)), http.StatusNotFound)
		if l := readLink(t, w, w.ep.Link); l.Attempts != 0 {
			t.Errorf("attempts = %d", l.Attempts)
		}
	})
	t.Run("happy path: four wrong dates, then the right one: a session, and the count starts again", func(t *testing.T) {
		p := w.anon.NewBrowser(t)
		for i := 1; i <= 4; i++ {
			r := p.Do("POST", linkPath(w.ep, "/dob"), dob(wrong))
			refused(t, r, http.StatusUnauthorized)
			if r.Header("Set-Cookie") != "" || strings.Contains(string(r.Body), w.ep.DateOfBirth) {
				t.Errorf("try %d: %s %s", i, r, r.Header("Set-Cookie"))
			}
			if left := 5 - i; !strings.Contains(r.Message(), "You have "+string(rune('0'+left))) {
				t.Errorf("try %d: message %q doesn't say %d left", i, r.Message(), left)
			}
		}
		if l := readLink(t, w, w.ep.Link); l.Attempts != 4 || l.Locked != nil {
			t.Fatalf("link = %+v", l)
		}
		r := p.Do("POST", linkPath(w.ep, "/dob"), dob(w.ep.DateOfBirth))
		r.Want(t, http.StatusOK)
		var got map[string]any
		r.Decode(t, &got)
		if len(got) != 1 || got["episodeId"] != w.ep.ID.String() {
			t.Errorf("got %s, want the episode's id only", r.Body)
		}
		cookie := r.Header("Set-Cookie")
		for _, want := range []string{"poised_patient=", "Path=/api/p", "HttpOnly", "SameSite=Strict", "Max-Age=1800"} {
			if !strings.Contains(cookie, want) {
				t.Errorf("cookie %q lacks %q", cookie, want)
			}
		}
		if l := readLink(t, w, w.ep.Link); l.Attempts != 0 || l.Used == nil || l.Locked != nil {
			t.Errorf("link = %+v", l)
		}
		p.Do("GET", patientPath(w.ep, ""), nil).Want(t, http.StatusOK)
		// The invite link keeps working (PAT-05): the same SMS again, days later.
		w.anon.AsPatient(t, w.ep.Link, w.ep.DateOfBirth)
	})
	t.Run("the fifth wrong date of birth locks the link, even for the right one after", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		for i := 1; i <= 4; i++ {
			refused(t, w.anon.Do("POST", linkPath(e, "/dob"), dob(wrong)), http.StatusUnauthorized)
		}
		r := w.anon.Do("POST", linkPath(e, "/dob"), dob(wrong))
		refused(t, r, http.StatusForbidden)
		if !strings.Contains(string(r.Body), `"code":"locked"`) {
			t.Errorf("5th: %s", r)
		}
		if l := readLink(t, w, e.Link); l.Attempts != 5 || l.Locked == nil {
			t.Errorf("link = %+v", l)
		}
		r = w.anon.Do("POST", linkPath(e, "/dob"), dob(e.DateOfBirth))
		refused(t, r, http.StatusForbidden)
		if r.Header("Set-Cookie") != "" {
			t.Error("a locked link started a session")
		}
		refused(t, w.anon.Do("POST", linkPath(e, "/continue"), nil), http.StatusForbidden)
		if l := readLink(t, w, e.Link); l.Attempts != 5 {
			t.Errorf("attempts = %d after the lock", l.Attempts)
		}
	})
	t.Run("another episode's link isn't touched", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, w.anon.Do("POST", linkPath(w.ep, "/dob"), dob(wrong)), http.StatusUnauthorized)
		if l := readLink(t, w, e.Link); l.Attempts != 0 {
			t.Errorf("attempts = %d", l.Attempts)
		}
	})
}

func TestPatientSession(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	mine := w.fx.Episode(t, w.hq, w.cara)

	t.Run("no sign-in: no session, no data", func(t *testing.T) {
		refused(t, w.anon.Do("GET", patientPath(mine, ""), nil), http.StatusUnauthorized)
		refused(t, w.anon.Do("PUT", patientPath(mine, "/answers/"+w.hq.Chapters[0].String()), map[string]any{"data": map[string]any{}, "revision": 0}), http.StatusUnauthorized)
		refused(t, w.anon.Do("POST", patientPath(mine, "/submit"), nil), http.StatusUnauthorized)
		if e := readEpisode(t, w, mine.ID); e.Submitted != nil {
			t.Error("submitted without a session")
		}
	})
	t.Run("wrong role: a signed-in clinician has no patient session", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", patientPath(mine, ""), nil), http.StatusUnauthorized)
	})
	t.Run("another patient's episode, in this trust or another, is 404", func(t *testing.T) {
		p := w.anon.AsPatient(t, mine.Link, mine.DateOfBirth)
		theirs := w.fx.Episode(t, w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json"), w.bea)
		for _, e := range []apitest.Episode{w.ep, theirs} {
			refused(t, p.Do("GET", patientPath(e, ""), nil), http.StatusNotFound)
			refused(t, p.Do("PUT", patientPath(e, "/answers/"+w.hq.Chapters[0].String()), map[string]any{"data": map[string]any{}, "revision": 0}), http.StatusNotFound)
			refused(t, p.Do("POST", patientPath(e, "/submit"), nil), http.StatusNotFound)
		}
		refused(t, p.Do("GET", "/api/p/episodes/abc", nil), http.StatusNotFound)
		refused(t, p.Do("GET", "/api/p/episodes/"+uuid.NewString(), nil), http.StatusNotFound)
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers`); n != 0 {
			t.Errorf("%d answer rows written", n)
		}
	})
	t.Run("the same patient's other episode opens with the same session", func(t *testing.T) {
		second := w.fx.EpisodeOf(t, w.hq, w.a, mine.PatientID, w.cara, "")
		w.anon.AsPatient(t, mine.Link, mine.DateOfBirth).Do("GET", patientPath(second, ""), nil).Want(t, http.StatusOK)
	})
	t.Run("the idle timeout: 30 minutes after the last request", func(t *testing.T) {
		p := w.anon.AsPatient(t, mine.Link, mine.DateOfBirth)
		for range 3 { // a request every 20 minutes keeps it open, an hour in all
			w.anon.Clock.Add(20 * time.Minute)
			p.Do("GET", patientPath(mine, ""), nil).Want(t, http.StatusOK)
		}
		w.anon.Clock.Add(31 * time.Minute)
		r := p.Do("GET", patientPath(mine, ""), nil)
		refused(t, r, http.StatusUnauthorized)
		if !strings.Contains(r.Message(), "open the link") {
			t.Errorf("message %q", r.Message())
		}
	})
	t.Run("signing out ends the session", func(t *testing.T) {
		p := w.anon.AsPatient(t, mine.Link, mine.DateOfBirth)
		p.Do("POST", "/api/p/logout", nil).Want(t, http.StatusNoContent)
		refused(t, p.Do("GET", patientPath(mine, ""), nil), http.StatusUnauthorized)
	})
	t.Run("a forged or staff cookie isn't a session", func(t *testing.T) {
		// Staff's sign-in cookie has another name and audience; a session for another secret fails.
		refused(t, w.clinician.Do("GET", patientPath(mine, ""), nil), http.StatusUnauthorized)
	})
}

func TestPatientHQ(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	saveAnswers(t, w, w.ep, about, nil, `{"q_smokes": "o_yes"}`)
	saveAnswers(t, w, w.ep, nurse, &w.cara, `{"q_secret_bp": "SECRET 120/80"}`)
	other := w.fx.Episode(t, w.hq, w.cara)
	saveAnswers(t, w, other, health, nil, `{"q_diabetes": true}`)
	if _, err := w.db.Exec(t.Context(), `UPDATE episode_answers SET revision = 2 WHERE episode_id = $1`, w.ep.ID); err != nil {
		t.Fatal(err)
	}
	// The patient's record holds identifiers they never type here (S14 answers 1).
	if _, err := w.db.Exec(t.Context(), `UPDATE patients SET national_id = '9434765919', national_id_type = 'nhs',
		mobile = '+447700900222', email = 'sam@example.org' WHERE id = $1`, w.ep.PatientID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET hospital_number = 'HN-777' WHERE id = $1`, w.ep.ID); err != nil {
		t.Fatal(err)
	}
	p := w.anon.AsPatient(t, w.ep.Link, w.ep.DateOfBirth)

	type hq struct {
		Patient struct {
			FirstName, Name, Sex string
			Age                  int
		}
		Hospital          string
		QuestionnaireName string
		Submitted         bool
		Chapters          []struct {
			ID      uuid.UUID
			Name    string
			Content json.RawMessage
		}
		Answers   map[uuid.UUID]json.RawMessage
		Revisions map[uuid.UUID]int
	}
	check := func(t *testing.T, r *apitest.Resp) hq {
		t.Helper()
		r.Want(t, http.StatusOK)
		// Nothing for clinicians reaches the patient (VEW-01): no clinician-only element, page or
		// set, no disclosure (clinicalOutputs), no test case, and no clinician answers. And none of
		// the identifiers the patient didn't type: not even the date of birth they did (S14 answers 1).
		for _, secret := range []string{"SECRET", "clinicianOnly", "clinicalOutputs", "testCases", "q_secret", "q_on_secret_page",
			"1970-01-01", "dateOfBirth", "9434765919", "nationalId", "+447700900222", "sam@example.org", "HN-777", w.ep.PatientID.String()} {
			if strings.Contains(string(r.Body), secret) {
				t.Errorf("the patient JSON has %q: %s", secret, r.Body)
			}
		}
		var got hq
		r.Decode(t, &got)
		return got
	}

	t.Run("happy path: the patient's Question Sets, stripped, and their answers", func(t *testing.T) {
		got := check(t, p.Do("GET", patientPath(w.ep, ""), nil))
		if len(got.Chapters) != 2 || got.Chapters[0].ID != about || got.Chapters[1].ID != health {
			t.Fatalf("chapters = %+v, want the 2 patient sets in order", got.Chapters)
		}
		content := string(got.Chapters[0].Content)
		for _, kept := range []string{"q_smokes", "g_heart", "q_heart", "q_meds", "q_med_name", "Yes"} {
			if !strings.Contains(content, kept) {
				t.Errorf("patient content lost %q: %s", kept, content)
			}
		}
		if got.QuestionnaireName != w.hq.Name || got.Submitted || got.Hospital == "" {
			t.Errorf("hq = %+v", got)
		}
		// The survey's variables, worked out here: {patientName}, {patientAge}, {patientSex}.
		wantAge := w.anon.Clock.Now().Year() - 1970
		if got.Patient.FirstName != "Test" || !strings.HasPrefix(got.Patient.Name, "Test Patient ") ||
			got.Patient.Age < wantAge-1 || got.Patient.Age > wantAge || got.Patient.Sex != "unknown" {
			t.Errorf("patient = %+v", got.Patient)
		}
		if len(got.Answers) != 1 || string(got.Answers[about]) != `{"q_smokes":"o_yes"}` {
			t.Errorf("answers = %s, want only the patient's own answer", got.Answers)
		}
		// Each saved set's revision, which the next save sends back (S02a). A set with no
		// answers yet has none: its first save sends 0.
		if len(got.Revisions) != 1 || got.Revisions[about] != 2 {
			t.Errorf("revisions = %v, want only the patient's own set, at 2", got.Revisions)
		}
	})
	t.Run("another episode of theirs: its own answers only", func(t *testing.T) {
		mine := w.fx.EpisodeOf(t, w.hq, w.a, w.ep.PatientID, w.cara, "")
		got := check(t, p.Do("GET", patientPath(mine, ""), nil))
		if len(got.Answers) != 0 {
			t.Errorf("answers = %s, want none", got.Answers)
		}
	})
}

func TestAgeOnTheHospitalsDay(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	// Born 2000-06-15. At 23:30 UTC on 14 June 2026 it's already the 15th in Auckland: 26 there,
	// 25 at a hospital on UTC.
	if _, err := w.db.Exec(t.Context(), `UPDATE patients SET date_of_birth = '2000-06-15' WHERE id = $1`, w.ep.PatientID); err != nil {
		t.Fatal(err)
	}
	p := w.anon.AsPatient(t, w.ep.Link, "2000-06-15")
	w.anon.Clock.Add(time.Date(2026, 6, 14, 23, 30, 0, 0, time.UTC).Sub(w.anon.Clock.Now()))
	age := func(zone string) int {
		t.Helper()
		if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET timezone = $2 WHERE id = $1`, w.a.ID, zone); err != nil {
			t.Fatal(err)
		}
		var got struct{ Patient struct{ Age int } }
		r := p.Do("GET", patientPath(w.ep, ""), nil)
		r.Want(t, http.StatusOK)
		r.Decode(t, &got)
		return got.Patient.Age
	}
	if a := age("Pacific/Auckland"); a != 26 {
		t.Errorf("Auckland: %d", a)
	}
	if a := age("UTC"); a != 25 {
		t.Errorf("UTC: %d", a)
	}
}

func TestSavePatientAnswers(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	about, health, nurse := w.hq.Chapters[0], w.hq.Chapters[1], w.hq.Chapters[2]
	p := w.anon.AsPatient(t, w.ep.Link, w.ep.DateOfBirth)
	save := func(e apitest.Episode, chapter uuid.UUID, body any) *apitest.Resp {
		return p.Do("PUT", patientPath(e, "/answers/"+chapter.String()), body)
	}
	// Each save sends the revision it started from: 0 for a set with no answers yet.
	answers := map[string]any{"data": map[string]any{"q_smokes": "o_no", "q_heart": true}, "revision": 0}
	revisionOf := func(chapter uuid.UUID) int {
		t.Helper()
		return count(t, w.db, `SELECT coalesce(max(revision), -1) FROM episode_answers
			WHERE episode_id = $1 AND chapter_id = $2 AND actor = 'patient'`, w.ep.ID, chapter)
	}

	t.Run("invalid input", func(t *testing.T) {
		refused(t, p.Do("PUT", patientPath(w.ep, "/answers/abc"), answers), http.StatusNotFound)
		refused(t, save(w.ep, uuid.New(), answers), http.StatusNotFound)
		refused(t, save(w.ep, about, `{"data":`), http.StatusBadRequest)
		refused(t, save(w.ep, about, `{"data": ["q_smokes"], "revision": 0}`), http.StatusBadRequest)
		refused(t, save(w.ep, about, `{"data": {}}`), http.StatusBadRequest) // no revision
		refused(t, save(w.ep, about, `{"data": {}, "revision": "0"}`), http.StatusBadRequest)
		refused(t, save(w.ep, about, `{"data": {}, "revision": -1}`), http.StatusBadRequest)
	})
	t.Run("wrong role: the clinician Question Set is not the patient's", func(t *testing.T) {
		refused(t, save(w.ep, nurse, map[string]any{"data": map[string]string{"q_secret_bp": "1"}, "revision": 0}), http.StatusNotFound)
	})
	t.Run("another hospital: a Question Set of another HQ", func(t *testing.T) {
		theirs := w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")
		refused(t, save(w.ep, theirs.Chapters[0], answers), http.StatusNotFound)
	})
	if n := count(t, w.db, "SELECT count(*) FROM episode_answers"); n != 0 {
		t.Fatalf("refusals saved %d answer rows", n)
	}

	t.Run("happy path: saved at revision 1, and saved again on the next autosave", func(t *testing.T) {
		if rev := savedRevision(t, save(w.ep, about, answers)); rev != 1 {
			t.Errorf("revision = %d, want 1", rev)
		}
		if rev := savedRevision(t, save(w.ep, about, map[string]any{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 1})); rev != 2 {
			t.Errorf("revision = %d, want 2", rev)
		}
		if got := patientAnswers(t, w, w.ep, about); got != `{"q_smokes": "o_yes"}` || revisionOf(about) != 2 {
			t.Errorf("answers = %s at revision %d, want the second save at 2", got, revisionOf(about))
		}
		if n := count(t, w.db, `SELECT count(*) FROM episode_answers WHERE episode_id = $1 AND updated_by IS NOT NULL`, w.ep.ID); n != 0 {
			t.Errorf("the patient's own answers are marked as entered by staff")
		}
	})
	t.Run("the HQ open on two devices: the older one's save is refused", func(t *testing.T) {
		// Both opened the set at revision 2. The phone saves first.
		savedRevision(t, save(w.ep, about, map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 2}))
		for _, body := range []map[string]any{
			{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 2}, // the laptop, which still shows Yes
			{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 0}, // a "first" save, but there is a row
			{"data": map[string]any{"q_smokes": "o_yes"}, "revision": 9}, // newer than the row
		} {
			r := save(w.ep, about, body)
			refused(t, r, http.StatusConflict)
			if !strings.Contains(r.Message(), "changed somewhere else") {
				t.Errorf("message = %q", r.Message())
			}
		}
		if got := patientAnswers(t, w, w.ep, about); got != `{"q_smokes": "o_no"}` || revisionOf(about) != 3 {
			t.Errorf("answers = %s at revision %d, want the phone's at 3", got, revisionOf(about))
		}
		// A revision for a set with no answers yet.
		refused(t, save(w.ep, health, map[string]any{"data": map[string]bool{"q_diabetes": true}, "revision": 1}), http.StatusConflict)
		if got := patientAnswers(t, w, w.ep, health); got != "" {
			t.Errorf("a stale save made a row: %s", got)
		}
	})
	t.Run("two first saves at once: one wins, the other gets a 409", func(t *testing.T) {
		r := meanwhile(t, w.db, `INSERT INTO episode_answers (episode_id, chapter_id, actor, data, revision)
			VALUES ($1, $2, 'patient', '{"q_diabetes": true}', 1)`, []any{w.ep.ID, health},
			func() *apitest.Resp {
				return save(w.ep, health, map[string]any{"data": map[string]bool{"q_diabetes": false}, "revision": 0})
			})
		refused(t, r, http.StatusConflict)
		if got := patientAnswers(t, w, w.ep, health); got != `{"q_diabetes": true}` || revisionOf(health) != 1 {
			t.Errorf("answers = %s at revision %d, want the first save's at 1", got, revisionOf(health))
		}
	})
	t.Run("answers to questions the patient doesn't get are dropped", func(t *testing.T) {
		savedRevision(t, save(w.ep, about, map[string]any{"data": map[string]any{
			"q_smokes": "o_yes", "q_smokes-Comment": "Pipe", "q_heart": false,
			"q_secret_note": "injected", "q_secret_ecg": "injected", "q_on_secret_page": "injected", "q_made_up": 1,
		}, "revision": 3}))
		var got map[string]any
		if err := json.Unmarshal([]byte(patientAnswers(t, w, w.ep, about)), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 || got["q_smokes"] != "o_yes" || got["q_smokes-Comment"] != "Pipe" || got["q_heart"] != false {
			t.Errorf("answers = %v, want q_smokes, q_smokes-Comment and q_heart only", got)
		}
	})
	t.Run("frozen after submit", func(t *testing.T) {
		p.Do("POST", patientPath(w.ep, "/submit"), nil).Want(t, http.StatusNoContent)
		r := save(w.ep, about, map[string]any{"data": map[string]any{"q_smokes": "o_no"}, "revision": 4})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "already been sent") {
			t.Errorf("message = %q", r.Message())
		}
		refused(t, save(w.ep, health, map[string]any{"data": map[string]bool{"q_diabetes": false}, "revision": 1}), http.StatusConflict)
		if got := patientAnswers(t, w, w.ep, health); got != `{"q_diabetes": true}` {
			t.Errorf("a Question Set was saved after submit: %s", got)
		}
		if got := patientAnswers(t, w, w.ep, about); !strings.Contains(got, `"q_smokes-Comment": "Pipe"`) {
			t.Errorf("answers changed after submit: %s", got)
		}
	})
}

func TestSubmitPatientHQ(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	other := w.fx.Episode(t, w.hq, w.cara)
	p := w.anon.AsPatient(t, w.ep.Link, w.ep.DateOfBirth)
	submit := func(e apitest.Episode) *apitest.Resp { return p.Do("POST", patientPath(e, "/submit"), nil) }

	t.Run("invalid input: another patient's episode, or none, is 404", func(t *testing.T) {
		refused(t, submit(other), http.StatusNotFound)
		refused(t, p.Do("POST", "/api/p/episodes/abc/submit", nil), http.StatusNotFound)
	})
	if e := readEpisode(t, w, w.ep.ID); e.Status != "hq_not_complete" || e.Submitted != nil {
		t.Fatalf("refusals changed the episode: %+v", e)
	}
	t.Run("happy path: Ready for review, with an event from the patient", func(t *testing.T) {
		submit(w.ep).Want(t, http.StatusNoContent)
		e := readEpisode(t, w, w.ep.ID)
		if e.Status != "ready_for_review" || e.Submitted == nil {
			t.Errorf("episode = %+v, want Ready for review with the submit time", e)
		}
		ev := readEvents(t, w, w.ep.ID)
		if len(ev) != 1 || ev[0].Kind != "event" || ev[0].Text != "HQ completed by the patient" || ev[0].User != nil {
			t.Errorf("events = %+v, want 'HQ completed by the patient' with no user", ev)
		}
		if n := count(t, w.db, `SELECT count(*) FROM episodes WHERE id = $1 AND patient_submitted_by IS NULL`, w.ep.ID); n != 1 {
			t.Error("the patient's submit is marked as staff's")
		}
	})
	t.Run("another episode isn't touched", func(t *testing.T) {
		if e := readEpisode(t, w, other.ID); e.Status != "hq_not_complete" || e.Submitted != nil {
			t.Errorf("the other episode = %+v", e)
		}
	})
	t.Run("a second submit is refused and writes nothing", func(t *testing.T) {
		refused(t, submit(w.ep), http.StatusConflict)
		if ev := readEvents(t, w, w.ep.ID); len(ev) != 1 {
			t.Errorf("events = %+v, want still 1", ev)
		}
	})
	t.Run("the link still opens, as submitted", func(t *testing.T) {
		r := w.anon.AsPatient(t, w.ep.Link, w.ep.DateOfBirth).Do("GET", patientPath(w.ep, ""), nil)
		r.Want(t, http.StatusOK)
		var got struct{ Submitted bool }
		r.Decode(t, &got)
		if !got.Submitted {
			t.Error("submitted = false after submit")
		}
	})
}

// The patient's session is held in the database (S14 answers 4, Rahul 13 Oct): each one belongs
// to the invite link it was opened with, so Sign out, a Resend (a new link) or a lock ends it at
// once, even for a copy of the cookie. The cookie names the session, never the patient's record.
func TestPatientSessionEndsOnTheServer(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	const cookie, path = "poised_patient", "/api/p"
	open := func(e apitest.Episode) (*apitest.Client, string) {
		t.Helper()
		p := w.anon.AsPatient(t, e.Link, e.DateOfBirth)
		v := p.Cookie(cookie, path+"/")
		if v == "" {
			t.Fatal("no session cookie")
		}
		return p, v
	}
	sessions := func(e apitest.Episode, where string) int {
		t.Helper()
		return count(t, w.db, `SELECT count(*) FROM patient_sessions s JOIN login_links l ON l.id = s.link_id
			WHERE l.episode_id = $1 AND `+where, e.ID)
	}

	t.Run("happy path: the right date of birth starts a session row; the cookie holds no record id", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		p, v := open(e)
		if n := sessions(e, "s.ended_at IS NULL AND s.account_id = (SELECT account_id FROM patients WHERE id = l.patient_id)"); n != 1 {
			t.Fatalf("live sessions = %d", n)
		}
		for _, part := range strings.Split(v, ".")[:2] {
			if b, err := base64.RawURLEncoding.DecodeString(part); err != nil || strings.Contains(string(b), e.PatientID.String()) {
				t.Errorf("cookie part %q: %v", b, err)
			}
		}
		p.Do("GET", patientPath(e, ""), nil).Want(t, http.StatusOK)
	})
	t.Run("Sign out ends the session: a copied cookie stops working", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		p, v := open(e)
		p.Do("POST", "/api/p/logout", nil).Want(t, http.StatusNoContent)
		refused(t, w.anon.WithCookie(t, cookie, v, path).Do("GET", patientPath(e, ""), nil), http.StatusUnauthorized)
		if n := sessions(e, "s.ended_at IS NOT NULL"); n != 1 {
			t.Errorf("ended sessions = %d", n)
		}
	})
	t.Run("a Resend ends sessions opened with the old link, not those of another episode's link", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		second := w.fx.EpisodeOf(t, w.hq, w.a, e.PatientID, w.cara, "")
		p, _ := open(e)
		other, _ := open(second)
		if _, err := w.db.Exec(t.Context(), `UPDATE patients SET email = 'resend.session@example.org' WHERE id = $1`, e.PatientID); err != nil {
			t.Fatal(err)
		}
		w.clinician.Do("POST", at(w.a, "/episodes/"+e.ID.String()+"/resend-invite"), nil).Want(t, http.StatusOK)
		refused(t, p.Do("GET", patientPath(e, ""), nil), http.StatusUnauthorized)
		other.Do("GET", patientPath(second, ""), nil).Want(t, http.StatusOK)
	})
	t.Run("a lock ends sessions opened with that link", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		p, _ := open(e)
		for range 5 {
			w.anon.Do("POST", linkPath(e, "/dob"), dob("1999-09-09"))
		}
		refused(t, p.Do("GET", patientPath(e, ""), nil), http.StatusUnauthorized)
		refused(t, p.Do("PUT", patientPath(e, "/answers/"+w.hq.Chapters[0].String()), map[string]any{"data": map[string]any{}, "revision": 0}), http.StatusUnauthorized)
	})
	t.Run("idle for 30 minutes on the server, even with a cookie that hasn't expired", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		p, _ := open(e)
		// The cookie is still valid, but the server last saw the session 31 minutes ago.
		if _, err := w.db.Exec(t.Context(), `UPDATE patient_sessions s SET last_seen_at = last_seen_at - interval '31 minutes'
			FROM login_links l WHERE l.id = s.link_id AND l.episode_id = $1`, e.ID); err != nil {
			t.Fatal(err)
		}
		refused(t, p.Do("GET", patientPath(e, ""), nil), http.StatusUnauthorized)
	})
	t.Run("invalid input: a forged, unknown or garbled session is 401, and logout without one is harmless", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, w.anon.WithCookie(t, cookie, "not.a.token", path).Do("GET", patientPath(e, ""), nil), http.StatusUnauthorized)
		w.anon.Do("POST", "/api/p/logout", nil).Want(t, http.StatusNoContent)
	})
	t.Run("patient responses are never cached", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		b := w.anon.NewBrowser(t)
		for _, r := range []*apitest.Resp{
			b.Do("POST", linkPath(e, "/continue"), nil),
			b.Do("POST", linkPath(e, "/dob"), dob("1999-09-09")),
			b.Do("POST", linkPath(e, "/dob"), dob(e.DateOfBirth)),
			b.Do("GET", patientPath(e, ""), nil),
			w.anon.Do("GET", patientPath(e, ""), nil),
		} {
			if got := r.Header("Cache-Control"); got != "no-store" {
				t.Errorf("%s: Cache-Control %q", r, got)
			}
		}
	})
}

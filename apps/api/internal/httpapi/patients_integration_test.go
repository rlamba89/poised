//go:build integration

package httpapi_test

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Patient records (plan C2, S13). A record is the trust's; a hospital lists and shows only its
// own patients (at least one episode there), and creating an episode for typed details links to
// the trust's record silently, showing nothing about it (decisions.md, 11 Oct; S13 answers).

// Made-up NHS numbers with valid check digits.
const (
	nhsAda   = "9434765919"
	nhsGrace = "4010232137"
	nhsFree  = "9990008000" // nobody's, in any trust
)

// ward is the clinic plus patients at each place a search must or mustn't reach.
type ward struct {
	*clinic
	ada      uuid.UUID // Ada Lovelace: an episode at A (hospital number HN-100) and one at A2
	adaHere  apitest.Episode
	adaThere apitest.Episode
	lovelock uuid.UUID // Ada Lovelock: an episode at A, no NHS number
	grace    uuid.UUID // Grace Lovell: an episode at A2 only (HN-200 there)
	mary     uuid.UUID // Mary Lovejoy: the trust's record, no episodes anywhere
	alan     uuid.UUID // Alan Lovett in trust B, with Ada's NHS number (another trust may hold it)
}

func newWard(t *testing.T) *ward {
	t.Helper()
	w := &ward{clinic: newClinic(t)}
	w.ada = w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Ada", LastName: "Lovelace", DateOfBirth: "1985-12-10",
		NationalID: nhsAda, Mobile: "+447700900100", Email: "ada@example.test"})
	w.adaHere = w.fx.EpisodeOf(t, w.hq, w.a, w.ada, w.cara, "HN-100")
	w.adaThere = w.fx.EpisodeOf(t, w.hq, w.a2, w.ada, w.cara, "A2-7")
	w.lovelock = w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Ada", LastName: "Lovelock", DateOfBirth: "1990-01-01"})
	w.fx.EpisodeOf(t, w.hq, w.a, w.lovelock, w.cara, "")
	w.grace = w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Grace", LastName: "Lovell", DateOfBirth: "1970-02-03",
		NationalID: nhsGrace, Mobile: "+447700900200", Email: "grace@example.test"})
	w.fx.EpisodeOf(t, w.hq, w.a2, w.grace, w.cara, "HN-200")
	w.mary = w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Mary", LastName: "Lovejoy", DateOfBirth: "1960-06-06",
		Mobile: "+447700900300", Email: "mary@example.test"})
	w.alan = w.fx.Patient(t, w.b.Org, apitest.PatientRecord{FirstName: "Alan", LastName: "Lovett", DateOfBirth: "1985-12-10", NationalID: nhsAda})
	w.fx.EpisodeOf(t, w.fx.PublishedHQ(t, w.b, w.bea, "basic.json"), w.b, w.alan, w.bob, "")
	return w
}

type searchResult struct {
	Patients []map[string]any
	More     bool
}

func search(t *testing.T, c *apitest.Client, h apitest.Hospital, q, dob string) searchResult {
	t.Helper()
	r := c.Do("GET", at(h, "/patients?q="+urlQuery(q)+"&dob="+dob), nil)
	r.Want(t, http.StatusOK)
	var got searchResult
	r.Decode(t, &got)
	return got
}

func urlQuery(s string) string { return url.QueryEscape(s) }

func resultIDs(res searchResult) []uuid.UUID {
	var out []uuid.UUID
	for _, p := range res.Patients {
		out = append(out, uuid.MustParse(p["id"].(string)))
	}
	return out
}

func TestSearchPatients(t *testing.T) {
	t.Parallel()
	w := newWard(t)

	t.Run("happy path: an NHS number, as typed with spaces", func(t *testing.T) {
		res := search(t, w.clinician, w.a, "943 476 5919", "")
		if got := resultIDs(res); !slices.Equal(got, []uuid.UUID{w.ada}) || res.More {
			t.Fatalf("results = %v (more %v), want only Ada", got, res.More)
		}
		// Name, date of birth, NHS number and this hospital's number: nothing else (no contacts).
		p := res.Patients[0]
		want := map[string]any{"id": w.ada.String(), "firstName": "Ada", "lastName": "Lovelace", "dateOfBirth": "1985-12-10",
			"nationalId": nhsAda, "hospitalNumber": "HN-100"}
		if fmt.Sprint(p) != fmt.Sprint(want) {
			t.Errorf("result = %v, want exactly %v", p, want)
		}
	})
	t.Run("happy path: a hospital number here, exactly, ignoring case", func(t *testing.T) {
		if got := resultIDs(search(t, w.clinician, w.a, "hn-100", "")); !slices.Equal(got, []uuid.UUID{w.ada}) {
			t.Errorf("results = %v, want only Ada", got)
		}
		if got := resultIDs(search(t, w.clinician, w.a, "HN-10", "")); len(got) != 0 {
			t.Errorf("part of a hospital number found %v", got)
		}
	})
	t.Run("happy path: the start of a surname finds this hospital's patients only", func(t *testing.T) {
		got := resultIDs(search(t, w.clinician, w.a, "LOV", ""))
		if !slices.Equal(got, []uuid.UUID{w.ada, w.lovelock}) { // by surname: Lovelace, Lovelock
			t.Errorf("results = %v, want Ada Lovelace and Ada Lovelock; never Grace (A2 only), Mary (no episodes) or trust B's Alan", got)
		}
	})
	t.Run("happy path: a short surname with a date of birth", func(t *testing.T) {
		if got := resultIDs(search(t, w.clinician, w.a, "Lo", "1985-12-10")); !slices.Equal(got, []uuid.UUID{w.ada}) {
			t.Errorf("results = %v, want only Ada", got)
		}
		if got := resultIDs(search(t, w.clinician, w.a, "Lovelace", "1985-12-11")); len(got) != 0 {
			t.Errorf("another date of birth found %v", got)
		}
	})
	t.Run("a patient known only at the trust's other hospital is never found", func(t *testing.T) {
		for _, q := range []string{nhsGrace, "HN-200", "Lovell"} {
			if got := resultIDs(search(t, w.clinician, w.a, q, "")); slices.Contains(got, w.grace) {
				t.Errorf("%q found Grace, who has no episode at A", q)
			}
		}
		// …but A2 finds her.
		if got := resultIDs(search(t, w.site2, w.a2, nhsGrace, "")); !slices.Equal(got, []uuid.UUID{w.grace}) {
			t.Errorf("A2: results = %v, want Grace", got)
		}
	})
	t.Run("at most 20, and it says there are more", func(t *testing.T) {
		for i := range 21 {
			p := w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: fmt.Sprint("P", i), LastName: "Manyton", DateOfBirth: "2000-01-01"})
			w.fx.EpisodeOf(t, w.hq, w.a, p, w.cara, "")
		}
		res := search(t, w.clinician, w.a, "many", "")
		if len(res.Patients) != 20 || !res.More {
			t.Errorf("%d results, more %v; want 20 and more", len(res.Patients), res.More)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", at(w.a, "/patients?q=lov"), nil), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, w.site2.Do("GET", at(w.a, "/patients?q=lov"), nil), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.otherClin.Do("GET", at(w.a, "/patients?q=lov"), nil), http.StatusNotFound)
		// B's search for Ada's NHS number finds B's own Alan, never A's Ada.
		if got := resultIDs(search(t, w.otherClin, w.b, nhsAda, "")); !slices.Equal(got, []uuid.UUID{w.alan}) {
			t.Errorf("B: results = %v, want only Alan", got)
		}
	})
	t.Run("invalid input: too broad, a bad date, a bad hospital id", func(t *testing.T) {
		for _, q := range []string{"", "?q=", "?q=lo", "?q=%20%20lo%20", "?dob=1985-12-10", "?q=Lovelace&dob=10/12/1985",
			"?q=" + strings.Repeat("x", 101)} {
			refused(t, w.clinician.Do("GET", at(w.a, "/patients"+q), nil), http.StatusBadRequest)
		}
		for _, p := range badIDs(w.a, "/patients?q=lov") {
			refused(t, w.clinician.Do("GET", p, nil), http.StatusNotFound)
		}
	})
	t.Run("the surname is matched as text: % and _ are not wildcards", func(t *testing.T) {
		if got := resultIDs(search(t, w.clinician, w.a, "L%v", "")); len(got) != 0 {
			t.Errorf("results = %v, want none", got)
		}
		if got := resultIDs(search(t, w.clinician, w.a, "Lov_l", "")); len(got) != 0 {
			t.Errorf("results = %v, want none", got)
		}
	})
}

func TestGetPatient(t *testing.T) {
	t.Parallel()
	w := newWard(t)
	get := func(c *apitest.Client, h apitest.Hospital, id uuid.UUID) *apitest.Resp {
		return c.Do("GET", at(h, "/patients/"+id.String()), nil)
	}

	t.Run("happy path: the trust's record, and their episodes at this hospital only", func(t *testing.T) {
		r := get(w.clinician, w.a, w.ada)
		r.Want(t, http.StatusOK)
		var got struct {
			Patient struct {
				ID                                        uuid.UUID
				FirstName, LastName, DateOfBirth, Sex     string
				NationalID, NationalIDType, Mobile, Email string
			}
			Episodes []struct {
				ID             uuid.UUID
				HospitalNumber string
			}
		}
		r.Decode(t, &got)
		p := got.Patient
		if p.ID != w.ada || p.LastName != "Lovelace" || p.DateOfBirth != "1985-12-10" || p.NationalID != nhsAda ||
			p.NationalIDType != "nhs" || p.Mobile != "+447700900100" || p.Email != "ada@example.test" {
			t.Errorf("patient = %+v", p)
		}
		if len(got.Episodes) != 1 || got.Episodes[0].ID != w.adaHere.ID || got.Episodes[0].HospitalNumber != "HN-100" {
			t.Errorf("episodes = %+v, want only A's %s (never A2's %s)", got.Episodes, w.adaHere.ID, w.adaThere.ID)
		}
		if strings.Contains(string(r.Body), "A2-7") || strings.Contains(string(r.Body), w.adaThere.ID.String()) {
			t.Errorf("the response mentions A2's episode: %s", r.Body)
		}
	})
	t.Run("a patient with no episode here is not found, though the trust holds them", func(t *testing.T) {
		refused(t, get(w.clinician, w.a, w.grace), http.StatusNotFound)
		refused(t, get(w.clinician, w.a, w.mary), http.StatusNotFound)
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, get(w.anon, w.a, w.ada), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, get(w.site2, w.a, w.ada), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, get(w.otherClin, w.a, w.ada), http.StatusNotFound)
		refused(t, get(w.otherClin, w.b, w.ada), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/patients/abc"), nil), http.StatusNotFound)
		refused(t, get(w.clinician, w.a, uuid.New()), http.StatusNotFound)
	})
}

// newPatient is a create-episode body for typed details at the HQ.
func newPatient(hq apitest.HQ, patient map[string]any) map[string]any {
	return map[string]any{"versionId": hq.VersionID, "procedure": "Knee replacement", "hospitalNumber": " K-1 ", "patient": patient}
}

func TestCreateEpisodeForANewPatient(t *testing.T) {
	t.Parallel()
	w := newWard(t)
	typed := map[string]any{"firstName": " Sam ", "lastName": " Smith ", "dateOfBirth": "1980-05-17", "sex": "female",
		"nationalId": " 999 000 8000 ", "mobile": " 07700 900123 ", "email": " Sam@Example.TEST "}
	with := func(k string, v any) map[string]any {
		b := map[string]any{}
		for key, val := range typed {
			b[key] = val
		}
		b[k] = v
		return b
	}
	create := func(c *apitest.Client, h apitest.Hospital, body any) *apitest.Resp {
		return c.Do("POST", at(h, "/episodes"), body)
	}
	rows := func(t *testing.T) [2]int {
		return [2]int{count(t, w.db, "SELECT count(*) FROM patients"), count(t, w.db, "SELECT count(*) FROM episodes")}
	}
	before := rows(t)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, create(w.anon, w.a, newPatient(w.hq, typed)), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, create(w.site2, w.a, newPatient(w.hq, typed)), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, create(w.otherClin, w.a, newPatient(w.hq, typed)), http.StatusNotFound)
		refused(t, create(w.otherClin, w.b, newPatient(w.hq, typed)), http.StatusBadRequest) // A's HQ
	})
	t.Run("invalid input", func(t *testing.T) {
		for name, bad := range map[string]any{
			"an NHS number with a wrong check digit": newPatient(w.hq, with("nationalId", "943 476 5918")),
			"an NHS number of 9 digits":              newPatient(w.hq, with("nationalId", "943476591")),
			"not a mobile number":                    newPatient(w.hq, with("mobile", "12345")),
			"not an email address":                   newPatient(w.hq, with("email", "sam@")),
			"no first name":                          newPatient(w.hq, with("firstName", " ")),
			"no surname":                             newPatient(w.hq, with("lastName", "")),
			"no date of birth":                       newPatient(w.hq, with("dateOfBirth", "")),
			"a date of birth that isn't a date":      newPatient(w.hq, with("dateOfBirth", "17/05/1980")),
			"no sex":                                 newPatient(w.hq, with("sex", "")),
			"a hospital number that's too long": map[string]any{"versionId": w.hq.VersionID, "patient": typed,
				"hospitalNumber": strings.Repeat("9", 51)},
			"both a found patient and typed details": map[string]any{"versionId": w.hq.VersionID, "patient": typed, "patientId": w.ada},
			"neither a found patient nor details":    map[string]any{"versionId": w.hq.VersionID},
			"no HQ":                                  map[string]any{"patient": typed},
			"not JSON":                               `{"patient":`,
		} {
			r := create(w.clinician, w.a, bad)
			if r.Code != http.StatusBadRequest || r.Message() == "" {
				t.Errorf("%s: %s, want 400 with a message", name, r)
			}
		}
		// No questionnaire chosen: the page sends an empty versionId (S13 test report 1, F-1).
		for _, v := range []any{"", nil, "not-a-uuid"} {
			r := create(w.clinician, w.a, map[string]any{"patient": typed, "versionId": v})
			if r.Code != http.StatusBadRequest || r.Message() != "Choose a published HQ." {
				t.Errorf("versionId %#v: %s, want 400 \"Choose a published HQ.\"", v, r)
			}
		}
		r := create(w.clinician, w.a, newPatient(w.hq, with("nationalId", "943 476 5918")))
		if !strings.Contains(r.Message(), "NHS number") {
			t.Errorf("a bad check digit: message %q doesn't name the NHS number", r.Message())
		}
	})
	if got := rows(t); got != before {
		t.Fatalf("refusals wrote patients/episodes: %v, want %v", got, before)
	}
	t.Run("happy path: the trust's record and the episode, cleaned up; the response is the episode's id only", func(t *testing.T) {
		r := create(w.clinician, w.a, newPatient(w.hq, typed))
		r.Want(t, http.StatusCreated)
		var got map[string]any
		r.Decode(t, &got)
		if len(got) != 1 || got["id"] == nil {
			t.Fatalf("response = %v, want only the episode's id", got)
		}
		var org uuid.UUID
		var first, last, dob, sex, id, typ, mobile, email, number, procedure string
		err := w.db.QueryRow(t.Context(), `SELECT p.org_id, p.first_name, p.last_name, p.date_of_birth::text, p.sex,
			p.national_id, p.national_id_type, p.mobile, p.email, e.hospital_number, e.procedure
			FROM episodes e JOIN patients p ON p.id = e.patient_id WHERE e.id = $1`, got["id"]).Scan(
			&org, &first, &last, &dob, &sex, &id, &typ, &mobile, &email, &number, &procedure)
		if err != nil {
			t.Fatal(err)
		}
		if org != w.a.Org || first != "Sam" || last != "Smith" || dob != "1980-05-17" || sex != "female" || id != nhsFree ||
			typ != "nhs" || mobile != "+447700900123" || email != "sam@example.test" || number != "K-1" || procedure != "Knee replacement" {
			t.Errorf("row = %v %q %q %q %q %q %q %q %q %q %q", org, first, last, dob, sex, id, typ, mobile, email, number, procedure)
		}
	})
	t.Run("happy path: no NHS number, no contacts (PAT-01a)", func(t *testing.T) {
		r := create(w.clinician, w.a, newPatient(w.hq, map[string]any{"firstName": "Noel", "lastName": "Nobody",
			"dateOfBirth": "1999-09-09", "sex": "unknown"}))
		r.Want(t, http.StatusCreated)
		if n := count(t, w.db, "SELECT count(*) FROM patients WHERE last_name = 'Nobody' AND national_id IS NULL AND mobile = '' AND email = ''"); n != 1 {
			t.Errorf("%d records without an NHS number, want 1", n)
		}
	})
	t.Run("happy path: a second open episode for a found patient (PAT-02)", func(t *testing.T) {
		r := create(w.clinician, w.a, map[string]any{"patientId": w.ada, "versionId": w.hq.VersionID, "hospitalNumber": "HN-101"})
		r.Want(t, http.StatusCreated)
		if n := count(t, w.db, "SELECT count(*) FROM episodes WHERE patient_id = $1 AND hospital_id = $2", w.ada, w.a.ID); n != 2 {
			t.Errorf("Ada has %d episodes at A, want 2", n)
		}
	})
	t.Run("a found patient must be this hospital's: one known only elsewhere is refused", func(t *testing.T) {
		before := rows(t)
		for _, id := range []uuid.UUID{w.grace, w.mary, w.alan, uuid.New()} {
			refused(t, create(w.clinician, w.a, map[string]any{"patientId": id, "versionId": w.hq.VersionID}), http.StatusBadRequest)
		}
		if got := rows(t); got != before {
			t.Errorf("refusals wrote %v, want %v", got, before)
		}
	})
}

// createdFor creates an episode at A for typed details and returns the patient it was linked to.
func createdFor(t *testing.T, w *ward, typed map[string]any) (uuid.UUID, *apitest.Resp) {
	t.Helper()
	r := w.clinician.Do("POST", at(w.a, "/episodes"), newPatient(w.hq, typed))
	r.Want(t, http.StatusCreated)
	var got struct{ ID uuid.UUID }
	r.Decode(t, &got)
	var pid uuid.UUID
	if err := w.db.QueryRow(t.Context(), "SELECT patient_id FROM episodes WHERE id = $1", got.ID).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid, r
}

// noStoredValues fails if a response holds any of the trust record's values staff didn't type.
func noStoredValues(t *testing.T, r *apitest.Resp, values ...string) {
	t.Helper()
	for _, v := range values {
		if strings.Contains(string(r.Body), v) {
			t.Errorf("the response shows the stored %q: %s", v, r.Body)
		}
	}
}

func TestCreateEpisodeMatchesTheTrustsRecord(t *testing.T) {
	t.Parallel()
	w := newWard(t)
	patients := func(t *testing.T) int { return count(t, w.db, "SELECT count(*) FROM patients") }

	t.Run("the NHS number, date of birth and surname agree: linked silently to A2's patient", func(t *testing.T) {
		before := patients(t)
		pid, r := createdFor(t, w, map[string]any{"firstName": "G", "lastName": " lovell ", "dateOfBirth": "1970-02-03",
			"sex": "female", "nationalId": "401 023 2137", "mobile": "07700 900999"})
		if pid != w.grace {
			t.Errorf("linked to %s, want Grace's record %s", pid, w.grace)
		}
		if n := patients(t); n != before {
			t.Errorf("%d records, want %d: nothing new", n, before)
		}
		noStoredValues(t, r, w.grace.String(), "Grace", "+447700900200", "grace@example.test", "HN-200")
		// The record isn't changed by what was typed.
		var first, mobile string
		if err := w.db.QueryRow(t.Context(), "SELECT first_name, mobile FROM patients WHERE id = $1", w.grace).Scan(&first, &mobile); err != nil {
			t.Fatal(err)
		}
		if first != "Grace" || mobile != "+447700900200" {
			t.Errorf("the record became %q %q", first, mobile)
		}
		// A now sees Grace through their own episode: A's search finds her, by A's hospital number.
		if got := resultIDs(search(t, w.clinician, w.a, "K-1", "")); !slices.Equal(got, []uuid.UUID{w.grace}) {
			t.Errorf("A's search = %v, want Grace", got)
		}
	})
	t.Run("the NHS number with another date of birth: 409, no stored values, nothing written", func(t *testing.T) {
		before := [2]int{patients(t), count(t, w.db, "SELECT count(*) FROM episodes")}
		for _, typed := range []map[string]any{
			{"firstName": "Ada", "lastName": "Lovelace", "dateOfBirth": "1985-12-11", "sex": "female", "nationalId": nhsAda},
			{"firstName": "Ada", "lastName": "Byron", "dateOfBirth": "1985-12-10", "sex": "female", "nationalId": nhsAda},
		} {
			r := w.clinician.Do("POST", at(w.a, "/episodes"), newPatient(w.hq, typed))
			refused(t, r, http.StatusConflict)
			var body map[string]any
			r.Decode(t, &body)
			if body["code"] != "mismatch" {
				t.Errorf("code = %v, want mismatch", body["code"])
			}
			noStoredValues(t, r, w.ada.String(), "1985-12-10", "10 Dec", "Lovelace", "+447700900100", "ada@example.test", "HN-100")
		}
		if got := [2]int{patients(t), count(t, w.db, "SELECT count(*) FROM episodes")}; got != before {
			t.Errorf("wrote %v, want %v", got, before)
		}
	})
	t.Run("the same NHS number in another trust is another record", func(t *testing.T) {
		r := w.otherClin.Do("POST", at(w.b, "/episodes"), map[string]any{"versionId": w.fx.PublishedHQ(t, w.b, w.bea, "basic.json").VersionID,
			"patient": map[string]any{"firstName": "Grace", "lastName": "Lovell", "dateOfBirth": "1970-02-03", "sex": "female", "nationalId": nhsGrace}})
		r.Want(t, http.StatusCreated)
		if n := count(t, w.db, "SELECT count(*) FROM patients WHERE national_id = $1", nhsGrace); n != 2 {
			t.Errorf("%d records with Grace's NHS number, want 2 (one per trust)", n)
		}
	})
	t.Run("no NHS number: the same name, date of birth and mobile link silently", func(t *testing.T) {
		before := patients(t)
		pid, r := createdFor(t, w, map[string]any{"firstName": "mary", "lastName": "LOVEJOY", "dateOfBirth": "1960-06-06",
			"sex": "female", "mobile": "+44 7700 900300"})
		if pid != w.mary || patients(t) != before {
			t.Errorf("linked to %s, want Mary's record %s and nothing new", pid, w.mary)
		}
		noStoredValues(t, r, w.mary.String(), "mary@example.test")
	})
	t.Run("no NHS number: the same name, date of birth and email link silently", func(t *testing.T) {
		pid, _ := createdFor(t, w, map[string]any{"firstName": "Ada", "lastName": "Lovelace", "dateOfBirth": "1985-12-10",
			"sex": "female", "email": "ADA@example.test"})
		if pid != w.ada {
			t.Errorf("linked to %s, want Ada's record %s", pid, w.ada)
		}
	})
	t.Run("the same name and date of birth, other contacts, known only elsewhere: a warning naming no one", func(t *testing.T) {
		olive := w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Olive", LastName: "Elsewhere", DateOfBirth: "1975-07-07",
			NationalID: "9990000018", Mobile: "+447700900400", Email: "olive@example.test"})
		w.fx.EpisodeOf(t, w.hq, w.a2, olive, w.cara, "HN-400")
		before := patients(t)
		typed := map[string]any{"firstName": "Olive", "lastName": "Elsewhere", "dateOfBirth": "1975-07-07", "sex": "female",
			"mobile": "07700 900888"}
		r := w.clinician.Do("POST", at(w.a, "/episodes"), newPatient(w.hq, typed))
		refused(t, r, http.StatusConflict)
		var body struct {
			Code    string
			Error   string
			Matches []map[string]any
		}
		r.Decode(t, &body)
		if body.Code != "possible_duplicate" || len(body.Matches) != 0 || !strings.Contains(body.Error, "may already be registered") {
			t.Errorf("body = %+v, want a possible_duplicate warning with no matches shown", body)
		}
		noStoredValues(t, r, olive.String(), "+447700900400", "olive@example.test", "HN-400", "9990000018")
		if n := patients(t); n != before {
			t.Errorf("a warning wrote %d records", n-before)
		}
		// A record with no episodes anywhere is warned about too (here at A2).
		mabel := w.fx.Patient(t, w.a.Org, apitest.PatientRecord{FirstName: "Mabel", LastName: "Nowhere", DateOfBirth: "1950-05-05"})
		r = w.site2.Do("POST", at(w.a2, "/episodes"), newPatient(w.hq, map[string]any{"firstName": "Mabel", "lastName": "Nowhere",
			"dateOfBirth": "1950-05-05", "sex": "female"}))
		refused(t, r, http.StatusConflict)
		noStoredValues(t, r, mabel.String())

		withAnyway := newPatient(w.hq, typed)
		withAnyway["createAnyway"] = true
		w.clinician.Do("POST", at(w.a, "/episodes"), withAnyway).Want(t, http.StatusCreated)
		if n := count(t, w.db, "SELECT count(*) FROM patients WHERE last_name = 'Elsewhere' AND org_id = $1", w.a.Org); n != 2 {
			t.Errorf("%d Olive Elsewheres in trust A, want 2 after Create anyway", n)
		}
	})
	t.Run("the same name and date of birth as this hospital's patient: the match is shown, to use or not", func(t *testing.T) {
		r := w.clinician.Do("POST", at(w.a, "/episodes"), newPatient(w.hq, map[string]any{"firstName": "Ada", "lastName": "Lovelock",
			"dateOfBirth": "1990-01-01", "sex": "female", "mobile": "07700 900777"}))
		refused(t, r, http.StatusConflict)
		var body struct {
			Code    string
			Matches []map[string]any
		}
		r.Decode(t, &body)
		want := map[string]any{"id": w.lovelock.String(), "firstName": "Ada", "lastName": "Lovelock", "dateOfBirth": "1990-01-01",
			"nationalId": nil, "hospitalNumber": ""}
		if body.Code != "possible_duplicate" || len(body.Matches) != 1 || fmt.Sprint(body.Matches[0]) != fmt.Sprint(want) {
			t.Errorf("body = %+v, want Ada Lovelock shown: %v", body, want)
		}
	})
	t.Run("two people typing the same new NHS number at once: the second is refused, not duplicated", func(t *testing.T) {
		typed := map[string]any{"firstName": "Ray", "lastName": "Race", "dateOfBirth": "2001-01-01", "sex": "male", "nationalId": nhsFree}
		r := meanwhile(t, w.db, `INSERT INTO patients (org_id, first_name, last_name, date_of_birth, sex, national_id, national_id_type)
			VALUES ($1, 'Ray', 'Race', '2001-01-01', 'male', $2, 'nhs')`, []any{w.a.Org, nhsFree},
			func() *apitest.Resp { return w.clinician.Do("POST", at(w.a, "/episodes"), newPatient(w.hq, typed)) })
		refused(t, r, http.StatusConflict)
		if n := count(t, w.db, "SELECT count(*) FROM patients WHERE national_id = $1", nhsFree); n != 1 {
			t.Errorf("%d records with the NHS number, want 1", n)
		}
	})
}

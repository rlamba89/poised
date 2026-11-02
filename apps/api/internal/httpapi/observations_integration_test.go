//go:build integration

package httpapi_test

import (
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Observations (plan C7, session S46; OBS-01–03): a clinician's readings on an episode, a history
// never edited (S46 answers 2):
//
//   - GET  …/episodes/{eid}/observations: the readings, oldest first, with the kinds, measures and
//     the ranges that apply (the trust's own, else the platform's; answers 1);
//   - POST …/episodes/{eid}/observations {kind, value, value2, note, takenAt};
//   - POST …/episodes/{eid}/observations/{obsid}/correct {value, value2, note, takenAt, reason};
//   - POST …/episodes/{eid}/observations/{obsid}/entered-in-error {reason}.

const (
	obsReasonNeeded = "Say why the reading is corrected."
	obsErrorReason  = "Say why the reading was entered in error."
	obsNotCounting  = "This reading has already been corrected or marked entered in error."
	obsOff          = "Observations aren't turned on at this hospital."
)

// obsRow is an observations row.
type obsRow struct {
	ID           uuid.UUID
	Kind         string
	Value        *float64
	Value2       *float64
	Unit, Note   *string
	EnteredBy    uuid.UUID
	Corrects     *uuid.UUID
	Reason       *string
	SupersededBy *uuid.UUID
	ErrorBy      *uuid.UUID
	ErrorReason  *string
}

func obsOf(t *testing.T, w *clinic, episode uuid.UUID) []obsRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT id, kind, value, value2, unit, note, entered_by, corrects, reason,
		superseded_by, error_by, error_reason FROM observations WHERE episode_id = $1 ORDER BY entered_at, id`, episode)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []obsRow
	for rows.Next() {
		var o obsRow
		if err := rows.Scan(&o.ID, &o.Kind, &o.Value, &o.Value2, &o.Unit, &o.Note, &o.EnteredBy, &o.Corrects, &o.Reason,
			&o.SupersededBy, &o.ErrorBy, &o.ErrorReason); err != nil {
			t.Fatal(err)
		}
		out = append(out, o)
	}
	return out
}

func noObs(t *testing.T, w *clinic, e apitest.Episode) {
	t.Helper()
	if got := obsOf(t, w, e.ID); len(got) != 0 {
		t.Errorf("%d readings were saved, want none", len(got))
	}
}

func obsPath(h apitest.Hospital, e apitest.Episode, rest string) string {
	return epPath(h, e, "/observations"+rest)
}

func addObs(c *apitest.Client, h apitest.Hospital, e apitest.Episode, body any) *apitest.Resp {
	return c.Do("POST", obsPath(h, e, ""), body)
}

func reading(kind string, value float64) map[string]any {
	return map[string]any{"kind": kind, "value": value, "takenAt": time.Now().Add(-time.Hour).Format(time.RFC3339)}
}

// record adds a reading as Cara and returns its id.
func record(t *testing.T, w *clinic, e apitest.Episode, body map[string]any) uuid.UUID {
	t.Helper()
	r := addObs(w.clinician, w.a, e, body)
	r.Want(t, http.StatusCreated)
	var out struct{ ID uuid.UUID }
	r.Decode(t, &out)
	return out.ID
}

// sqlExec runs one statement on the test database.
func sqlExec(t *testing.T, w *clinic, sql string, args ...any) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestAddObservation(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	t.Run("happy path: each kind, with its stored unit", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		record(t, w, e, reading("weight", 82.5))
		bp := reading("bp", 128)
		bp["value2"] = 82
		record(t, w, e, bp)
		record(t, w, e, reading("heartRate", 72))
		record(t, w, e, map[string]any{"kind": "ecg", "note": "  Sinus rhythm  ", "takenAt": time.Now().Format(time.RFC3339)})
		got := obsOf(t, w, e.ID)
		if len(got) != 4 {
			t.Fatalf("%d rows, want 4", len(got))
		}
		if g := got[0]; g.Kind != "weight" || *g.Value != 82.5 || *g.Unit != "kg" || g.Value2 != nil || g.Note != nil || g.EnteredBy != w.cara.ID {
			t.Errorf("weight = %+v", g)
		}
		if g := got[1]; *g.Value != 128 || *g.Value2 != 82 || *g.Unit != "mmHg" {
			t.Errorf("bp = %+v", g)
		}
		if g := got[3]; g.Value != nil || g.Unit != nil || *g.Note != "Sinus rhythm" {
			t.Errorf("ecg = %+v", g)
		}
	})
	t.Run("a time before the patient's date of birth is refused, and nothing saved", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		for _, at := range []string{"0026-10-30T09:00:00Z", "1800-10-30T09:00:00Z"} {
			r := addObs(w.clinician, w.a, e, map[string]any{"kind": "weight", "value": 80, "takenAt": at})
			refused(t, r, http.StatusBadRequest)
			if !strings.Contains(r.Message(), "date of birth") {
				t.Errorf("%s: message %q", at, r.Message())
			}
		}
		noObs(t, w, e)
	})
	t.Run("an implausible value is only a warning: it's saved", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		record(t, w, e, reading("heartRate", 220))
		if got := obsOf(t, w, e.ID); len(got) != 1 {
			t.Errorf("%d rows", len(got))
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, addObs(w.anon, w.a, e, reading("weight", 80)), http.StatusUnauthorized)
		noObs(t, w, e)
	})
	t.Run("the role: a clinician at the trust's other hospital can't add one here (404)", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, addObs(w.site2, w.a, e, reading("weight", 80)), http.StatusNotFound)
		refused(t, addObs(w.site2, w.a2, e, reading("weight", 80)), http.StatusNotFound)
		noObs(t, w, e)
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		refused(t, addObs(w.otherClin, w.a, e, reading("weight", 80)), http.StatusNotFound)
		refused(t, addObs(w.otherClin, w.b, e, reading("weight", 80)), http.StatusNotFound)
		noObs(t, w, e)
	})
	t.Run("invalid input: hard limits and malformed readings are 400", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		future := reading("weight", 80)
		future["takenAt"] = time.Now().Add(2 * time.Hour).Format(time.RFC3339)
		diastolic := reading("bp", 80)
		diastolic["value2"] = 120
		for _, c := range []struct {
			body any
			want string
		}{
			{reading("weight", -3), "Weight must be from"},
			{reading("heartRate", 400), "Heart rate must be from"},
			{reading("pulse", 70), "Choose what was measured"},
			{future, "future"},
			{diastolic, "lower than the systolic"},
			{reading("bp", 120), "Enter the diastolic"},
			{map[string]any{"kind": "weight", "value": 80}, "When was it measured"},
			{map[string]any{"kind": "weight", "value": 80, "takenAt": "yesterday"}, "could not be read"},
			{map[string]any{"kind": "ecg", "note": "", "takenAt": time.Now().Format(time.RFC3339)}, "Enter the ECG"},
			{"not an object", "could not be read"},
		} {
			r := addObs(w.clinician, w.a, e, c.body)
			refused(t, r, http.StatusBadRequest)
			if !strings.Contains(r.Message(), c.want) {
				t.Errorf("%v: %q, want %q", c.body, r.Message(), c.want)
			}
		}
		noObs(t, w, e)
	})
	t.Run("the trust's own hard limits apply", func(t *testing.T) {
		w := newClinic(t) // its own trust, so its settings change nothing else
		sqlExec(t, w, `UPDATE orgs SET settings = settings || '{"observations.ranges": {"heartRate": {"softMin": 50, "softMax": 100, "hardMin": 30, "hardMax": 150}}}' WHERE id = $1`, w.a.Org)
		e := w.fx.Episode(t, w.hq, w.cara)
		r := addObs(w.clinician, w.a, e, reading("heartRate", 160))
		refused(t, r, http.StatusBadRequest)
		if !strings.Contains(r.Message(), "30 to 150") {
			t.Errorf("message %q, want the trust's limits", r.Message())
		}
		record(t, w, e, reading("heartRate", 140))
	})
	t.Run("a cancelled or archived episode takes no readings (409)", func(t *testing.T) {
		for _, status := range []string{"cancelled", "archived"} {
			e := w.fx.Episode(t, w.hq, w.cara)
			setStatus(t, w, e, status)
			refused(t, addObs(w.clinician, w.a, e, reading("weight", 80)), http.StatusConflict)
			noObs(t, w, e)
		}
	})
	t.Run("observations turned off at the hospital: 403, and the list too", func(t *testing.T) {
		w := newClinic(t)
		sqlExec(t, w, `UPDATE hospitals SET settings = settings || '{"feature.observations": false}' WHERE id = $1`, w.a.ID)
		e := w.fx.Episode(t, w.hq, w.cara)
		r := addObs(w.clinician, w.a, e, reading("weight", 80))
		refused(t, r, http.StatusForbidden)
		if r.Message() != obsOff {
			t.Errorf("message %q", r.Message())
		}
		refused(t, w.clinician.Do("GET", obsPath(w.a, e, ""), nil), http.StatusForbidden)
		noObs(t, w, e)
	})
}

// obsView is GET …/observations.
type obsView struct {
	MaxNote  int `json:"maxNote"`
	Readings []struct {
		ID            uuid.UUID
		Kind          string
		Value, Value2 *float64
		Unit, Note    *string
		EnteredByName string
		Corrects      *uuid.UUID
		Reason        *string
		SupersededBy  *uuid.UUID
		ErrorAt       *time.Time
		ErrorByName   *string
		ErrorReason   *string
	}
	Kinds    []struct{ Key, Label string }
	Measures []struct {
		Key, Label, Unit string
		Decimals         int
	}
	Ranges map[string]struct{ SoftMin, SoftMax, HardMin, HardMax float64 }
}

func TestEpisodeSaysWhetherObservationsAreOn(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	on := func() bool {
		var v struct {
			ObservationsOn *bool `json:"observationsOn"`
		}
		r := w.clinician.Do("GET", epPath(w.a, e, ""), nil)
		r.Want(t, http.StatusOK)
		r.Decode(t, &v)
		if v.ObservationsOn == nil {
			t.Fatalf("no observationsOn: %s", r)
		}
		return *v.ObservationsOn
	}
	if !on() {
		t.Error("on by default")
	}
	sqlExec(t, w, `UPDATE hospitals SET settings = settings || '{"feature.observations": false}' WHERE id = $1`, w.a.ID)
	if on() {
		t.Error("off at this hospital")
	}
}

func TestListObservations(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	first := record(t, w, e, reading("weight", 82.5))
	second := record(t, w, e, reading("height", 170))
	t.Run("happy path: readings, kinds, measures and the ranges that apply", func(t *testing.T) {
		r := w.clinician.Do("GET", obsPath(w.a, e, ""), nil)
		r.Want(t, http.StatusOK)
		var v obsView
		r.Decode(t, &v)
		if len(v.Readings) != 2 || v.Readings[0].ID != first || v.Readings[1].ID != second {
			t.Fatalf("readings = %+v", v.Readings)
		}
		if v.Readings[0].EnteredByName == "" || *v.Readings[0].Unit != "kg" {
			t.Errorf("reading = %+v", v.Readings[0])
		}
		if v.MaxNote != 2000 {
			t.Errorf("maxNote = %d", v.MaxNote)
		}
		if len(v.Kinds) != 11 || len(v.Measures) != 10 || v.Ranges["heartRate"].HardMax == 0 {
			t.Errorf("kinds %d, measures %d, ranges %+v", len(v.Kinds), len(v.Measures), v.Ranges)
		}
	})
	t.Run("another hospital of the trust: 404", func(t *testing.T) {
		refused(t, w.site2.Do("GET", obsPath(w.a, e, ""), nil), http.StatusNotFound)
		refused(t, w.site2.Do("GET", obsPath(w.a2, e, ""), nil), http.StatusNotFound)
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		refused(t, w.anon.Do("GET", obsPath(w.a, e, ""), nil), http.StatusUnauthorized)
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		refused(t, w.otherClin.Do("GET", obsPath(w.a, e, ""), nil), http.StatusNotFound)
		refused(t, w.otherClin.Do("GET", obsPath(w.b, e, ""), nil), http.StatusNotFound)
	})
	t.Run("invalid input: a malformed episode id is 404", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/not-an-id/observations"), nil), http.StatusNotFound)
	})
	t.Run("the patient's API has no such route", func(t *testing.T) {
		r := w.anon.Do("GET", "/api/p/episodes/"+e.ID.String()+"/observations", nil)
		if r.Code != http.StatusNotFound && r.Code != http.StatusUnauthorized {
			t.Errorf("patient path: %d", r.Code)
		}
	})
}

func correct(c *apitest.Client, h apitest.Hospital, e apitest.Episode, id uuid.UUID, body any) *apitest.Resp {
	return c.Do("POST", obsPath(h, e, "/"+id.String()+"/correct"), body)
}

func markObs(c *apitest.Client, h apitest.Hospital, e apitest.Episode, id uuid.UUID, body any) *apitest.Resp {
	return c.Do("POST", obsPath(h, e, "/"+id.String()+"/entered-in-error"), body)
}

func TestCorrectObservation(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	fix := map[string]any{"value": 80.5, "reason": "Scale not zeroed"}
	t.Run("happy path: a new row corrects the old, which is superseded and kept", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		old := record(t, w, e, reading("weight", 82.5))
		r := correct(w.clinician, w.a, e, old, fix)
		r.Want(t, http.StatusCreated)
		var out struct{ ID uuid.UUID }
		r.Decode(t, &out)
		got := obsOf(t, w, e.ID)
		if len(got) != 2 {
			t.Fatalf("%d rows, want 2", len(got))
		}
		if got[0].SupersededBy == nil || *got[0].SupersededBy != out.ID || *got[0].Value != 82.5 {
			t.Errorf("old = %+v", got[0])
		}
		if n := got[1]; n.ID != out.ID || n.Kind != "weight" || *n.Value != 80.5 || *n.Corrects != old || *n.Reason != "Scale not zeroed" {
			t.Errorf("new = %+v", n)
		}
		// A correction of a correction works too; the first is still kept.
		correct(w.clinician, w.a, e, out.ID, map[string]any{"value": 80.6, "reason": "Re-weighed"}).Want(t, http.StatusCreated)
		if got := obsOf(t, w, e.ID); len(got) != 3 || got[1].SupersededBy == nil {
			t.Errorf("after a second correction: %+v", got)
		}
	})
	t.Run("a blood pressure keeps both values; the time stays the original's unless given", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		bp := reading("bp", 128)
		bp["value2"] = 82
		old := record(t, w, e, bp)
		correct(w.clinician, w.a, e, old, map[string]any{"value": 132, "value2": 84, "reason": "Misread"}).Want(t, http.StatusCreated)
		var same bool
		if err := w.db.QueryRow(t.Context(), `SELECT a.taken_at = b.taken_at FROM observations a, observations b
			WHERE a.id = $1 AND b.corrects = $1`, old).Scan(&same); err != nil || !same {
			t.Errorf("the correction's time should be the original's: %v %v", same, err)
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		old := record(t, w, e, reading("weight", 82.5))
		refused(t, correct(w.anon, w.a, e, old, fix), http.StatusUnauthorized)
	})
	t.Run("the role: another hospital's clinician: 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		old := record(t, w, e, reading("weight", 82.5))
		refused(t, correct(w.site2, w.a, e, old, fix), http.StatusNotFound)
		refused(t, correct(w.site2, w.a2, e, old, fix), http.StatusNotFound)
		if len(obsOf(t, w, e.ID)) != 1 {
			t.Error("nothing should be added")
		}
	})
	t.Run("another trust's: 404, and a reading of another episode is 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		old := record(t, w, e, reading("weight", 82.5))
		refused(t, correct(w.otherClin, w.a, e, old, fix), http.StatusNotFound)
		refused(t, correct(w.otherClin, w.b, e, old, fix), http.StatusNotFound)
		other := w.fx.Episode(t, w.hq, w.cara)
		refused(t, correct(w.clinician, w.a, other, old, fix), http.StatusNotFound)
		if len(obsOf(t, w, e.ID)) != 1 || len(obsOf(t, w, other.ID)) != 0 {
			t.Error("nothing should be added")
		}
	})
	t.Run("invalid input: no reason (422), a hard limit (400), not JSON (400)", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		old := record(t, w, e, reading("weight", 82.5))
		r := correct(w.clinician, w.a, e, old, map[string]any{"value": 80.5, "reason": "  "})
		refused(t, r, http.StatusUnprocessableEntity)
		if r.Message() != obsReasonNeeded {
			t.Errorf("message %q", r.Message())
		}
		refused(t, correct(w.clinician, w.a, e, old, map[string]any{"value": -1, "reason": "x"}), http.StatusBadRequest)
		refused(t, correct(w.clinician, w.a, e, old, "x"), http.StatusBadRequest)
		refused(t, correct(w.clinician, w.a, e, old, map[string]any{"value": 80, "reason": strings.Repeat("x", 1001)}), http.StatusBadRequest)
		if len(obsOf(t, w, e.ID)) != 1 {
			t.Error("nothing should be added")
		}
	})
	t.Run("a reading already corrected or entered in error can't be corrected (409)", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		old := record(t, w, e, reading("weight", 82.5))
		correct(w.clinician, w.a, e, old, fix).Want(t, http.StatusCreated)
		r := correct(w.clinician, w.a, e, old, fix)
		refused(t, r, http.StatusConflict)
		if r.Message() != obsNotCounting {
			t.Errorf("message %q", r.Message())
		}
		wrong := record(t, w, e, reading("height", 170))
		markObs(w.clinician, w.a, e, wrong, map[string]any{"reason": "Wrong patient"}).Want(t, http.StatusNoContent)
		refused(t, correct(w.clinician, w.a, e, wrong, fix), http.StatusConflict)
		if len(obsOf(t, w, e.ID)) != 3 {
			t.Error("no further row")
		}
	})
}

func TestCorrectObservationAtTheSameMoment(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	// S46 verification, F-3: the loser is told it's already corrected (409, which reloads the
	// panel), not a server error, however the requests interleave.
	for round := range 5 {
		e := w.fx.Episode(t, w.hq, w.cara)
		old := record(t, w, e, reading("weight", 82.5))
		codes := make([]int, 6)
		var wg sync.WaitGroup
		for i := range codes {
			wg.Add(1)
			go func() {
				defer wg.Done()
				codes[i] = correct(w.clinician, w.a, e, old, map[string]any{"value": 80 + i, "reason": "Scale"}).Code
			}()
		}
		wg.Wait()
		won := 0
		for _, c := range codes {
			switch c {
			case http.StatusCreated:
				won++
			case http.StatusConflict:
			default:
				t.Errorf("round %d: %v", round, codes)
			}
		}
		if won != 1 {
			t.Errorf("round %d: %d corrections won: %v", round, won, codes)
		}
		if got := obsOf(t, w, e.ID); len(got) != 2 {
			t.Errorf("round %d: %d rows, want 2", round, len(got))
		}
	}
}

func TestMarkObservationInError(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	why := map[string]any{"reason": "Wrong patient's reading"}
	t.Run("happy path: marked, kept, with who and why", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := record(t, w, e, reading("heartRate", 72))
		markObs(w.clinician, w.a, e, id, why).Want(t, http.StatusNoContent)
		got := obsOf(t, w, e.ID)
		if len(got) != 1 || got[0].ErrorBy == nil || *got[0].ErrorBy != w.cara.ID || *got[0].ErrorReason != "Wrong patient's reading" {
			t.Errorf("row = %+v", got)
		}
		var v obsView
		r := w.clinician.Do("GET", obsPath(w.a, e, ""), nil)
		r.Want(t, http.StatusOK)
		r.Decode(t, &v)
		if v.Readings[0].ErrorAt == nil || v.Readings[0].ErrorByName == nil {
			t.Errorf("view = %+v", v.Readings[0])
		}
	})
	t.Run("no sign-in: 401", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := record(t, w, e, reading("heartRate", 72))
		refused(t, markObs(w.anon, w.a, e, id, why), http.StatusUnauthorized)
	})
	t.Run("the role: another hospital's clinician: 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := record(t, w, e, reading("heartRate", 72))
		refused(t, markObs(w.site2, w.a, e, id, why), http.StatusNotFound)
		refused(t, markObs(w.site2, w.a2, e, id, why), http.StatusNotFound)
	})
	t.Run("another trust's: 404", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := record(t, w, e, reading("heartRate", 72))
		refused(t, markObs(w.otherClin, w.a, e, id, why), http.StatusNotFound)
		refused(t, markObs(w.otherClin, w.b, e, id, why), http.StatusNotFound)
		if got := obsOf(t, w, e.ID); got[0].ErrorBy != nil {
			t.Error("it shouldn't be marked")
		}
	})
	t.Run("invalid input: no reason (422), not JSON, an unknown reading (404)", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := record(t, w, e, reading("heartRate", 72))
		r := markObs(w.clinician, w.a, e, id, map[string]any{"reason": ""})
		refused(t, r, http.StatusUnprocessableEntity)
		if r.Message() != obsErrorReason {
			t.Errorf("message %q", r.Message())
		}
		refused(t, markObs(w.clinician, w.a, e, id, "x"), http.StatusBadRequest)
		refused(t, markObs(w.clinician, w.a, e, uuid.New(), why), http.StatusNotFound)
		if got := obsOf(t, w, e.ID); got[0].ErrorBy != nil {
			t.Error("it shouldn't be marked")
		}
	})
	t.Run("twice, or after a correction: 409", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		id := record(t, w, e, reading("heartRate", 72))
		markObs(w.clinician, w.a, e, id, why).Want(t, http.StatusNoContent)
		refused(t, markObs(w.clinician, w.a, e, id, why), http.StatusConflict)
		other := record(t, w, e, reading("weight", 82))
		correct(w.clinician, w.a, e, other, map[string]any{"value": 81, "reason": "x"}).Want(t, http.StatusCreated)
		refused(t, markObs(w.clinician, w.a, e, other, why), http.StatusConflict)
	})
}

// The table itself refuses any edit but the one supersede or entered-in-error mark, and any delete
// but the episode's own (migration 00037), whatever the code asks.
func TestObservationsNeverEdited(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	id := record(t, w, e, reading("weight", 82.5))
	for _, sql := range []string{
		`UPDATE observations SET value = 70 WHERE id = $1`,
		`UPDATE observations SET taken_at = now() - interval '1 day' WHERE id = $1`,
		`DELETE FROM observations WHERE id = $1`,
	} {
		if _, err := w.db.Exec(t.Context(), sql, id); err == nil {
			t.Errorf("%s: allowed", sql)
		}
	}
	markObs(w.clinician, w.a, e, id, map[string]any{"reason": "x"}).Want(t, http.StatusNoContent)
	if _, err := w.db.Exec(t.Context(), `UPDATE observations SET error_at = NULL, error_by = NULL, error_reason = NULL WHERE id = $1`, id); err == nil {
		t.Error("unmarking was allowed")
	}
	sqlExec(t, w, `DELETE FROM login_links WHERE episode_id = $1`, e.ID) // no cascade there
	sqlExec(t, w, `DELETE FROM episodes WHERE id = $1`, e.ID)            // the episode's own deletion takes them
	if len(obsOf(t, w, e.ID)) != 0 {
		t.Error("the episode's readings should go with it")
	}
}

// Each route writes one audit row naming the episode (S11): readings are patient data.
func TestObservationsAudited(t *testing.T) {
	t.Parallel()
	w := newClinic(t)
	e := w.fx.Episode(t, w.hq, w.cara)
	for _, c := range []struct {
		action string
		req    func() *apitest.Resp
		want   int
	}{
		{"observation.add", func() *apitest.Resp { return addObs(w.clinician, w.a, e, reading("weight", 82)) }, http.StatusCreated},
		{"observation.list", func() *apitest.Resp { return w.clinician.Do("GET", obsPath(w.a, e, ""), nil) }, http.StatusOK},
		{"observation.correct", func() *apitest.Resp {
			return correct(w.clinician, w.a, e, obsOf(t, w, e.ID)[0].ID, map[string]any{"value": 81, "reason": "x"})
		}, http.StatusCreated},
		{"observation.error", func() *apitest.Resp {
			return markObs(w.clinician, w.a, e, obsOf(t, w, e.ID)[1].ID, map[string]any{"reason": "x"})
		}, http.StatusNoContent},
	} {
		a := oneAudit(t, w, c.action, c.req, c.want)
		if !slices.Contains(a.Episodes, e.ID) {
			t.Errorf("%s: audit = %+v", c.action, a)
		}
	}
}

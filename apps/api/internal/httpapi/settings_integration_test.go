//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Trust settings (plan C1, session S10; ORG-04, ORG-05). A trust's whole-trust admins set its
// settings and each hospital's overrides at /api/o/{oid}/settings. Trust-only settings can't be
// set for a hospital, and the opt-ins are the platform admin's (PRM-08).

type settingsHospital struct {
	ID        uuid.UUID
	Name      string
	Status    string
	Values    map[string]json.RawMessage
	Effective map[string]json.RawMessage
}

type settingsView struct {
	Defs []struct {
		Key, Label, Group, Level, Kind string
		OptIn                          bool
	}
	Trust          map[string]json.RawMessage
	Effective      map[string]json.RawMessage
	Hospitals      []settingsHospital
	Questionnaires []struct {
		ID   uuid.UUID
		Name string
	}
}

func (v *settingsView) hospital(id uuid.UUID) *settingsHospital {
	for i := range v.Hospitals {
		if v.Hospitals[i].ID == id {
			return &v.Hospitals[i]
		}
	}
	return nil
}

type settingsWorld struct {
	c                  *apitest.Client
	fx                 *apitest.Fixtures
	a1, a2, b1         apitest.Hospital
	ada, sup, hosAdmin apitest.User
	adaC, supC, hosC   *apitest.Client
	beaC               *apitest.Client
}

func newSettingsWorld(t *testing.T) *settingsWorld {
	t.Helper()
	c, fx := apitest.New(t)
	w := &settingsWorld{c: c, fx: fx, a1: fx.Hospital(t), b1: fx.Hospital(t)}
	w.a2 = fx.HospitalIn(t, w.a1.Org)
	w.ada = fx.TrustUser(t, w.a1.Org, "admin")
	w.sup = fx.TrustUser(t, w.a1.Org, "super_clinician")
	w.hosAdmin = fx.User(t, w.a1, "admin")
	w.adaC, w.supC, w.hosC = c.LoginAs(t, w.ada), c.LoginAs(t, w.sup), c.LoginAs(t, w.hosAdmin)
	w.beaC = c.LoginAs(t, fx.TrustUser(t, w.b1.Org, "admin"))
	return w
}

func settingsPath(org uuid.UUID) string { return orgPath(org, "/settings") }

func getSettings(t *testing.T, c *apitest.Client, org uuid.UUID) settingsView {
	t.Helper()
	r := c.Do("GET", settingsPath(org), nil)
	r.Want(t, http.StatusOK)
	var v settingsView
	r.Decode(t, &v)
	return v
}

func patchSettings(t *testing.T, w *settingsWorld, hospital any, values map[string]any, want int) (settingsView, string) {
	t.Helper()
	r := w.adaC.Do("PATCH", settingsPath(w.a1.Org), map[string]any{"hospitalId": hospital, "values": values})
	r.Want(t, want)
	var v settingsView
	if want == http.StatusOK {
		r.Decode(t, &v)
	}
	return v, r.Message()
}

func TestGetTrustSettings(t *testing.T) {
	w := newSettingsWorld(t)
	hq := w.fx.PublishedHQ(t, w.a1, w.ada, "basic.json")
	w.fx.DraftHQ(t, w.a1, w.ada, "basic.json") // a draft isn't a choice for the default HQ

	t.Run("happy path: every setting, the trust's values, and each hospital's", func(t *testing.T) {
		v := getSettings(t, w.adaC, w.a1.Org)
		if len(v.Defs) < 20 || len(v.Trust) != 0 || string(v.Effective["retentionYears"]) != "8" {
			t.Errorf("defs %d trust %v effective %v", len(v.Defs), v.Trust, v.Effective["retentionYears"])
		}
		if len(v.Hospitals) != 2 || v.hospital(w.a1.ID) == nil || v.hospital(w.a2.ID) == nil || v.hospital(w.b1.ID) != nil {
			t.Errorf("hospitals %+v", v.Hospitals)
		}
		if string(v.hospital(w.a1.ID).Effective["reminders.days"]) != "[2,5]" {
			t.Errorf("a1 %+v", v.hospital(w.a1.ID).Effective)
		}
		if len(v.Questionnaires) != 1 || v.Questionnaires[0].ID != hq.ID {
			t.Errorf("questionnaires %+v", v.Questionnaires)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("GET", settingsPath(w.a1.Org), nil).Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role: a whole-trust super clinician and an admin of one hospital get 403", func(t *testing.T) {
		w.supC.Do("GET", settingsPath(w.a1.Org), nil).Want(t, http.StatusForbidden)
		w.hosC.Do("GET", settingsPath(w.a1.Org), nil).Want(t, http.StatusForbidden)
	})
	t.Run("another trust's admin gets 404", func(t *testing.T) {
		w.beaC.Do("GET", settingsPath(w.a1.Org), nil).Want(t, http.StatusNotFound)
	})
	t.Run("a malformed trust id gets 404", func(t *testing.T) {
		w.adaC.Do("GET", "/api/o/nope/settings", nil).Want(t, http.StatusNotFound)
	})
}

func TestPatchTrustSettings(t *testing.T) {
	w := newSettingsWorld(t)
	hq := w.fx.PublishedHQ(t, w.a1, w.ada, "basic.json")
	otherHQ := w.fx.PublishedHQ(t, w.b1, w.ada, "basic.json")

	t.Run("happy path: the trust's values, which both hospitals follow", func(t *testing.T) {
		v, _ := patchSettings(t, w, nil, map[string]any{
			"retentionYears": 10, "contact.phone": " 0100 000 0000 ", "reminders.days": []int{5, 1},
			"defaultHq": hq.ID.String(), "templates.inviteSms": "Hi {{firstName}}: {{link}}",
		}, http.StatusOK)
		if string(v.Trust["retentionYears"]) != "10" || string(v.Trust["contact.phone"]) != `"0100 000 0000"` ||
			string(v.Trust["reminders.days"]) != "[1,5]" || string(v.Effective["defaultHq"]) != `"`+hq.ID.String()+`"` {
			t.Errorf("trust %v", v.Trust)
		}
		for _, h := range []uuid.UUID{w.a1.ID, w.a2.ID} {
			if string(v.hospital(h).Effective["contact.phone"]) != `"0100 000 0000"` {
				t.Errorf("%s doesn't follow the trust: %v", h, v.hospital(h).Effective["contact.phone"])
			}
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND settings->>'retentionYears' = '10'`, w.a1.Org); n != 1 {
			t.Error("not stored")
		}
	})
	t.Run("a hospital's own value; the other hospital still follows the trust", func(t *testing.T) {
		v, _ := patchSettings(t, w, w.a2.ID, map[string]any{"contact.phone": "0200 000 0000", "whatHappensNext": "See you soon."}, http.StatusOK)
		if string(v.hospital(w.a2.ID).Effective["contact.phone"]) != `"0200 000 0000"` ||
			string(v.hospital(w.a1.ID).Effective["contact.phone"]) != `"0100 000 0000"` ||
			string(v.hospital(w.a2.ID).Values["whatHappensNext"]) != `"See you soon."` {
			t.Errorf("a1 %v / a2 %v", v.hospital(w.a1.ID).Effective["contact.phone"], v.hospital(w.a2.ID).Values)
		}
		if string(v.Trust["contact.phone"]) != `"0100 000 0000"` {
			t.Error("the trust's value changed")
		}
	})
	t.Run("null puts the hospital back to following the trust", func(t *testing.T) {
		v, _ := patchSettings(t, w, w.a2.ID, map[string]any{"contact.phone": nil}, http.StatusOK)
		if _, ok := v.hospital(w.a2.ID).Values["contact.phone"]; ok || string(v.hospital(w.a2.ID).Effective["contact.phone"]) != `"0100 000 0000"` {
			t.Errorf("a2 %v", v.hospital(w.a2.ID).Values)
		}
	})
	t.Run("S21: a status message without {{link}} is accepted, switched on for one hospital; an invite without one isn't", func(t *testing.T) {
		v, _ := patchSettings(t, w, w.a1.ID, map[string]any{"statusMessages.readyForAdmissionSms": true,
			"templates.readyForAdmissionSms": "{{hospital}}: you're ready, {{firstName}}.", "worklists.soonDays": 10}, http.StatusOK)
		if string(v.hospital(w.a1.ID).Effective["statusMessages.readyForAdmissionSms"]) != "true" ||
			string(v.hospital(w.a2.ID).Effective["statusMessages.readyForAdmissionSms"]) != "false" ||
			string(v.hospital(w.a1.ID).Effective["worklists.soonDays"]) != "10" || string(v.hospital(w.a2.ID).Effective["worklists.soonDays"]) != "14" {
			t.Errorf("a1 %v / a2 %v", v.hospital(w.a1.ID).Values, v.hospital(w.a2.ID).Effective["statusMessages.readyForAdmissionSms"])
		}
		_, msg := patchSettings(t, w, w.a1.ID, map[string]any{"templates.inviteSms": "{{hospital}}: you're ready, {{firstName}}."}, http.StatusBadRequest)
		if !strings.Contains(msg, "{{link}}") {
			t.Errorf("message %q, want it to say {{link}} is needed", msg)
		}
	})
	t.Run("trust-only settings can't be set for a hospital", func(t *testing.T) {
		_, msg := patchSettings(t, w, w.a1.ID, map[string]any{"retentionYears": 3}, http.StatusBadRequest)
		if msg == "" {
			t.Error("no message")
		}
		// S27: crossTrustPrefill is gone, for the trust too (another trust's answers are always copied).
		patchSettings(t, w, nil, map[string]any{"crossTrustPrefill": true}, http.StatusBadRequest)
	})
	t.Run("the opt-ins are the platform admin's", func(t *testing.T) {
		patchSettings(t, w, nil, map[string]any{"feature.proms": true}, http.StatusForbidden)
		patchSettings(t, w, w.a1.ID, map[string]any{"instrument.eq5d5l": true}, http.StatusForbidden)
		if n := count(t, w.c.DB, `SELECT count(*) FROM hospitals WHERE id = $1 AND settings ? 'instrument.eq5d5l'`, w.a1.ID); n != 0 {
			t.Error("written")
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		w.c.Do("PATCH", settingsPath(w.a1.Org), map[string]any{"hospitalId": nil, "values": map[string]any{"retentionYears": 9}}).
			Want(t, http.StatusUnauthorized)
	})
	t.Run("wrong role", func(t *testing.T) {
		body := map[string]any{"hospitalId": w.a1.ID, "values": map[string]any{"contact.phone": "0300 000 0000"}}
		w.supC.Do("PATCH", settingsPath(w.a1.Org), body).Want(t, http.StatusForbidden)
		w.hosC.Do("PATCH", settingsPath(w.a1.Org), body).Want(t, http.StatusForbidden)
	})
	t.Run("another trust: its admin gets 404, and its hospital or questionnaire can't be named", func(t *testing.T) {
		w.beaC.Do("PATCH", settingsPath(w.a1.Org), map[string]any{"hospitalId": nil, "values": map[string]any{"retentionYears": 9}}).
			Want(t, http.StatusNotFound)
		patchSettings(t, w, w.b1.ID, map[string]any{"contact.phone": "0400 000 0000"}, http.StatusBadRequest)
		patchSettings(t, w, nil, map[string]any{"defaultHq": otherHQ.ID.String()}, http.StatusBadRequest)
		if n := count(t, w.c.DB, `SELECT count(*) FROM hospitals WHERE id = $1 AND settings ? 'contact.phone'`, w.b1.ID); n != 0 {
			t.Error("trust B's hospital was changed")
		}
	})
	t.Run("invalid input: nothing is written", func(t *testing.T) {
		before := count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND settings->>'retentionYears' = '10'`, w.a1.Org)
		for name, values := range map[string]map[string]any{
			"an unknown key":          {"nope": 1},
			"out of range":            {"retentionYears": 0},
			"a good and a bad value":  {"retentionYears": 20, "contact.email": "nope"},
			"a template with no link": {"templates.reminderSms": "Hello {{firstName}}"},
			"nothing to change":       {},
			"a draft as default HQ":   {"defaultHq": uuid.New().String()},
		} {
			if _, msg := patchSettings(t, w, nil, values, http.StatusBadRequest); msg == "" {
				t.Errorf("%s: no message", name)
			}
		}
		w.adaC.Do("PATCH", settingsPath(w.a1.Org), `{"values":`).Want(t, http.StatusBadRequest)
		w.adaC.Do("PATCH", settingsPath(w.a1.Org), map[string]any{"hospitalId": "nope", "values": map[string]any{"contact.phone": "0100"}}).
			Want(t, http.StatusBadRequest)
		if count(t, w.c.DB, `SELECT count(*) FROM orgs WHERE id = $1 AND settings->>'retentionYears' = '10'`, w.a1.Org) != before {
			t.Error("the bad request changed something")
		}
	})
}

// The observations' ranges (S46 answers 1): the platform's until the trust's whole-trust admins set
// their own, for the whole trust only, within the rules no trust can change.
func TestObservationRangesSetting(t *testing.T) {
	t.Parallel()
	w := newSettingsWorld(t)
	mine := map[string]any{"heartRate": map[string]any{"softMin": 50, "softMax": 100, "hardMin": 30, "hardMax": 150}}
	t.Run("the platform's by default", func(t *testing.T) {
		v := getSettings(t, w.adaC, w.a1.Org)
		var r map[string]map[string]float64
		if err := json.Unmarshal(v.Effective["observations.ranges"], &r); err != nil || r["heartRate"]["hardMax"] != 300 || len(r) != 10 {
			t.Errorf("effective = %s (%v)", v.Effective["observations.ranges"], err)
		}
	})
	t.Run("an admin sets the trust's own; every hospital follows it", func(t *testing.T) {
		v, _ := patchSettings(t, w, nil, map[string]any{"observations.ranges": mine}, http.StatusOK) // nothing set yet, so nothing to say it loaded
		var r map[string]map[string]float64
		if err := json.Unmarshal(v.hospital(w.a2.ID).Effective["observations.ranges"], &r); err != nil || r["heartRate"]["hardMax"] != 150 {
			t.Errorf("hospital A2's = %s", v.hospital(w.a2.ID).Effective["observations.ranges"])
		}
	})
	t.Run("a fixed rule broken: 400, nothing stored", func(t *testing.T) {
		bad := map[string]any{"heartRate": map[string]any{"softMin": 50, "softMax": 100, "hardMin": -1, "hardMax": 150}}
		_, msg := patchSettings(t, w, nil, map[string]any{"observations.ranges": bad}, http.StatusBadRequest)
		if !strings.Contains(msg, "below 0") {
			t.Errorf("message %q", msg)
		}
	})
	t.Run("not for one hospital: it's the whole trust's", func(t *testing.T) {
		_, msg := patchSettings(t, w, w.a1.ID, map[string]any{"observations.ranges": mine}, http.StatusBadRequest)
		if !strings.Contains(msg, "whole trust only") {
			t.Errorf("message %q", msg)
		}
	})
	t.Run("a super clinician can't set it (403)", func(t *testing.T) {
		w.supC.Do("PATCH", settingsPath(w.a1.Org), map[string]any{"hospitalId": nil, "values": map[string]any{"observations.ranges": mine}}).
			Want(t, http.StatusForbidden)
	})
}

// S46 test report 5: the ranges are one setting, so a page left open would put back every row as
// it last read them. A save that changes them says what it loaded, and is refused if that's gone.
func TestObservationRangesLeftOpen(t *testing.T) {
	t.Parallel()
	w := newSettingsWorld(t)
	row := func(softMax int) map[string]any {
		return map[string]any{"softMin": 50, "softMax": softMax, "hardMin": 30, "hardMax": 150}
	}
	save := func(loaded, ranges any) *apitest.Resp {
		return w.adaC.Do("PATCH", settingsPath(w.a1.Org), map[string]any{"hospitalId": nil,
			"values": map[string]any{"observations.ranges": ranges}, "loaded": map[string]any{"observations.ranges": loaded}})
	}
	first := map[string]any{"heartRate": row(100)}
	// As the API returns it: the database's key order, which a page sends back unchanged.
	stored := map[string]any{"heartRate": map[string]any{"hardMax": 150, "hardMin": 30, "softMax": 100, "softMin": 50}}
	theirs := map[string]any{"heartRate": row(105)}
	save(nil, first).Want(t, http.StatusOK)
	t.Run("what it loaded is what's stored, in any key order: saved", func(t *testing.T) {
		save(stored, first).Want(t, http.StatusOK)
	})
	save(first, theirs).Want(t, http.StatusOK) // the other admin's save
	t.Run("the page left open: refused, and nothing stored", func(t *testing.T) {
		r := save(first, map[string]any{"heartRate": row(100), "weight": map[string]any{"softMin": 35, "softMax": 190, "hardMin": 1, "hardMax": 400}})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "Someone else changed the observation ranges") {
			t.Errorf("message %q", r.Message())
		}
		var got map[string]map[string]float64
		v := getSettings(t, w.adaC, w.a1.Org)
		if err := json.Unmarshal(v.Trust["observations.ranges"], &got); err != nil || got["heartRate"]["softMax"] != 105 || len(got) != 1 {
			t.Errorf("stored = %s (%v)", v.Trust["observations.ranges"], err)
		}
	})
	t.Run("back to the default from a page left open: refused too", func(t *testing.T) {
		refused(t, save(first, nil), http.StatusConflict)
	})
	t.Run("loaded said nothing was set, but it is: refused", func(t *testing.T) {
		refused(t, save(nil, first), http.StatusConflict)
	})
	t.Run("another setting saves from a page left open", func(t *testing.T) {
		r := w.adaC.Do("PATCH", settingsPath(w.a1.Org), map[string]any{"hospitalId": nil,
			"values": map[string]any{"reminders.notRespondingDay": 9}})
		r.Want(t, http.StatusOK)
	})
}

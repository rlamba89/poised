//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/asarules"
)

// ASA rules on an HQ (S40; ASA-02, ASA-03). Go stores and checks their shape; which fire is
// packages/clinical's.

type rulesDetail struct {
	Questionnaire struct {
		VersionID     uuid.UUID
		Status        string
		Rules         json.RawMessage
		RulesRevision int
	}
}

func getRules(t *testing.T, c *apitest.Client, h apitest.Hospital, qid uuid.UUID) rulesDetail {
	t.Helper()
	var d rulesDetail
	r := c.Do("GET", at(h, "/questionnaires/"+qid.String()), nil)
	r.Want(t, http.StatusOK)
	r.Decode(t, &d)
	return d
}

func storedRules(t *testing.T, w *cast, versionID uuid.UUID) (asarules.RuleSet, int) {
	t.Helper()
	var raw []byte
	var rev int
	if err := w.db.QueryRow(t.Context(), "SELECT rules, rules_revision FROM questionnaire_versions WHERE id = $1", versionID).Scan(&raw, &rev); err != nil {
		t.Fatal(err)
	}
	var set asarules.RuleSet
	if raw != nil {
		if err := json.Unmarshal(raw, &set); err != nil {
			t.Fatal(err)
		}
	}
	return set, rev
}

func defaults(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(asarules.Defaults(), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNewHQGetsTheDefaultRules(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	r := w.author.Do("POST", at(w.a, "/questionnaires"), map[string]string{"name": "Rules HQ", "description": "x"})
	r.Want(t, http.StatusCreated)
	var created struct{ ID uuid.UUID }
	r.Decode(t, &created)

	t.Run("the questionnaire comes with its rules", func(t *testing.T) {
		d := getRules(t, w.author, w.a, created.ID)
		set, problem := asarules.Check(d.Questionnaire.Rules)
		if problem != "" || len(set.Rules) != 2 || set.Rules[0].Name != "Low risk" || set.Rules[1].Name != "High risk" || !set.NoGradeAutoTriage {
			t.Errorf("rules = %s (%s), want the two defaults", d.Questionnaire.Rules, problem)
		}
		if d.Questionnaire.RulesRevision != 0 {
			t.Errorf("revision = %d, want 0", d.Questionnaire.RulesRevision)
		}
	})
	t.Run("a clinician reads them too (read-only)", func(t *testing.T) {
		if d := getRules(t, w.clin, w.a, created.ID); len(d.Questionnaire.Rules) == 0 || string(d.Questionnaire.Rules) == "null" {
			t.Error("a clinician got no rules")
		}
	})
	t.Run("another trust can't read them", func(t *testing.T) {
		refused(t, w.other.Do("GET", at(w.a, "/questionnaires/"+created.ID.String()), nil), http.StatusNotFound)
		refused(t, w.other.Do("GET", at(w.b, "/questionnaires/"+created.ID.String()), nil), http.StatusNotFound)
	})
}

func TestNewVersionCopiesTheRules(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	newVersion := func(t *testing.T, hq apitest.HQ) uuid.UUID {
		t.Helper()
		r := w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/versions"), nil)
		r.Want(t, http.StatusCreated)
		var v struct{ ID uuid.UUID }
		r.Decode(t, &v)
		return v.ID
	}
	t.Run("the published version's own rules", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		own := defaults(t)
		own["noGradeAutoTriage"] = false
		own["rules"] = own["rules"].([]any)[:1]
		b, _ := json.Marshal(own)
		if _, err := w.db.Exec(t.Context(), "UPDATE questionnaire_versions SET rules = $2 WHERE id = $1", hq.VersionID, b); err != nil {
			t.Fatal(err)
		}
		set, rev := storedRules(t, w, newVersion(t, hq))
		if len(set.Rules) != 1 || set.NoGradeAutoTriage || rev != 0 {
			t.Errorf("v2's rules = %+v (revision %d), want v1's one rule with the setting off", set, rev)
		}
	})
	t.Run("an older version with no rules: the defaults (S40 answers 1)", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		if set, _ := storedRules(t, w, hq.VersionID); set.Rules != nil {
			t.Fatal("the fixture's version already has rules")
		}
		set, _ := storedRules(t, w, newVersion(t, hq))
		if len(set.Rules) != 2 || !set.NoGradeAutoTriage {
			t.Errorf("v2's rules = %+v, want the two defaults", set)
		}
		if set, _ := storedRules(t, w, hq.VersionID); set.Rules != nil {
			t.Error("the published version was given rules")
		}
	})
}

func TestSaveRules(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	path := at(w.a, "/questionnaires/"+hq.ID.String()+"/rules")
	var procedure, otherTrusts uuid.UUID
	if err := w.db.QueryRow(t.Context(), "INSERT INTO procedures (org_id, name) VALUES ($1, 'Hip replacement') RETURNING id", w.a.Org).Scan(&procedure); err != nil {
		t.Fatal(err)
	}
	if err := w.db.QueryRow(t.Context(), "INSERT INTO procedures (org_id, name) VALUES ($1, 'Knee replacement') RETURNING id", w.b.Org).Scan(&otherTrusts); err != nil {
		t.Fatal(err)
	}
	body := func(revision int, change func(m map[string]any)) map[string]any {
		m := defaults(t)
		if change != nil {
			change(m)
		}
		return map[string]any{"rules": m, "revision": revision, "versionId": hq.VersionID}
	}
	firstWhen := func(m map[string]any) map[string]any {
		return m["rules"].([]any)[0].(map[string]any)["when"].(map[string]any)
	}

	t.Run("happy path: the draft's rules are stored, checked and trimmed, with a new revision", func(t *testing.T) {
		r := w.author.Do("PUT", path, body(0, func(m map[string]any) {
			low := m["rules"].([]any)[0].(map[string]any)
			low["then"] = []any{map[string]any{"kind": "autoTriage", "target": "ready_for_poa"}}
			low["name"] = "  Low risk "
			firstWhen(m)["procedures"] = []any{procedure.String()}
			m["noGradeAutoTriage"] = false
			m["testCases"] = []any{map[string]any{"id": "t1", "name": "Fit and well", "patient": map[string]any{"name": "Sam Sample", "age": 54},
				"answers": map[string]any{"q_smokes": "no"}, "expected": []any{"No suggested ASA", "Rule fires: Low risk: auto-triage to Ready for POA"}}}
		}))
		r.Want(t, http.StatusOK)
		var got struct {
			Revision int
			Rules    asarules.RuleSet
		}
		r.Decode(t, &got)
		if got.Rules.Rules[0].Name != "Low risk" {
			t.Errorf("the reply's rules = %+v, want them as stored (trimmed)", got.Rules)
		}
		set, rev := storedRules(t, w, hq.VersionID)
		if got.Revision != 1 || rev != 1 {
			t.Errorf("revision = %d (stored %d), want 1", got.Revision, rev)
		}
		if set.Rules[0].Name != "Low risk" || set.Rules[0].Then[0].Target != "ready_for_poa" || set.NoGradeAutoTriage || len(set.TestCases) != 1 {
			t.Errorf("stored = %+v", set)
		}
		if d := getRules(t, w.author, w.a, hq.ID); d.Questionnaire.RulesRevision != 1 || !strings.Contains(string(d.Questionnaire.Rules), "Fit and well") {
			t.Errorf("GET = %s (revision %d)", d.Questionnaire.Rules, d.Questionnaire.RulesRevision)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("PUT", path, body(1, nil)), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician, and a super clinician of one hospital (the trust's HQ)", func(t *testing.T) {
		refused(t, w.clin.Do("PUT", path, body(1, nil)), http.StatusForbidden)
		sue := w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician"))
		refused(t, sue.Do("PUT", path, body(1, nil)), http.StatusForbidden)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.other.Do("PUT", path, body(1, nil)), http.StatusNotFound)
		refused(t, w.other.Do("PUT", at(w.b, "/questionnaires/"+hq.ID.String()+"/rules"), body(1, nil)), http.StatusNotFound)
		// A rule may name only the trust's own procedures.
		refused(t, w.author.Do("PUT", path, body(1, func(m map[string]any) { firstWhen(m)["procedures"] = []any{otherTrusts.String()} })), http.StatusBadRequest)
	})
	t.Run("invalid input: a bad shape, in plain words", func(t *testing.T) {
		for _, c := range []struct {
			body   any
			saying string
		}{
			{body(1, func(m map[string]any) {
				m["rules"].([]any)[1].(map[string]any)["then"].([]any)[0].(map[string]any)["title"] = ""
			}), "Rule “High risk”: Enter the task."},
			{body(1, func(m map[string]any) { m["rules"].([]any)[0].(map[string]any)["then"] = []any{} }), "add at least one action"},
			{body(1, func(m map[string]any) { firstWhen(m)["asa"] = map[string]any{"op": "<=", "grade": 9} }), "I to VI"},
			{map[string]any{"revision": 1, "versionId": hq.VersionID}, "could not be read"},
			{map[string]any{"rules": defaults(t), "revision": 1}, "could not be read"}, // no version named
			{`{"rules": `, ""},
		} {
			r := w.author.Do("PUT", path, c.body)
			refused(t, r, http.StatusBadRequest)
			if !strings.Contains(r.Message(), c.saying) {
				t.Errorf("message = %q, want it to say %q", r.Message(), c.saying)
			}
		}
		refused(t, w.author.Do("PUT", at(w.a, "/questionnaires/abc/rules"), body(1, nil)), http.StatusNotFound)
		if _, rev := storedRules(t, w, hq.VersionID); rev != 1 {
			t.Errorf("a refused save changed the revision to %d", rev)
		}
	})
	t.Run("a stale revision: 409, nothing saved", func(t *testing.T) {
		r := w.author.Do("PUT", path, body(0, nil))
		refused(t, r, http.StatusConflict)
		if set, rev := storedRules(t, w, hq.VersionID); rev != 1 || set.NoGradeAutoTriage {
			t.Errorf("a stale save changed the rules (revision %d)", rev)
		}
	})
	t.Run("a page opened on an older version: 409, the new draft unchanged (verification 1, F1)", func(t *testing.T) {
		old := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		oldPath := at(w.a, "/questionnaires/"+old.ID.String())
		// v1 is published and v2 made while a page still shows v1 at revision 0.
		w.author.Do("POST", oldPath+"/publish", map[string]any{"versionId": old.VersionID}).Want(t, http.StatusNoContent)
		w.author.Do("POST", oldPath+"/versions", nil).Want(t, http.StatusCreated)
		v2 := getRules(t, w.author, w.a, old.ID).Questionnaire.VersionID
		stale := defaults(t)
		stale["rules"] = stale["rules"].([]any)[:1]
		r := w.author.Do("PUT", oldPath+"/rules", map[string]any{"rules": stale, "revision": 0, "versionId": old.VersionID})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "changed since you opened it") {
			t.Errorf("message = %q", r.Message())
		}
		if set, rev := storedRules(t, w, v2); len(set.Rules) != 2 || rev != 0 {
			t.Errorf("v2's rules = %+v (revision %d), want the two defaults untouched", set, rev)
		}
	})
	t.Run("a published version is read-only: 409", func(t *testing.T) {
		pub := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		r := w.author.Do("PUT", at(w.a, "/questionnaires/"+pub.ID.String()+"/rules"), body(0, nil))
		refused(t, r, http.StatusConflict)
		if set, _ := storedRules(t, w, pub.VersionID); set.Rules != nil {
			t.Error("the published version's rules changed")
		}
	})
}

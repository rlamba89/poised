//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/keys"
)

// Question keys (plan C4, session S23): the editor's label and option lock (GET …/question-keys),
// and the two checks publishing runs (a key used twice in an HQ; a published keyed question's
// options changed, in any HQ of the trust). Another trust's HQs never count.

// setContent replaces a Question Set's JSON, as an author's saves would.
func setContent(t *testing.T, w *cast, chapterID uuid.UUID, content string) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), "UPDATE chapters SET content = $2 WHERE id = $1", chapterID, content); err != nil {
		t.Fatal(err)
	}
}

func page(elements string) string {
	return `{"pages":[{"name":"p_1","elements":[` + elements + `]}]}`
}

// smokes is workflow.json's q_smokes with these options.
func smokes(options string) string {
	return `{"type":"radiogroup","name":"q_smokes","title":"Do you smoke?","choices":[` + options + `]}`
}

const smokesYesNo = `{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"}`

type keysResponse struct {
	Questions map[string][]keys.Match
	Problems  []keys.Problem
}

func TestQuestionKeys(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	get := func(c *apitest.Client, h apitest.Hospital, hq apitest.HQ) *apitest.Resp {
		return c.Do("GET", at(h, "/questionnaires/"+hq.ID.String()+"/question-keys"), nil)
	}

	// Trust A: "X" published shares q_smokes by key; "Z" asks about
	// alcohol in the same words under another key; the draft being edited is "mine".
	x := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
	z := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
	setContent(t, w, z.Chapters[0], page(`{"type":"radiogroup","name":"q_drink","title":"how often do you drink alcohol","choices":[{"value":"o_1","text":"never"},{"value":"o_2","text":"Sometimes"},{"value":"o_3","text":"Often"}]}`))
	setContent(t, w, z.Chapters[1], page(``))
	mine := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
	setContent(t, w, mine.Chapters[0], page(smokes(smokesYesNo)+`,
		{"type":"radiogroup","name":"q_alcohol","title":"How often do you drink alcohol?","choices":[{"value":"o_a","text":"Never"},{"value":"o_b","text":"Sometimes"},{"value":"o_c","text":"Often"}]},
		{"type":"text","name":"q_brand_new","title":"Something nobody else asks"},
		{"type":"text","name":"q_nurse","title":"Pulse","clinicianOnly":true}`))
	// Trust B has the same HQ, keys and all: it never counts for trust A.
	w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")
	w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")

	t.Run("happy path: matches by key and by same wording, and the published options", func(t *testing.T) {
		r := get(w.author, w.a, mine)
		r.Want(t, http.StatusOK)
		var got keysResponse
		r.Decode(t, &got)
		if m := got.Questions["q_smokes"]; !reflect.DeepEqual(m, []keys.Match{{QuestionnaireID: x.ID.String(), Name: x.Name, By: "key"}}) {
			t.Errorf("q_smokes matches %v, want X by key only (not Z, not trust B's)", m)
		}
		if m := got.Questions["q_alcohol"]; !reflect.DeepEqual(m, []keys.Match{{QuestionnaireID: z.ID.String(), Name: z.Name, By: "wording"}}) {
			t.Errorf("q_alcohol matches %v, want Z by wording", m)
		}
		if m, ok := got.Questions["q_brand_new"]; !ok || len(m) != 0 {
			t.Errorf("q_brand_new = %v (listed %v), want a new question", m, ok)
		}
		if _, ok := got.Questions["q_nurse"]; ok {
			t.Error("a clinician-only question has a pre-fill label")
		}
		if len(got.Problems) != 0 {
			t.Errorf("problems %v, want none", got.Problems)
		}
	})
	t.Run("happy path: the draft's key problems, as Publish would give them", func(t *testing.T) {
		draft := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, draft.Chapters[0], page(smokes(`{"value":"o_yes","text":"Yes"}`)+`,{"type":"text","name":"q_dup","title":"A"},{"type":"text","name":"q_dup","title":"B"}`))
		r := get(w.author, w.a, draft)
		r.Want(t, http.StatusOK)
		var got keysResponse
		r.Decode(t, &got)
		all := ""
		for _, p := range got.Problems {
			all += p.Set + " · " + p.Message + "\n"
		}
		if len(got.Problems) != 2 || !strings.Contains(all, "the option “No” was removed") || !strings.Contains(all, "The question ID q_dup is used") {
			t.Errorf("problems:\n%s want the removed option (from X, not trust B) and the duplicate", all)
		}
	})
	t.Run("happy path: a super clinician of one hospital can see them too", func(t *testing.T) {
		sue := w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician"))
		get(sue, w.a, mine).Want(t, http.StatusOK)
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, get(w.anon, w.a, mine), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician", func(t *testing.T) {
		refused(t, get(w.clin, w.a, mine), http.StatusForbidden)
		refused(t, get(w.anon.LoginAs(t, w.fx.TrustUser(t, w.a.Org, "clinician")), w.a, mine), http.StatusForbidden)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, get(w.other, w.a, mine), http.StatusNotFound)
		refused(t, get(w.other, w.b, mine), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("GET", at(w.a, "/questionnaires/abc/question-keys"), nil), http.StatusNotFound)
		refused(t, w.author.Do("GET", at(w.a, "/questionnaires/"+uuid.NewString()+"/question-keys"), nil), http.StatusNotFound)
	})
}

type publishRefusal struct {
	Error    string
	Problems []keys.Problem
}

func TestPublishKeyChecks(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	publish := func(hq apitest.HQ, versionID uuid.UUID) *apitest.Resp {
		return w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/publish"), map[string]any{"versionId": versionID})
	}
	newVersion := func(t *testing.T, hq apitest.HQ) (uuid.UUID, uuid.UUID) {
		t.Helper()
		r := w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/versions"), nil)
		r.Want(t, http.StatusCreated)
		var v struct{ ID uuid.UUID }
		r.Decode(t, &v)
		var first uuid.UUID
		if err := w.db.QueryRow(t.Context(), "SELECT id FROM chapters WHERE version_id = $1 AND position = 1", v.ID).Scan(&first); err != nil {
			t.Fatal(err)
		}
		return v.ID, first
	}
	refusedWith := func(t *testing.T, r *apitest.Resp, versionID uuid.UUID, want ...string) []keys.Problem {
		t.Helper()
		refused(t, r, http.StatusUnprocessableEntity)
		var body publishRefusal
		r.Decode(t, &body)
		all := ""
		for _, p := range body.Problems {
			all += p.Set + " · " + p.Message + "\n"
		}
		for _, s := range want {
			if !strings.Contains(all, s) {
				t.Errorf("problems %q don't mention %q", all, s)
			}
		}
		if s := versionStatus(t, w.db, versionID); s != "draft" {
			t.Errorf("a refused version is %s, want draft", s)
		}
		return body.Problems
	}

	t.Run("the same HQ's new version: a removed option is refused", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		v2, set := newVersion(t, hq)
		setContent(t, w, set, page(smokes(`{"value":"o_yes","text":"Yes"}`)))
		p := refusedWith(t, publish(hq, v2), v2, "About you", "“Do you smoke?”", "the option “No” was removed")
		if len(p) != 1 {
			t.Errorf("problems %v, want one", p)
		}
	})
	t.Run("the same HQ's new version: an added or reworded option is refused too (frozen, S23 answers 2)", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		v2, set := newVersion(t, hq)
		setContent(t, w, set, page(smokes(smokesYesNo+`,{"value":"o_ex","text":"I used to"}`)))
		refusedWith(t, publish(hq, v2), v2, "the option “I used to” was added")
		setContent(t, w, set, page(smokes(`{"value":"o_yes","text":"Yes, daily"},{"value":"o_no","text":"No"}`)))
		refusedWith(t, publish(hq, v2), v2, "the option “Yes” was reworded to “Yes, daily”")
	})
	t.Run("happy path: the question reworded and its options reordered publishes", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		v2, set := newVersion(t, hq)
		setContent(t, w, set, page(`{"type":"radiogroup","name":"q_smokes","title":"Do you currently smoke?","choices":[{"value":"o_no","text":"No"},{"value":"o_yes","text":"Yes"}]}`))
		publish(hq, v2).Want(t, http.StatusNoContent)
	})
	t.Run("a different HQ of the trust with the key: a changed option is refused", func(t *testing.T) {
		w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		other := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, other.Chapters[0], page(smokes(`{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"Never"}`)))
		refusedWith(t, publish(other, other.VersionID), other.VersionID, "the option “No” was reworded to “Never”", "(published in “Workflow HQ")
	})
	t.Run("happy path: another trust's HQs never lock a key", func(t *testing.T) {
		// In trust B, q_smokes is a Yes/No switch with no options at all.
		w.fx.PublishedHQ(t, w.b, w.bea, "basic.json")
		mine := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, mine.Chapters[0], page(smokes(smokesYesNo)))
		// Trust A's own published copies (above) have Yes/No, so this publishes.
		publish(mine, mine.VersionID).Want(t, http.StatusNoContent)
	})
	t.Run("a published key can't change its kind of answer or unit (after verification 1)", func(t *testing.T) {
		src := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, src.Chapters[0], page(`{"type":"text","name":"q_wt","title":"Weight","inputType":"number","unit":"kg"},
			{"type":"matrix","name":"q_grid","title":"How often do you","rows":["Run","Swim"],"columns":["Never","Always"]}`))
		setContent(t, w, src.Chapters[1], page(``))
		publish(src, src.VersionID).Want(t, http.StatusNoContent)

		mine := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, mine.Chapters[0], page(`{"type":"text","name":"q_wt","title":"Weight","inputType":"date"},
			{"type":"matrix","name":"q_grid","title":"How often do you","rows":["Run"],"columns":["Never","Always"]}`))
		setContent(t, w, mine.Chapters[1], page(``))
		refusedWith(t, publish(mine, mine.VersionID), mine.VersionID,
			"“Weight”: its kind of answer changed from number to date", "the row “Swim” was removed")
		setContent(t, w, mine.Chapters[0], page(`{"type":"text","name":"q_wt","title":"Weight","inputType":"number","unit":"lb"}`))
		refusedWith(t, publish(mine, mine.VersionID), mine.VersionID, "its unit changed from “kg” to “lb”")
	})
	t.Run("a published date's format and a text question's input format are frozen (after verification 2)", func(t *testing.T) {
		src := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, src.Chapters[0], page(`{"type":"text","name":"q_opdate","title":"Date of your operation","inputType":"date"},
			{"type":"text","name":"q_contact","title":"Contact"}`))
		setContent(t, w, src.Chapters[1], page(``))
		publish(src, src.VersionID).Want(t, http.StatusNoContent)

		mine := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, mine.Chapters[0], page(`{"type":"text","name":"q_opdate","title":"Date of your operation","inputType":"number","dateFormat":"y"},
			{"type":"text","name":"q_contact","title":"Contact","inputType":"email","validators":[{"type":"email"}]}`))
		setContent(t, w, mine.Chapters[1], page(``))
		refusedWith(t, publish(mine, mine.VersionID), mine.VersionID,
			"its date format changed from day, month and year to year only", "its answer format changed from any text to email address")
	})
	t.Run("happy path: a published chapter with plain-string grid columns doesn't stop publishing (after verification 1)", func(t *testing.T) {
		odd := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, odd.Chapters[1], page(`{"type":"matrix","name":"q_odd","title":"Odd","rows":["A"],"columns":["Never","Sometimes"]},{"type":"matrixdynamic","name":"q_odd2","columns":["what"]}`))
		mine := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, mine.Chapters[1], page(`{"type":"matrix","name":"q_odd","title":"Odd","rows":["A"],"columns":["Never","Sometimes"]}`))
		publish(mine, mine.VersionID).Want(t, http.StatusNoContent)
	})
	t.Run("a key published in shapes that disagree is one problem, not a list of contradictions (test report 3)", func(t *testing.T) {
		// Before the lock, two HQs published q_conflict differently.
		for _, el := range []string{`{"type":"boolean","name":"q_conflict","title":"Diabetes?"}`, `{"type":"radiogroup","name":"q_conflict","title":"Diabetes?","choices":["Yes","No"]}`} {
			old := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
			setContent(t, w, old.Chapters[1], page(el))
		}
		mine := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, mine.Chapters[1], page(`{"type":"radiogroup","name":"q_conflict","title":"Diabetes?","choices":["Yes","No"]}`))
		p := refusedWith(t, publish(mine, mine.VersionID), mine.VersionID, "“Diabetes?”: the trust has already published this question with different answers")
		if len(p) != 1 {
			t.Errorf("problems %v, want one", p)
		}
	})
	t.Run("a key used twice in one HQ is refused, naming both Question Sets", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, hq.Chapters[1], page(`{"type":"boolean","name":"q_heart","title":"Heart trouble?"}`)) // q_heart is in "About you" too (the same kind, so only the duplicate is wrong)
		p := refusedWith(t, publish(hq, hq.VersionID), hq.VersionID, "q_heart", "“About you”", "“Your health”")
		if len(p) != 1 || p[0].Set != "Your health" {
			t.Errorf("problems %v, want one, on Your health", p)
		}
	})
}

// Keys survive "New version" (LCY-06/07): every element keeps its name, so pre-fill finds them.
func TestNewVersionKeepsKeys(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
	r := w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/versions"), nil)
	r.Want(t, http.StatusCreated)
	var v2 struct{ ID uuid.UUID }
	r.Decode(t, &v2)

	names := func(versionID uuid.UUID) []string {
		rows, err := w.db.Query(t.Context(), "SELECT content FROM chapters WHERE version_id = $1 ORDER BY position", versionID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var content json.RawMessage
			if err := rows.Scan(&content); err != nil {
				t.Fatal(err)
			}
			qs, err := keys.Walk(content)
			if err != nil {
				t.Fatal(err)
			}
			for _, q := range qs {
				out = append(out, q.Name)
			}
		}
		return out
	}
	v1Names, v2Names := names(hq.VersionID), names(v2.ID)
	if len(v1Names) < 8 || !slices.Equal(v1Names, v2Names) {
		t.Errorf("v1 keys %v, v2 keys %v; want the same", v1Names, v2Names)
	}
}

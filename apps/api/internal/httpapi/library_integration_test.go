//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/keys"
)

// The question library (plan C4, session S24) at /api/o/{oid}/library-questions: the platform's
// starter set (seeded; org_id NULL) and each trust's own, which whole-trust super clinicians and
// admins save from the editor and remove (S24 answers 2). And "Same as…" at
// …/questionnaires/{qid}/question-keys/search (answers 3–5).

type libraryItem struct {
	ID            uuid.UUID
	Platform      bool
	Key, Title    string
	Element       json.RawMessage
	Tags          []string
	CreatedByName string
}

func libraryPath(org uuid.UUID, p string) string { return orgPath(org, "/library-questions"+p) }

// platformQuestion seeds a platform library question, as cmd/seed does.
func platformQuestion(t *testing.T, w *cast, key, title, element string, tags ...string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if tags == nil {
		tags = []string{}
	}
	if err := w.db.QueryRow(t.Context(), `INSERT INTO library_questions (key, title, element, tags) VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING RETURNING id`, key, title, element, tags).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

const smokerElement = `{"type":"radiogroup","name":"smoker","title":"Do you smoke?","choices":[{"value":"never","text":"Never"},{"value":"ex","text":"I used to"},{"value":"current","text":"Yes, I smoke now"}]}`

func listLibrary(t *testing.T, c *apitest.Client, org uuid.UUID, q string) []libraryItem {
	t.Helper()
	r := c.Do("GET", libraryPath(org, "?q="+url.QueryEscape(q)), nil)
	r.Want(t, http.StatusOK)
	var got []libraryItem
	r.Decode(t, &got)
	return got
}

func libraryKeys(items []libraryItem) []string {
	var out []string
	for _, l := range items {
		out = append(out, l.Key)
	}
	return out
}

func TestListLibrary(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	platformQuestion(t, w, "smoker", "Do you smoke?", smokerElement, "Lifestyle")
	platformQuestion(t, w, "snoring", "Do you snore?", `{"type":"radiogroup","name":"snoring","title":"Do you snore?","choices":[{"value":"yes","text":"Yes"},{"value":"no","text":"No"}]}`, "Sleep")
	w.author.Do("POST", libraryPath(w.a.Org, ""), map[string]any{
		"element": map[string]any{"type": "text", "name": "q_job", "title": "What is your job?"}, "tags": []string{"Work"},
	}).Want(t, http.StatusCreated)
	w.other.Do("POST", libraryPath(w.b.Org, ""), map[string]any{
		"element": map[string]any{"type": "text", "name": "q_b_only", "title": "Trust B asks this"},
	}).Want(t, http.StatusCreated)

	t.Run("happy path: the platform's and the trust's own, by title", func(t *testing.T) {
		got := listLibrary(t, w.author, w.a.Org, "")
		if k := libraryKeys(got); !slices.Equal(k, []string{"smoker", "snoring", "q_job"}) {
			t.Fatalf("library = %v, want the platform's and trust A's (never trust B's), by title", k)
		}
		if !got[0].Platform || got[2].Platform || got[2].Title != "What is your job?" || !slices.Equal(got[2].Tags, []string{"Work"}) ||
			got[2].CreatedByName == "" || !strings.Contains(string(got[0].Element), `"value":"ex"`) {
			t.Errorf("items = %+v", got)
		}
		if k := libraryKeys(listLibrary(t, w.other, w.b.Org, "")); !slices.Equal(k, []string{"smoker", "snoring", "q_b_only"}) {
			t.Errorf("trust B's library = %v", k)
		}
	})
	t.Run("happy path: searching by title or tag, in any case", func(t *testing.T) {
		if k := libraryKeys(listLibrary(t, w.author, w.a.Org, "SNOR")); !slices.Equal(k, []string{"snoring"}) {
			t.Errorf("by title: %v", k)
		}
		if k := libraryKeys(listLibrary(t, w.author, w.a.Org, "lifestyle")); !slices.Equal(k, []string{"smoker"}) {
			t.Errorf("by tag: %v", k)
		}
		if k := libraryKeys(listLibrary(t, w.author, w.a.Org, "%")); len(k) != 0 {
			t.Errorf("a %% matched %v: it isn't a wildcard", k)
		}
	})
	t.Run("happy path: a whole-trust admin", func(t *testing.T) {
		admin := w.anon.LoginAs(t, w.fx.TrustUser(t, w.a.Org, "admin"))
		admin.Do("GET", libraryPath(w.a.Org, ""), nil).Want(t, http.StatusOK)
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", libraryPath(w.a.Org, ""), nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician, a whole-trust clinician, a super clinician of one hospital", func(t *testing.T) {
		refused(t, w.clin.Do("GET", libraryPath(w.a.Org, ""), nil), http.StatusForbidden)
		refused(t, w.anon.LoginAs(t, w.fx.TrustUser(t, w.a.Org, "clinician")).Do("GET", libraryPath(w.a.Org, ""), nil), http.StatusForbidden)
		refused(t, w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician")).Do("GET", libraryPath(w.a.Org, ""), nil), http.StatusForbidden)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.other.Do("GET", libraryPath(w.a.Org, ""), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("GET", "/api/o/abc/library-questions", nil), http.StatusNotFound)
	})
}

func TestSaveToLibrary(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	platformQuestion(t, w, "smoker", "Do you smoke?", smokerElement)
	save := func(c *apitest.Client, org uuid.UUID, body any) *apitest.Resp {
		return c.Do("POST", libraryPath(org, ""), body)
	}
	stored := func(t *testing.T, key string) (int, json.RawMessage) {
		t.Helper()
		var el json.RawMessage
		n := count(t, w.db, "SELECT count(*) FROM library_questions WHERE org_id = $1 AND key = $2", w.a.Org, key)
		if n == 1 {
			if err := w.db.QueryRow(t.Context(), "SELECT element FROM library_questions WHERE org_id = $1 AND key = $2", w.a.Org, key).Scan(&el); err != nil {
				t.Fatal(err)
			}
		}
		return n, el
	}

	t.Run("happy path: kept with its key, options and tags; its conditions and link dropped", func(t *testing.T) {
		r := save(w.author, w.a.Org, map[string]any{"tags": []string{" Lifestyle ", "", "lifestyle", "Diet"}, "element": json.RawMessage(
			`{"type":"radiogroup","name":"q_diet","title":"Do you follow a special diet?","visibleIf":"{q_other} = 'o_x'","alwaysFresh":true,
			  "linkedFrom":{"questionnaire":"Other HQ","title":"Diet?"},
			  "choices":[{"value":"o_vegan","text":"Vegan"},{"value":"o_none","text":"No"}]}`)})
		r.Want(t, http.StatusCreated)
		n, el := stored(t, "q_diet")
		if n != 1 || strings.Contains(string(el), "visibleIf") || strings.Contains(string(el), "linkedFrom") || !strings.Contains(string(el), `"o_vegan"`) || !strings.Contains(string(el), "alwaysFresh") {
			t.Fatalf("stored %d: %s", n, el)
		}
		var row struct {
			Title     string
			Tags      []string
			CreatedBy uuid.UUID
		}
		if err := w.db.QueryRow(t.Context(), "SELECT title, tags, created_by FROM library_questions WHERE key = 'q_diet'").Scan(&row.Title, &row.Tags, &row.CreatedBy); err != nil {
			t.Fatal(err)
		}
		if row.Title != "Do you follow a special diet?" || !slices.Equal(row.Tags, []string{"Lifestyle", "Diet"}) || row.CreatedBy != w.alex.ID {
			t.Errorf("row = %+v", row)
		}
	})
	t.Run("happy path: a published question, as published; a repeating group", func(t *testing.T) {
		w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(smokes(smokesYesNo))}).Want(t, http.StatusCreated)
		save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(`{"type":"paneldynamic","name":"q_ops","title":"Operations","templateElements":[{"type":"text","name":"q_op","title":"Operation"}]}`)}).
			Want(t, http.StatusCreated)
	})
	t.Run("a key the library already has (the trust's or the platform's) is refused", func(t *testing.T) {
		refused(t, save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(`{"type":"text","name":"q_diet","title":"Again"}`)}), http.StatusConflict)
		refused(t, save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(smokerElement)}), http.StatusConflict)
		if n, _ := stored(t, "smoker"); n != 0 {
			t.Errorf("trust A has %d smoker items, want none (the platform's)", n)
		}
	})
	t.Run("a question of a repeating list can't go in on its own (test report 1, found 1)", func(t *testing.T) {
		// workflow.json's q_med_name is a row question of the repeating group q_meds, published in trust A.
		w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		r := save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(`{"type":"text","name":"q_med_name","title":"Medicine"}`)})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "repeating list") {
			t.Errorf("message %q", r.Message())
		}
		// …nor a row question of a library repeating list (or any question inside a library item).
		platformQuestion(t, w, "allergies", "Your allergies", `{"type":"paneldynamic","name":"allergies","title":"Your allergies","templateElements":[{"type":"text","name":"allergies_what","title":"What are you allergic to?"}]}`)
		refused(t, save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(`{"type":"text","name":"allergies_what","title":"What are you allergic to?"}`)}), http.StatusConflict)
		if n, _ := stored(t, "q_med_name"); n != 0 {
			t.Error("a row question was stored")
		}
	})
	t.Run("a question that isn't as the trust published it is refused", func(t *testing.T) {
		w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
		r := save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(
			`{"type":"boolean","name":"q_diabetes","title":"Do you have diabetes?","labelTrue":"I do"}`)})
		refused(t, r, http.StatusConflict)
		if !strings.Contains(r.Message(), "the label “Yes” was reworded to “I do”") {
			t.Errorf("message %q", r.Message())
		}
	})
	t.Run("happy path: an admin of the whole trust", func(t *testing.T) {
		admin := w.anon.LoginAs(t, w.fx.TrustUser(t, w.a.Org, "admin"))
		save(admin, w.a.Org, map[string]any{"element": json.RawMessage(`{"type":"text","name":"q_admin","title":"Admin's"}`)}).Want(t, http.StatusCreated)
	})
	body := map[string]any{"element": json.RawMessage(`{"type":"text","name":"q_refused","title":"Refused"}`)}
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, save(w.anon, w.a.Org, body), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician, a super clinician of one hospital", func(t *testing.T) {
		refused(t, save(w.clin, w.a.Org, body), http.StatusForbidden)
		refused(t, save(w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician")), w.a.Org, body), http.StatusForbidden)
		if n, _ := stored(t, "q_refused"); n != 0 {
			t.Error("a refused save was stored")
		}
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, save(w.other, w.a.Org, body), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, el := range []string{`"text"`, `[]`, `{"type":"text","title":"No ID"}`, `{"type":"html","name":"q_html","html":"x"}`,
			`{"type":"panel","name":"g_group","elements":[{"type":"text","name":"q_in"}]}`, `{"type":"signaturepad","name":"q_sig"}`} {
			refused(t, save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(el)}), http.StatusBadRequest)
		}
		long := strings.Repeat("x", 41)
		refused(t, save(w.author, w.a.Org, map[string]any{"element": json.RawMessage(`{"type":"text","name":"q_t","title":"T"}`), "tags": []string{long}}), http.StatusBadRequest)
		refused(t, w.author.Do("POST", libraryPath(w.a.Org, ""), "not an object"), http.StatusBadRequest)
		if n := count(t, w.db, "SELECT count(*) FROM library_questions WHERE org_id = $1 AND key IN ('q_in','q_sig','q_t','q_html')", w.a.Org); n != 0 {
			t.Errorf("%d invalid items stored", n)
		}
	})
}

func TestRemoveFromLibrary(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	platform := platformQuestion(t, w, "smoker", "Do you smoke?", smokerElement)
	add := func(t *testing.T, key string) uuid.UUID {
		t.Helper()
		r := w.author.Do("POST", libraryPath(w.a.Org, ""), map[string]any{"element": map[string]any{"type": "text", "name": key, "title": key}})
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		return got.ID
	}
	remove := func(c *apitest.Client, org, id uuid.UUID) *apitest.Resp {
		return c.Do("DELETE", libraryPath(org, "/"+id.String()), nil)
	}

	t.Run("happy path: gone from the library; a questionnaire it was inserted into keeps its copy", func(t *testing.T) {
		id := add(t, "q_inserted")
		hq := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
		setContent(t, w, hq.Chapters[1], page(`{"type":"text","name":"q_inserted","title":"q_inserted"}`))
		remove(w.author, w.a.Org, id).Want(t, http.StatusNoContent)
		if n := count(t, w.db, "SELECT count(*) FROM library_questions WHERE id = $1", id); n != 0 {
			t.Error("still in the library")
		}
		if n := count(t, w.db, "SELECT count(*) FROM chapters WHERE id = $1 AND content::text LIKE '%q_inserted%'", hq.Chapters[1]); n != 1 {
			t.Error("the questionnaire's copy changed")
		}
	})
	t.Run("the platform's can't be removed", func(t *testing.T) {
		refused(t, remove(w.author, w.a.Org, platform), http.StatusForbidden)
		if n := count(t, w.db, "SELECT count(*) FROM library_questions WHERE id = $1", platform); n != 1 {
			t.Error("the platform's item was removed")
		}
	})
	id := add(t, "q_kept")
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, remove(w.anon, w.a.Org, id), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician, a super clinician of one hospital", func(t *testing.T) {
		refused(t, remove(w.clin, w.a.Org, id), http.StatusForbidden)
		refused(t, remove(w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician")), w.a.Org, id), http.StatusForbidden)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, remove(w.other, w.a.Org, id), http.StatusNotFound)
		refused(t, remove(w.other, w.b.Org, id), http.StatusNotFound)
		if n := count(t, w.db, "SELECT count(*) FROM library_questions WHERE id = $1", id); n != 1 {
			t.Error("removed by another trust")
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("DELETE", libraryPath(w.a.Org, "/abc"), nil), http.StatusNotFound)
		refused(t, remove(w.author, w.a.Org, uuid.New()), http.StatusNotFound)
	})
}

type candidatesResponse struct {
	Candidates []keys.Candidate
	LinkedTo   string // S29: the text of the question its key already links to, or ""
}

func TestSameAsSearch(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	platformQuestion(t, w, "smoker", "Do you smoke?", smokerElement)
	// Trust A has published "Do you smoke?" (q_smokes, Yes/No) in Workflow HQ; trust B the same.
	src := w.fx.PublishedHQ(t, w.a, w.alex, "workflow.json")
	w.fx.PublishedHQ(t, w.b, w.bea, "workflow.json")
	w.other.Do("POST", libraryPath(w.b.Org, ""), map[string]any{"element": json.RawMessage(`{"type":"radiogroup","name":"q_b_smoke","title":"Smoking (trust B)","choices":["Yes"]}`)}).
		Want(t, http.StatusCreated)
	mine := w.fx.DraftHQ(t, w.a, w.alex, "workflow.json")
	setContent(t, w, mine.Chapters[1], page(`{"type":"radiogroup","name":"q_new","title":"Do you currently smoke?","choices":[{"value":"o_a","text":"Option 1"}]}`))
	question := json.RawMessage(`{"type":"radiogroup","name":"q_new","title":"Do you currently smoke?","choices":[{"value":"o_a","text":"Option 1"}]}`)
	search := func(c *apitest.Client, h apitest.Hospital, hq apitest.HQ, body any) *apitest.Resp {
		return c.Do("POST", at(h, "/questionnaires/"+hq.ID.String()+"/question-keys/search"), body)
	}

	t.Run("happy path: the trust's published keys and the library, compatible or not, with reasons", func(t *testing.T) {
		r := search(w.author, w.a, mine, map[string]any{"question": question, "q": "smok"})
		r.Want(t, http.StatusOK)
		var got candidatesResponse
		r.Decode(t, &got)
		byKey := map[string]keys.Candidate{}
		for _, c := range got.Candidates {
			byKey[c.Key] = c
		}
		if s := byKey["smoker"]; !s.Compatible || !s.Library || !slices.Equal(s.Sources, []string{"Library"}) || !strings.Contains(string(s.Element), `"ex"`) {
			t.Errorf("smoker = %+v", s)
		}
		// q_smokes is in this HQ (the draft is a copy of Workflow HQ), so it can't be adopted again.
		if s := byKey["q_smokes"]; s.Compatible || !s.InThisHQ || !slices.Equal(s.Sources, []string{src.Name}) || !strings.Contains(s.Reason, "“About you”") {
			t.Errorf("q_smokes = %+v", s)
		}
		if _, ok := byKey["q_b_smoke"]; ok {
			t.Error("trust B's library question was offered")
		}
		if _, ok := byKey["q_new"]; ok {
			t.Error("the question's own key was offered")
		}
	})
	t.Run("happy path: a question already linked says to what, so the window can warn (S29)", func(t *testing.T) {
		for _, c := range []struct {
			question, want string
		}{
			{string(question), ""}, // a new question: no warning
			// From a published HQ (this questionnaire's own earlier version is one): that question's text.
			{`{"type":"radiogroup","name":"q_smokes","title":"Smoking now?","choices":[{"value":"o_yes","text":"Yes"},{"value":"o_no","text":"No"}]}`, "Do you smoke?"},
			// From the library: the library's text, not this question's.
			{`{"type":"radiogroup","name":"smoker","title":"Smoking status?","choices":[{"value":"never","text":"Never"}]}`, "Do you smoke?"},
		} {
			r := search(w.author, w.a, mine, map[string]any{"question": json.RawMessage(c.question), "q": "zzz-nothing"})
			r.Want(t, http.StatusOK)
			var got candidatesResponse
			r.Decode(t, &got)
			if got.LinkedTo != c.want || !strings.Contains(string(r.Body), `"linkedTo":`) {
				t.Errorf("%s: linkedTo %q, want %q", c.question, got.LinkedTo, c.want)
			}
		}
		// Trust B's own library question means nothing to trust A.
		r := search(w.author, w.a, mine, map[string]any{"question": json.RawMessage(`{"type":"radiogroup","name":"q_b_smoke","title":"x","choices":["Yes"]}`)})
		var got candidatesResponse
		r.Decode(t, &got)
		if got.LinkedTo != "" {
			t.Errorf("trust B's library key: linkedTo %q", got.LinkedTo)
		}
	})
	t.Run("happy path: an incompatible kind is offered greyed, with its reason", func(t *testing.T) {
		r := search(w.author, w.a, mine, map[string]any{"question": json.RawMessage(`{"type":"text","name":"q_new","title":"x"}`), "q": "do you smoke"})
		r.Want(t, http.StatusOK)
		var got candidatesResponse
		r.Decode(t, &got)
		i := slices.IndexFunc(got.Candidates, func(c keys.Candidate) bool { return c.Key == "smoker" })
		if i < 0 || got.Candidates[i].Compatible || got.Candidates[i].Reason != "Its answer is single choice; this question's is text." {
			t.Errorf("candidates = %+v", got.Candidates)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, search(w.anon, w.a, mine, map[string]any{"question": question}), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician; a super clinician of one hospital can't change the trust's questionnaire", func(t *testing.T) {
		refused(t, search(w.clin, w.a, mine, map[string]any{"question": question}), http.StatusForbidden)
		refused(t, search(w.anon.LoginAs(t, w.fx.User(t, w.a, "super_clinician")), w.a, mine, map[string]any{"question": question}), http.StatusForbidden)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, search(w.other, w.a, mine, map[string]any{"question": question}), http.StatusNotFound)
		refused(t, search(w.other, w.b, mine, map[string]any{"question": question}), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, q := range []string{`"x"`, `{"type":"text"}`, `{"type":"html","name":"q_h"}`, `{"type":"panel","name":"g_x","elements":[{"type":"text","name":"q_y"}]}`} {
			refused(t, search(w.author, w.a, mine, map[string]any{"question": json.RawMessage(q)}), http.StatusBadRequest)
		}
		refused(t, search(w.author, w.a, mine, "nope"), http.StatusBadRequest)
		refused(t, w.author.Do("POST", at(w.a, "/questionnaires/abc/question-keys/search"), map[string]any{"question": question}), http.StatusNotFound)
	})
}

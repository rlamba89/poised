package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/keys"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// Question keys (plan C4, session S23). Go finds where the trust uses a key by walking the JSON of
// its published and retired Question Sets on each request, with no table of keys (S23 answers, 1).
//   - GET …/questionnaires/{qid}/question-keys: the editor's "Pre-fills from N other HQs" label
//     (PRE-05), and the key problems its Publish window shows before the author clicks.
//   - publishQuestionnaire refuses a key used twice in the HQ, or a keyed question whose answer
//     (kind, unit, options, rows, scale, columns) differs from how it was published anywhere in the
//     trust (keyProblems).

// trustVersion is one published or retired version of one of the trust's questionnaires.
type trustVersion struct {
	keys.Version
	VersionID uuid.UUID
	VersionNo int
}

// trustVersions walks every published and retired version of the trust's questionnaires, newest
// version first within each questionnaire. Another trust's are never read.
func trustVersions(ctx context.Context, q db.Querier, org uuid.UUID) ([]trustVersion, error) {
	rows, err := q.ListTrustPublishedContents(ctx, org)
	if err != nil {
		return nil, err
	}
	var out []trustVersion
	for _, r := range rows {
		if len(out) == 0 || out[len(out)-1].VersionID != r.VersionID {
			out = append(out, trustVersion{
				Version:   keys.Version{QuestionnaireID: r.QuestionnaireID.String(), Questionnaire: r.QuestionnaireName},
				VersionID: r.VersionID, VersionNo: int(r.VersionNo),
			})
		}
		qs, err := keys.WalkSets([]keys.Set{{Name: r.Name, Audience: r.Audience, Content: r.Content}})
		if err != nil {
			return nil, err
		}
		v := &out[len(out)-1]
		v.Questions = append(v.Questions, qs...)
	}
	return out, nil
}

// versionQuestions walks one version's Question Sets in order, read with ListVersionContents.
func versionQuestions(rows []db.ListVersionContentsRow) ([]keys.Placed, error) {
	sets := make([]keys.Set, len(rows))
	for i, r := range rows {
		sets[i] = keys.Set{Name: r.Name, Audience: r.Audience, Content: r.Content}
	}
	return keys.WalkSets(sets)
}

// publishedQuestions is, for each of these keys, its question in the newest version of each of
// the trust's questionnaires that holds it (the questionnaire being edited included).
func publishedQuestions(versions []trustVersion, names map[string]bool) map[string][]keys.Published {
	out := map[string][]keys.Published{}
	taken := map[[2]string]bool{} // (questionnaire, key) already found in a newer version
	for _, v := range versions {
		for _, q := range v.Questions {
			k := [2]string{v.QuestionnaireID, q.Name}
			if !names[q.Name] || taken[k] {
				continue
			}
			taken[k] = true
			out[q.Name] = append(out[q.Name], keys.Published{Questionnaire: v.Questionnaire, VersionNo: v.VersionNo, Question: q.Question})
		}
	}
	return out
}

func namesOf(qs []keys.Placed) map[string]bool {
	out := map[string]bool{}
	for _, q := range qs {
		out[q.Name] = true
	}
	return out
}

// keyProblems are the key checks that block publishing a version (S23 answers, 2 and 3, and
// "after verification 1"): a key used twice in it, or a keyed question whose answer differs from
// how it was published anywhere in the trust.
func keyProblems(draft []keys.Placed, versions []trustVersion) []keys.Problem {
	problems := keys.Duplicates(draft)
	if problems == nil {
		problems = []keys.Problem{}
	}
	return append(problems, keys.LockProblems(draft, publishedQuestions(versions, namesOf(draft)))...)
}

// questionKeys is GET …/questionnaires/{qid}/question-keys, for the editor (super clinicians and
// above; no patient data, so not audited). For the questionnaire's latest version, as saved:
//   - questions: each patient question's matches among the trust's other questionnaires (by key,
//     else by the same wording; PRE-05), an empty list for a new question;
//   - problems: the key problems Publish would refuse (keyProblems), for a draft, so the Publish
//     window shows them before the author clicks. The editor fetches this again after every save.
func (s *server) questionKeys(w http.ResponseWriter, r *http.Request) {
	row, ok := s.loadQuestionnaire(w, r)
	if !ok {
		return
	}
	if !can(r, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins can see where questions are used.")
		return
	}
	ctx := r.Context()
	rows, err := s.q.ListVersionContents(ctx, row.VersionID)
	if err != nil {
		serverError(w, "list contents", err)
		return
	}
	mine, err := versionQuestions(rows)
	if err != nil {
		serverError(w, "walk questionnaire", err)
		return
	}
	versions, err := trustVersions(ctx, s.q, row.OrgID)
	if err != nil {
		serverError(w, "walk trust questionnaires", err)
		return
	}
	var others []keys.Version
	for _, v := range versions {
		if v.QuestionnaireID != row.ID.String() {
			others = append(others, v.Version)
		}
	}
	problems := []keys.Problem{}
	if row.Status == "draft" {
		problems = keyProblems(mine, versions)
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": keys.Matches(mine, others), "problems": problems})
}

// searchQuestionKeys is POST …/questionnaires/{qid}/question-keys/search, the editor's "Same as…"
// (S24): the keys the question in the body could take, from the trust's published and retired
// questionnaires (each key as its newest version holding it asks it; this questionnaire's too) and
// the library (the platform's and the trust's), matching `q` by text or key. Each comes with
// whether it may be adopted (keys.Compatible, and not already in this HQ) and why not. Only those
// who may change the questionnaire ask: adopting changes it.
func (s *server) searchQuestionKeys(w http.ResponseWriter, r *http.Request) {
	row, ok := s.loadQuestionnaire(w, r)
	if !ok {
		return
	}
	if !canChange(r, row.HospitalID, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins who can change this questionnaire can link its questions.")
		return
	}
	var in struct {
		Question json.RawMessage `json:"question"`
		Q        string          `json:"q"`
	}
	if !readJSON(w, r, 256<<10, &in) {
		return
	}
	var top struct {
		Type string `json:"type"`
	}
	mine, err := keys.Element(in.Question)
	if err != nil || json.Unmarshal(in.Question, &top) != nil || top.Type == "panel" || len(mine) == 0 || mine[0].Name == "" || !mine[0].Answerable {
		writeError(w, http.StatusBadRequest, "Choose a question to link.")
		return
	}
	ctx := r.Context()
	rows, err := s.q.ListVersionContents(ctx, row.VersionID)
	if err != nil {
		serverError(w, "list contents", err)
		return
	}
	here, err := versionQuestions(rows)
	if err != nil {
		serverError(w, "walk questionnaire", err)
		return
	}
	versions, err := trustVersions(ctx, s.q, row.OrgID)
	if err != nil {
		serverError(w, "walk trust questionnaires", err)
		return
	}
	library, err := s.q.ListLibraryQuestions(ctx, db.ListLibraryQuestionsParams{OrgID: uuid.NullUUID{UUID: row.OrgID, Valid: true}})
	if err != nil {
		serverError(w, "list library", err)
		return
	}
	sources, err := keySources(versions, library)
	if err != nil {
		serverError(w, "walk library", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": keys.Candidates(mine[0], here, sources, in.Q)})
}

// keySources are the trust's questionnaires, each with every key's question from its newest
// published or retired version holding it (as the lock compares them), then the library's
// questions, under the name "Library".
func keySources(versions []trustVersion, library []db.ListLibraryQuestionsRow) ([]keys.Source, error) {
	var out []keys.Source
	index := map[string]int{}     // questionnaire → its source
	taken := map[[2]string]bool{} // (questionnaire, key) found in a newer version
	for _, v := range versions {
		i, ok := index[v.QuestionnaireID]
		if !ok {
			i = len(out)
			index[v.QuestionnaireID] = i
			out = append(out, keys.Source{Name: v.Questionnaire})
		}
		for _, q := range v.Questions {
			k := [2]string{v.QuestionnaireID, q.Name}
			if taken[k] {
				continue
			}
			taken[k] = true
			out[i].Questions = append(out[i].Questions, q)
		}
	}
	lib := keys.Source{Name: "Library", Library: true}
	for _, l := range library {
		qs, err := keys.Element(l.Element)
		if err != nil {
			return nil, err
		}
		for _, q := range qs {
			lib.Questions = append(lib.Questions, keys.Placed{Question: q, Set: "Library"})
		}
	}
	return append(out, lib), nil
}

package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/content"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/keys"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// The question library (plan C4, session S24; §6 of saas-requirements.md, PNL-06 as changed):
// GET/POST /api/o/{oid}/library-questions and DELETE …/{id}.
//   - The platform's starter set (org_id NULL) is seeded (cmd/seed) and can't be changed here. A
//     platform admin may retire one (S33; content.go): it isn't listed here any more.
//   - Whole-trust super clinicians and admins, who may change the trust's questionnaires, list
//     the library, save a question from the editor to the trust's own, and remove the trust's
//     own (S24 answers 2). Nobody else sees it: it's only used while authoring.
//   - A question keeps its key and its options' values in the library, so inserting it links the
//     questionnaires it's in, for pre-fill (S25). Removing an item changes no questionnaire.
//   - No patient data, so not audited.

const (
	libraryRole     = "Only super clinicians and admins of the whole trust can use the question library."
	libraryNotFound = "Library question not found."
)

// libraryQuestion is one item as the editor lists it; the editor works out its kind from Element.
type libraryQuestion struct {
	ID            uuid.UUID       `json:"id"`
	Platform      bool            `json:"platform"`
	Key           string          `json:"key"`
	Title         string          `json:"title"`
	Element       json.RawMessage `json:"element"`
	Tags          []string        `json:"tags"`
	CreatedByName string          `json:"createdByName"`
}

func (s *server) listLibrary(w http.ResponseWriter, r *http.Request) {
	if !here(r).TrustRole.AtLeast(role.SuperClinician) {
		writeError(w, http.StatusForbidden, libraryRole)
		return
	}
	rows, err := s.q.ListLibraryQuestions(r.Context(), db.ListLibraryQuestionsParams{
		OrgID: uuid.NullUUID{UUID: orgID(r), Valid: true}, Search: strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if err != nil {
		serverError(w, "list library", err)
		return
	}
	out := make([]libraryQuestion, len(rows))
	for i, l := range rows {
		out[i] = libraryQuestion{ID: l.ID, Platform: l.Platform, Key: l.Key, Title: l.Title, Element: l.Element, Tags: l.Tags, CreatedByName: l.CreatedByName}
	}
	writeJSON(w, http.StatusOK, out)
}

// saveToLibrary is the editor's "Save to library": one question, with its key and options, into
// the trust's own library.
func (s *server) saveToLibrary(w http.ResponseWriter, r *http.Request) {
	if !here(r).TrustRole.AtLeast(role.SuperClinician) {
		writeError(w, http.StatusForbidden, libraryRole)
		return
	}
	var in struct {
		Element json.RawMessage `json:"element"`
		Tags    []string        `json:"tags"`
	}
	if !readJSON(w, r, 256<<10, &in) {
		return
	}
	element, q, msg := libraryElement(in.Element)
	if msg == "" {
		in.Tags, msg = content.Tags(in.Tags) // the content library's rule (S33): the same limits
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx, org := r.Context(), orgID(r)
	var id uuid.UUID
	ok := s.inTx(w, r, func(tx *db.Queries) bool {
		// The same lock as publishing, so a publish can't change the key's answers under the check.
		if err := tx.LockOrgQuestionnaires(ctx, org); err != nil {
			serverError(w, "lock", err)
			return false
		}
		taken, err := tx.LibraryKeyTaken(ctx, db.LibraryKeyTakenParams{Key: q.Name, OrgID: uuid.NullUUID{UUID: org, Valid: true}})
		if err != nil {
			serverError(w, "library key", err)
			return false
		}
		if taken {
			writeError(w, http.StatusConflict, fmt.Sprintf("The library already has a question with the ID %s.", q.Name))
			return false
		}
		// The library's copy must be the question as the trust published it, or every questionnaire
		// it's inserted into would be refused at publishing (S23's lock).
		versions, err := trustVersions(ctx, tx, org)
		if err != nil {
			serverError(w, "walk trust questionnaires", err)
			return false
		}
		// A repeating list's row question is answered in the list's rows: on its own, the same key
		// would mean a different answer (test report 1, found 1). Save the repeating group instead.
		for _, v := range versions {
			for _, p := range v.Questions {
				if p.Name == q.Name && p.InList != "" {
					writeError(w, http.StatusConflict, fmt.Sprintf("“%s” is a question of a repeating list in “%s”, so it can't go in the library on its own. Save the repeating group instead.", q.Label(), v.Questionnaire))
					return false
				}
			}
		}
		// Retired starter questions too (S33): their keys are still the platform's.
		library, err := tx.ListLibraryQuestions(ctx, db.ListLibraryQuestionsParams{OrgID: uuid.NullUUID{UUID: org, Valid: true}, WithRetired: true})
		if err != nil {
			serverError(w, "list library", err)
			return false
		}
		for _, l := range library {
			inside, err := keys.Element(l.Element)
			if err != nil {
				serverError(w, "walk library", err)
				return false
			}
			if slices.ContainsFunc(inside, func(x keys.Question) bool { return x.Name == q.Name }) {
				writeError(w, http.StatusConflict, fmt.Sprintf("The library already has a question with the ID %s, in “%s”.", q.Name, l.Title))
				return false
			}
		}
		mine := []keys.Placed{{Question: q, Set: "Library"}}
		if p := keys.LockProblems(mine, publishedQuestions(versions, namesOf(mine))); len(p) > 0 {
			writeError(w, http.StatusConflict, "This question isn't the same as the trust's published one, so it can't go in the library. "+p[0].Message)
			return false
		}
		id, err = tx.CreateLibraryQuestion(ctx, db.CreateLibraryQuestionParams{
			OrgID: uuid.NullUUID{UUID: org, Valid: true}, Key: q.Name, Title: q.Title, Element: element, Tags: in.Tags,
			CreatedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
		})
		if isUniqueViolation(err) { // saved at the same moment by someone else
			writeError(w, http.StatusConflict, fmt.Sprintf("The library already has a question with the ID %s.", q.Name))
			return false
		}
		if err != nil {
			serverError(w, "save library question", err)
			return false
		}
		return true
	})
	if ok {
		writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
	}
}

// libraryElement checks a question for the library and returns the JSON to keep (without its
// conditions, which name questions of its own questionnaire, nor where "Same as…" linked it from)
// and the question; or a plain-language problem.
func libraryElement(raw json.RawMessage) (json.RawMessage, keys.Question, string) {
	var el map[string]json.RawMessage
	if json.Unmarshal(raw, &el) != nil || el == nil {
		return nil, keys.Question{}, "Choose a question to save."
	}
	qs, err := keys.Element(raw)
	switch {
	case err != nil:
		return nil, keys.Question{}, "Choose a question to save."
	case len(qs) == 0 || qs[0].Type == "" || string(el["type"]) == `"panel"`:
		return nil, keys.Question{}, "Only a single question can be saved to the library, not a group."
	case qs[0].Name == "":
		return nil, keys.Question{}, "The question has no ID."
	case !qs[0].Answerable:
		return nil, keys.Question{}, "Information blocks, calculations and the profile hold no answer, so they can't go in the library."
	case keys.OneOff(qs[0].Type):
		return nil, keys.Question{}, "Signatures, file uploads and the BMI block are never pre-filled, so they don't go in the library."
	}
	for _, k := range []string{"visibleIf", "enableIf", "requiredIf", "linkedFrom"} {
		delete(el, k)
	}
	out, err := json.Marshal(el)
	if err != nil {
		return nil, keys.Question{}, "Choose a question to save."
	}
	return out, qs[0], ""
}

// removeFromLibrary removes one of the trust's own questions. Questionnaires it was inserted into
// keep their copies, key and all.
func (s *server) removeFromLibrary(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "lid", libraryNotFound)
	if !ok {
		return
	}
	if !here(r).TrustRole.AtLeast(role.SuperClinician) {
		writeError(w, http.StatusForbidden, libraryRole)
		return
	}
	org := uuid.NullUUID{UUID: orgID(r), Valid: true}
	owner, err := s.q.GetLibraryQuestionOwner(r.Context(), db.GetLibraryQuestionOwnerParams{ID: id, OrgID: org})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, libraryNotFound)
		return
	}
	if err != nil {
		serverError(w, "library owner", err)
		return
	}
	if !owner.Valid {
		writeError(w, http.StatusForbidden, "The platform's starter questions can't be removed.")
		return
	}
	if _, err := s.q.DeleteLibraryQuestion(r.Context(), db.DeleteLibraryQuestionParams{ID: id, OrgID: org}); err != nil {
		serverError(w, "remove library question", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

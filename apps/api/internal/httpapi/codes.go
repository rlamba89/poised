package httpapi

import (
	"net/http"
	"strings"

	"github.com/rlamba89/poised/apps/api/internal/asaguide"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

var codeSets = map[string]bool{"SNOMED": true, "ICD10": true}

// codeResult is a code as the search sends it, with its choices from the ASA guide (S39).
type codeResult struct {
	db.SearchCodesRow
	AsaChoices []asaguide.Choice `json:"asaChoices"`
}

// searchCodes is CLN-09: up to 20 active codes in one code set, each with the examples the ASA
// guide gives for it (S39), which the editor offers as the disclosure's ASA grade.
func (s *server) searchCodes(w http.ResponseWriter, r *http.Request) {
	set := r.URL.Query().Get("set")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if !codeSets[set] {
		writeError(w, http.StatusBadRequest, "Choose a code set: SNOMED or ICD10.")
		return
	}
	if q == "" {
		writeJSON(w, http.StatusOK, []codeResult{})
		return
	}
	rows, err := s.q.SearchCodes(r.Context(), db.SearchCodesParams{CodeSet: set, Query: q})
	if err != nil {
		serverError(w, "search codes", err)
		return
	}
	out := make([]codeResult, len(rows))
	for i, row := range rows {
		out[i] = codeResult{SearchCodesRow: row, AsaChoices: asaguide.ChoicesFor(row.CodeSet, row.Code)}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) listCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListCategories(r.Context())
	if err != nil {
		serverError(w, "list categories", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

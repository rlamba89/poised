package httpapi

import (
	"net/http"
	"strings"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

var codeSets = map[string]bool{"SNOMED": true, "ICD10": true}

// searchCodes is CLN-09: up to 20 active codes in one code set.
func (s *server) searchCodes(w http.ResponseWriter, r *http.Request) {
	set := r.URL.Query().Get("set")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if !codeSets[set] {
		writeError(w, http.StatusBadRequest, "Choose a code set: SNOMED or ICD10.")
		return
	}
	if q == "" {
		writeJSON(w, http.StatusOK, []db.SearchCodesRow{})
		return
	}
	rows, err := s.q.SearchCodes(r.Context(), db.SearchCodesParams{CodeSet: set, Query: q})
	if err != nil {
		serverError(w, "search codes", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *server) listCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListCategories(r.Context())
	if err != nil {
		serverError(w, "list categories", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

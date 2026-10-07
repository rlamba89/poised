package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

const (
	optionListNotFound = "Option list not found."
	maxOptionListBytes = 256 << 10
	maxOptions         = 500
)

// optionListInput is what authors save: a name and the options, without option IDs
// (each question that uses the list gets its own IDs, OPT-02).
type optionListInput struct {
	Name    string          `json:"name"`
	Options json.RawMessage `json:"options"`
}

// validate trims the name and checks the options are 1 to 500 objects, each with a label.
func (in *optionListInput) validate() string {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return "Enter a name for the option list."
	}
	var options []map[string]any
	if err := json.Unmarshal(in.Options, &options); err != nil {
		return "The options could not be read."
	}
	if len(options) == 0 || len(options) > maxOptions {
		return "An option list needs between 1 and 500 options."
	}
	for _, o := range options {
		text, _ := o["text"].(string)
		if strings.TrimSpace(text) == "" {
			return "Every option needs a label."
		}
	}
	return ""
}

func (s *server) listOptionLists(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListOptionLists(r.Context(), hospitalID(r))
	if err != nil {
		serverError(w, "list option lists", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *server) createOptionList(w http.ResponseWriter, r *http.Request) {
	in, ok := readOptionList(w, r)
	if !ok {
		return
	}
	id, err := s.q.CreateOptionList(r.Context(), db.CreateOptionListParams{
		HospitalID: hospitalID(r), Name: in.Name, Options: in.Options, UpdatedBy: currentUser(r).ID,
	})
	if isUniqueViolation(err) {
		writeError(w, http.StatusConflict, "An option list with this name already exists.")
		return
	}
	if err != nil {
		serverError(w, "create option list", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// updateOptionList replaces a list. Questions that already use it keep their own copy.
func (s *server) updateOptionList(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "lid", optionListNotFound)
	if !ok {
		return
	}
	in, ok := readOptionList(w, r)
	if !ok {
		return
	}
	n, err := s.q.UpdateOptionList(r.Context(), db.UpdateOptionListParams{
		ID: id, HospitalID: hospitalID(r), Name: in.Name, Options: in.Options, UpdatedBy: currentUser(r).ID,
	})
	switch {
	case isUniqueViolation(err):
		writeError(w, http.StatusConflict, "An option list with this name already exists.")
	case err != nil:
		serverError(w, "update option list", err)
	case n == 0:
		writeError(w, http.StatusNotFound, optionListNotFound)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) deleteOptionList(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "lid", optionListNotFound)
	if !ok {
		return
	}
	if !atLeast(r, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins can change option lists.")
		return
	}
	n, err := s.q.DeleteOptionList(r.Context(), db.DeleteOptionListParams{ID: id, HospitalID: hospitalID(r)})
	switch {
	case err != nil:
		serverError(w, "delete option list", err)
	case n == 0:
		writeError(w, http.StatusNotFound, optionListNotFound)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// readOptionList checks the caller is a super clinician (or above) and reads a valid list, answering errors itself.
func readOptionList(w http.ResponseWriter, r *http.Request) (optionListInput, bool) {
	var in optionListInput
	if !atLeast(r, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins can change option lists.")
		return in, false
	}
	if !readJSON(w, r, maxOptionListBytes, &in) {
		return in, false
	}
	if msg := in.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return in, false
	}
	return in, true
}

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

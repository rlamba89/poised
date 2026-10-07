package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

const pageSize = 20

// listQuestionnaires is FRM-01: the hospital's questionnaires, searchable by name
// and description, newest change first, 20 per page.
func (s *server) listQuestionnaires(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	page = max(page, 1)
	rows, err := s.q.ListQuestionnaires(r.Context(), db.ListQuestionnairesParams{
		HospitalID: hospitalID(r),
		Search:     strings.TrimSpace(r.URL.Query().Get("q")),
		PageSize:   pageSize,
		PageOffset: int32((page - 1) * pageSize),
	})
	if err != nil {
		serverError(w, "list questionnaires", err)
		return
	}
	var total int64
	if len(rows) > 0 {
		total = rows[0].Total
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rows, "total": total, "page": page, "pageSize": pageSize})
}

type questionnaireInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// validate trims the fields and returns a plain-language problem, or "".
func (in *questionnaireInput) validate() string {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	switch {
	case in.Name == "":
		return "Enter a name for the questionnaire."
	case in.Description == "":
		return "Enter a description for the questionnaire."
	}
	return ""
}

const draftNameTaken = "A draft questionnaire with this name already exists."

// createQuestionnaire is FRM-04: a questionnaire and its draft version 1.
// Two drafts in one hospital can't share a name.
func (s *server) createQuestionnaire(w http.ResponseWriter, r *http.Request) {
	if !atLeast(r, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins can create questionnaires.")
		return
	}
	var in questionnaireInput
	if !readJSON(w, r, 16<<10, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx, hid, uid := r.Context(), hospitalID(r), currentUser(r).ID

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	if err := q.LockHospitalQuestionnaires(ctx, hid); err != nil {
		serverError(w, "lock", err)
		return
	}
	taken, err := q.DraftNameExists(ctx, db.DraftNameExistsParams{HospitalID: hid, Name: in.Name, ExcludeID: uuid.Nil})
	if err != nil {
		serverError(w, "check name", err)
		return
	}
	if taken {
		writeError(w, http.StatusConflict, draftNameTaken)
		return
	}
	id, err := q.CreateQuestionnaire(ctx, db.CreateQuestionnaireParams{HospitalID: hid, Name: in.Name, Description: in.Description, CreatedBy: uid})
	if err != nil {
		serverError(w, "create questionnaire", err)
		return
	}
	if _, err := q.CreateVersion(ctx, db.CreateVersionParams{QuestionnaireID: id, VersionNo: 1, UpdatedBy: uid}); err != nil {
		serverError(w, "create version", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// loadQuestionnaire fetches {qid} within the hospital, answering 404 itself.
func (s *server) loadQuestionnaire(w http.ResponseWriter, r *http.Request) (db.GetQuestionnaireRow, bool) {
	qid, ok := pathID(w, r, "qid", "Questionnaire not found.")
	if !ok {
		return db.GetQuestionnaireRow{}, false
	}
	row, err := s.q.GetQuestionnaire(r.Context(), db.GetQuestionnaireParams{ID: qid, HospitalID: hospitalID(r)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Questionnaire not found.")
		return row, false
	}
	if err != nil {
		serverError(w, "get questionnaire", err)
		return row, false
	}
	return row, true
}

func (s *server) getQuestionnaire(w http.ResponseWriter, r *http.Request) {
	row, ok := s.loadQuestionnaire(w, r)
	if !ok {
		return
	}
	chapters, err := s.q.ListChapters(r.Context(), row.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questionnaire": row, "chapters": chapters})
}

// deleteQuestionnaire is FRM-06: drafts only, by their creator or an admin.
func (s *server) deleteQuestionnaire(w http.ResponseWriter, r *http.Request) {
	row, ok := s.loadQuestionnaire(w, r)
	if !ok {
		return
	}
	if row.CreatedBy != currentUser(r).ID && !atLeast(r, role.Admin) {
		writeError(w, http.StatusForbidden, "Only the person who created this questionnaire or an admin can delete it.")
		return
	}
	published, err := s.q.HasNonDraftVersion(r.Context(), row.ID)
	if err != nil {
		serverError(w, "check versions", err)
		return
	}
	if published {
		writeError(w, http.StatusConflict, "Only drafts can be deleted. Published versions can be retired instead.")
		return
	}
	if err := s.q.DeleteQuestionnaire(r.Context(), db.DeleteQuestionnaireParams{ID: row.ID, HospitalID: row.HospitalID}); err != nil {
		serverError(w, "delete questionnaire", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// publishQuestionnaire publishes the latest version, which must be a draft (LCY-01). There is
// no sign-off yet (SGN comes later). The browser has already checked logic problems and test
// cases (plan 2.1). `versionId` is the version it checked, so a newer one isn't published by mistake.
func (s *server) publishQuestionnaire(w http.ResponseWriter, r *http.Request) {
	if !atLeast(r, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins can publish questionnaires.")
		return
	}
	row, ok := s.loadQuestionnaire(w, r)
	if !ok {
		return
	}
	var in struct {
		VersionID uuid.UUID `json:"versionId"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	if in.VersionID != row.VersionID || row.Status != "draft" {
		writeError(w, http.StatusConflict, "This version has changed since you opened it. Reload and try again.")
		return
	}
	chapters, err := s.q.ListChapters(r.Context(), row.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	if len(chapters) == 0 {
		writeError(w, http.StatusBadRequest, "Add at least one Question Set before publishing.")
		return
	}
	n, err := s.q.PublishVersion(r.Context(), db.PublishVersionParams{VersionID: row.VersionID, UpdatedBy: currentUser(r).ID})
	if err != nil {
		serverError(w, "publish", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "This version has changed since you opened it. Reload and try again.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// createVersion starts a new draft from the latest published version (LCY-06/07). Its chapters
// are copied unchanged, so stable IDs and test cases carry over.
func (s *server) createVersion(w http.ResponseWriter, r *http.Request) {
	if !atLeast(r, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins can create versions.")
		return
	}
	row, ok := s.loadQuestionnaire(w, r)
	if !ok {
		return
	}
	if row.Status == "draft" {
		writeError(w, http.StatusConflict, "This questionnaire already has a draft.")
		return
	}
	ctx := r.Context()
	taken, err := s.q.DraftNameExists(ctx, db.DraftNameExistsParams{HospitalID: row.HospitalID, Name: row.Name, ExcludeID: row.ID})
	if err != nil {
		serverError(w, "check name", err)
		return
	}
	if taken {
		writeError(w, http.StatusConflict, draftNameTaken)
		return
	}
	id, err := s.q.CreateNextVersion(ctx, db.CreateNextVersionParams{FromVersionID: row.VersionID, UpdatedBy: currentUser(r).ID})
	if err != nil {
		serverError(w, "create version", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/chapter"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

var (
	iconName  = regexp.MustCompile(`^[a-z0-9-]{1,50}$`)
	audiences = map[string]bool{"patient": true, "clinician": true, "clinician_document": true}
)

const (
	notDraft         = "Only draft versions can be changed."
	chapterNotFound  = "Chapter not found."
	staleRevisionMsg = "Someone else changed this chapter since you opened it. Reload to see their changes; your latest edits were not saved."
)

// canEdit writes 403/409 itself and reports whether the caller may change a chapter
// of a version with this status.
func canEdit(w http.ResponseWriter, r *http.Request, versionStatus string) bool {
	if !atLeast(r, role.SuperClinician) {
		writeError(w, http.StatusForbidden, "Only super clinicians and admins can change chapters.")
		return false
	}
	if versionStatus != "draft" {
		writeError(w, http.StatusConflict, notDraft)
		return false
	}
	return true
}

func (s *server) loadChapterMeta(w http.ResponseWriter, r *http.Request) (db.GetChapterMetaRow, bool) {
	cid, ok := pathID(w, r, "cid", chapterNotFound)
	if !ok {
		return db.GetChapterMetaRow{}, false
	}
	row, err := s.q.GetChapterMeta(r.Context(), db.GetChapterMetaParams{ID: cid, HospitalID: hospitalID(r)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, chapterNotFound)
		return row, false
	}
	if err != nil {
		serverError(w, "get chapter", err)
		return row, false
	}
	return row, true
}

// addChapter appends a chapter to the questionnaire's draft (FRM-07).
func (s *server) addChapter(w http.ResponseWriter, r *http.Request) {
	q, ok := s.loadQuestionnaire(w, r)
	if !ok || !canEdit(w, r, q.Status) {
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "Enter a name for the chapter.")
		return
	}
	uid := currentUser(r).ID
	id, err := s.q.CreateChapter(r.Context(), db.CreateChapterParams{VersionID: q.VersionID, Name: in.Name, UpdatedBy: uid})
	if err != nil {
		serverError(w, "create chapter", err)
		return
	}
	s.touch(r, q.VersionID)
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// updateChapter renames, describes, and sets the icon and audience (FRM-07).
func (s *server) updateChapter(w http.ResponseWriter, r *http.Request) {
	c, ok := s.loadChapterMeta(w, r)
	if !ok || !canEdit(w, r, c.VersionStatus) {
		return
	}
	// Fields left out keep their current value.
	in := struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Icon        *string `json:"icon"`
		Audience    *string `json:"audience"`
	}{&c.Name, &c.Description, &c.Icon, &c.Audience}
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	name, desc := strings.TrimSpace(*in.Name), strings.TrimSpace(*in.Description)
	switch {
	case name == "":
		writeError(w, http.StatusBadRequest, "Enter a name for the chapter.")
		return
	case !iconName.MatchString(*in.Icon):
		writeError(w, http.StatusBadRequest, "Choose an icon from the list.")
		return
	case !audiences[*in.Audience]:
		writeError(w, http.StatusBadRequest, "Choose who the chapter is for: patient, clinician or clinician document.")
		return
	}
	err := s.q.UpdateChapterMeta(r.Context(), db.UpdateChapterMetaParams{
		ID: c.ID, Name: name, Description: desc, Icon: *in.Icon, Audience: *in.Audience, UpdatedBy: currentUser(r).ID,
	})
	if err != nil {
		serverError(w, "update chapter", err)
		return
	}
	s.touch(r, c.VersionID)
	w.WriteHeader(http.StatusNoContent)
}

// reorderChapters takes every chapter id of the draft in the new order.
func (s *server) reorderChapters(w http.ResponseWriter, r *http.Request) {
	q, ok := s.loadQuestionnaire(w, r)
	if !ok || !canEdit(w, r, q.Status) {
		return
	}
	var order []uuid.UUID
	if !readJSON(w, r, 64<<10, &order) {
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return
	}
	defer tx.Rollback(ctx)
	qtx := db.New(tx)
	current, err := qtx.ListChapters(ctx, q.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	if !sameIDs(current, order) {
		writeError(w, http.StatusConflict, "The chapter list has changed. Reload and try again.")
		return
	}
	for i, id := range order {
		if err := qtx.SetChapterPosition(ctx, db.SetChapterPositionParams{ID: id, VersionID: q.VersionID, Position: int32(i + 1)}); err != nil {
			serverError(w, "set position", err)
			return
		}
	}
	if err := qtx.TouchVersion(ctx, db.TouchVersionParams{ID: q.VersionID, UpdatedBy: currentUser(r).ID}); err != nil {
		serverError(w, "touch version", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sameIDs reports whether order is a permutation of the current chapter ids.
func sameIDs(current []db.ListChaptersRow, order []uuid.UUID) bool {
	if len(current) != len(order) {
		return false
	}
	seen := make(map[uuid.UUID]bool, len(order))
	for _, id := range order {
		seen[id] = true
	}
	for _, c := range current {
		if !seen[c.ID] {
			return false
		}
	}
	return len(seen) == len(order)
}

func (s *server) deleteChapter(w http.ResponseWriter, r *http.Request) {
	c, ok := s.loadChapterMeta(w, r)
	if !ok || !canEdit(w, r, c.VersionStatus) {
		return
	}
	if err := s.q.DeleteChapter(r.Context(), c.ID); err != nil {
		serverError(w, "delete chapter", err)
		return
	}
	s.touch(r, c.VersionID)
	w.WriteHeader(http.StatusNoContent)
}

// getChapter returns metadata, the SurveyJS JSON and its revision.
func (s *server) getChapter(w http.ResponseWriter, r *http.Request) {
	cid, ok := pathID(w, r, "cid", chapterNotFound)
	if !ok {
		return
	}
	row, err := s.q.GetChapter(r.Context(), db.GetChapterParams{ID: cid, HospitalID: hospitalID(r)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, chapterNotFound)
		return
	}
	if err != nil {
		serverError(w, "get chapter", err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

// saveContent stores the SurveyJS JSON if `revision` is still current, else 409 (LCY-04).
func (s *server) saveContent(w http.ResponseWriter, r *http.Request) {
	c, ok := s.loadChapterMeta(w, r)
	if !ok || !canEdit(w, r, c.VersionStatus) {
		return
	}
	var in struct {
		Content  json.RawMessage `json:"content"`
		Revision int32           `json:"revision"`
	}
	if !readJSON(w, r, chapter.MaxContentBytes, &in) {
		return
	}
	if err := chapter.ValidateContent(in.Content); err != nil {
		writeError(w, http.StatusBadRequest, "The chapter content is not a valid form definition.")
		return
	}
	rev, err := s.q.SaveChapterContent(r.Context(), db.SaveChapterContentParams{
		ID: c.ID, Content: in.Content, Revision: in.Revision, UpdatedBy: currentUser(r).ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, staleRevisionMsg)
		return
	}
	if err != nil {
		serverError(w, "save content", err)
		return
	}
	s.touch(r, c.VersionID)
	writeJSON(w, http.StatusOK, map[string]int32{"revision": rev})
}

// touch records who last changed the version (FRM-01 "last changed"). Failure is logged only.
func (s *server) touch(r *http.Request, versionID uuid.UUID) {
	if err := s.q.TouchVersion(r.Context(), db.TouchVersionParams{ID: versionID, UpdatedBy: currentUser(r).ID}); err != nil {
		log.Printf("touch version: %v", err)
	}
}

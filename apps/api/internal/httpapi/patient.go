package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/chapter"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// The patient's routes (plan-workflow.md Step 3) have no sign-in: the token in their link is the key.

var tokenFormat = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

const linkNotFound = "This link doesn't work. Please check it, or contact the hospital."

func (s *server) loadByToken(w http.ResponseWriter, r *http.Request) (db.GetEpisodeByTokenRow, bool) {
	token := r.PathValue("token")
	if !tokenFormat.MatchString(token) {
		writeError(w, http.StatusNotFound, linkNotFound)
		return db.GetEpisodeByTokenRow{}, false
	}
	e, err := s.q.GetEpisodeByToken(r.Context(), token)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, linkNotFound)
		return e, false
	}
	if err != nil {
		serverError(w, "episode by token", err)
		return e, false
	}
	a := auditOf(r) // the token names an episode: from here on the request is in its trust's log
	a.org, a.hospital = e.OrgID, e.HospitalID
	a.touched(e.PatientID, e.ID)
	if !e.Open { // the trust or hospital is suspended (ORG-03): the answers are kept
		writeError(w, http.StatusForbidden, "This questionnaire isn't available at the moment. Please contact the hospital.")
		return e, false
	}
	return e, true
}

type patientChapter struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	Content     json.RawMessage `json:"content"`
}

// patientHQ returns the patient's Question Sets, stripped for patients, and their answers so far.
func (s *server) patientHQ(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadByToken(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := s.q.ListVersionChapters(ctx, e.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	chapters := []patientChapter{}
	for _, c := range rows {
		if c.Audience != "patient" {
			continue
		}
		content, _, err := chapter.ForPatient(c.Content)
		if err != nil {
			serverError(w, "strip chapter", err)
			return
		}
		chapters = append(chapters, patientChapter{ID: c.ID, Name: c.Name, Description: c.Description, Icon: c.Icon, Content: content})
	}
	saved, err := s.q.ListEpisodeAnswers(ctx, e.ID)
	if err != nil {
		serverError(w, "list answers", err)
		return
	}
	answers := map[uuid.UUID]json.RawMessage{}
	revisions := map[uuid.UUID]int32{} // what the next save of each set sends back (S02a)
	for _, a := range saved {
		if a.Actor == "patient" {
			answers[a.ChapterID] = a.Data
			revisions[a.ChapterID] = a.Revision
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"patient":           map[string]any{"firstName": e.FirstName, "lastName": e.LastName, "dateOfBirth": e.DateOfBirth, "sex": e.Sex},
		"questionnaireName": e.QuestionnaireName,
		"submitted":         e.PatientSubmittedAt.Valid,
		"chapters":          chapters,
		"answers":           answers,
		"revisions":         revisions,
	})
}

// savePatientAnswers autosaves one Question Set's answers (SurveyJS data). Only answers to the
// patient's own questions are kept, and nothing is saved once the HQ has been submitted. It
// answers with the new revision, or 409 if the revision sent is stale (the link is open twice).
func (s *server) savePatientAnswers(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadByToken(w, r)
	if !ok {
		return
	}
	cid, ok := pathID(w, r, "cid", chapterNotFound)
	if !ok {
		return
	}
	c, err := s.q.GetVersionChapter(r.Context(), db.GetVersionChapterParams{ID: cid, VersionID: e.VersionID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && c.Audience != "patient") {
		writeError(w, http.StatusNotFound, chapterNotFound)
		return
	}
	if err != nil {
		serverError(w, "get chapter", err)
		return
	}
	var in answersInput
	if !readJSON(w, r, chapter.MaxContentBytes, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	_, names, err := chapter.ForPatient(c.Content)
	if err != nil {
		serverError(w, "strip chapter", err)
		return
	}
	data, dropped := chapter.FilterAnswers(in.Data, names)
	if dropped {
		log.Printf("patient answers for episode %s: dropped answers to questions patients don't get", e.ID)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		serverError(w, "encode answers", err)
		return
	}
	rev, err := s.q.SavePatientAnswers(r.Context(), db.SavePatientAnswersParams{EpisodeID: e.ID, ChapterID: c.ID, Data: raw, Revision: *in.Revision})
	if errors.Is(err, pgx.ErrNoRows) {
		if e.PatientSubmittedAt.Valid {
			writeError(w, http.StatusConflict, "Your answers have already been sent, so they can't be changed here.")
		} else {
			writeError(w, http.StatusConflict, staleAnswersMsg)
		}
		return
	}
	if err != nil {
		serverError(w, "save answers", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int32{"revision": rev})
}

// submitPatientHQ freezes the patient's answers and moves the episode to Ready for review
// (triage). The browser has checked every required question (plan 2.1).
func (s *server) submitPatientHQ(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadByToken(w, r)
	if !ok {
		return
	}
	n, err := s.q.SubmitPatientHQ(r.Context(), e.ID)
	if err != nil {
		serverError(w, "submit", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "Your answers have already been sent.")
		return
	}
	s.event(r, e.ID, "HQ completed by the patient", uuid.Nil)
	w.WriteHeader(http.StatusNoContent)
}

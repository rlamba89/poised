package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/chapter"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// Patient sign-in (PAT-04/05, auth-first Step 5). The link alone shows nothing: the patient
// taps Continue, then confirms their date of birth, which starts a patient session. The HQ
// routes need that session.

var tokenFormat = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

const (
	linkNotFound = "This link doesn't work. Please check it, or contact the hospital."
	linkLocked   = "This link is locked after too many tries. Please contact the hospital for a new one."
	// dobAttempts wrong dates of birth lock a link (PAT-05).
	dobAttempts = 5
)

// loadLink finds the link in {token}, answering 404 (unknown) or 403 (locked) itself.
func (s *server) loadLink(w http.ResponseWriter, r *http.Request) (db.GetLoginLinkRow, bool) {
	token := r.PathValue("token")
	if !tokenFormat.MatchString(token) {
		writeError(w, http.StatusNotFound, linkNotFound)
		return db.GetLoginLinkRow{}, false
	}
	l, err := s.q.GetLoginLink(r.Context(), auth.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, linkNotFound)
		return l, false
	}
	if err != nil {
		serverError(w, "get link", err)
		return l, false
	}
	if l.LockedAt.Valid {
		writeError(w, http.StatusForbidden, linkLocked)
		return l, false
	}
	return l, true
}

// continueLink is the patient's Continue. It checks the link without showing anything (so an
// email scanner opening the page uses nothing up) and says whether this browser is already
// signed in for the link's episode, in which case the date of birth is skipped.
func (s *server) continueLink(w http.ResponseWriter, r *http.Request) {
	l, ok := s.loadLink(w, r)
	if !ok {
		return
	}
	episode, err := s.patientSession(r)
	if err != nil && !errors.Is(err, errNoSession) {
		serverError(w, "load patient session", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"signedIn": err == nil && episode == l.EpisodeID})
}

// confirmDateOfBirth starts a patient session when the date of birth matches the link's
// patient. Each wrong one counts towards the lock.
func (s *server) confirmDateOfBirth(w http.ResponseWriter, r *http.Request) {
	l, ok := s.loadLink(w, r)
	if !ok {
		return
	}
	var in struct {
		DateOfBirth string `json:"dateOfBirth"` // YYYY-MM-DD
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	given, err := time.Parse(time.DateOnly, in.DateOfBirth)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Enter your date of birth as a day, month and year.")
		return
	}
	ctx := r.Context()
	if !l.DateOfBirth.Valid || l.DateOfBirth.Time.Format(time.DateOnly) != given.Format(time.DateOnly) {
		locked, err := s.q.RecordWrongDateOfBirth(ctx, db.RecordWrongDateOfBirthParams{ID: l.ID, MaxAttempts: dobAttempts})
		if errors.Is(err, pgx.ErrNoRows) { // locked by a try sent at the same time
			writeError(w, http.StatusForbidden, linkLocked)
			return
		}
		if err != nil {
			serverError(w, "record wrong date of birth", err)
			return
		}
		if locked {
			s.event(r, l.EpisodeID, "Patient link locked after 5 wrong dates of birth", uuid.Nil)
			writeError(w, http.StatusForbidden, linkLocked)
			return
		}
		writeError(w, http.StatusBadRequest, "That date of birth doesn't match our records. Please try again.")
		return
	}
	if err := s.q.ResetDateOfBirthTries(ctx, l.ID); err != nil {
		serverError(w, "reset tries", err)
		return
	}
	id, hash, err := auth.NewSessionID()
	if err != nil {
		serverError(w, "new session", err)
		return
	}
	err = s.q.CreatePatientSession(ctx, db.CreatePatientSessionParams{IDHash: hash, EpisodeID: l.EpisodeID, MaxAgeSeconds: s.sessionMaxAge.Seconds()})
	if err != nil {
		serverError(w, "create patient session", err)
		return
	}
	http.SetCookie(w, s.cookie(s.patientCookieName(), id, int(s.sessionMaxAge.Seconds())))
	w.WriteHeader(http.StatusNoContent)
}

// patientLogout ends the patient's session and clears its cookie.
func (s *server) patientLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.patientCookieName()); err == nil {
		if err := s.q.DeleteSession(r.Context(), auth.HashToken(c.Value)); err != nil {
			serverError(w, "delete session", err)
			return
		}
	}
	http.SetCookie(w, s.cookie(s.patientCookieName(), "", -1))
	w.WriteHeader(http.StatusNoContent)
}

// errNoSession means the request carries no live patient session.
var errNoSession = errors.New("no patient session")

// patientSession returns the episode of the request's live patient session, touching it.
func (s *server) patientSession(r *http.Request) (uuid.UUID, error) {
	c, err := r.Cookie(s.patientCookieName())
	if err != nil {
		return uuid.Nil, errNoSession
	}
	hash := auth.HashToken(c.Value)
	row, err := s.q.GetPatientSession(r.Context(), db.GetPatientSessionParams{IDHash: hash, IdleSeconds: s.sessionIdle.Seconds()})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errNoSession
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get patient session: %w", err)
	}
	if time.Since(row.LastSeenAt) > touchEvery {
		if err := s.q.TouchSession(r.Context(), hash); err != nil {
			log.Printf("touch patient session: %v", err) // the request can still go ahead
		}
	}
	return row.EpisodeID, nil
}

// loadPatientEpisode is the signed-in patient's episode, answering 401 itself without a session.
func (s *server) loadPatientEpisode(w http.ResponseWriter, r *http.Request) (db.GetPatientEpisodeRow, bool) {
	id, err := s.patientSession(r)
	if errors.Is(err, errNoSession) {
		writeError(w, http.StatusUnauthorized, "Please open the link from the hospital again.")
		return db.GetPatientEpisodeRow{}, false
	}
	if err != nil {
		serverError(w, "load patient session", err)
		return db.GetPatientEpisodeRow{}, false
	}
	e, err := s.q.GetPatientEpisode(r.Context(), id)
	if err != nil {
		serverError(w, "patient episode", err)
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

// patientHQ returns the signed-in patient's Question Sets, stripped for patients, and their
// answers so far. Their date of birth is included: they have just confirmed it.
func (s *server) patientHQ(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadPatientEpisode(w, r)
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
	for _, a := range saved {
		if a.Actor == "patient" {
			answers[a.ChapterID] = a.Data
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"patient":           map[string]any{"firstName": e.FirstName, "lastName": e.LastName, "dateOfBirth": e.DateOfBirth, "sex": e.Sex},
		"questionnaireName": e.QuestionnaireName,
		"submitted":         e.PatientSubmittedAt.Valid,
		"chapters":          chapters,
		"answers":           answers,
	})
}

// savePatientAnswers autosaves one Question Set's answers (SurveyJS data). Only answers to the
// patient's own questions are kept, and nothing is saved once the HQ has been submitted.
func (s *server) savePatientAnswers(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadPatientEpisode(w, r)
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
	var in struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if !readJSON(w, r, chapter.MaxContentBytes, &in) {
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
	n, err := s.q.SavePatientAnswers(r.Context(), db.SavePatientAnswersParams{EpisodeID: e.ID, ChapterID: c.ID, Data: raw})
	if err != nil {
		serverError(w, "save answers", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "Your answers have already been sent, so they can't be changed here.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// submitPatientHQ freezes the patient's answers and moves the episode to Ready for review
// (triage). The browser has checked every required question (plan 2.1).
func (s *server) submitPatientHQ(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadPatientEpisode(w, r)
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

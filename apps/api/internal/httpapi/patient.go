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
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/chapter"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// The patient's side (plan C2, session S14; A-4, PAT-04, PAT-05). Every session starts with a
// link from an SMS or email, then Continue (a POST, so a link scanner opening the page can't use
// it), then the patient's date of birth. Only then is a session cookie set; the link alone returns
// nothing but the hospital's name (S14 answers 1). Five wrong dates of birth lock the link.
//
// The session is a patient_sessions row tied to the invite link it was opened with (S14 answers
// 4): Sign out, a resend (the link replaced) or a lock (the link locked) ends it at once, and so
// do 30 minutes without a request. The cookie names only the session. A session opens its
// patient's episodes only. Every patient response is no-store: a shared phone's browser mustn't
// keep it.

var tokenFormat = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

const (
	linkNotFound = "This link doesn't work. Please check it, or contact the hospital."
	linkLocked   = "This link is locked because the date of birth was entered wrongly too many times. Please contact the hospital to send you a new link."
	formClosed   = "This questionnaire isn't available at the moment. Please contact the hospital."
	sessionEnded = "For your security, you've been signed out. Please open the link in your text message or email again."
	// maxDobAttempts wrong dates of birth lock a link (PAT-05).
	maxDobAttempts = 5
)

// loadLink finds the working link named by {token}, answering itself when there's none, or when
// it's locked or its trust or hospital is suspended.
func (s *server) loadLink(w http.ResponseWriter, r *http.Request) (db.GetLinkByHashRow, bool) {
	token := r.PathValue("token")
	if !tokenFormat.MatchString(token) {
		writeError(w, http.StatusNotFound, linkNotFound)
		return db.GetLinkByHashRow{}, false
	}
	l, err := s.q.GetLinkByHash(r.Context(), db.GetLinkByHashParams{TokenHash: auth.HashLinkToken(token), Now: s.clock.Now()})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, linkNotFound)
		return l, false
	}
	if err != nil {
		serverError(w, "link", err)
		return l, false
	}
	a := auditOf(r) // the link names an episode: from here on the request is in its trust's log
	a.org, a.hospital = l.OrgID, l.HospitalID
	a.touched(l.PatientID, l.EpisodeID.UUID)
	if !l.Open { // the trust or hospital is suspended (ORG-03): the answers are kept
		writeError(w, http.StatusForbidden, formClosed)
		return l, false
	}
	if l.LockedAt.Valid {
		writeLocked(w, linkLocked)
		return l, false
	}
	return l, true
}

// writeLocked is a locked link's refusal; code "locked" lets the page say what to do.
func writeLocked(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusForbidden, map[string]string{"error": msg, "code": "locked"})
}

// linkContinue is the Continue button: it says whether the link works, and returns only the
// hospital's name, for the date-of-birth screen. It changes nothing.
func (s *server) linkContinue(w http.ResponseWriter, r *http.Request) {
	l, ok := s.loadLink(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"hospital": l.HospitalName})
}

// linkDob checks the date of birth. Right: the session cookie, and the episode to open. Wrong:
// 401 with the tries left, and the link locks at the fifth. A date that can't be read isn't a try.
func (s *server) linkDob(w http.ResponseWriter, r *http.Request) {
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
	dob, err := time.Parse(time.DateOnly, in.DateOfBirth)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Enter your date of birth as a real date, for example 27 3 1985.")
		return
	}
	ctx := r.Context()
	if !sameDate(dob, l.DateOfBirth) {
		// The episode's live links share the tries, and lock together (S16 test report 1).
		n, err := s.q.FailInviteDob(ctx, db.FailInviteDobParams{ID: l.ID, MaxAttempts: maxDobAttempts})
		s.wrongDob(w, n, err, linkLocked, "")
		return
	}
	n, err := s.q.PassInviteDob(ctx, l.ID)
	if err != nil {
		serverError(w, "right date of birth", err)
		return
	}
	if n == 0 { // locked by a wrong date sent at the same moment
		writeLocked(w, linkLocked)
		return
	}
	// The record joins the person's account (or starts one), with their other records that
	// share its contacts and date of birth (S15, PAT-07); the session is the account's.
	account, err := s.joinAccount(r, l.PatientID, l.DateOfBirth)
	if err != nil {
		serverError(w, "join account", err)
		return
	}
	if !s.startSession(w, r, l.ID, account) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]uuid.UUID{"episodeId": l.EpisodeID.UUID})
}

// wrongDob answers a wrong date of birth, given the tries used so far (n) and the error from
// counting it (pgx.ErrNoRows: already locked): 401 with the tries left, or the lock at the fifth
// (locked is what the lock's message says to do). mismatch is what didn't match ("": the date of
// birth). An invite's tries are its episode's (FailInviteDob); a sign-in link's are its own
// (FailLinkDob).
func (s *server) wrongDob(w http.ResponseWriter, n int32, err error, locked, mismatch string) {
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && n >= maxDobAttempts) {
		writeLocked(w, locked)
		return
	}
	if err != nil {
		serverError(w, "wrong date of birth", err)
		return
	}
	left := maxDobAttempts - n
	tries := "tries"
	if left == 1 {
		tries = "try"
	}
	if mismatch == "" {
		mismatch = "That date of birth doesn't match our records. Please check it and try again."
	}
	writeError(w, http.StatusUnauthorized, fmt.Sprintf("%s You have %d %s left.", mismatch, left, tries))
}

// startSession starts the account's session, opened with the link, and sets its cookie.
func (s *server) startSession(w http.ResponseWriter, r *http.Request, link, account uuid.UUID) bool {
	session, err := s.q.StartPatientSession(r.Context(), db.StartPatientSessionParams{LinkID: link, AccountID: account, Now: s.clock.Now()})
	if err != nil {
		serverError(w, "start session", err)
		return false
	}
	return s.setPatientSession(w, session)
}

// sameDate compares a typed date with a stored one, by calendar date.
func sameDate(typed time.Time, stored pgtype.Date) bool {
	if !stored.Valid {
		return false
	}
	y1, m1, d1 := typed.Date()
	y2, m2, d2 := stored.Time.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// setPatientSession sets (or renews) the session's cookie: HTTP-only, sent only to the patient
// API, and Secure when the app is served over https.
func (s *server) setPatientSession(w http.ResponseWriter, session uuid.UUID) bool {
	token, err := auth.IssuePatient(s.secret, session, s.clock.Now())
	if err != nil {
		serverError(w, "patient session", err)
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.PatientCookieName, Value: token, Path: "/api/p", HttpOnly: true, SameSite: http.SameSiteStrictMode,
		Secure: s.secureCookies(), MaxAge: int(auth.PatientIdle.Seconds()),
	})
	return true
}

// sessionOf is the session named by the request's cookie, or uuid.Nil.
func (s *server) sessionOf(r *http.Request) uuid.UUID {
	c, err := r.Cookie(auth.PatientCookieName)
	if err != nil {
		return uuid.Nil
	}
	id, err := auth.VerifyPatient(s.secret, c.Value, s.clock.Now())
	if err != nil {
		return uuid.Nil
	}
	return id
}

// requirePatient lets a request through only with a live session: its cookie checks out, and its
// row (one query, which also counts the request as activity) isn't signed out, idle, or on a
// link that was replaced or locked. It renews the cookie, and puts the patient's record id in the
// context.
func (s *server) requirePatient(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.sessionOf(r)
		if session == uuid.Nil {
			writeError(w, http.StatusUnauthorized, sessionEnded)
			return
		}
		now := s.clock.Now()
		account, err := s.q.TouchPatientSession(r.Context(), db.TouchPatientSessionParams{ID: session, Now: now, IdleSince: now.Add(-auth.PatientIdle)})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, sessionEnded)
			return
		}
		if err != nil {
			serverError(w, "patient session", err)
			return
		}
		if !s.setPatientSession(w, session) {
			return
		}
		next(w, r.WithContext(withAccount(r.Context(), account)))
	}
}

// patientLogout ends the patient's session in the database, so a copy of the cookie stops
// working too, and clears the cookie. Without a session it just clears the cookie.
func (s *server) patientLogout(w http.ResponseWriter, r *http.Request) {
	if session := s.sessionOf(r); session != uuid.Nil {
		if err := s.q.EndPatientSession(r.Context(), db.EndPatientSessionParams{ID: session, Now: pgtype.Timestamptz{Time: s.clock.Now(), Valid: true}}); err != nil {
			serverError(w, "end session", err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: auth.PatientCookieName, Value: "", Path: "/api/p", HttpOnly: true, Secure: s.secureCookies(), MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

// noStore marks a patient response as never to be cached (a shared phone's browser mustn't keep
// it, S14 answers 4).
func noStore(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next(w, r)
	}
}

// form is an episode's HQ being filled in for the patient: by the patient (their session), or by
// staff on their behalf (PAT-09), when enteredBy is the staff member.
type form struct {
	db.GetEpisodeFormRow
	enteredBy uuid.UUID // uuid.Nil: the patient
}

// loadPatientForm is {eid} for the patient's session: an episode of one of their records, in any
// trust (all joined to their account, S15), or 404.
func (s *server) loadPatientForm(w http.ResponseWriter, r *http.Request) (form, bool) {
	f, ok := s.loadForm(w, r)
	if ok && !f.ownedBy(accountOf(r)) {
		writeError(w, http.StatusNotFound, episodeNotFound)
		return f, false
	}
	return f, ok
}

// ownedBy reports whether the episode's patient record has joined the account.
func (f form) ownedBy(account uuid.UUID) bool {
	return account != uuid.Nil && f.AccountID.Valid && f.AccountID.UUID == account
}

// loadStaffForm is {eid} at the staff member's hospital, or 404.
func (s *server) loadStaffForm(w http.ResponseWriter, r *http.Request) (form, bool) {
	f, ok := s.loadForm(w, r)
	if ok && f.HospitalID != hospitalID(r) {
		writeError(w, http.StatusNotFound, episodeNotFound)
		return f, false
	}
	f.enteredBy = currentUser(r).ID
	return f, ok
}

// loadForm reads {eid}; the callers check whose it is before anything is sent or changed.
func (s *server) loadForm(w http.ResponseWriter, r *http.Request) (form, bool) {
	eid, ok := pathID(w, r, "eid", episodeNotFound)
	if !ok {
		return form{}, false
	}
	e, err := s.q.GetEpisodeForm(r.Context(), eid)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, episodeNotFound)
		return form{}, false
	}
	if err != nil {
		serverError(w, "episode form", err)
		return form{}, false
	}
	f := form{GetEpisodeFormRow: e}
	if accountOf(r) == uuid.Nil && e.HospitalID != hospitalID(r) {
		return f, true // staff at another hospital: refused by loadStaffForm, and not in this trust's log
	}
	if accountOf(r) != uuid.Nil && !f.ownedBy(accountOf(r)) {
		return f, true // another person's: refused by loadPatientForm, and logged nowhere
	}
	a := auditOf(r)
	a.org, a.hospital = e.OrgID, e.HospitalID
	a.touched(e.PatientID, e.ID)
	if !e.Open { // the trust or hospital is suspended (ORG-03): the answers are kept
		writeError(w, http.StatusForbidden, formClosed)
		return f, false
	}
	return f, true
}

type patientChapter struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	Content     json.RawMessage `json:"content"`
}

// formPatient is who the HQ is for, as the form needs them: the first name for "Hello Sam", and
// the survey variables {patientName}, {patientAge} and {patientSex}. Never the date of birth
// (the answer to the sign-in check) or an identifier the patient didn't type (S14 answers 1).
type formPatient struct {
	FirstName string `json:"firstName"`
	Name      string `json:"name"`
	Age       int    `json:"age"`
	Sex       string `json:"sex"`
}

// writeForm sends the HQ's patient Question Sets, stripped for patients, with the answers so far.
func (s *server) writeForm(w http.ResponseWriter, r *http.Request, f form) {
	ctx := r.Context()
	rows, err := s.q.ListVersionChapters(ctx, f.VersionID)
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
	saved, err := s.q.ListEpisodeAnswers(ctx, f.ID)
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
		"patient": formPatient{FirstName: f.FirstName, Name: f.FirstName + " " + f.LastName,
			Age: ageOn(f.DateOfBirth, s.today(f.Timezone)), Sex: f.Sex},
		"hospital":          f.HospitalName,
		"questionnaireName": f.QuestionnaireName,
		"submitted":         f.PatientSubmittedAt.Valid,
		"chapters":          chapters,
		"answers":           answers,
		"revisions":         revisions,
	})
}

// today is the date now at the hospital (its time zone, or the deployment's: A-20).
func (s *server) today(zone *string) time.Time {
	name := s.cfg.Country.TimeZone
	if zone != nil {
		name = *zone
	}
	loc, err := time.LoadLocation(name)
	if err != nil { // checked when set; the deployment's was checked at start-up
		loc = time.UTC
	}
	return s.clock.Now().In(loc)
}

// ageOn is the age in whole years on the day today.
func ageOn(dob pgtype.Date, today time.Time) int {
	if !dob.Valid {
		return 0
	}
	y, m, d := dob.Time.Date()
	ty, tm, td := today.Date()
	age := ty - y
	if tm < m || (tm == m && td < d) {
		age--
	}
	return age
}

// saveForm autosaves one Question Set's answers (SurveyJS data). Only answers to the patient's
// own questions are kept, and nothing is saved once the HQ has been sent. It answers with the
// new revision, or 409 if the revision sent is stale (the HQ is open twice).
func (s *server) saveForm(w http.ResponseWriter, r *http.Request, f form) {
	cid, ok := pathID(w, r, "cid", chapterNotFound)
	if !ok {
		return
	}
	c, err := s.q.GetVersionChapter(r.Context(), db.GetVersionChapterParams{ID: cid, VersionID: f.VersionID})
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
		log.Printf("patient answers for episode %s: dropped answers to questions patients don't get", f.ID)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		serverError(w, "encode answers", err)
		return
	}
	rev, err := s.q.SavePatientAnswers(r.Context(), db.SavePatientAnswersParams{
		EpisodeID: f.ID, ChapterID: c.ID, Data: raw, Revision: *in.Revision,
		EnteredBy: uuid.NullUUID{UUID: f.enteredBy, Valid: f.enteredBy != uuid.Nil},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		if f.PatientSubmittedAt.Valid {
			writeError(w, http.StatusConflict, "These answers have already been sent, so they can't be changed here.")
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

// submitForm freezes the patient's answers and moves the episode to Ready for review (triage).
// The browser has checked every required question (plan 2.1).
func (s *server) submitForm(w http.ResponseWriter, r *http.Request, f form) {
	n, err := s.q.SubmitPatientHQ(r.Context(), db.SubmitPatientHQParams{
		ID: f.ID, SubmittedBy: uuid.NullUUID{UUID: f.enteredBy, Valid: f.enteredBy != uuid.Nil},
	})
	if err != nil {
		serverError(w, "submit", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "These answers have already been sent.")
		return
	}
	if f.enteredBy == uuid.Nil {
		s.event(r, f.ID, "HQ completed by the patient", uuid.Nil)
	} else {
		s.event(r, f.ID, "HQ completed on the patient's behalf", f.enteredBy)
	}
	w.WriteHeader(http.StatusNoContent)
}

// The patient's routes: GET, PUT …/answers/{cid} and POST …/submit on /api/p/episodes/{eid}.

func (s *server) patientForm(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadPatientForm(w, r); ok {
		s.writeForm(w, r, f)
	}
}

func (s *server) savePatientForm(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadPatientForm(w, r); ok {
		s.saveForm(w, r, f)
	}
}

func (s *server) submitPatientForm(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadPatientForm(w, r); ok {
		s.submitForm(w, r, f)
	}
}

// Staff completing the HQ on the patient's behalf (PAT-09), e.g. on the phone: the same form, on
// …/episodes/{eid}/patient-form. The answers are the patient's, marked "entered by" the staff member.

func (s *server) staffPatientForm(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadStaffForm(w, r); ok {
		s.writeForm(w, r, f)
	}
}

func (s *server) staffSavePatientForm(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadStaffForm(w, r); ok {
		s.saveForm(w, r, f)
	}
}

func (s *server) staffSubmitPatientForm(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadStaffForm(w, r); ok {
		s.submitForm(w, r, f)
	}
}

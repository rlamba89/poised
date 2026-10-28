package httpapi

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/chapter"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/evaluate"
	"github.com/rlamba89/poised/apps/api/internal/role"
	"github.com/rlamba89/poised/apps/api/internal/triage"
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
	linkNotFound  = "This link doesn't work. Please check it, or contact the hospital."
	linkLocked    = "This link is locked because the date of birth was entered wrongly too many times. Please contact the hospital to send you a new link."
	formClosed    = "This questionnaire isn't available at the moment. Please contact the hospital."
	formCancelled = "This questionnaire was cancelled by the hospital, so there's nothing more to fill in. Please contact the hospital if you have any questions."
	formArchived  = "This questionnaire is closed, so there's nothing more to fill in. Please contact the hospital if you have any questions."
	sessionEnded  = "For your security, you've been signed out. Please open the link in your text message or email again."
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
	// Archived since it was cancelled, the patient's form still says cancelled, as their home does
	// (S29; Rahul, 25 Oct), sent or not. Staff get the archived episode's 409 below.
	archivedCancelled := e.Status == episode.Archived && e.ArchivedFrom != nil && *e.ArchivedFrom == episode.Cancelled
	if e.Status == episode.Cancelled || (archivedCancelled && accountOf(r) != uuid.Nil) { // S20 answers 5: no form; code "cancelled" lets the page say so
		writeJSON(w, http.StatusForbidden, map[string]string{"error": formCancelled, "code": "cancelled"})
		return f, false
	}
	if e.Status == episode.Archived { // S22: read-only. The patient's form is closed; staff get the 409 of every write
		if accountOf(r) != uuid.Nil && e.PatientSubmittedAt.Valid && r.Method == http.MethodGet {
			return f, true // S27: what the patient sent stays theirs to view, read-only, as before it was archived
		}
		if accountOf(r) != uuid.Nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": formArchived, "code": "archived"})
		} else {
			writeError(w, http.StatusConflict, archivedMsg)
		}
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
// On the form's first open, sets with no answers yet are pre-filled from the patient's earlier HQs
// (S25; S28: the first open only), and the
// answers come with where each pre-filled one came from (`prefill`, by Question Set), another
// trust's included (S26; S27: always, without asking).
func (s *server) writeForm(w http.ResponseWriter, r *http.Request, f form) {
	ctx := r.Context()
	rows, err := s.q.ListVersionChapters(ctx, f.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	if err := s.prefillForm(ctx, f, auditOf(r), rows); err != nil {
		serverError(w, "pre-fill", err)
		return
	}
	saved, err := s.q.ListEpisodeAnswers(ctx, f.FormID)
	if err != nil {
		serverError(w, "list answers", err)
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
	answers := map[uuid.UUID]json.RawMessage{}
	revisions := map[uuid.UUID]int32{} // what the next save of each set sends back (S02a)
	prefilled := map[uuid.UUID]setPrefill{}
	for _, a := range saved {
		if a.Actor == "patient" {
			answers[a.ChapterID] = a.Data
			revisions[a.ChapterID] = a.Revision
			if p, ok := prefillOf(a); ok {
				prefilled[a.ChapterID] = p
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"patient": formPatient{FirstName: f.FirstName, Name: f.FirstName + " " + f.LastName,
			Age: ageOn(f.DateOfBirth, s.today(f.Timezone)), Sex: f.Sex},
		"hospital":          f.HospitalName,
		"questionnaireName": f.QuestionnaireName,
		"submitted":         f.PatientSubmittedAt.Valid,
		"archived":          f.Status == episode.Archived, // S28: the read-only View of a sent HQ says no more about a review
		"chapters":          chapters,
		"answers":           answers,
		"revisions":         revisions,
		"prefill":           prefilled,
	})
}

// today is the date now at the hospital (its time zone, or the deployment's: A-20).
func (s *server) today(zone *string) time.Time {
	return s.inZone(zone, s.clock.Now())
}

// inZone is t at the hospital (its time zone, or the deployment's: A-20).
func (s *server) inZone(zone *string, t time.Time) time.Time {
	name := s.cfg.Country.TimeZone
	if zone != nil {
		name = *zone
	}
	loc, err := time.LoadLocation(name)
	if err != nil { // checked when set; the deployment's was checked at start-up
		loc = time.UTC
	}
	return t.In(loc)
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
		FormID: f.FormID, ChapterID: c.ID, Data: raw, Revision: *in.Revision, Cleared: in.Cleared,
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

// submitForm submits the HQ form, freezing the patient's answers, and moves the episode to Ready
// for review (triage), through the transition table (S20), recording the change. Then the HQ's
// rules may move it straight on (S42, autoTriage).
// The browser checks every required question (plan 2.1); Go also holds it back while a required
// video tick every patient sees isn't ticked (S34), and while the evaluate service says a set the
// patient's form shows isn't complete (S44: nothing is frozen, the status is unchanged). What's
// frozen is only what the patient's form showed (S44a, keepSent). If the service can't be reached
// the send goes through as before, with every answer kept (S41's fail-safe).
func (s *server) submitForm(w http.ResponseWriter, r *http.Request, f form) {
	who := role.None // the patient
	if f.enteredBy != uuid.Nil {
		who = here(r).Role
	}
	if _, err := episode.Check(f.Status, episode.ReadyForReview, episode.Submit, who); err != nil {
		writeError(w, http.StatusConflict, "These answers have already been sent.")
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	chapters, err := q.ListVersionChapters(ctx, f.VersionID)
	if err != nil {
		serverError(w, "list chapters", err)
		return
	}
	if msg, err := uncheckedTicks(ctx, q, f, chapters); err != nil {
		serverError(w, "check ticks", err)
		return
	} else if msg != "" {
		writeError(w, http.StatusConflict, msg)
		return
	}
	n, err := q.SubmitPatientHQ(ctx, db.SubmitPatientHQParams{
		FormID: f.FormID, SubmittedBy: uuid.NullUUID{UUID: f.enteredBy, Valid: f.enteredBy != uuid.Nil},
	})
	if err != nil {
		serverError(w, "submit", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "These answers have already been sent.")
		return
	}
	note := "HQ completed by the patient"
	if f.enteredBy != uuid.Nil {
		note = "HQ completed on the patient's behalf"
	}
	if err := recordStatus(ctx, q, f.ID, episode.Submit, episode.HQNotComplete, episode.ReadyForReview, "", f.enteredBy, note); err != nil {
		serverError(w, "status history", err)
		return
	}
	// One call does both (S44): the check here, and the result stored and acted on below. It's made
	// after the freeze, in the same transaction, so the answers checked are the ones frozen; a
	// refusal rolls it all back.
	res, err := s.evaluateSent(ctx, q, f, chapters)
	if err != nil {
		serverError(w, "store outputs", err)
		return
	}
	if res != nil {
		if msg := unfinishedSets(*res, chapters); msg != "" {
			writeError(w, http.StatusConflict, msg)
			return
		}
		if err := keepSent(ctx, q, f.FormID, chapters, *res); err != nil {
			serverError(w, "keep what was sent", err)
			return
		}
	}
	message := "" // the status the patient is told about now, if any (EPI-08)
	if res != nil {
		if err := ruleActions(ctx, q, f.ID, *res); err != nil {
			serverError(w, "rule actions", err)
			return
		}
		if message, err = s.autoTriage(ctx, q, f, *res); err != nil {
			serverError(w, "auto-triage", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return
	}
	if message != "" {
		s.automaticStatusMessage(r.Context(), f.ID, message)
	}
	w.WriteHeader(http.StatusNoContent)
}

// autoTriage acts on the rules that fired (S42; ASA-02, A-8), in the send's transaction: if a rule
// auto-triages and every guard holds (internal/triage), the episode moves on from Ready for review
// to the rule's target, recorded as the automatic auto_triage move naming the rule, with an
// automatic General notes line; a guard that stopped it gets a line saying why. It returns the
// status whose message the patient gets once the send is saved: the target's, unless the HQ holds
// the message until a clinician clicks Checked (S42 answers 2); else "".
func (s *server) autoTriage(ctx context.Context, q *db.Queries, f form, res evaluate.Result) (string, error) {
	d := triage.Decide(res)
	if d.Rule == "" {
		return "", nil
	}
	if d.Target == episode.ReadyForAdmission { // S43: a requirement standing from an earlier send holds it too
		now, err := q.LockEpisodeStatus(ctx, f.ID)
		if err != nil {
			return "", err
		}
		if now.AnaesthetistAsaRequiredBy != nil {
			grade, err := countingGrade(ctx, q, f.ID) // S44b: the grade that counts
			if err != nil {
				return "", err
			}
			if needsAnaesthetistASA(now.AnaesthetistAsaRequiredBy, grade) {
				d = triage.Decision{Rule: d.Rule, Blocked: triage.AnaesthetistASAGuard}
			}
		}
	}
	if !d.Moves() {
		return "", q.AddAutomaticEpisodeEventNow(ctx, db.AddAutomaticEpisodeEventNowParams{EpisodeID: f.ID, Text: d.Note()})
	}
	if err := episode.CheckAutomatic(episode.ReadyForReview, d.Target, episode.AutoTriage); err != nil {
		return "", err // asarules.Targets and the table disagree: a bug
	}
	var hq struct {
		MessageAfterCheck bool `json:"messageAfterCheck"`
	}
	if len(f.VersionRules) > 0 {
		_ = json.Unmarshal(f.VersionRules, &hq) // checked when saved (asarules.Check); unreadable: send at once, as by default
	}
	stem := episode.StatusMessage(episode.AutoTriage, d.Target)
	held := hq.MessageAfterCheck && stem != ""
	n, err := q.AutoTriageEpisode(ctx, db.AutoTriageEpisodeParams{ID: f.ID, Status: d.Target, Rule: &d.Rule, Summary: &d.Summary, MessagePending: held})
	if err != nil || n == 0 {
		return "", cmp.Or(err, errors.New("the episode isn't Ready for review"))
	}
	rule := d.Rule
	if err := q.AddStatusEvent(ctx, db.AddStatusEventParams{EpisodeID: f.ID, Move: string(episode.AutoTriage),
		FromStatus: ptrTo(episode.ReadyForReview), ToStatus: d.Target, Rule: &rule}); err != nil {
		return "", err
	}
	if err := q.AddAutomaticEpisodeEventNow(ctx, db.AddAutomaticEpisodeEventNowParams{EpisodeID: f.ID, Text: d.Note()}); err != nil {
		return "", err
	}
	if held || stem == "" {
		return "", nil
	}
	return d.Target, nil
}

func ptrTo[T any](v T) *T { return &v }

// ruleActions makes the fired rules' other writes (S43; ASA-03, TSK-01), in the send's transaction,
// before autoTriage: each rule's task (for its team, naming the rule) unless that rule's task is
// still open from an earlier send; each flag unless it's still standing; the requirement of the
// anaesthetist's ASA grade before Ready for admission; and the statuses the rules suggest (the
// latest send's replace any before). Each task, flag and new requirement gets an automatic General
// notes line.
func ruleActions(ctx context.Context, q *db.Queries, eid uuid.UUID, res evaluate.Result) error {
	a := triage.ActionsOf(res)
	note := func(text string) error {
		return q.AddAutomaticEpisodeEventNow(ctx, db.AddAutomaticEpisodeEventNowParams{EpisodeID: eid, Text: text})
	}
	for _, t := range a.Tasks {
		n, err := q.CreateRuleTask(ctx, db.CreateRuleTaskParams{EpisodeID: eid, Title: t.Title, Team: t.Team, Rule: t.Rule})
		if err == nil && n > 0 {
			err = note(t.Note())
		}
		if err != nil {
			return err
		}
	}
	for _, f := range a.Flags {
		n, err := q.AddRuleFlag(ctx, db.AddRuleFlagParams{EpisodeID: eid, Name: f.Name, Rule: f.Rule})
		if err == nil && n > 0 {
			err = note(f.Note())
		}
		if err != nil {
			return err
		}
	}
	if a.RequireASA != "" {
		n, err := q.RequireAnaesthetistASA(ctx, db.RequireAnaesthetistASAParams{ID: eid, Rule: a.RequireASA})
		if err == nil && n > 0 {
			err = note(triage.RequireNote(a.RequireASA))
		}
		if err != nil {
			return err
		}
	}
	if a.Suggestions == nil {
		a.Suggestions = []triage.Suggestion{}
	}
	suggestions, err := json.Marshal(a.Suggestions)
	if err != nil {
		return err
	}
	return q.SetRuleSuggestions(ctx, db.SetRuleSuggestionsParams{ID: eid, Suggestions: suggestions})
}

// checksDidntRun is the General notes line when the evaluate service couldn't work out a sent HQ.
const checksDidntRun = "The automatic checks didn't run. Review this HQ by hand."

// evaluateSent asks the evaluate service what the sent answers give (S41, D-1) and stores it in
// form_outputs, in the send's transaction, after the answers are frozen; autoTriage acts on it (S42). The service gets the published version whole (each set with its audience, and the rules),
// the patient's answers, their age today at the hospital and their sex, and the episode's procedure:
// nothing else about the patient. If the call fails (the service down, too slow, a bad reply), the
// send still goes through: the error is stored, one EVALUATE_FAILED line is logged (the form's id
// and the error, never an answer) and the General notes ask for the HQ to be reviewed by hand.
//
// It returns the result, or nil when the call failed (S42 then leaves the episode Ready for review).
func (s *server) evaluateSent(ctx context.Context, q *db.Queries, f form, chapters []db.ListVersionChaptersRow) (*evaluate.Result, error) {
	saved, err := q.ListEpisodeAnswers(ctx, f.FormID)
	if err != nil {
		return nil, err
	}
	in := evaluate.Input{FormID: f.FormID, Rules: f.VersionRules, Answers: map[uuid.UUID]json.RawMessage{},
		Patient: evaluate.Patient{Age: ageOn(f.DateOfBirth, s.today(f.Timezone)), Sex: f.Sex}}
	if f.ProcedureID.Valid {
		in.ProcedureID = &f.ProcedureID.UUID
	}
	for _, c := range chapters {
		in.Sets = append(in.Sets, evaluate.Set{ID: c.ID, Name: c.Name, Audience: c.Audience, Content: c.Content})
	}
	for _, a := range saved {
		if a.Actor == "patient" {
			in.Answers[a.ChapterID] = a.Data
		}
	}
	res, err := s.eval.Evaluate(ctx, in)
	if err != nil {
		log.Printf("%s form %s: %v", evaluate.FailedMarker, f.FormID, err)
		msg := err.Error()
		if err := q.AddFormOutput(ctx, db.AddFormOutputParams{FormID: f.FormID, Error: &msg}); err != nil {
			return nil, err
		}
		// Written as automatic, shown "Automatic" like a reminder's (test report 1, F-1), whoever sent it.
		return nil, q.AddAutomaticEpisodeEventNow(ctx, db.AddAutomaticEpisodeEventNowParams{EpisodeID: f.ID, Text: checksDidntRun})
	}
	return &res, q.AddFormOutput(ctx, db.AddFormOutputParams{FormID: f.FormID, Result: res.Stored(), EvaluatorVersion: &res.Evaluator})
}

// keepSent keeps in the frozen patient answers only what the patient's form showed when they sent
// (S44a, Rahul 27 Oct), as the evaluate service worked it out (res.Sent): the answers to questions
// a condition hid, and every answer of a set its condition hid, are dropped, with where each was
// pre-filled from. So a set a clinician's correction brings back later starts empty. Go never
// evaluates: it only removes the keys the service left out, and never adds or changes an answer. A
// set the reply doesn't name, or whose answers aren't an object, keeps everything.
func keepSent(ctx context.Context, q *db.Queries, formID uuid.UUID, chapters []db.ListVersionChaptersRow, res evaluate.Result) error {
	if len(res.Sent) == 0 {
		return nil
	}
	saved, err := q.ListEpisodeAnswers(ctx, formID)
	if err != nil {
		return err
	}
	patientSet := map[uuid.UUID]bool{}
	for _, c := range chapters {
		patientSet[c.ID] = c.Audience == "patient"
	}
	for _, a := range saved {
		sent, named := res.Sent[a.ChapterID]
		if a.Actor != "patient" || !named || !patientSet[a.ChapterID] {
			continue
		}
		var keep, data map[string]json.RawMessage
		if json.Unmarshal(sent, &keep) != nil || keep == nil || json.Unmarshal(a.Data, &data) != nil {
			continue
		}
		n := len(data)
		for k := range data {
			if _, ok := keep[k]; !ok {
				delete(data, k)
			}
		}
		if len(data) == n {
			continue
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if err := q.KeepSentAnswers(ctx, db.KeepSentAnswersParams{Data: raw, FormID: formID, ChapterID: a.ChapterID}); err != nil {
			return err
		}
	}
	return nil
}

// uncheckedTicks is why the HQ can't be sent yet, or "": a required "I have watched this video" tick
// (S34, INF-02) that every patient sees isn't ticked. The browser checks every required question
// first (plan 2.1); this holds back only what Go can know without evaluating a condition.
func uncheckedTicks(ctx context.Context, q *db.Queries, f form, rows []db.ListVersionChaptersRow) (string, error) {
	saved, err := q.ListEpisodeAnswers(ctx, f.FormID)
	if err != nil {
		return "", err
	}
	answers := map[uuid.UUID]map[string]json.RawMessage{}
	for _, a := range saved {
		if a.Actor == "patient" {
			var data map[string]json.RawMessage
			_ = json.Unmarshal(a.Data, &data) // answers that aren't an object hold no tick
			answers[a.ChapterID] = data
		}
	}
	var titles []string
	for _, c := range rows {
		if c.Audience != "patient" {
			continue
		}
		ticks, err := chapter.RequiredTicks(c.Content)
		if err != nil {
			return "", err
		}
		for _, t := range chapter.Unticked(ticks, answers[c.ID]) {
			titles = append(titles, "“"+cmp.Or(t.Title, t.Name)+"”")
		}
	}
	if len(titles) == 0 {
		return "", nil
	}
	return "Please tick “I have watched this video” for " + strings.Join(titles, ", ") + " before sending your answers.", nil
}

// unfinishedSets is why the HQ can't be sent yet, or "": the evaluate service says a Question Set
// the patient's form shows isn't complete (S44), named as the patient sees it.
func unfinishedSets(res evaluate.Result, chapters []db.ListVersionChaptersRow) string {
	var names []string
	for _, ps := range res.PatientSets {
		if ps.Complete {
			continue
		}
		name := "a Question Set"
		for _, c := range chapters {
			if c.ID == ps.ID {
				name = c.Name
			}
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return ""
	}
	return "Please finish " + quotedList(names) + " before sending your answers."
}

// quotedList is “A”, “B” and “C”.
func quotedList(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = "“" + n + "”"
	}
	if len(q) == 1 {
		return q[0]
	}
	return strings.Join(q[:len(q)-1], ", ") + " and " + q[len(q)-1]
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

// staffPatientForm is the form for staff to fill in. Before its first pre-fill, a record that
// hasn't joined a sign-in account joins the one that matches it on all four fields (S28): its
// date of birth, mobile or email, first name and surname. Then other trusts' answers fill it in
// as they do for the patient (patients filled in for often can't open a link: Rahul, 24 Oct).
func (s *server) staffPatientForm(w http.ResponseWriter, r *http.Request) {
	f, ok := s.loadStaffForm(w, r)
	if !ok {
		return
	}
	if !f.AccountID.Valid && !f.PatientSubmittedAt.Valid && !f.PrefilledAt.Valid {
		account, err := s.joinOnBehalf(r, f)
		if err != nil {
			serverError(w, "join account", err)
			return
		}
		f.AccountID = account
	}
	s.writeForm(w, r, f)
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

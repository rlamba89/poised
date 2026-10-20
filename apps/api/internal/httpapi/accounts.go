package httpapi

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/patient"
)

// Patient accounts across trusts, sign-in links and the patient's home (plan C2, session S15;
// A-3, PAT-06, PAT-07, PAT-10, PAT-11).
//
// An account is one person: their sign-in contacts and their date of birth. Each trust's record of
// them joins it when they prove their date of birth through a link that reached them at a contact
// the record holds: an invite (the record's own), or a sign-in link (the contact they typed). The
// date of birth is part of every match, so two people who share a phone or an email stay apart.
//
// A returning patient types their mobile or email and gets a sign-in link (/s/<token>), which
// works once, within 15 minutes, then Continue and their date of birth, as an invite does. The
// session is the account's: their home lists every trust's episodes, and opens each one.
//
// They can change their mobile or email (PAT-10, S15 answers 1): a link goes to the new contact
// first (/v/<token>), and only once it's opened does the account, and every trust's record of
// them, change. The old contact is told.
//
// Sign-in and verification messages are rate-limited (S15 answers 2): 3 per contact per 15
// minutes, 10 per IP address per hour. Texts go only to the deployment's country (PAT-11).

const (
	signinTTL = 15 * time.Minute // PAT-05
	// maxNameLength is the longest first name or surname the sign-in link's name step reads.
	maxNameLength = 100
	verifyTTL = 24 * time.Hour

	signinLinkExpired = "This sign-in link has expired or has already been used. Please ask for a new one."
	signinLocked      = "This link is locked because the details were entered wrongly too many times. Please ask for a new sign-in link."
	verifyLinkExpired = "This link has expired or has already been used. Sign in and change your details again to get a new one."
	signinLinkUnknown = "This link doesn't work. Please check it, or ask for a new sign-in link."
	verifyLinkUnknown = "This link doesn't work. Please check it, or sign in and change your details again."
)

// joinAccount joins an invited record to the person's account, making one from the record if
// there's none with its date of birth, its first name and surname (S15 answers 6 and 8) and its
// mobile or email, and joins their other records with that name, those contacts and date of birth too. Records
// that join take the account's contacts they lack. It returns the account. Every record that
// joins is written to its trust's audit log (account.join).
func (s *server) joinAccount(r *http.Request, patientID uuid.UUID, dob pgtype.Date) (uuid.UUID, error) {
	ctx := context.WithoutCancel(r.Context()) // the date of birth was right: finish what's started
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	if err := q.LockBirthDate(ctx, dob); err != nil {
		return uuid.Nil, err
	}
	rec, err := q.LockPatientForAccount(ctx, patientID)
	if err != nil {
		return uuid.Nil, err
	}
	account := rec.AccountID.UUID
	if !rec.AccountID.Valid {
		account, err = q.FindAccountForRecord(ctx, db.FindAccountForRecordParams{
			DateOfBirth: rec.DateOfBirth, FirstName: rec.FirstName, LastName: rec.LastName, Mobile: rec.Mobile, Email: rec.Email})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			account, err = q.CreateAccount(ctx, db.CreateAccountParams{
				Mobile: rec.Mobile, Email: rec.Email, DateOfBirth: rec.DateOfBirth, FirstName: rec.FirstName, LastName: rec.LastName})
		case err == nil:
			err = q.FillAccountContacts(ctx, db.FillAccountContactsParams{ID: account, Mobile: rec.Mobile, Email: rec.Email})
		}
		if err != nil {
			return uuid.Nil, err
		}
		if err := q.JoinAccount(ctx, db.JoinAccountParams{ID: rec.ID, AccountID: uuid.NullUUID{UUID: account, Valid: true}}); err != nil {
			return uuid.Nil, err
		}
		if err := q.FillRecordContacts(ctx, []uuid.UUID{rec.ID}); err != nil {
			return uuid.Nil, err
		}
		auditOf(r).also("account.join", rec.OrgID, uuid.Nil).touched(rec.ID, uuid.Nil) // S15 answers 7
	}
	if err := s.joinMatching(ctx, r, q, account, rec.DateOfBirth, rec.FirstName, rec.LastName, rec.Mobile, rec.Email); err != nil {
		return uuid.Nil, err
	}
	return account, tx.Commit(ctx)
}

// joinMatching joins the person's records not yet in an account that have their date of birth,
// their first name and surname, and one of contacts (those that just reached them). Each joined record takes
// the account's contacts it lacks, and is in its trust's log.
func (s *server) joinMatching(ctx context.Context, r *http.Request, q *db.Queries, account uuid.UUID, dob pgtype.Date, firstName, lastName string, contacts ...string) error {
	var proved []string
	for _, c := range contacts {
		if c != "" {
			proved = append(proved, c)
		}
	}
	if len(proved) == 0 {
		return nil
	}
	joined, err := q.JoinMatchingRecords(ctx, db.JoinMatchingRecordsParams{
		AccountID: uuid.NullUUID{UUID: account, Valid: true}, DateOfBirth: dob, FirstName: firstName, LastName: lastName, Contacts: proved})
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(joined))
	for _, j := range joined {
		ids = append(ids, j.ID)
		auditOf(r).also("account.join", j.OrgID, uuid.Nil).touched(j.ID, uuid.Nil)
	}
	if len(ids) == 0 {
		return nil
	}
	return q.FillRecordContacts(ctx, ids)
}

// --- Sign-in links (PAT-06) ---

// requestSignin is POST /api/p/signin {contact}: a sign-in link by text or email to a contact an
// account signs in with. The answer is the same whether there's an account or not, so nobody
// learns who has one; only the rate limits and a mobile outside the country are refused.
func (s *server) requestSignin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Contact string `json:"contact"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	contact, email, msg := patient.Contact(in.Contact, s.cfg.Country.PhoneCountry)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if !s.allowMessage(w, r, contact) {
		return
	}
	has, err := s.q.AccountHasContact(r.Context(), contact)
	if err != nil {
		serverError(w, "account by contact", err)
		return
	}
	if has {
		// A send that fails gets the same answer as no account (S17: real sending can fail), so a
		// failure can't tell anyone the contact has an account. It's logged, and a refused send
		// also counts for the delivery-failure alarm; the person can ask again.
		if err := s.sendContactLink(r.Context(), "signin", contact, email, uuid.Nil); err != nil {
			log.Printf("sign-in link: %v", err)
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"channel": channelOf(email)})
}

func channelOf(email bool) string {
	if email {
		return "email"
	}
	return "sms"
}

// sendContactLink makes a sign-in link (kind signin) or a new contact's verification link (kind
// verify, for account) and sends it to contact, by email or text.
func (s *server) sendContactLink(ctx context.Context, kind, contact string, email bool, account uuid.UUID) error {
	token, err := auth.NewLinkToken()
	if err != nil {
		return err
	}
	now := s.clock.Now()
	ttl, path := signinTTL, "/s/"
	if kind == "verify" {
		ttl, path = verifyTTL, "/v/"
	}
	if _, err := s.q.CreateContactLink(ctx, db.CreateContactLinkParams{
		TokenHash: auth.HashLinkToken(token), Kind: kind, Contact: &contact,
		AccountID: uuid.NullUUID{UUID: account, Valid: account != uuid.Nil},
		ExpiresAt: pgtype.Timestamptz{Time: now.Add(ttl), Valid: true}, Now: now,
	}); err != nil {
		return err
	}
	m := contactMessages(kind, s.cfg.Country.AppURL+path+token, email)
	if email {
		_, err = s.email.SendEmail(ctx, contact, m.subject, m.body)
	} else {
		_, err = s.sms.SendSMS(ctx, contact, m.body)
	}
	return err
}

type contactMessage struct{ subject, body string }

// contactMessages are the sign-in and verification messages. They name Preopsis, not a hospital:
// an account spans every trust.
func contactMessages(kind, link string, email bool) contactMessage {
	switch {
	case kind == "signin" && email:
		return contactMessage{"Your Preopsis sign-in link",
			"Hello,\n\nHere is your link to sign in to Preopsis and see your health questionnaires:\n\n" + link +
				"\n\nIt works once, within 15 minutes. If you didn't ask for it, you can ignore this email."}
	case kind == "signin":
		return contactMessage{"", "Preopsis: your sign-in link. It works once, within 15 minutes: " + link +
			" If you didn't ask for it, ignore this message."}
	case email:
		return contactMessage{"Confirm your new email address for Preopsis",
			"Hello,\n\nTo use this email address for Preopsis, open this link:\n\n" + link +
				"\n\nIt works for 24 hours. If you didn't ask for this, you can ignore this email."}
	default:
		return contactMessage{"", "Preopsis: to use this mobile number for Preopsis, open this link: " + link +
			" If you didn't ask for this, ignore this message."}
	}
}

// loadContactLink finds the sign-in or verification link named by {token}, answering itself when
// it's unknown (404), used or expired (404, code "expired"), or locked (403, code "locked").
func (s *server) loadContactLink(w http.ResponseWriter, r *http.Request, kind string) (db.GetContactLinkByHashRow, bool) {
	expired, locked, unknown := signinLinkExpired, signinLocked, signinLinkUnknown
	if kind == "verify" {
		expired, unknown = verifyLinkExpired, verifyLinkUnknown
	}
	token := r.PathValue("token")
	if !tokenFormat.MatchString(token) {
		writeError(w, http.StatusNotFound, unknown)
		return db.GetContactLinkByHashRow{}, false
	}
	l, err := s.q.GetContactLinkByHash(r.Context(), db.GetContactLinkByHashParams{TokenHash: auth.HashLinkToken(token), Kind: kind})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, unknown)
		return l, false
	}
	if err != nil {
		serverError(w, "contact link", err)
		return l, false
	}
	if l.LockedAt.Valid {
		writeLocked(w, locked)
		return l, false
	}
	if l.UsedAt.Valid || l.RevokedAt.Valid || !l.ExpiresAt.Time.After(s.clock.Now()) { // revoked: a newer request replaced it
		writeJSON(w, http.StatusNotFound, map[string]string{"error": expired, "code": "expired"})
		return l, false
	}
	return l, true
}

// signinContinue is the sign-in link's Continue: whether it still works. It returns nothing about
// anyone, and changes nothing.
func (s *server) signinContinue(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.loadContactLink(w, r, "signin"); ok {
		writeJSON(w, http.StatusOK, map[string]string{})
	}
}

// signinDob checks the date of birth against the accounts that sign in with the link's contact.
// Right: the link is used up, the person's records with that contact, date of birth and name join
// their account, and their session starts. Wrong: as for an invite, five lock the link.
// When two accounts with that contact share the date of birth (twins, or cousins, on a parent's
// phone: S15 answers 6 and 8), the first name and surname are asked for (409, code "name"), which
// isn't a try; a wrong name is. Nothing about anyone is said before the date of birth is right.
// A contact no account uses any more (changed since the link went) is an expired link.
func (s *server) signinDob(w http.ResponseWriter, r *http.Request) {
	l, ok := s.loadContactLink(w, r, "signin")
	if !ok {
		return
	}
	var in struct {
		DateOfBirth string `json:"dateOfBirth"`
		FirstName   string `json:"firstName"` // only when asked for
		LastName    string `json:"lastName"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	d, err := time.Parse(time.DateOnly, in.DateOfBirth)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Enter your date of birth as a real date, for example 27 3 1985.")
		return
	}
	in.FirstName, in.LastName = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName)
	named := in.FirstName != "" || in.LastName != ""
	switch {
	case named && (in.FirstName == "" || in.LastName == ""):
		writeError(w, http.StatusBadRequest, "Enter your first name and your surname.")
		return
	case utf8.RuneCountInString(in.FirstName) > maxNameLength || utf8.RuneCountInString(in.LastName) > maxNameLength:
		writeError(w, http.StatusBadRequest, "That name is too long. Enter your first name and surname as the hospital has them.")
		return
	}
	ctx := context.WithoutCancel(r.Context())
	dob := pgtype.Date{Time: d, Valid: true}
	accounts, err := s.q.AccountsForSignin(ctx, db.AccountsForSigninParams{Contact: *l.Contact, DateOfBirth: dob,
		Named: named, FirstName: in.FirstName, LastName: in.LastName})
	if err != nil {
		serverError(w, "account for sign-in", err)
		return
	}
	if len(accounts) == 0 {
		still, err := s.q.AccountHasContact(ctx, *l.Contact)
		if err != nil {
			serverError(w, "account by contact", err)
			return
		}
		switch {
		case !still: // the contact was changed away from since the link went
			writeJSON(w, http.StatusNotFound, map[string]string{"error": signinLinkExpired, "code": "expired"})
		case named:
			n, err := s.q.FailLinkDob(r.Context(), db.FailLinkDobParams{ID: l.ID, MaxAttempts: maxDobAttempts})
			s.wrongDob(w, n, err, signinLocked, "That date of birth and name don't match our records. Please check them and try again.")
		default:
			n, err := s.q.FailLinkDob(r.Context(), db.FailLinkDobParams{ID: l.ID, MaxAttempts: maxDobAttempts})
			s.wrongDob(w, n, err, signinLocked, "")
		}
		return
	}
	if len(accounts) > 1 {
		writeJSON(w, http.StatusConflict, map[string]string{"code": "name",
			"error": "More than one person with this date of birth uses this " + contactName(strings.Contains(*l.Contact, "@")) +
				". Please also enter your first name and surname."})
		return
	}
	account, firstKey, lastKey := accounts[0].ID, accounts[0].FirstNameKey, accounts[0].LastNameKey
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "sign in", err)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	n, err := q.UseContactLink(ctx, db.UseContactLinkParams{ID: l.ID, Now: pgtype.Timestamptz{Time: s.clock.Now(), Valid: true}})
	if err != nil {
		serverError(w, "use sign-in link", err)
		return
	}
	if n == 0 { // used or locked at the same moment
		writeJSON(w, http.StatusNotFound, map[string]string{"error": signinLinkExpired, "code": "expired"})
		return
	}
	if err := q.LockBirthDate(ctx, dob); err != nil {
		serverError(w, "sign in", err)
		return
	}
	if err := s.joinMatching(ctx, r, q, account, dob, firstKey, lastKey, *l.Contact); err != nil {
		serverError(w, "join records", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "sign in", err)
		return
	}
	if !s.startSession(w, r, l.ID, account) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{})
}

// --- The home (PAT-06, PX-08) ---

type homeEpisode struct {
	ID                uuid.UUID `json:"id"`
	PatientName       string    `json:"patientName"` // whose questionnaire it is (S15 answers 6)
	Hospital          string    `json:"hospital"`
	QuestionnaireName string    `json:"questionnaireName"`
	Procedure         string    `json:"procedure"`
	CreatedAt         time.Time `json:"createdAt"`
	State             string    `json:"state"` // to_do, in_progress or sent
}

// patientHome is GET /api/p/home: every episode of every trust's record of the person, newest
// first, with the hospital's name, and their own contact details shown in part. One query for
// the episodes. Each hospital it lists gets a row in its trust's audit log.
func (s *server) patientHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	acc, err := s.q.GetAccount(ctx, accountOf(r))
	if err != nil {
		serverError(w, "account", err)
		return
	}
	rows, err := s.q.ListHome(ctx, uuid.NullUUID{UUID: acc.ID, Valid: true})
	if err != nil {
		serverError(w, "home", err)
		return
	}
	episodes := []homeEpisode{}
	firstName := ""
	for _, e := range rows {
		auditOf(r).also("home.view", e.OrgID, e.HospitalID).touched(e.PatientID, e.ID)
		firstName = greeting(firstName, e.FirstName)
		state := "to_do"
		switch {
		case e.PatientSubmittedAt.Valid:
			state = "sent"
		case e.Started:
			state = "in_progress"
		}
		episodes = append(episodes, homeEpisode{ID: e.ID, PatientName: strings.TrimSpace(e.FirstName) + " " + strings.TrimSpace(e.LastName), Hospital: e.HospitalName, QuestionnaireName: e.QuestionnaireName,
			Procedure: e.Procedure, CreatedAt: e.CreatedAt, State: state})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"firstName": firstName,
		"episodes":  episodes,
		"contacts":  map[string]string{"mobile": patient.MaskMobile(acc.Mobile), "email": patient.MaskEmail(acc.Email)},
	})
}

// greeting is the first name to greet the person by, given the one chosen so far ("" for none)
// and the next record's, newest first: the newest one written with a capital (a hospital may have
// typed "sam"), else the newest, capitalised (S15 test report 2).
func greeting(sofar, next string) string {
	next = strings.TrimSpace(next)
	switch {
	case sofar != "" && sofar != strings.ToLower(sofar):
		return sofar
	case next != "" && next != strings.ToLower(next):
		return next
	case sofar != "":
		return sofar
	}
	r, n := utf8.DecodeRuneInString(next)
	if n == 0 {
		return ""
	}
	return string(unicode.ToUpper(r)) + next[n:]
}

// --- Changing a contact (PAT-10) ---

// requestContactChange is POST /api/p/contact {contact, kind}: a link to the new mobile or email,
// which changes nothing until it's opened. A newer request replaces an older one of the same kind.
// kind is the box it was typed in, "mobile" or "email" ("" for either): each box takes only its
// own kind (S15 answers 7).
func (s *server) requestContactChange(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Contact string `json:"contact"`
		Kind    string `json:"kind"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	typedEmail := strings.Contains(in.Contact, "@")
	switch {
	case in.Kind != "" && in.Kind != "mobile" && in.Kind != "email":
		writeError(w, http.StatusBadRequest, "Choose a mobile number or an email address.")
		return
	case in.Kind == "email" && !typedEmail:
		writeError(w, http.StatusBadRequest, "Enter an email address, for example name@example.com.")
		return
	case in.Kind == "mobile" && typedEmail:
		writeError(w, http.StatusBadRequest, "Enter a mobile number.")
		return
	}
	contact, email, msg := patient.Contact(in.Contact, s.cfg.Country.PhoneCountry)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	acc, err := s.q.GetAccount(ctx, accountOf(r))
	if err != nil {
		serverError(w, "account", err)
		return
	}
	if (email && contact == acc.Email) || (!email && contact == acc.Mobile) {
		writeError(w, http.StatusBadRequest, "That's already your "+contactName(email)+".")
		return
	}
	if !s.allowMessage(w, r, contact) {
		return
	}
	if err := s.q.RevokeVerifyLinks(ctx, db.RevokeVerifyLinksParams{
		AccountID: uuid.NullUUID{UUID: acc.ID, Valid: true}, Email: email, Now: pgtype.Timestamptz{Time: s.clock.Now(), Valid: true}}); err != nil {
		serverError(w, "revoke verify links", err)
		return
	}
	if err := s.sendContactLink(ctx, "verify", contact, email, acc.ID); err != nil {
		log.Printf("verify link: %v", err)
		writeError(w, http.StatusBadGateway, "We couldn't send the link just now. Please try again in a few minutes.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"channel": channelOf(email)})
}

func contactName(email bool) string {
	if email {
		return "email address"
	}
	return "mobile number"
}

// confirmContact is POST /api/p/contact/{token}/confirm: the new contact's link was opened, so it
// becomes the account's, and every trust's record of the person takes it. No session is needed:
// the link proves the new contact is theirs, and it may be opened on another device. The old
// contact is told, in case it wasn't them.
func (s *server) confirmContact(w http.ResponseWriter, r *http.Request) {
	l, ok := s.loadContactLink(w, r, "verify")
	if !ok {
		return
	}
	contact := *l.Contact
	_, email, _ := patient.Contact(contact, s.cfg.Country.PhoneCountry) // stored as checked
	ctx := context.WithoutCancel(r.Context())
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "confirm contact", err)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	acc, err := q.LockAccount(ctx, l.AccountID.UUID)
	if err != nil {
		serverError(w, "account", err)
		return
	}
	taken, err := q.ContactTaken(ctx, db.ContactTakenParams{ID: acc.ID, DateOfBirth: acc.DateOfBirth,
		FirstNameKey: acc.FirstNameKey, LastNameKey: acc.LastNameKey, Contact: contact})
	if err != nil {
		serverError(w, "contact taken", err)
		return
	}
	if taken {
		writeError(w, http.StatusConflict, "This "+contactName(email)+" is already used by another account with your details. Please contact your hospital.")
		return
	}
	n, err := q.UseContactLink(ctx, db.UseContactLinkParams{ID: l.ID, Now: pgtype.Timestamptz{Time: s.clock.Now(), Valid: true}})
	if err != nil {
		serverError(w, "use verify link", err)
		return
	}
	if n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": verifyLinkExpired, "code": "expired"})
		return
	}
	if err := q.SetAccountContact(ctx, db.SetAccountContactParams{ID: acc.ID, Email: email, Contact: contact}); err != nil {
		serverError(w, "set account contact", err)
		return
	}
	records, err := q.SetRecordsContact(ctx, db.SetRecordsContactParams{AccountID: uuid.NullUUID{UUID: acc.ID, Valid: true}, Email: email, Contact: contact})
	if err != nil {
		serverError(w, "set records contact", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "confirm contact", err)
		return
	}
	for _, rec := range records {
		auditOf(r).also("contact.update", rec.OrgID, uuid.Nil).touched(rec.ID, uuid.Nil)
	}
	s.tellContacts(ctx, acc.Mobile, acc.Email, email)
	writeJSON(w, http.StatusOK, map[string]string{"channel": channelOf(email)})
}

// tellContacts tells the contacts the account had before a change (its mobile and its email),
// whether the change replaced one or added one where there was none (S15 answers 7): a session
// left open on a shared phone mustn't move the account to someone else's contact unnoticed.
func (s *server) tellContacts(ctx context.Context, oldMobile, oldEmail string, email bool) {
	verb := "changed"
	if (email && oldEmail == "") || (!email && oldMobile == "") {
		verb = "added"
	}
	what := contactName(email)
	if oldEmail != "" {
		if _, err := s.email.SendEmail(ctx, oldEmail, "Your Preopsis contact details were "+verb,
			"Hello,\n\nThe "+what+" you use for Preopsis was "+verb+".\n\nIf you didn't do this, please contact your hospital."); err != nil {
			log.Printf("tell old email: %v", err)
		}
	}
	if oldMobile != "" && patient.SMSAllowed(oldMobile, s.cfg.Country.PhoneCountry) {
		if _, err := s.sms.SendSMS(ctx, oldMobile, "Preopsis: the "+what+" you use for Preopsis was "+verb+
			". If you didn't do this, please contact your hospital."); err != nil {
			log.Printf("tell old mobile: %v", err)
		}
	}
}

// --- Rate limits (PAT-11, S15 answers 2) ---

type rateLimit struct {
	key    string
	max    int
	window time.Duration
	msg    string
}

// allowMessage counts a request for a message to contact against the rate limits, answering 429
// itself when one is reached: 3 per contact per 15 minutes, and 10 per IP address per hour.
func (s *server) allowMessage(w http.ResponseWriter, r *http.Request, contact string) bool {
	over, err := s.hit(r.Context(), []rateLimit{
		{"contact:" + contact, 3, 15 * time.Minute,
			"You've asked for 3 links in the last 15 minutes. Please use the newest one, or wait 15 minutes and try again."},
		{"ip:" + clientIP(r), 10, time.Hour,
			"There have been too many requests from this network. Please try again in an hour."},
	})
	if err != nil {
		serverError(w, "rate limit", err)
		return false
	}
	if over != nil {
		writeError(w, http.StatusTooManyRequests, over.msg)
		return false
	}
	return true
}

// hit records a hit on every limit's key, unless one of them is already at its maximum within its
// window: then nothing is recorded and that limit is returned. Keys are locked (always in the same
// order) while they're counted, so requests at the same moment can't both slip under a limit.
func (s *server) hit(ctx context.Context, limits []rateLimit) (*rateLimit, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	now := s.clock.Now()
	for i, l := range limits {
		if err := q.LockRateKey(ctx, l.key); err != nil {
			return nil, err
		}
		n, err := q.CountHits(ctx, db.CountHitsParams{Key: l.key, Since: now.Add(-l.window)})
		if err != nil {
			return nil, err
		}
		if int(n) >= l.max {
			return &limits[i], nil
		}
	}
	for _, l := range limits {
		if err := q.DropOldHits(ctx, db.DropOldHitsParams{Key: l.key, Before: now.Add(-24 * time.Hour)}); err != nil {
			return nil, err
		}
		if err := q.AddHit(ctx, db.AddHitParams{Key: l.key, At: now}); err != nil {
			return nil, err
		}
	}
	return nil, tx.Commit(ctx)
}

// clientIP is the address the request came from. Behind the proxies on AWS (Amplify's rewrite,
// API Gateway), internal/serve sets RemoteAddr to the browser's address from X-Forwarded-For
// (config FORWARDED_HOPS, S06); on Lambda it has no port.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

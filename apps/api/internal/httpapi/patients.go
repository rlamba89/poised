package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/patient"
)

// Patient records (plan C2, S13). A record is the trust's (org_id), but a hospital lists and
// shows only its own patients: those with at least one episode there. Creating an episode for
// typed details links silently to the trust's existing record when the identifiers agree, and
// shows nothing about it (decisions.md, 11 Oct; S13 answers 2).

const (
	patientNotFound = "Patient not found."
	searchLimit     = 20 // at most this many results: nobody can list everyone
)

// searchPatients is the New episode page's search: this hospital's patients by national ID,
// hospital number here, or surname (with an optional date of birth).
func (s *server) searchPatients(w http.ResponseWriter, r *http.Request) {
	scheme := s.cfg.Country.NationalID()
	q := r.URL.Query()
	in, msg := patient.ParseSearch(q.Get("q"), q.Get("dob"), scheme)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	arg := db.SearchPatientsParams{
		OrgID: orgID(r), HospitalID: hospitalID(r), NationalID: in.NationalID, NationalIDType: scheme.Type,
		Term: in.Term, SurnamePrefix: likePrefix(in.Term), MaxRows: searchLimit + 1,
	}
	if in.Birth != nil {
		arg.Birth = pgtype.Date{Time: *in.Birth, Valid: true}
	}
	rows, err := s.q.SearchPatients(r.Context(), arg)
	if err != nil {
		serverError(w, "search patients", err)
		return
	}
	more := len(rows) > searchLimit
	if more {
		rows = rows[:searchLimit]
	}
	for _, p := range rows {
		auditOf(r).touched(p.ID, uuid.Nil)
	}
	writeJSON(w, http.StatusOK, map[string]any{"patients": rows, "more": more})
}

// likePrefix is a LIKE pattern for values starting with s, in lower case, with LIKE's own
// characters (% _ \) matched as text.
func likePrefix(s string) string {
	s = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(s))
	return s + "%"
}

// getPatient is a patient's page: the trust's record (S08 answers: a hospital's staff see the
// patient's details) and their episodes at this hospital only.
func (s *server) getPatient(w http.ResponseWriter, r *http.Request) {
	pid, ok := pathID(w, r, "pid", patientNotFound)
	if !ok {
		return
	}
	p, err := s.q.GetHospitalPatient(r.Context(), db.GetHospitalPatientParams{ID: pid, OrgID: orgID(r), HospitalID: hospitalID(r)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, patientNotFound)
		return
	}
	if err != nil {
		serverError(w, "get patient", err)
		return
	}
	episodes, err := s.q.ListPatientEpisodesHere(r.Context(), db.ListPatientEpisodesHereParams{PatientID: p.ID, HospitalID: hospitalID(r)})
	if err != nil {
		serverError(w, "patient episodes", err)
		return
	}
	auditOf(r).touched(p.ID, uuid.Nil)
	for _, e := range episodes {
		auditOf(r).touched(p.ID, e.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"patient": p, "episodes": episodes})
}

var sexes = map[string]bool{"female": true, "male": true, "other": true, "unknown": true}

// patientInput is a new patient's details as staff typed them on the New episode page.
type patientInput struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	DateOfBirth string `json:"dateOfBirth"` // YYYY-MM-DD
	Sex         string `json:"sex"`
	NationalID  string `json:"nationalId"` // optional (PAT-01a)
	Mobile      string `json:"mobile"`     // optional; stored in E.164
	Email       string `json:"email"`      // optional; stored in lower case
}

// details checks and cleans up what was typed, or returns a plain-language problem.
func (s *server) details(in patientInput, now time.Time) (patient.Details, string, string) {
	scheme := s.cfg.Country.NationalID()
	d := patient.Details{FirstName: strings.TrimSpace(in.FirstName), LastName: strings.TrimSpace(in.LastName)}
	dob, err := time.Parse(time.DateOnly, strings.TrimSpace(in.DateOfBirth))
	d.DateOfBirth = dob
	switch {
	case d.FirstName == "" || d.LastName == "":
		return d, "", "Enter the patient's first name and surname."
	case len(d.FirstName) > 100 || len(d.LastName) > 100:
		return d, "", "The patient's name is too long."
	case err != nil || dob.After(now) || dob.Year() < 1900:
		return d, "", "Enter a valid date of birth."
	case !sexes[in.Sex]:
		return d, "", "Choose the patient's sex."
	}
	if id := scheme.Normalise(strings.TrimSpace(in.NationalID)); id != "" {
		if !scheme.Valid(id) {
			return d, "", "That isn't a valid " + scheme.Label + ". Check it, or leave it empty if the patient doesn't have one."
		}
		d.NationalID = id
	}
	var ok bool
	if d.Mobile, ok = patient.Mobile(in.Mobile, s.cfg.Country.PhoneCountry); !ok {
		return d, "", "Enter a valid mobile number, or leave it empty."
	}
	if d.Email, ok = patient.Email(in.Email); !ok {
		return d, "", "Enter a valid email address, or leave it empty."
	}
	return d, in.Sex, ""
}

// duplicateMatch is one of this hospital's patients shown in a possible-duplicate warning, so
// staff can use that record instead. Patients known only elsewhere are never shown.
type duplicateMatch struct {
	ID             uuid.UUID   `json:"id"`
	FirstName      string      `json:"firstName"`
	LastName       string      `json:"lastName"`
	DateOfBirth    pgtype.Date `json:"dateOfBirth"`
	NationalID     *string     `json:"nationalId"`
	HospitalNumber string      `json:"hospitalNumber"`
}

// errAnswered means resolvePatient has written the response itself (a 409).
var errAnswered = errors.New("answered")

// resolvePatient finds or creates the trust's record for typed details, inside the episode's
// transaction (patient.Decide has the rules). On a mismatch or a possible duplicate it writes
// the 409 itself and returns errAnswered; neither shows a stored value staff didn't type, apart
// from this hospital's own patients in a possible-duplicate warning.
func (s *server) resolvePatient(ctx context.Context, w http.ResponseWriter, r *http.Request, q *db.Queries,
	typed patient.Details, sex string, createAnyway bool) (uuid.UUID, error) {
	scheme := s.cfg.Country.NationalID()
	var byID *patient.Record
	if typed.NationalID != "" {
		rec, err := q.PatientByNationalID(ctx, db.PatientByNationalIDParams{OrgID: orgID(r), NationalIDType: &scheme.Type, NationalID: &typed.NationalID})
		if err == nil {
			byID = &patient.Record{ID: rec.ID, FirstName: rec.FirstName, LastName: rec.LastName, DateOfBirth: rec.DateOfBirth.Time,
				NationalID: deref(rec.NationalID), Mobile: rec.Mobile, Email: rec.Email}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, err
		}
	}
	born, err := q.PatientsBornOn(ctx, db.PatientsBornOnParams{OrgID: orgID(r), HospitalID: hospitalID(r),
		DateOfBirth: pgtype.Date{Time: typed.DateOfBirth, Valid: true}})
	if err != nil {
		return uuid.Nil, err
	}
	records := make([]patient.Record, len(born))
	here := map[uuid.UUID]db.PatientsBornOnRow{}
	for i, b := range born {
		records[i] = patient.Record{ID: b.ID, FirstName: b.FirstName, LastName: b.LastName, DateOfBirth: b.DateOfBirth.Time,
			NationalID: deref(b.NationalID), Mobile: b.Mobile, Email: b.Email}
		if b.Here {
			here[b.ID] = b
		}
	}

	d := patient.Decide(typed, byID, records, createAnyway)
	switch d.Outcome {
	case patient.Link:
		return d.Record.ID, nil
	case patient.Mismatch:
		writeJSON(w, http.StatusConflict, map[string]string{"code": "mismatch",
			"error": "These details don't match the trust's record for this " + scheme.Label +
				". Check the " + scheme.Label + ", date of birth and surname. Nothing was created."})
		return uuid.Nil, errAnswered
	case patient.PossibleDuplicate:
		shown := []duplicateMatch{}
		for _, m := range d.Matches {
			if b, ok := here[m.ID]; ok {
				shown = append(shown, duplicateMatch{ID: b.ID, FirstName: b.FirstName, LastName: b.LastName,
					DateOfBirth: b.DateOfBirth, NationalID: b.NationalID, HospitalNumber: b.HospitalNumber})
				auditOf(r).touched(b.ID, uuid.Nil)
			}
		}
		msg := "This patient may already be registered in the trust. Check their " + scheme.Label +
			" or contact details, or create a new record anyway."
		if len(shown) > 0 {
			msg = "This hospital already has a patient with this name and date of birth. Use that record, or create a new one anyway."
		}
		writeJSON(w, http.StatusConflict, map[string]any{"code": "possible_duplicate", "error": msg, "matches": shown})
		return uuid.Nil, errAnswered
	}

	var id, typ *string
	if typed.NationalID != "" {
		id, typ = &typed.NationalID, &scheme.Type
	}
	pid, err := q.CreatePatient(ctx, db.CreatePatientParams{
		OrgID: orgID(r), FirstName: typed.FirstName, LastName: typed.LastName,
		DateOfBirth: pgtype.Date{Time: typed.DateOfBirth, Valid: true}, Sex: sex,
		NationalID: id, NationalIDType: typ, Mobile: typed.Mobile, Email: typed.Email,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "patients_national_id_key" {
		// Someone added a record with this national ID since it was looked up.
		writeError(w, http.StatusConflict, "A record with this "+scheme.Label+" was added just now. Try again.")
		return uuid.Nil, errAnswered
	}
	return pid, err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

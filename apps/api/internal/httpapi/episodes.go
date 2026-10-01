package httpapi

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

const episodeNotFound = "Episode not found."

// clinicianOnly writes 403 itself unless the caller is a clinician in this hospital.
func clinicianOnly(w http.ResponseWriter, r *http.Request) bool {
	if !hasRole(r, "clinician") {
		writeError(w, http.StatusForbidden, "Only clinicians can see patients and episodes.")
		return false
	}
	return true
}

var sexes = map[string]bool{"female": true, "male": true, "other": true, "unknown": true}

type patientInput struct {
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	DateOfBirth    string `json:"dateOfBirth"` // YYYY-MM-DD
	Sex            string `json:"sex"`
	HospitalNumber string `json:"hospitalNumber"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
}

// validate trims the fields and returns the date of birth, or a plain-language problem.
func (in *patientInput) validate(now time.Time) (time.Time, string) {
	for _, f := range []*string{&in.FirstName, &in.LastName, &in.DateOfBirth, &in.HospitalNumber, &in.Phone, &in.Email} {
		*f = strings.TrimSpace(*f)
	}
	dob, err := time.Parse(time.DateOnly, in.DateOfBirth)
	switch {
	case in.FirstName == "" || in.LastName == "":
		return dob, "Enter the patient's first and last name."
	case err != nil || dob.After(now) || dob.Year() < 1900:
		return dob, "Enter a valid date of birth."
	case !sexes[in.Sex]:
		return dob, "Choose the patient's sex."
	}
	return dob, ""
}

func (s *server) listPatients(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	rows, err := s.q.ListPatients(r.Context(), hospitalID(r))
	if err != nil {
		serverError(w, "list patients", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *server) createPatient(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	var in patientInput
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	dob, msg := in.validate(time.Now())
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	id, err := s.q.CreatePatient(r.Context(), db.CreatePatientParams{
		HospitalID: hospitalID(r), FirstName: in.FirstName, LastName: in.LastName,
		DateOfBirth: pgtype.Date{Time: dob, Valid: true}, Sex: in.Sex,
		HospitalNumber: in.HospitalNumber, Phone: in.Phone, Email: in.Email,
	})
	if err != nil {
		serverError(w, "create patient", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// listPublishedHQs lists the questionnaires an episode can be given: their published versions.
func (s *server) listPublishedHQs(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	rows, err := s.q.ListPublishedVersions(r.Context(), hospitalID(r))
	if err != nil {
		serverError(w, "list published", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *server) listEpisodes(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	rows, err := s.q.ListEpisodes(r.Context(), db.ListEpisodesParams{HospitalID: hospitalID(r), Status: r.URL.Query().Get("status")})
	if err != nil {
		serverError(w, "list episodes", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// createEpisode gives a patient a published HQ and a link to fill it in.
func (s *server) createEpisode(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	var in struct {
		PatientID   uuid.UUID `json:"patientId"`
		VersionID   uuid.UUID `json:"versionId"`
		Procedure   string    `json:"procedure"`
		Anaesthetic string    `json:"anaesthetic"`
		Consultant  string    `json:"consultant"`
	}
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	token, err := episode.NewToken()
	if err != nil {
		serverError(w, "token", err)
		return
	}
	id, err := s.q.CreateEpisode(r.Context(), db.CreateEpisodeParams{
		HospitalID: hospitalID(r), PatientID: in.PatientID, VersionID: in.VersionID,
		Procedure: strings.TrimSpace(in.Procedure), Anaesthetic: strings.TrimSpace(in.Anaesthetic),
		Consultant: strings.TrimSpace(in.Consultant), PatientToken: token, CreatedBy: currentUser(r).ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusBadRequest, "Choose a patient and a published HQ.")
		return
	}
	if err != nil {
		serverError(w, "create episode", err)
		return
	}
	s.event(r, id, "Episode created", currentUser(r).ID)
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// loadEpisode fetches {eid} within the hospital, answering 404 itself.
func (s *server) loadEpisode(w http.ResponseWriter, r *http.Request) (db.GetEpisodeRow, bool) {
	eid, ok := pathID(w, r, "eid", episodeNotFound)
	if !ok {
		return db.GetEpisodeRow{}, false
	}
	row, err := s.q.GetEpisode(r.Context(), db.GetEpisodeParams{ID: eid, HospitalID: hospitalID(r)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, episodeNotFound)
		return row, false
	}
	if err != nil {
		serverError(w, "get episode", err)
		return row, false
	}
	return row, true
}

// getEpisode returns the episode with its patient and its General notes.
func (s *server) getEpisode(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	events, err := s.q.ListEpisodeEvents(r.Context(), e.ID)
	if err != nil {
		serverError(w, "list events", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"episode": e, "events": events})
}

// updateEpisode changes the status (by hand, see episode.CanSetStatus), the procedure details
// and the ASA grades. Fields left out keep their value; an ASA grade of 0 clears it.
func (s *server) updateEpisode(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	// Defaults point at copies, so e keeps the current values to compare with.
	status, procedure, anaesthetic, consultant := e.Status, e.Procedure, e.Anaesthetic, e.Consultant
	in := struct {
		Status          *string `json:"status"`
		Procedure       *string `json:"procedure"`
		Anaesthetic     *string `json:"anaesthetic"`
		Consultant      *string `json:"consultant"`
		NurseAsa        *int32  `json:"nurseAsa"`
		AnaesthetistAsa *int32  `json:"anaesthetistAsa"`
	}{Status: &status, Procedure: &procedure, Anaesthetic: &anaesthetic, Consultant: &consultant}
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	if !episode.CanSetStatus(e.Status, *in.Status) {
		writeError(w, http.StatusConflict, "The status can't be changed to that now.")
		return
	}
	nurse, nurseMsg, okN := asaChange(e.NurseAsa, in.NurseAsa, "Nurse")
	anaes, anaesMsg, okA := asaChange(e.AnaesthetistAsa, in.AnaesthetistAsa, "Anaesthetist")
	if !okN || !okA {
		writeError(w, http.StatusBadRequest, "Choose an ASA grade from 1 to 6.")
		return
	}
	err := s.q.UpdateEpisode(r.Context(), db.UpdateEpisodeParams{
		ID: e.ID, Status: *in.Status, Procedure: strings.TrimSpace(*in.Procedure),
		Anaesthetic: strings.TrimSpace(*in.Anaesthetic), Consultant: strings.TrimSpace(*in.Consultant),
		NurseAsa: nurse, AnaesthetistAsa: anaes,
	})
	if err != nil {
		serverError(w, "update episode", err)
		return
	}
	uid := currentUser(r).ID
	if *in.Status != e.Status {
		s.event(r, e.ID, "Status changed to "+episode.Labels[*in.Status], uid)
	}
	for _, msg := range []string{nurseMsg, anaesMsg} {
		if msg != "" {
			s.event(r, e.ID, msg, uid)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// asaChange applies a requested ASA grade (nil keeps, 0 clears, 1–6 sets) and describes the
// change for the General notes. ok is false for a grade out of range.
func asaChange(current, requested *int32, who string) (next *int32, msg string, ok bool) {
	switch {
	case requested == nil:
		return current, "", true
	case *requested == 0:
		if current == nil {
			return nil, "", true
		}
		return nil, who + " ASA grade cleared", true
	case *requested < 1 || *requested > 6:
		return current, "", false
	case current != nil && *current == *requested:
		return current, "", true
	}
	return requested, fmt.Sprintf("%s ASA grade set to %d", who, *requested), true
}

// addEpisodeNote adds a clinician's comment to the General notes.
func (s *server) addEpisodeNote(w http.ResponseWriter, r *http.Request) {
	if !clinicianOnly(w, r) {
		return
	}
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, 16<<10, &in) {
		return
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "Enter a comment.")
		return
	}
	uid := currentUser(r).ID
	err := s.q.AddEpisodeEvent(r.Context(), db.AddEpisodeEventParams{EpisodeID: e.ID, Kind: "comment", Text: text, UserID: uuid.NullUUID{UUID: uid, Valid: true}})
	if err != nil {
		serverError(w, "add note", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// event records an automatic General note. A failure is logged only: the change itself is saved.
func (s *server) event(r *http.Request, episodeID uuid.UUID, text string, userID uuid.UUID) {
	user := uuid.NullUUID{UUID: userID, Valid: userID != uuid.Nil}
	if err := s.q.AddEpisodeEvent(r.Context(), db.AddEpisodeEventParams{EpisodeID: episodeID, Kind: "event", Text: text, UserID: user}); err != nil {
		log.Printf("episode event: %v", err)
	}
}

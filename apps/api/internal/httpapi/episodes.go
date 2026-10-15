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

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/patient"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

const episodeNotFound = "Episode not found."

// Patients and episodes need no role check: everyone in a hospital is at least a clinician, and
// sees all its data (A-5). The middleware has checked they're a member here.

// listPublishedHQs lists the questionnaires an episode can be given: the published versions of
// those this hospital uses.
func (s *server) listPublishedHQs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListPublishedVersions(r.Context(), db.ListPublishedVersionsParams{OrgID: orgID(r), HospitalID: hospitalID(r)})
	if err != nil {
		serverError(w, "list published", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// listEpisodes is one hospital's episodes, newest first, optionally of one status.
func (s *server) listEpisodes(w http.ResponseWriter, r *http.Request) {
	s.writeEpisodes(w, r, db.ListEpisodesParams{OrgID: orgID(r), HospitalIds: []uuid.UUID{hospitalID(r)}})
}

// listTrustEpisodes is "All hospitals" (STF-03): every hospital's episodes for a trust-wide
// member, else those of the hospitals the caller is a member of.
func (s *server) listTrustEpisodes(w http.ResponseWriter, r *http.Request) {
	p := here(r)
	s.writeEpisodes(w, r, db.ListEpisodesParams{OrgID: p.Org, AllHospitals: p.Role != role.None, HospitalIds: p.Hospitals})
}

func (s *server) writeEpisodes(w http.ResponseWriter, r *http.Request, arg db.ListEpisodesParams) {
	arg.Status = r.URL.Query().Get("status")
	if arg.HospitalIds == nil {
		arg.HospitalIds = []uuid.UUID{}
	}
	rows, err := s.q.ListEpisodes(r.Context(), arg)
	if err != nil {
		serverError(w, "list episodes", err)
		return
	}
	for _, e := range rows {
		auditOf(r).touched(e.PatientID, e.ID)
	}
	writeJSON(w, http.StatusOK, rows)
}

// createEpisode gives a patient a published HQ and a link to fill it in (the New episode page).
// The patient is either one of this hospital's, found by the search (patientId), or typed in
// (patient): then the trust's existing record is linked to silently when the identifiers agree,
// or a new one is created (resolvePatient). The response is the episode's id only.
func (s *server) createEpisode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PatientID      *uuid.UUID    `json:"patientId"`
		Patient        *patientInput `json:"patient"`
		CreateAnyway   bool          `json:"createAnyway"`   // after a possible-duplicate warning
		VersionID      string        `json:"versionId"`      // a string, so "" (nothing chosen) gets its own message
		HospitalNumber string        `json:"hospitalNumber"` // this hospital's label for the patient (PAT-01b)
		Procedure      string        `json:"procedure"`
		Anaesthetic    string        `json:"anaesthetic"`
		Consultant     string        `json:"consultant"`
	}
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	in.HospitalNumber = strings.TrimSpace(in.HospitalNumber)
	versionID, versionErr := uuid.Parse(in.VersionID)
	var typed patient.Details
	var sex, msg string
	switch {
	case (in.PatientID == nil) == (in.Patient == nil):
		msg = "Choose a patient from the search, or enter a new patient's details."
	case versionErr != nil || versionID == uuid.Nil:
		msg = "Choose a published HQ."
	case len(in.HospitalNumber) > 50:
		msg = "The hospital number is too long (50 characters at most)."
	case in.Patient != nil:
		typed, sex, msg = s.details(*in.Patient, time.Now())
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
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
	var pid uuid.UUID
	if in.PatientID != nil {
		p, err := q.GetHospitalPatient(ctx, db.GetHospitalPatientParams{ID: *in.PatientID, OrgID: orgID(r), HospitalID: hospitalID(r)})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "That isn't one of this hospital's patients. Search again, or enter their details.")
			return
		}
		if err != nil {
			serverError(w, "get patient", err)
			return
		}
		pid = p.ID
	} else {
		pid, err = s.resolvePatient(ctx, w, r, q, typed, sex, in.CreateAnyway)
		if errors.Is(err, errAnswered) {
			return
		}
		if err != nil {
			serverError(w, "patient record", err)
			return
		}
	}
	id, err := q.CreateEpisode(ctx, db.CreateEpisodeParams{
		OrgID: orgID(r), HospitalID: hospitalID(r), PatientID: pid, VersionID: versionID,
		Procedure: strings.TrimSpace(in.Procedure), Anaesthetic: strings.TrimSpace(in.Anaesthetic),
		Consultant: strings.TrimSpace(in.Consultant), HospitalNumber: in.HospitalNumber,
		CreatedBy: currentUser(r).ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusBadRequest, "Choose a published HQ.")
		return
	}
	if err != nil {
		serverError(w, "create episode", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return
	}
	auditOf(r).touched(pid, id)
	s.event(r, id, "Episode created", currentUser(r).ID)
	s.inviteAfterCreate(r, id) // by SMS and email (PAT-03); a failure is noted, never refused
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
	auditOf(r).touched(row.PatientID, row.ID)
	return row, true
}

// getEpisode returns the episode with its patient, its General notes and its invite.
func (s *server) getEpisode(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	events, err := s.q.ListEpisodeEvents(r.Context(), e.ID)
	if err != nil {
		serverError(w, "list events", err)
		return
	}
	// The patient's invite (S14): null until one is sent; lockedAt once five wrong dates of birth lock it.
	var invite *db.GetEpisodeInviteRow
	inv, err := s.q.GetEpisodeInvite(r.Context(), uuid.NullUUID{UUID: e.ID, Valid: true})
	switch {
	case err == nil:
		invite = &inv
	case !errors.Is(err, pgx.ErrNoRows):
		serverError(w, "episode invite", err)
		return
	}
	// The texts and emails that didn't reach the patient since the latest invite or Resend (S17, OPS-05).
	undelivered, err := s.q.ListUndeliveredMessages(r.Context(), e.ID)
	if err != nil {
		serverError(w, "undelivered messages", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"episode": e, "events": events, "invite": invite, "undelivered": undelivered})
}

// episodeChange is what updateEpisode changes. A field sent as null decodes to a nil pointer;
// for an ASA grade that keeps the current grade (see asaChange).
type episodeChange struct {
	Status          *string `json:"status"`
	Procedure       *string `json:"procedure"`
	Anaesthetic     *string `json:"anaesthetic"`
	Consultant      *string `json:"consultant"`
	NurseAsa        *int32  `json:"nurseAsa"`
	AnaesthetistAsa *int32  `json:"anaesthetistAsa"`
}

// validate trims the procedure details and returns a plain-language problem, or "".
func (in *episodeChange) validate() string {
	if in.Status == nil || in.Procedure == nil || in.Anaesthetic == nil || in.Consultant == nil {
		return "The request could not be read."
	}
	for _, f := range []*string{in.Procedure, in.Anaesthetic, in.Consultant} {
		*f = strings.TrimSpace(*f)
	}
	return ""
}

// updateEpisode changes the status (by hand, see episode.CanSetStatus), the procedure details
// and the ASA grades. Fields left out keep their value; an ASA grade of 0 clears it.
func (s *server) updateEpisode(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	// Defaults point at copies, so e keeps the current values to compare with.
	status, procedure, anaesthetic, consultant := e.Status, e.Procedure, e.Anaesthetic, e.Consultant
	in := episodeChange{Status: &status, Procedure: &procedure, Anaesthetic: &anaesthetic, Consultant: &consultant}
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
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
	n, err := s.q.UpdateEpisode(r.Context(), db.UpdateEpisodeParams{
		ID: e.ID, LoadedStatus: e.Status, Status: *in.Status, Procedure: *in.Procedure, Anaesthetic: *in.Anaesthetic,
		Consultant: *in.Consultant, NurseAsa: nurse, AnaesthetistAsa: anaes,
	})
	if err != nil {
		serverError(w, "update episode", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "This episode changed since you opened it. Reload to see the latest; your change was not saved.")
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

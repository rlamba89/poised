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
	"github.com/rlamba89/poised/apps/api/internal/patient"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

const episodeNotFound = "Episode not found."

// Patients and episodes need no role check: everyone in a hospital is at least a clinician, and
// sees all its data (A-5). The middleware has checked they're a member here.

// publishedHQ is an HQ an episode can be given. isDefault marks the hospital's default HQ (its
// defaultHq setting, S10), which the New episode page chooses first (S19).
type publishedHQ struct {
	VersionID       uuid.UUID `json:"versionId"`
	VersionNo       int32     `json:"versionNo"`
	Name            string    `json:"name"`
	QuestionnaireID uuid.UUID `json:"questionnaireId"`
	IsDefault       bool      `json:"isDefault"`
}

// listPublishedHQs lists the questionnaires an episode can be given: the published versions of
// those this hospital uses.
func (s *server) listPublishedHQs(w http.ResponseWriter, r *http.Request) {
	rows, def, err := s.hospitalHQs(r.Context(), orgID(r), hospitalID(r))
	if err != nil {
		serverError(w, "list published", err)
		return
	}
	out := []publishedHQ{}
	for _, h := range rows {
		out = append(out, publishedHQ{VersionID: h.VersionID, VersionNo: h.VersionNo, Name: h.Name, QuestionnaireID: h.QuestionnaireID,
			IsDefault: def.Valid && def.UUID == h.QuestionnaireID})
	}
	writeJSON(w, http.StatusOK, out)
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

// writeEpisodes answers a list, with ?status= and ?worklist= (S21, EPI-02) applied.
func (s *server) writeEpisodes(w http.ResponseWriter, r *http.Request, arg db.ListEpisodesParams) {
	arg.Status = r.URL.Query().Get("status")
	arg.Worklist = r.URL.Query().Get("worklist")
	arg.IncludeArchived = r.URL.Query().Get("archived") == "true" // "Show archived" (S22): left out otherwise
	if arg.Worklist != "" && !episode.IsWorklist(arg.Worklist) {
		writeError(w, http.StatusBadRequest, "Choose a worklist.")
		return
	}
	if arg.HospitalIds == nil {
		arg.HospitalIds = []uuid.UUID{}
	}
	arg.Now, arg.DefaultZone, arg.SoonDays = s.clock.Now(), s.cfg.Country.TimeZone, defaultSoonDays()
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
		ProcedureID    string        `json:"procedureId"`    // one the trust's list offers here (S19), or "" for none
		PlannedDate    string        `json:"plannedDate"`    // YYYY-MM-DD, or "" (EPI-01)
		PoaDate        string        `json:"poaDate"`
		Anaesthetic    string        `json:"anaesthetic"`
		Consultant     string        `json:"consultant"`
	}
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	in.HospitalNumber = strings.TrimSpace(in.HospitalNumber)
	versionID, versionErr := uuid.Parse(in.VersionID)
	dates, msg := readEpisodeDates(in.ProcedureID, in.PlannedDate, in.PoaDate)
	var typed patient.Details
	var sex string
	switch {
	case msg != "":
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
	if dates.procedure.Valid && !procedureOffered(w, r, q, dates.procedure.UUID) {
		return
	}
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
		ProcedureID: dates.procedure, PlannedDate: dates.planned, PoaDate: dates.poa, Anaesthetic: strings.TrimSpace(in.Anaesthetic),
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
	// The procedure's Information list is copied onto the episode (S36 answers 2): a booked patient
	// keeps the list from when they were booked.
	if dates.procedure.Valid {
		if err := q.CopyProcedureInfo(ctx, db.CopyProcedureInfoParams{EpisodeID: id, ProcedureID: dates.procedure.UUID, OrgID: orgID(r)}); err != nil {
			serverError(w, "copy information", err)
			return
		}
	}
	// Its first status, in the history (EPI-03); General notes say "Episode created" below.
	if err := recordStatus(ctx, q, id, episode.Create, "", episode.HQNotComplete, "", currentUser(r).ID, ""); err != nil {
		serverError(w, "status history", err)
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
	// The status (S20): the moves this user may make from here, the undo, and the history.
	status, err := s.statusView(r.Context(), e, here(r).Role)
	if err != nil {
		serverError(w, "status", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"episode": e, "events": events, "invite": invite, "undelivered": undelivered,
		"moves": status.Moves, "undo": status.Undo, "unarchive": status.Unarchive, "statusHistory": status.History})
}

// episodeChange is what updateEpisode changes. A field sent as null decodes to a nil pointer;
// for an ASA grade that keeps the current grade (see asaChange), and for the procedure and the
// dates it clears them. Revision is the episode's revision the page loaded (S20): required.
type episodeChange struct {
	Revision        *int32  `json:"revision"`
	Status          *string `json:"status"`      // refused: the status has its own routes (S20), so an old page can't think it saved one
	ProcedureID     *string `json:"procedureId"` // S19: an id, or "" for none
	PlannedDate     *string `json:"plannedDate"` // YYYY-MM-DD, or ""
	PoaDate         *string `json:"poaDate"`
	Anaesthetic     *string `json:"anaesthetic"`
	Consultant      *string `json:"consultant"`
	NurseAsa        *int32  `json:"nurseAsa"`
	AnaesthetistAsa *int32  `json:"anaesthetistAsa"`
}

// validate trims the procedure details and returns a plain-language problem, or "".
func (in *episodeChange) validate() string {
	if in.Revision == nil || in.Anaesthetic == nil || in.Consultant == nil {
		return "The request could not be read."
	}
	if in.Status != nil {
		return "The status is changed on its own (PATCH …/status)."
	}
	for _, f := range []*string{in.Anaesthetic, in.Consultant} {
		*f = strings.TrimSpace(*f)
	}
	return ""
}

// updateEpisode changes the procedure details and the ASA grades; the status has its own routes
// (statuses.go, S20). Fields left out keep their value; an ASA grade of 0 clears it. The
// procedure and its dates can be changed until the HQ review is complete (S19); a new procedure
// must be one the trust's list offers here, while the one the episode has stays even if retired.
// The revision sent must be the episode's current one (409 otherwise), so of two tabs, the second
// to save is told, not silently overwriting the first (pending.md §1a).
func (s *server) updateEpisode(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	// Defaults point at copies, so e keeps the current values to compare with.
	anaesthetic, consultant := e.Anaesthetic, e.Consultant
	was := episodeDates{procedure: e.ProcedureID, planned: e.PlannedDate, poa: e.PoaDate}
	procedure, planned, poa := was.strings()
	in := episodeChange{ProcedureID: &procedure, PlannedDate: &planned, PoaDate: &poa,
		Anaesthetic: &anaesthetic, Consultant: &consultant}
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	msg := in.validate()
	var next episodeDates
	if msg == "" {
		next, msg = readEpisodeDates(deref(in.ProcedureID), deref(in.PlannedDate), deref(in.PoaDate))
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if e.Revision != *in.Revision {
		writeStale(w)
		return
	}
	if msg := readOnly(e.Status); msg != "" {
		writeError(w, http.StatusConflict, msg)
		return
	}
	if next != was && e.ReviewCompletedAt.Valid {
		writeError(w, http.StatusConflict, "The procedure and its dates can't be changed once the HQ review is complete.")
		return
	}
	procedureName := e.Procedure
	if next.procedure != was.procedure && next.procedure.Valid {
		if procedureName, ok = offeredProcedureName(w, r, s.q, next.procedure.UUID); !ok {
			return
		}
	}
	nurse, nurseMsg, okN := asaChange(e.NurseAsa, in.NurseAsa, "Nurse")
	anaes, anaesMsg, okA := asaChange(e.AnaesthetistAsa, in.AnaesthetistAsa, "Anaesthetist")
	if !okN || !okA {
		writeError(w, http.StatusBadRequest, "Choose an ASA grade from 1 to 6.")
		return
	}
	saved := s.inTx(w, r, func(q *db.Queries) bool {
		n, err := q.UpdateEpisode(r.Context(), db.UpdateEpisodeParams{
			ID: e.ID, Revision: *in.Revision, ProcedureID: next.procedure, PlannedDate: next.planned,
			PoaDate: next.poa, Anaesthetic: *in.Anaesthetic,
			Consultant: *in.Consultant, NurseAsa: nurse, AnaesthetistAsa: anaes,
		})
		if err != nil {
			serverError(w, "update episode", err)
			return false
		}
		if n == 0 {
			writeStale(w)
			return false
		}
		if next.procedure != was.procedure { // S36: the items copied from the old procedure give way to the new one's
			return s.swapProcedureInfo(w, r, q, e.ID, next.procedure)
		}
		return true
	})
	if !saved {
		return
	}
	uid := currentUser(r).ID
	for _, msg := range append(was.changes(next, procedureName), nurseMsg, anaesMsg) {
		if msg != "" {
			s.event(r, e.ID, msg, uid)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// swapProcedureInfo replaces the Information items an episode copied from its procedure with its
// new procedure's list (none if it has none now). The items a clinician added stay (S36).
func (s *server) swapProcedureInfo(w http.ResponseWriter, r *http.Request, q *db.Queries, episodeID uuid.UUID, procedure uuid.NullUUID) bool {
	if err := q.ClearProcedureInfo(r.Context(), episodeID); err != nil {
		serverError(w, "clear information", err)
		return false
	}
	if !procedure.Valid {
		return true
	}
	if err := q.CopyProcedureInfo(r.Context(), db.CopyProcedureInfoParams{EpisodeID: episodeID, ProcedureID: procedure.UUID, OrgID: orgID(r)}); err != nil {
		serverError(w, "copy information", err)
		return false
	}
	return true
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
	if e.Status == episode.Archived { // S22 answers 2: notes too (a cancelled episode still takes them)
		writeError(w, http.StatusConflict, archivedMsg)
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

// readOnly is why an episode in this status can't be changed (cancelled, S20; archived, S22), or "".
func readOnly(status string) string {
	switch status {
	case episode.Cancelled:
		return cancelledMsg
	case episode.Archived:
		return archivedMsg
	}
	return ""
}

// event records an automatic General note. A failure is logged only: the change itself is saved.
func (s *server) event(r *http.Request, episodeID uuid.UUID, text string, userID uuid.UUID) {
	user := uuid.NullUUID{UUID: userID, Valid: userID != uuid.Nil}
	if err := s.q.AddEpisodeEvent(r.Context(), db.AddEpisodeEventParams{EpisodeID: episodeID, Kind: "event", Text: text, UserID: user}); err != nil {
		log.Printf("episode event: %v", err)
	}
}

// episodeDates are an episode's procedure (S19) and its two dates (EPI-01), each optional.
type episodeDates struct {
	procedure    uuid.NullUUID
	planned, poa pgtype.Date
}

// readEpisodeDates reads a procedure id and the planned procedure and POA appointment dates
// (YYYY-MM-DD), each "" for none. It returns a plain-language problem, or "".
func readEpisodeDates(procedure, planned, poa string) (episodeDates, string) {
	var d episodeDates
	if p := strings.TrimSpace(procedure); p != "" {
		id, err := uuid.Parse(p)
		if err != nil {
			return d, procedureNotOffered
		}
		d.procedure = uuid.NullUUID{UUID: id, Valid: true}
	}
	var ok bool
	if d.planned, ok = optionalDate(planned); !ok {
		return d, "Enter the planned procedure date as a date."
	}
	if d.poa, ok = optionalDate(poa); !ok {
		return d, "Enter the POA appointment date as a date."
	}
	return d, ""
}

// optionalDate reads a YYYY-MM-DD date in the years 1900 to 2199, or "" for none.
func optionalDate(s string) (pgtype.Date, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Date{}, true
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil || t.Year() < 1900 || t.Year() > 2199 {
		return pgtype.Date{}, false
	}
	return pgtype.Date{Time: t, Valid: true}, true
}

// strings are the fields as the PATCH body sends them: an id or "", a date or "".
func (d episodeDates) strings() (procedure, planned, poa string) {
	if d.procedure.Valid {
		procedure = d.procedure.UUID.String()
	}
	return procedure, dateString(d.planned), dateString(d.poa)
}

func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(time.DateOnly)
}

// changes describe the change from d to next for the General notes; name is next's procedure's.
func (d episodeDates) changes(next episodeDates, name string) []string {
	var out []string
	if next.procedure != d.procedure {
		if next.procedure.Valid {
			out = append(out, "Procedure set to "+name)
		} else {
			out = append(out, "Procedure cleared")
		}
	}
	for _, c := range []struct {
		label     string
		was, next pgtype.Date
	}{{"Planned procedure date", d.planned, next.planned}, {"POA appointment date", d.poa, next.poa}} {
		switch {
		case c.was == c.next:
		case c.next.Valid:
			out = append(out, c.label+" set to "+dateString(c.next))
		default:
			out = append(out, c.label+" cleared")
		}
	}
	return out
}

const procedureNotOffered = "Choose a procedure from the trust's list for this hospital."

// offeredProcedureName is the procedure's name if an episode here can be given it now (not
// retired, the trust's or this hospital's own), answering 400 itself if it can't.
func offeredProcedureName(w http.ResponseWriter, r *http.Request, q db.Querier, id uuid.UUID) (string, bool) {
	name, err := q.OfferedProcedureName(r.Context(), db.OfferedProcedureNameParams{ID: id, OrgID: orgID(r), HospitalID: hospitalID(r)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusBadRequest, procedureNotOffered)
		return "", false
	}
	if err != nil {
		serverError(w, "procedure", err)
		return "", false
	}
	return name, true
}

func procedureOffered(w http.ResponseWriter, r *http.Request, q db.Querier, id uuid.UUID) bool {
	_, ok := offeredProcedureName(w, r, q, id)
	return ok
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/prefill"
)

// Pre-fill (plan C4, session S25; PRE-01–03, PRE-06). When the HQ form is opened, by the patient or
// by staff on their behalf (PAT-09), each patient Question Set with no answers yet is filled from
// the patient's earlier submitted HQs in the same trust (internal/prefill has the rules), and the
// result is stored as the patient's answers in the same request: once per set, never over anything
// saved. Each set's opening card then asks the patient to check them, and records the button
// pressed (prefillChoice).

// prefillForm fills the form's unanswered patient Question Sets (chapters: the form's version's,
// saved: its answers so far). It reports whether it tried to store any, so the caller reads the
// answers again: a second open at the same moment (another tab, React's double effect in
// development) stores nothing itself, but must still send what the first one stored. Only the
// form's own trust is read (S26 adds other trusts, deliberately).
func (s *server) prefillForm(ctx context.Context, f form, a *auditEntry, chapters []db.ListVersionChaptersRow, saved []db.ListEpisodeAnswersRow) (bool, error) {
	if f.PatientSubmittedAt.Valid {
		return false, nil
	}
	answered := map[uuid.UUID]bool{}
	for _, row := range saved {
		if row.Actor == "patient" {
			answered[row.ChapterID] = true
		}
	}
	target := make([]prefill.Chapter, len(chapters))
	fill := map[string]bool{}
	for i, c := range chapters {
		target[i] = prefill.Chapter{ID: c.ID.String(), Audience: c.Audience, Content: c.Content}
		if c.Audience == "patient" && !answered[c.ID] {
			fill[c.ID.String()] = true
		}
	}
	if len(fill) == 0 {
		return false, nil
	}
	sources, err := s.prefillSources(ctx, f.OrgID, f, a)
	if err != nil || len(sources) == 0 {
		return false, err
	}
	filled, err := prefill.Fill(target, fill, sources)
	if err != nil {
		return false, err
	}
	for cid, got := range filled {
		data, err := json.Marshal(got.Data)
		if err != nil {
			return false, err
		}
		meta, err := json.Marshal(got.Prefill)
		if err != nil {
			return false, err
		}
		// No row: the patient (or another open) saved this set first, and that stays.
		if _, err := s.q.PrefillPatientAnswers(ctx, db.PrefillPatientAnswersParams{FormID: f.FormID, ChapterID: uuid.MustParse(cid), Data: data, Prefill: meta}); err != nil {
			return false, err
		}
	}
	return len(filled) > 0, nil
}

// prefillSources reads the patient's earlier submitted HQ forms in one trust, newest first, each
// with its Question Sets and answers (ListPrefillSources). Their episodes go in the request's
// audit row: their answers were read.
func (s *server) prefillSources(ctx context.Context, org uuid.UUID, f form, a *auditEntry) ([]prefill.Source, error) {
	rows, err := s.q.ListPrefillSources(ctx, db.ListPrefillSourcesParams{OrgID: org, FormID: f.FormID, PatientID: f.PatientID, AccountID: f.AccountID})
	if err != nil {
		return nil, err
	}
	var out []prefill.Source
	for _, r := range rows {
		if len(out) == 0 || out[len(out)-1].FormID != r.FormID.String() {
			out = append(out, prefill.Source{FormID: r.FormID.String(), SubmittedAt: r.SubmittedAt.Time,
				Patient: map[string]json.RawMessage{}, Clinician: map[string]json.RawMessage{}})
			a.touched(r.PatientID, r.EpisodeID)
		}
		src := &out[len(out)-1]
		cid := r.ChapterID.String()
		src.Chapters = append(src.Chapters, prefill.Chapter{ID: cid, Audience: r.Audience, Content: r.Content})
		if r.PatientData != nil {
			src.Patient[cid] = r.PatientData
		}
		if r.ClinicianData != nil {
			src.Clinician[cid] = r.ClinicianData
		}
	}
	return out, nil
}

// setPrefill is what the form sends about one pre-filled Question Set: where each answer came from
// (prefill.Entry by key) and the opening card's button, once pressed.
type setPrefill struct {
	Answers  json.RawMessage `json:"answers"`
	Choice   *string         `json:"choice"`
	ChoiceAt *time.Time      `json:"choiceAt"`
}

// prefillOf is a patient row's pre-fill, if Go filled anything in it.
func prefillOf(row db.ListEpisodeAnswersRow) (setPrefill, bool) {
	var entries map[string]json.RawMessage
	if json.Unmarshal(row.Prefill, &entries) != nil || len(entries) == 0 {
		return setPrefill{}, false
	}
	p := setPrefill{Answers: row.Prefill, Choice: row.PrefillChoice}
	if row.PrefillChoiceAt.Valid {
		p.ChoiceAt = &row.PrefillChoiceAt.Time
	}
	return p, true
}

// The opening card's buttons (PRE-02, S25 answers 5).
var prefillChoices = map[string]bool{"check_each": true, "still_correct": true}

// prefillChoice records the opening card's button for one pre-filled Question Set: POST
// …/prefill/{cid} {choice: check_each | still_correct}, by the patient or staff for them. The first
// button pressed is kept; pressing again (another tab) changes nothing.
func (s *server) prefillChoice(w http.ResponseWriter, r *http.Request, f form) {
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
	var in struct {
		Choice string `json:"choice"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	if !prefillChoices[in.Choice] {
		writeError(w, http.StatusBadRequest, "Choose “Check each answer” or “Everything is still correct”.")
		return
	}
	ctx := r.Context()
	n, err := s.q.RecordPrefillChoice(ctx, db.RecordPrefillChoiceParams{
		Choice: in.Choice, By: uuid.NullUUID{UUID: f.enteredBy, Valid: f.enteredBy != uuid.Nil}, FormID: f.FormID, ChapterID: cid,
	})
	if err != nil {
		serverError(w, "record choice", err)
		return
	}
	if n == 0 {
		if f.PatientSubmittedAt.Valid {
			writeError(w, http.StatusConflict, "These answers have already been sent.")
			return
		}
		rows, err := s.q.ListEpisodeAnswers(ctx, f.FormID)
		if err != nil {
			serverError(w, "list answers", err)
			return
		}
		for _, row := range rows {
			if p, ok := prefillOf(row); ok && row.Actor == "patient" && row.ChapterID == cid && p.Choice != nil {
				w.WriteHeader(http.StatusNoContent) // already recorded: the first button stays
				return
			}
		}
		writeError(w, http.StatusConflict, "Nothing in this section was filled in from earlier answers.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) patientPrefillChoice(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadPatientForm(w, r); ok {
		s.prefillChoice(w, r, f)
	}
}

func (s *server) staffPrefillChoice(w http.ResponseWriter, r *http.Request) {
	if f, ok := s.loadStaffForm(w, r); ok {
		s.prefillChoice(w, r, f)
	}
}

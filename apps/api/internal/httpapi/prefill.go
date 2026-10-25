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
// pressed (prefillChoice). Other trusts' answers (S26) are always used since S27 (Rahul, 24 Oct),
// by the patient and by staff for them, with no setting and no question: only for what the form's
// own trust's forms didn't ask, read one trust per query and audited under that trust too. A set
// holding them opens on a card naming the hospital, with Start fresh as a third button.

// prefillForm fills the form's unanswered patient Question Sets (chapters: the form's version's)
// on its first open only (S28 answers 3): the whole pre-fill, this trust's answers and other
// trusts', runs once per form, and the form records it (episode_forms.prefilled_at), so later
// opens never read earlier HQs again and each source trust gets one prefill.read row per form. The
// form is locked while it runs: a second open at the same moment (another tab, React's double
// effect in development) waits, then finds it done. The caller reads the answers afterwards.
//
// Answers come from the form's own trust, then from every other trust where the person's account
// has a sent HQ (S27: always, for the patient and for staff on their behalf), each read with its
// own query; another trust's go only where the form's own trust's forms asked nothing.
func (s *server) prefillForm(ctx context.Context, f form, a *auditEntry, chapters []db.ListVersionChaptersRow) error {
	if f.PatientSubmittedAt.Valid || f.PrefilledAt.Valid {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	done, err := q.LockFormPrefill(ctx, f.FormID)
	if err != nil || done {
		return err
	}
	if err := s.fillForm(ctx, q, f, a, chapters); err != nil {
		return err
	}
	if err := q.MarkFormPrefilled(ctx, f.FormID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// fillForm is prefillForm's work, inside its transaction (q).
func (s *server) fillForm(ctx context.Context, q *db.Queries, f form, a *auditEntry, chapters []db.ListVersionChaptersRow) error {
	saved, err := q.ListEpisodeAnswers(ctx, f.FormID)
	if err != nil {
		return err
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
		return nil
	}
	sources, err := prefillSources(ctx, q, f.OrgID, f, a, false)
	if err != nil {
		return err
	}
	var others []uuid.UUID
	if f.AccountID.Valid { // other trusts' records are found only through the person's account (S15)
		if others, err = q.ListPrefillTrusts(ctx, db.ListPrefillTrustsParams{AccountID: f.AccountID, OrgID: f.OrgID}); err != nil {
			return err
		}
	}
	for _, org := range others {
		// One trust per query, with its own org_id; its log gets a row of its own (S26 answers 5).
		theirs, err := prefillSources(ctx, q, org, f, a, true)
		if err != nil {
			return err
		}
		sources = append(sources, theirs...)
	}
	if len(sources) == 0 {
		return nil
	}
	// Another trust's question matches by key only for a platform library key (S26 answers, after
	// verification 1): read them only when another trust's answers are used.
	var library map[string]bool
	if len(others) > 0 {
		elements, err := q.ListPlatformLibraryElements(ctx)
		if err != nil {
			return err
		}
		library = prefill.LibraryKeys(elements)
	}
	filled, err := prefill.Fill(target, fill, sources, library)
	if err != nil {
		return err
	}
	for cid, got := range filled {
		data, err := json.Marshal(got.Data)
		if err != nil {
			return err
		}
		meta, err := json.Marshal(got.Prefill)
		if err != nil {
			return err
		}
		// No row: the patient saved this set first (in another tab), and that stays.
		if _, err := q.PrefillPatientAnswers(ctx, db.PrefillPatientAnswersParams{FormID: f.FormID, ChapterID: uuid.MustParse(cid), Data: data, Prefill: meta}); err != nil {
			return err
		}
	}
	return nil
}

// prefillSources reads the patient's earlier submitted HQ forms in one trust, newest first, each
// with its Question Sets and answers (ListPrefillSources). Their episodes go in the request's
// audit row: their answers were read. Another trust's (otherTrust) go in a row of that trust's own,
// at the source episode's hospital, never in this trust's (S26 answers 5).
func prefillSources(ctx context.Context, q *db.Queries, org uuid.UUID, f form, a *auditEntry, otherTrust bool) ([]prefill.Source, error) {
	rows, err := q.ListPrefillSources(ctx, db.ListPrefillSourcesParams{OrgID: org, FormID: f.FormID, PatientID: f.PatientID, AccountID: f.AccountID})
	if err != nil {
		return nil, err
	}
	var out []prefill.Source
	for _, r := range rows {
		if len(out) == 0 || out[len(out)-1].FormID != r.FormID.String() {
			src := prefill.Source{FormID: r.FormID.String(), SubmittedAt: r.SubmittedAt.Time, Questionnaire: r.QuestionnaireName,
				Patient: map[string]json.RawMessage{}, Clinician: map[string]json.RawMessage{}}
			if otherTrust {
				src.OtherTrust, src.Hospital = true, r.HospitalName
				a.also("prefill.read", org, r.HospitalID).touched(r.PatientID, r.EpisodeID)
			} else {
				a.touched(r.PatientID, r.EpisodeID)
			}
			out = append(out, src)
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

// The opening card's buttons (PRE-02, S25 answers 5), and Start fresh on a card for another
// trust's answers (S27).
var prefillChoices = map[string]bool{"check_each": true, "still_correct": true, "start_fresh": true}

// prefillChoice records the opening card's button for one pre-filled Question Set: POST
// …/prefill/{cid} {choice: check_each | still_correct | start_fresh}, by the patient or staff for
// them. The first button pressed is kept; pressing it again (another tab) changes nothing. Start
// fresh also clears the set's answers (startFresh), so it's refused once another button was
// pressed: the patient would think the answers had gone.
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
		writeError(w, http.StatusBadRequest, "Choose “Check each answer”, “Everything is still correct” or “Start fresh”.")
		return
	}
	// The button pressed names the row (S29; Rahul, 25 Oct): behalf.prefill_start_fresh and so on,
	// so the log alone shows who confirmed the answers and who cleared them. A request refused
	// before this point keeps the route's own action.
	a := auditOf(r)
	a.action += "_" + in.Choice
	ctx := r.Context()
	by := uuid.NullUUID{UUID: f.enteredBy, Valid: f.enteredBy != uuid.Nil}
	if in.Choice == "start_fresh" {
		s.startFresh(w, r, f, cid, by)
		return
	}
	n, err := s.q.RecordPrefillChoice(ctx, db.RecordPrefillChoiceParams{Choice: in.Choice, By: by, FormID: f.FormID, ChapterID: cid})
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
				if *p.Choice == "start_fresh" {
					// Start fresh emptied the set elsewhere (another tab, or the patient while staff
					// had the card open): this card's answers are gone, so it mustn't go on as if
					// they'd been checked (S27 verification 1, finding 1).
					writeError(w, http.StatusConflict, startedFreshMsg)
					return
				}
				w.WriteHeader(http.StatusNoContent) // already recorded: the first button stays
				return
			}
		}
		writeError(w, http.StatusConflict, "Nothing in this section was filled in from earlier answers.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// startFresh is the card's Start fresh for one Question Set (S27 answers 1–2): its pre-filled
// answers, from any trust, are cleared and the button recorded, only while the set holds another
// trust's answers and is as Go filled it. A second press of Start fresh changes nothing (204).
func (s *server) startFresh(w http.ResponseWriter, r *http.Request, f form, cid uuid.UUID, by uuid.NullUUID) {
	ctx := r.Context()
	n, err := s.q.StartFreshPrefill(ctx, db.StartFreshPrefillParams{By: by, FormID: f.FormID, ChapterID: cid})
	if err != nil {
		serverError(w, "start fresh", err)
		return
	}
	if n > 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
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
		if row.Actor != "patient" || row.ChapterID != cid {
			continue
		}
		p, ok := prefillOf(row)
		switch {
		case !ok:
		case p.Choice != nil && *p.Choice == "start_fresh":
			w.WriteHeader(http.StatusNoContent) // already done (another tab)
			return
		case !fromOtherTrust(p.Answers):
			writeError(w, http.StatusConflict, "Start fresh is only for answers copied from another hospital's questionnaire.")
			return
		default: // another button was pressed, or the answers were changed since
			writeError(w, http.StatusConflict, "The answers in this section have changed since you opened it, so they weren't cleared. Please reload the page.")
			return
		}
	}
	writeError(w, http.StatusConflict, "Nothing in this section was filled in from earlier answers.")
}

// startedFreshMsg refuses a card's other button once Start fresh has cleared the set elsewhere.
const startedFreshMsg = "The answers in this section were cleared with “Start fresh” somewhere else, so there's nothing here to check. Please reload the page."

// fromOtherTrust reports whether a set's pre-fill map holds another trust's answer.
func fromOtherTrust(entries json.RawMessage) bool {
	var m map[string]prefill.Entry
	if json.Unmarshal(entries, &m) != nil {
		return false
	}
	for _, e := range m {
		if e.OtherTrust {
			return true
		}
	}
	return false
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

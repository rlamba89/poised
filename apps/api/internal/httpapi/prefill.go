package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/prefill"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// Pre-fill (plan C4, session S25; PRE-01–03, PRE-06). When the HQ form is opened, by the patient or
// by staff on their behalf (PAT-09), each patient Question Set with no answers yet is filled from
// the patient's earlier submitted HQs in the same trust (internal/prefill has the rules), and the
// result is stored as the patient's answers in the same request: once per set, never over anything
// saved. Each set's opening card then asks the patient to check them, and records the button
// pressed (prefillChoice). Other trusts' answers (S26) are used only with both trusts' setting on
// and the patient's Yes, never on their behalf (crossTrust, prefillConsent).

// prefillForm fills the form's unanswered patient Question Sets (chapters: the form's version's,
// saved: its answers so far). It reports whether it tried to store any, so the caller reads the
// answers again: a second open at the same moment (another tab, React's double effect in
// development) stores nothing itself, but must still send what the first one stored.
//
// Answers come from the form's own trust, and (S26) from other trusts the patient agreed to, only
// when the patient opens it themselves (crossTrust). While another trust's question is still to be
// answered, nothing is filled and the questions are returned: the patient answers them first (the
// consent screen comes before the first Question Set), and the next open fills from every source at
// once, so another trust's answers go only where the form's own trust has nothing.
func (s *server) prefillForm(ctx context.Context, f form, a *auditEntry, chapters []db.ListVersionChaptersRow, saved []db.ListEpisodeAnswersRow) (bool, []consentQuestion, error) {
	if f.PatientSubmittedAt.Valid {
		return false, nil, nil
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
		return false, nil, nil
	}
	var agreed []uuid.UUID
	if f.enteredBy == uuid.Nil { // never on the patient's behalf (S26 answers 4)
		questions, yes, err := s.crossTrust(ctx, f)
		if err != nil || len(questions) > 0 {
			return false, questions, err
		}
		agreed = yes
	}
	sources, err := s.prefillSources(ctx, f.OrgID, f, a, false)
	if err != nil {
		return false, nil, err
	}
	for _, org := range agreed {
		// One trust per query, with its own org_id; its log gets a row of its own (S26 answers 5).
		theirs, err := s.prefillSources(ctx, org, f, a, true)
		if err != nil {
			return false, nil, err
		}
		sources = append(sources, theirs...)
	}
	if len(sources) == 0 {
		return false, nil, nil
	}
	// Another trust's question matches by key only for a platform library key (S26 answers, after
	// verification 1): read them only when another trust's answers are used.
	var library map[string]bool
	if len(agreed) > 0 {
		elements, err := s.q.ListPlatformLibraryElements(ctx)
		if err != nil {
			return false, nil, err
		}
		library = prefill.LibraryKeys(elements)
	}
	filled, err := prefill.Fill(target, fill, sources, library)
	if err != nil {
		return false, nil, err
	}
	for cid, got := range filled {
		data, err := json.Marshal(got.Data)
		if err != nil {
			return false, nil, err
		}
		meta, err := json.Marshal(got.Prefill)
		if err != nil {
			return false, nil, err
		}
		// No row: the patient (or another open) saved this set first, and that stays.
		if _, err := s.q.PrefillPatientAnswers(ctx, db.PrefillPatientAnswersParams{FormID: f.FormID, ChapterID: uuid.MustParse(cid), Data: data, Prefill: meta}); err != nil {
			return false, nil, err
		}
	}
	return len(filled) > 0, nil, nil
}

// consentQuestion is "Use the answers you gave <hospital> in <month>?" for one other trust (S26
// answers 3): its id (the answer names it), and the hospital and date of its newest sent HQ.
type consentQuestion struct {
	OrgID       uuid.UUID `json:"orgId"`
	Hospital    string    `json:"hospital"`
	SubmittedAt time.Time `json:"submittedAt"`
}

// crossTrust is, for the patient's own form, the other trusts whose answers could pre-fill it
// (S26 answers 2 and 3): only if this trust's crossTrustPrefill is on, and then each active trust
// where the person's account has a sent HQ and whose own setting is on. It returns the ones still
// to ask about, and the ones the person said Yes to. It reads no answers.
func (s *server) crossTrust(ctx context.Context, f form) ([]consentQuestion, []uuid.UUID, error) {
	if !f.AccountID.Valid {
		return nil, nil, nil
	}
	mine, err := s.hospitalSettings(ctx, f.OrgID, f.HospitalID)
	if err != nil || !crossTrustOn(mine) {
		return nil, nil, err
	}
	rows, err := s.q.ListPrefillOffers(ctx, db.ListPrefillOffersParams{AccountID: f.AccountID.UUID, OrgID: f.OrgID})
	if err != nil {
		return nil, nil, err
	}
	var ask []consentQuestion
	var yes []uuid.UUID
	for _, r := range rows {
		theirs, err := settings.Decode(r.OrgSettings)
		if err != nil { // that trust's problem mustn't stop this form opening: unreadable is off
			log.Printf("cross-trust pre-fill: trust %s's settings can't be read, so it isn't offered: %v", r.OrgID, err)
			continue
		}
		if !crossTrustOn(settings.Effective(settings.Platform(), theirs, nil)) {
			continue
		}
		switch {
		case r.Agreed == nil:
			ask = append(ask, consentQuestion{OrgID: r.OrgID, Hospital: r.HospitalName, SubmittedAt: r.SubmittedAt.Time})
		case *r.Agreed:
			yes = append(yes, r.OrgID)
		}
	}
	// The newest answers first, when several trusts are asked about (one at a time).
	slices.SortStableFunc(ask, func(a, b consentQuestion) int { return b.SubmittedAt.Compare(a.SubmittedAt) })
	return ask, yes, nil
}

// crossTrustOn reads a trust's crossTrustPrefill (a trust-only setting, so a hospital's effective
// value is its trust's). Unreadable is off.
func crossTrustOn(v settings.Values) bool {
	var on bool
	return settings.Get(v, "crossTrustPrefill", &on) == nil && on
}

// prefillSources reads the patient's earlier submitted HQ forms in one trust, newest first, each
// with its Question Sets and answers (ListPrefillSources). Their episodes go in the request's
// audit row: their answers were read. Another trust's (otherTrust) go in a row of that trust's own,
// at the source episode's hospital, never in this trust's (S26 answers 5).
func (s *server) prefillSources(ctx context.Context, org uuid.UUID, f form, a *auditEntry, otherTrust bool) ([]prefill.Source, error) {
	rows, err := s.q.ListPrefillSources(ctx, db.ListPrefillSourcesParams{OrgID: org, FormID: f.FormID, PatientID: f.PatientID, AccountID: f.AccountID})
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

// prefillConsent records the patient's answer to "Use the answers you gave …?" for one other
// trust: POST /api/p/episodes/{eid}/prefill-consent {orgId, agree}, from the patient's own session
// only (staff can't agree for them, S26 answers 4). Once per account and trust: it can't be
// changed (409), and only a trust the form is asking about can be answered (409 otherwise: the
// question went away, e.g. a setting was switched off).
func (s *server) prefillConsent(w http.ResponseWriter, r *http.Request) {
	f, ok := s.loadPatientForm(w, r)
	if !ok {
		return
	}
	var in struct {
		OrgID string `json:"orgId"`
		Agree *bool  `json:"agree"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	org, err := uuid.Parse(in.OrgID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "We couldn't tell which answers this is about. Please reload the page.")
		return
	}
	if in.Agree == nil {
		writeError(w, http.StatusBadRequest, "Choose “Yes, use them” or “No, thanks”.")
		return
	}
	if f.PatientSubmittedAt.Valid {
		writeError(w, http.StatusConflict, "These answers have already been sent.")
		return
	}
	ctx := r.Context()
	ask, _, err := s.crossTrust(ctx, f)
	if err != nil {
		serverError(w, "cross-trust pre-fill", err)
		return
	}
	if !slices.ContainsFunc(ask, func(q consentQuestion) bool { return q.OrgID == org }) {
		writeError(w, http.StatusConflict, "This question has already been answered, or no longer applies. Please reload the page.")
		return
	}
	n, err := s.q.DecidePrefillConsent(ctx, db.DecidePrefillConsentParams{AccountID: f.AccountID.UUID, OrgID: org, Agreed: *in.Agree,
		EpisodeID: uuid.NullUUID{UUID: f.ID, Valid: true}})
	if err != nil {
		serverError(w, "record consent", err)
		return
	}
	if n == 0 { // answered at the same moment in another tab
		writeError(w, http.StatusConflict, "This question has already been answered. Please reload the page.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

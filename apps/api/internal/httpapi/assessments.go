package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/assessment"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// Assessments (plan C7, session S47; §9 "Assessments through the HQ"): validated instruments such
// as STOP-Bang, which a clinician adds to an episode (S47 answers 1), completes and confirms.
//
//   - GET  …/episodes/{eid}/assessments: the episode's assessments, each with its content, the
//     clinician's answers, where each came from (sources) and the Confirm (who, when); the ones
//     still to add; and the trust's observations.precedence, which the browser's pre-fill uses;
//   - POST …/episodes/{eid}/assessments {assessment}: an episode_forms row of kind 'assessment' on
//     the assessment's published platform version (made, or made anew if its wording changed);
//   - PUT  …/episodes/{eid}/assessments/{fid} {data, sources, revision}: the clinician's answers,
//     saved with the revision check; it clears a Confirm, so a changed item needs a new one;
//   - POST …/episodes/{eid}/assessments/{fid}/confirm {data, sources, revision}: the same, confirmed.
//
// The browser pre-fills each item from the HQ, the observations and the patient's profile, and
// works out the score (packages/clinical, assessments.ts): Go only checks the answers' shape
// (internal/assessment) and stores them. Any clinician at the hospital may do it; a cancelled or
// archived episode takes nothing (409); with the hospital's feature.assessments off, every route is
// a 403. Patients never see an assessment: their routes find only the episode's HQ.

const (
	assessmentNotFound = "Assessment not found."
	assessmentsOffMsg  = "Assessments aren't turned on at this hospital."
)

// assessmentsOn says whether the hospital has assessments; the episode's GET sends it.
func assessmentsOn(v settings.Values) (bool, error) {
	var on bool
	err := settings.Get(v, "feature.assessments", &on)
	return on, err
}

// assessmentSettings answers 403 itself when the hospital has assessments off, else returns
// whether observations take precedence in scores (the trust's setting, OBS-04).
func (s *server) assessmentSettings(w http.ResponseWriter, r *http.Request) (bool, bool) {
	v, err := s.hospitalSettings(r.Context(), orgID(r), hospitalID(r))
	if err != nil {
		serverError(w, "settings", err)
		return false, false
	}
	on, err := assessmentsOn(v)
	if err != nil {
		serverError(w, "settings", err)
		return false, false
	}
	if !on {
		writeError(w, http.StatusForbidden, assessmentsOffMsg)
		return false, false
	}
	var precedence bool
	if err := settings.Get(v, "observations.precedence", &precedence); err != nil {
		serverError(w, "settings", err)
		return false, false
	}
	return precedence, true
}

func (s *server) listAssessments(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	precedence, ok := s.assessmentSettings(w, r)
	if !ok {
		return
	}
	rows, err := s.q.ListAssessmentForms(r.Context(), e.ID)
	if err != nil {
		serverError(w, "list assessments", err)
		return
	}
	available := []assessment.Def{}
	for _, d := range assessment.All {
		if !slices.ContainsFunc(rows, func(f db.ListAssessmentFormsRow) bool { return f.Assessment == d.ID }) {
			available = append(available, d)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"assessments": rows, "available": available, "obsPrecedence": precedence})
}

func (s *server) addAssessment(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	if _, ok := s.assessmentSettings(w, r); !ok {
		return
	}
	var in struct {
		Assessment string `json:"assessment"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	def, found := assessment.Lookup(in.Assessment)
	if !found {
		writeError(w, http.StatusBadRequest, "There's no such assessment.")
		return
	}
	var id uuid.UUID
	if !s.lockedEpisode(w, r, e.ID, "add assessment", func(q *db.Queries) (bool, error) {
		version, err := publishedAssessment(r, q, def)
		if err != nil {
			return false, err
		}
		id, err = q.AddAssessmentForm(r.Context(), db.AddAssessmentFormParams{EpisodeID: e.ID, VersionID: version, Assessment: &def.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, def.Name+" is already on this episode.")
			return false, nil
		}
		return err == nil, err
	}) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// publishedAssessment is the assessment's platform version with its content as Go holds it: any
// version with that content (published, or retired by another build serving at once), else a new
// one, published with the others retired, made the first time and whenever the wording changed.
// Forms on an older version keep it, so each shows what its clinician saw.
func publishedAssessment(r *http.Request, q *db.Queries, def assessment.Def) (uuid.UUID, error) {
	ctx := r.Context()
	if err := q.LockAssessmentDefinition(ctx, def.ID); err != nil {
		return uuid.Nil, err
	}
	same, err := q.MatchingAssessmentVersion(ctx, db.MatchingAssessmentVersionParams{Content: def.Content, Assessment: &def.ID})
	if err == nil {
		return same, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	qid, err := q.AssessmentQuestionnaire(ctx, db.AssessmentQuestionnaireParams{Assessment: &def.ID, Name: def.Name})
	if err != nil {
		return uuid.Nil, err
	}
	if err := q.RetireAssessmentVersions(ctx, qid); err != nil {
		return uuid.Nil, err
	}
	made, err := q.PublishAssessmentVersion(ctx, db.PublishAssessmentVersionParams{QuestionnaireID: qid, Name: def.Name, Content: def.Content})
	return made.VersionID, err
}

// assessmentInput is a save's or a Confirm's body.
type assessmentInput struct {
	Data     map[string]json.RawMessage `json:"data"`
	Sources  map[string]string          `json:"sources"`
	Revision *int32                     `json:"revision"`
}

func (s *server) saveAssessment(w http.ResponseWriter, r *http.Request) {
	s.storeAssessment(w, r, false)
}

func (s *server) confirmAssessment(w http.ResponseWriter, r *http.Request) {
	s.storeAssessment(w, r, true)
}

// storeAssessment saves the clinician's answers and their sources, confirmed or not.
func (s *server) storeAssessment(w http.ResponseWriter, r *http.Request, confirm bool) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	if _, ok := s.assessmentSettings(w, r); !ok {
		return
	}
	fid, ok := pathID(w, r, "fid", assessmentNotFound)
	if !ok {
		return
	}
	form, err := s.q.GetAssessmentForm(r.Context(), db.GetAssessmentFormParams{ID: fid, EpisodeID: e.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, assessmentNotFound)
		return
	} else if err != nil {
		serverError(w, "get assessment", err)
		return
	}
	var in assessmentInput
	if !readJSON(w, r, 64<<10, &in) {
		return
	}
	if in.Revision == nil || *in.Revision < 0 {
		writeError(w, http.StatusBadRequest, "The request could not be read.")
		return
	}
	if in.Data == nil {
		in.Data = map[string]json.RawMessage{}
	}
	if in.Sources == nil {
		in.Sources = map[string]string{}
	}
	if msg := assessment.Check(form.Content, in.Data, in.Sources); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	// A "Profile" age or sex is the record's: the age today at the hospital, as the browser works it
	// out (S48 verification 1, Rahul 7 Nov).
	if msg := assessment.ProfileProblem(in.Data, ageOn(e.DateOfBirth, s.today(e.HospitalTimeZone)), e.Sex); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if missing := assessment.Unanswered(form.Content, in.Data); confirm && missing != "" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Answer every item before confirming: %s has no answer.", missing))
		return
	}
	data, err := json.Marshal(in.Data)
	if err != nil {
		serverError(w, "encode answers", err)
		return
	}
	sources, err := json.Marshal(in.Sources)
	if err != nil {
		serverError(w, "encode sources", err)
		return
	}
	var rev int32
	if !s.lockedEpisode(w, r, e.ID, "save assessment", func(q *db.Queries) (bool, error) {
		rev, err = q.SaveAssessmentAnswers(r.Context(), db.SaveAssessmentAnswersParams{
			FormID: form.ID, ChapterID: form.ChapterID, Data: data, Sources: sources, Confirmed: confirm, Revision: *in.Revision,
			UpdatedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
		})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, staleAnswersMsg)
			return false, nil
		}
		return err == nil, err
	}) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]int32{"revision": rev})
}

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

const patientToken = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ" // 43 characters

var clinicianSetID = uuid.MustParse("55555555-5555-4555-8555-555555555555")

// patientFake holds one episode reachable by patientToken, with a patient Question Set
// (chapterID) and a clinician one (clinicianSetID).
type patientFake struct {
	*fakeQ
	submitted bool
	saved     []db.SavePatientAnswersParams
	events    []db.AddEpisodeEventParams
}

const patientSet = `{"pages":[{"name":"p_1","elements":[
  {"type":"radiogroup","name":"q_smoke","choices":[{"value":"o_y","clinicalOutputs":[{"id":"out_1"}]}]},
  {"type":"comment","name":"q_clin","clinicianOnly":true}
]}]}`

func (f *patientFake) GetEpisodeByToken(_ context.Context, token string) (db.GetEpisodeByTokenRow, error) {
	if token != patientToken {
		return db.GetEpisodeByTokenRow{}, pgx.ErrNoRows
	}
	return db.GetEpisodeByTokenRow{ID: episodeID, VersionID: versionID, PatientSubmittedAt: pgtype.Timestamptz{Valid: f.submitted}}, nil
}

func (f *patientFake) ListVersionChapters(context.Context, uuid.UUID) ([]db.ListVersionChaptersRow, error) {
	return []db.ListVersionChaptersRow{
		{ID: chapterID, Name: "About you", Audience: "patient", Content: json.RawMessage(patientSet)},
		{ID: clinicianSetID, Name: "Assessment", Audience: "clinician", Content: json.RawMessage(`{}`)},
	}, nil
}

func (f *patientFake) GetVersionChapter(_ context.Context, arg db.GetVersionChapterParams) (db.GetVersionChapterRow, error) {
	switch {
	case arg.VersionID != versionID:
		return db.GetVersionChapterRow{}, pgx.ErrNoRows
	case arg.ID == chapterID:
		return db.GetVersionChapterRow{ID: chapterID, Audience: "patient", Content: json.RawMessage(patientSet)}, nil
	case arg.ID == clinicianSetID:
		return db.GetVersionChapterRow{ID: clinicianSetID, Audience: "clinician", Content: json.RawMessage(`{}`)}, nil
	}
	return db.GetVersionChapterRow{}, pgx.ErrNoRows
}

func (f *patientFake) ListEpisodeAnswers(context.Context, uuid.UUID) ([]db.ListEpisodeAnswersRow, error) {
	return []db.ListEpisodeAnswersRow{
		{ChapterID: chapterID, Actor: "patient", Data: json.RawMessage(`{"q_smoke":"o_y"}`)},
		{ChapterID: chapterID, Actor: "clinician", Data: json.RawMessage(`{"q_clin":"seen"}`)},
	}, nil
}

func (f *patientFake) SavePatientAnswers(_ context.Context, arg db.SavePatientAnswersParams) (int64, error) {
	if f.submitted {
		return 0, nil
	}
	f.saved = append(f.saved, arg)
	return 1, nil
}

func (f *patientFake) SubmitPatientHQ(context.Context, uuid.UUID) (int64, error) {
	if f.submitted {
		return 0, nil
	}
	f.submitted = true
	return 1, nil
}

func (f *patientFake) AddEpisodeEvent(_ context.Context, arg db.AddEpisodeEventParams) error {
	f.events = append(f.events, arg)
	return nil
}

// asPatient sends a request with no cookie, as the patient's browser does.
func asPatient(t *testing.T, f *patientFake, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	routes(&server{q: f, secret: secret}).ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestPatientHQ(t *testing.T) {
	f := &patientFake{fakeQ: newFake()}
	if rec := asPatient(t, f, "GET", "/api/p/not-a-token", ""); rec.Code != http.StatusNotFound {
		t.Errorf("bad token: got %d, want 404", rec.Code)
	}
	if rec := asPatient(t, f, "GET", "/api/p/"+strings.Repeat("x", 43), ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown token: got %d, want 404", rec.Code)
	}
	rec := asPatient(t, f, "GET", "/api/p/"+patientToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d (%s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, gone := range []string{"q_clin", "clinicalOutputs", "Assessment", "seen"} {
		if strings.Contains(body, gone) {
			t.Errorf("the patient gets %q: %s", gone, body)
		}
	}
	if !strings.Contains(body, `"q_smoke":"o_y"`) {
		t.Errorf("the patient's own answers are missing: %s", body)
	}
}

func TestSavePatientAnswers(t *testing.T) {
	path := "/api/p/" + patientToken + "/answers/"
	f := &patientFake{fakeQ: newFake()}
	rec := asPatient(t, f, "PUT", path+chapterID.String(), `{"data":{"q_smoke":"o_y","q_clin":"injected"}}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := string(f.saved[0].Data); got != `{"q_smoke":"o_y"}` {
		t.Errorf("saved %s: the clinician-only answer should be dropped", got)
	}
	if rec := asPatient(t, f, "PUT", path+clinicianSetID.String(), `{"data":{}}`); rec.Code != http.StatusNotFound {
		t.Errorf("clinician Question Set: got %d, want 404", rec.Code)
	}

	if rec := asPatient(t, f, "POST", "/api/p/"+patientToken+"/submit", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("submit: got %d", rec.Code)
	}
	if len(f.events) != 1 || f.events[0].UserID.Valid {
		t.Errorf("events = %+v: want one, by the patient (no user)", f.events)
	}
	if rec := asPatient(t, f, "PUT", path+chapterID.String(), `{"data":{"q_smoke":"o_n"}}`); rec.Code != http.StatusConflict {
		t.Errorf("save after submit: got %d, want 409", rec.Code)
	}
	if rec := asPatient(t, f, "POST", "/api/p/"+patientToken+"/submit", ""); rec.Code != http.StatusConflict {
		t.Errorf("second submit: got %d, want 409", rec.Code)
	}
}

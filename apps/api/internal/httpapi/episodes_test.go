package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

var episodeID = uuid.MustParse("44444444-4444-4444-8444-444444444444")

// episodeFake holds one episode in hospital A and records writes.
type episodeFake struct {
	*fakeQ
	episode  db.GetEpisodeRow
	patients []db.CreatePatientParams
	updates  []db.UpdateEpisodeParams
	events   []db.AddEpisodeEventParams
}

func newEpisodeFake(status string, roles ...string) *episodeFake {
	f := newFake()
	f.roles[hospitalA] = roles
	return &episodeFake{fakeQ: f, episode: db.GetEpisodeRow{ID: episodeID, Status: status}}
}

func (f *episodeFake) CreatePatient(_ context.Context, arg db.CreatePatientParams) (uuid.UUID, error) {
	f.patients = append(f.patients, arg)
	return uuid.New(), nil
}

func (f *episodeFake) GetEpisode(_ context.Context, arg db.GetEpisodeParams) (db.GetEpisodeRow, error) {
	if arg.ID != episodeID || arg.HospitalID != hospitalA {
		return db.GetEpisodeRow{}, pgx.ErrNoRows
	}
	return f.episode, nil
}

func (f *episodeFake) UpdateEpisode(_ context.Context, arg db.UpdateEpisodeParams) error {
	f.updates = append(f.updates, arg)
	return nil
}

func (f *episodeFake) AddEpisodeEvent(_ context.Context, arg db.AddEpisodeEventParams) error {
	f.events = append(f.events, arg)
	return nil
}

func hospitalPath(suffix string) string { return hp(hospitalA) + suffix }

func TestCreatePatient(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Format(time.DateOnly)
	tests := []struct {
		name, body string
		want       int
	}{
		{"valid", `{"firstName":" Jo ","lastName":"Bloggs","dateOfBirth":"1974-12-11","sex":"male"}`, http.StatusCreated},
		{"no last name", `{"firstName":"Jo","lastName":" ","dateOfBirth":"1974-12-11","sex":"male"}`, http.StatusBadRequest},
		{"date not a date", `{"firstName":"Jo","lastName":"Bloggs","dateOfBirth":"11/12/1974","sex":"male"}`, http.StatusBadRequest},
		{"born tomorrow", `{"firstName":"Jo","lastName":"Bloggs","dateOfBirth":"` + tomorrow + `","sex":"male"}`, http.StatusBadRequest},
		{"unknown sex value", `{"firstName":"Jo","lastName":"Bloggs","dateOfBirth":"1974-12-11","sex":"x"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newEpisodeFake("", "clinician")
			if rec := call(t, f, "POST", hospitalPath("/patients"), tt.body); rec.Code != tt.want {
				t.Fatalf("got %d (%s), want %d", rec.Code, rec.Body.String(), tt.want)
			}
			if tt.want == http.StatusCreated && f.patients[0].FirstName != "Jo" {
				t.Errorf("first name %q was not trimmed", f.patients[0].FirstName)
			}
		})
	}
}

func TestUpdateEpisode(t *testing.T) {
	path := hospitalPath("/episodes/" + episodeID.String())
	tests := []struct {
		name, status, body string
		want               int
		events             []string
	}{
		{"ready for admission after the review", episode.ReadyForPOA, `{"status":"ready_for_admission"}`, http.StatusNoContent, []string{"Status changed to Ready for admission"}},
		{"not before the review", episode.ReadyForReview, `{"status":"ready_for_admission"}`, http.StatusConflict, nil},
		{"ASA grades", episode.ReadyForReview, `{"nurseAsa":2,"anaesthetistAsa":3}`, http.StatusNoContent, []string{"Nurse ASA grade set to 2", "Anaesthetist ASA grade set to 3"}},
		{"ASA out of range", episode.ReadyForReview, `{"nurseAsa":7}`, http.StatusBadRequest, nil},
		{"procedure only", episode.HQNotComplete, `{"procedure":"Hip"}`, http.StatusNoContent, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newEpisodeFake(tt.status, "clinician")
			if rec := call(t, f, "PATCH", path, tt.body); rec.Code != tt.want {
				t.Fatalf("got %d (%s), want %d", rec.Code, rec.Body.String(), tt.want)
			}
			var got []string
			for _, e := range f.events {
				got = append(got, e.Text)
			}
			if len(got) != len(tt.events) {
				t.Fatalf("events = %q, want %q", got, tt.events)
			}
			for i := range got {
				if got[i] != tt.events[i] {
					t.Errorf("event %d = %q, want %q", i, got[i], tt.events[i])
				}
			}
		})
	}
}

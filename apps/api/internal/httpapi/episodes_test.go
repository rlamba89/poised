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
	created  []db.CreateEpisodeParams
	events   []db.AddEpisodeEventParams
	// CreateEpisode finds no row unless the version is this one (published, in the hospital).
	published uuid.UUID
}

func newEpisodeFake(status string, roles ...string) *episodeFake {
	f := newFake()
	f.roles[hospitalA] = roles
	return &episodeFake{fakeQ: f, episode: db.GetEpisodeRow{ID: episodeID, Status: status}, published: versionID}
}

func (f *episodeFake) CreatePatient(_ context.Context, arg db.CreatePatientParams) (uuid.UUID, error) {
	f.patients = append(f.patients, arg)
	return uuid.New(), nil
}

func (f *episodeFake) CreateEpisode(_ context.Context, arg db.CreateEpisodeParams) (uuid.UUID, error) {
	if arg.VersionID != f.published {
		return uuid.Nil, pgx.ErrNoRows
	}
	f.created = append(f.created, arg)
	return episodeID, nil
}

func (f *episodeFake) GetEpisode(_ context.Context, arg db.GetEpisodeParams) (db.GetEpisodeRow, error) {
	if arg.ID != episodeID || arg.HospitalID != hospitalA {
		return db.GetEpisodeRow{}, pgx.ErrNoRows
	}
	return f.episode, nil
}

func (f *episodeFake) AddEpisodeEvent(_ context.Context, arg db.AddEpisodeEventParams) error {
	f.events = append(f.events, arg)
	return nil
}

func hospitalPath(suffix string) string { return "/api/h/" + hospitalA.String() + suffix }

func TestEpisodesNeedClinician(t *testing.T) {
	f := newEpisodeFake(episode.HQNotComplete, "author", "publisher")
	if rec := call(t, f, "GET", hospitalPath("/episodes/"+episodeID.String()), ""); rec.Code != http.StatusForbidden {
		t.Errorf("author: got %d, want 403", rec.Code)
	}
}

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

func TestCreateEpisode(t *testing.T) {
	f := newEpisodeFake("", "clinician")
	body := `{"patientId":"` + uuid.NewString() + `","versionId":"` + versionID.String() + `","procedure":"Knee replacement"}`
	if rec := call(t, f, "POST", hospitalPath("/episodes"), body); rec.Code != http.StatusCreated {
		t.Fatalf("got %d (%s)", rec.Code, rec.Body.String())
	}
	if len(f.created[0].PatientToken) != 43 {
		t.Errorf("token %q: want 43 characters", f.created[0].PatientToken)
	}
	if len(f.events) != 1 || f.events[0].Text != "Episode created" {
		t.Errorf("events = %+v", f.events)
	}

	body = `{"patientId":"` + uuid.NewString() + `","versionId":"` + uuid.NewString() + `"}`
	if rec := call(t, f, "POST", hospitalPath("/episodes"), body); rec.Code != http.StatusBadRequest {
		t.Errorf("unpublished version: got %d, want 400", rec.Code)
	}
}

// A PATCH sends only what it changes. A text field sent as null is refused, not a crash (found
// by S01's code review: the handler dereferenced the nil pointer and dropped the connection).
func TestEpisodeChangeValidate(t *testing.T) {
	s := func(v string) *string { return &v }
	valid := func() episodeChange {
		return episodeChange{Status: s("ready_for_poa"), Procedure: s(" Hip replacement "), Anaesthetic: s(" GA "), Consultant: s(" Dr Who ")}
	}

	in := valid()
	if msg := in.validate(); msg != "" || *in.Procedure != "Hip replacement" || *in.Anaesthetic != "GA" || *in.Consultant != "Dr Who" {
		t.Errorf("valid change: %q, %q %q %q; want no problem, trimmed", msg, *in.Procedure, *in.Anaesthetic, *in.Consultant)
	}
	for name, change := range map[string]func(*episodeChange){
		"null status":      func(c *episodeChange) { c.Status = nil },
		"null procedure":   func(c *episodeChange) { c.Procedure = nil },
		"null anaesthetic": func(c *episodeChange) { c.Anaesthetic = nil },
		"null consultant":  func(c *episodeChange) { c.Consultant = nil },
	} {
		in := valid()
		change(&in)
		if in.validate() == "" {
			t.Errorf("%s: no problem reported", name)
		}
	}
}

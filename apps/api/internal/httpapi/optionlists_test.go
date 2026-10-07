package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

// listFake records option-list writes; names already taken fail like the unique index.
type listFake struct {
	*fakeQ
	created []db.CreateOptionListParams
	taken   map[string]bool
}

func (f *listFake) CreateOptionList(_ context.Context, arg db.CreateOptionListParams) (uuid.UUID, error) {
	if f.taken[arg.Name] {
		return uuid.Nil, &pgconn.PgError{Code: "23505"}
	}
	f.created = append(f.created, arg)
	return uuid.New(), nil
}

func (f *listFake) DeleteOptionList(_ context.Context, arg db.DeleteOptionListParams) (int64, error) {
	if arg.HospitalID != hospitalA {
		return 0, nil
	}
	return 1, nil
}

func doList(t *testing.T, f *listFake, method, path, body string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	signedIn(req)
	rec := httptest.NewRecorder()
	routes(&server{q: f}).ServeHTTP(rec, req)
	return rec.Code
}

// OPT-07: authors save a named list of options for their hospital.
func TestCreateOptionList(t *testing.T) {
	path := hp(hospitalA) + "/option-lists"
	tests := []struct {
		name  string
		roles []string
		body  string
		want  int
	}{
		{"saved", []string{"super_clinician"}, `{"name":" Anaesthetic types ","options":[{"text":"General"},{"text":"Local","score":1}]}`, http.StatusCreated},
		{"clinician can't", []string{"clinician"}, `{"name":"X","options":[{"text":"A"}]}`, http.StatusForbidden},
		{"needs a name", []string{"super_clinician"}, `{"name":"  ","options":[{"text":"A"}]}`, http.StatusBadRequest},
		{"needs options", []string{"super_clinician"}, `{"name":"X","options":[]}`, http.StatusBadRequest},
		{"options need labels", []string{"super_clinician"}, `{"name":"X","options":[{"text":""}]}`, http.StatusBadRequest},
		{"options must be a list", []string{"super_clinician"}, `{"name":"X","options":{"text":"A"}}`, http.StatusBadRequest},
		{"name in use", []string{"super_clinician"}, `{"name":"Taken","options":[{"text":"A"}]}`, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &listFake{fakeQ: newFake(), taken: map[string]bool{"Taken": true}}
			f.roles[hospitalA] = tt.roles
			if got := doList(t, f, "POST", path, tt.body); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
			if tt.want == http.StatusCreated && (len(f.created) != 1 || f.created[0].Name != "Anaesthetic types" || f.created[0].HospitalID != hospitalA) {
				t.Errorf("created %+v", f.created)
			}
			if tt.want != http.StatusCreated && len(f.created) != 0 {
				t.Error("a list was saved")
			}
		})
	}
}

func TestDeleteOptionListScopedToHospital(t *testing.T) {
	f := &listFake{fakeQ: newFake()}
	f.roles[hospitalB] = []string{"super_clinician"}
	if got := doList(t, f, "DELETE", hp(hospitalA)+"/option-lists/"+uuid.NewString(), ""); got != http.StatusNoContent {
		t.Errorf("own hospital: got %d, want 204", got)
	}
	if got := doList(t, f, "DELETE", hp(hospitalB)+"/option-lists/"+uuid.NewString(), ""); got != http.StatusNotFound {
		t.Errorf("other hospital's list: got %d, want 404", got)
	}
}

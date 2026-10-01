package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

var (
	questionnaireID = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	versionID       = uuid.MustParse("33333333-3333-4333-8333-333333333333")
)

// versionFake holds one questionnaire in hospital A, showing its latest version.
type versionFake struct {
	*fakeQ
	latest    db.GetQuestionnaireRow
	chapters  int
	published []db.PublishVersionParams
	copied    []db.CreateNextVersionParams
	nameTaken bool
}

func newVersionFake(status string, roles ...string) *versionFake {
	f := newFake()
	f.roles[hospitalA] = roles
	return &versionFake{
		fakeQ:    f,
		latest:   db.GetQuestionnaireRow{ID: questionnaireID, HospitalID: hospitalA, Name: "HJE Full HQ", VersionID: versionID, VersionNo: 1, Status: status},
		chapters: 2,
	}
}

func (f *versionFake) GetQuestionnaire(_ context.Context, arg db.GetQuestionnaireParams) (db.GetQuestionnaireRow, error) {
	if arg.ID != questionnaireID || arg.HospitalID != hospitalA {
		return db.GetQuestionnaireRow{}, pgx.ErrNoRows
	}
	return f.latest, nil
}

func (f *versionFake) ListChapters(context.Context, uuid.UUID) ([]db.ListChaptersRow, error) {
	return make([]db.ListChaptersRow, f.chapters), nil
}

func (f *versionFake) PublishVersion(_ context.Context, arg db.PublishVersionParams) (int64, error) {
	f.published = append(f.published, arg)
	return 1, nil
}

func (f *versionFake) DraftNameExists(context.Context, db.DraftNameExistsParams) (bool, error) {
	return f.nameTaken, nil
}

func (f *versionFake) CreateNextVersion(_ context.Context, arg db.CreateNextVersionParams) (uuid.UUID, error) {
	f.copied = append(f.copied, arg)
	return uuid.New(), nil
}

// call sends an authenticated request as userA to the routes backed by q.
func call(t *testing.T, q db.Querier, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	token, err := auth.Issue(secret, userA, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	rec := httptest.NewRecorder()
	routes(&server{q: q, secret: secret}).ServeHTTP(rec, req)
	return rec
}

func questionnairePath(suffix string) string {
	return "/api/h/" + hospitalA.String() + "/questionnaires/" + questionnaireID.String() + suffix
}

// LCY-01: the publisher publishes the draft they checked.
func TestPublish(t *testing.T) {
	body := `{"versionId":"` + versionID.String() + `"}`
	tests := []struct {
		name     string
		status   string
		roles    []string
		chapters int
		body     string
		want     int
	}{
		{"publisher publishes the draft", "draft", []string{"publisher"}, 2, body, http.StatusNoContent},
		{"author alone can't publish", "draft", []string{"author"}, 2, body, http.StatusForbidden},
		{"already published", "published", []string{"publisher"}, 2, body, http.StatusConflict},
		{"a different version was checked", "draft", []string{"publisher"}, 2, `{"versionId":"` + uuid.NewString() + `"}`, http.StatusConflict},
		{"no Question Sets", "draft", []string{"publisher"}, 0, body, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newVersionFake(tt.status, tt.roles...)
			f.chapters = tt.chapters
			rec := call(t, f, "POST", questionnairePath("/publish"), tt.body)
			if rec.Code != tt.want {
				t.Fatalf("got %d (%s), want %d", rec.Code, rec.Body.String(), tt.want)
			}
			if published := len(f.published) == 1; published != (tt.want == http.StatusNoContent) {
				t.Errorf("published = %v", published)
			}
			if tt.want == http.StatusNoContent && f.published[0].VersionID != versionID {
				t.Errorf("published version %s, want %s", f.published[0].VersionID, versionID)
			}
		})
	}
}

// LCY-06/07: a new draft is copied from the published version.
func TestCreateVersion(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		roles     []string
		nameTaken bool
		want      int
	}{
		{"from published", "published", []string{"author"}, false, http.StatusCreated},
		{"a draft already exists", "draft", []string{"author"}, false, http.StatusConflict},
		{"viewer can't", "published", []string{"viewer"}, false, http.StatusForbidden},
		{"another draft has the name", "published", []string{"author"}, true, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newVersionFake(tt.status, tt.roles...)
			f.nameTaken = tt.nameTaken
			rec := call(t, f, "POST", questionnairePath("/versions"), "")
			if rec.Code != tt.want {
				t.Fatalf("got %d (%s), want %d", rec.Code, rec.Body.String(), tt.want)
			}
			if tt.want == http.StatusCreated && (len(f.copied) != 1 || f.copied[0].FromVersionID != versionID) {
				t.Errorf("copied = %+v, want one copy of %s", f.copied, versionID)
			}
		})
	}
}

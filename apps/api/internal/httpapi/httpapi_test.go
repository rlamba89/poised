package httpapi

import (
	"context"
	"encoding/json"
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
	secret    = []byte("test-secret")
	userA     = auth.User{ID: uuid.MustParse("00000000-0000-4000-8000-0000000000a1"), Name: "Alex Author"}
	hospitalA = uuid.MustParse("00000000-0000-4000-8000-00000000000a")
	hospitalB = uuid.MustParse("00000000-0000-4000-8000-00000000000b")
	chapterID = uuid.MustParse("11111111-1111-4111-8111-111111111111")
)

// fakeQ implements the queries these tests reach; any other call panics
// through the nil embedded interface.
type fakeQ struct {
	db.Querier
	roles   map[uuid.UUID][]string // hospital → caller's roles
	chapter *db.GetChapterMetaRow  // in hospital A
	// SaveChapterContent succeeds only when the submitted revision is current.
	revision int32
	saved    []db.SaveChapterContentParams
}

func (f *fakeQ) ListRolesInHospital(_ context.Context, arg db.ListRolesInHospitalParams) ([]string, error) {
	return f.roles[arg.HospitalID], nil
}

func (f *fakeQ) GetChapterMeta(_ context.Context, arg db.GetChapterMetaParams) (db.GetChapterMetaRow, error) {
	if f.chapter == nil || arg.ID != f.chapter.ID || arg.HospitalID != hospitalA {
		return db.GetChapterMetaRow{}, pgx.ErrNoRows
	}
	return *f.chapter, nil
}

func (f *fakeQ) SaveChapterContent(_ context.Context, arg db.SaveChapterContentParams) (int32, error) {
	if arg.Revision != f.revision {
		return 0, pgx.ErrNoRows // mirrors "WHERE revision = $n" matching no row
	}
	f.revision++
	f.saved = append(f.saved, arg)
	return f.revision, nil
}

func (f *fakeQ) TouchVersion(context.Context, db.TouchVersionParams) error { return nil }

func newFake() *fakeQ {
	return &fakeQ{
		roles:    map[uuid.UUID][]string{hospitalA: {"author"}},
		chapter:  &db.GetChapterMetaRow{ID: chapterID, VersionStatus: "draft"},
		revision: 3,
	}
}

func do(t *testing.T, q *fakeQ, method, path, body string, withCookie bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if withCookie {
		token, err := auth.Issue(secret, userA, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	routes(&server{q: q, secret: secret}).ServeHTTP(rec, req)
	return rec
}

func errorOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct{ Error string }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not a JSON error: %v", rec.Body.String(), err)
	}
	return body.Error
}

func contentPath(h uuid.UUID) string {
	return "/api/h/" + h.String() + "/chapters/" + chapterID.String() + "/content"
}

func TestRequireUser(t *testing.T) {
	q := newFake()
	if rec := do(t, q, "PUT", contentPath(hospitalA), `{}`, false); rec.Code != http.StatusUnauthorized {
		t.Errorf("no cookie: got %d, want 401", rec.Code)
	}

	req := httptest.NewRequest("PUT", contentPath(hospitalA), strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "forged"})
	rec := httptest.NewRecorder()
	routes(&server{q: q, secret: secret}).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad token: got %d, want 401", rec.Code)
	}
}

func TestHospitalScope(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"member", contentPath(hospitalA), http.StatusOK},
		{"not a member", contentPath(hospitalB), http.StatusNotFound},
		{"unknown hospital", contentPath(uuid.New()), http.StatusNotFound},
		{"not a uuid", "/api/h/abc/chapters/" + chapterID.String() + "/content", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newFake(), "PUT", tt.path, `{"content":{},"revision":3}`, true)
			if rec.Code != tt.want {
				t.Errorf("got %d %s, want %d", rec.Code, rec.Body, tt.want)
			}
		})
	}
}

func TestChapterFromAnotherHospitalIsNotFound(t *testing.T) {
	q := newFake()
	q.roles[hospitalB] = []string{"author"} // member of both, but the chapter is in A
	rec := do(t, q, "PUT", contentPath(hospitalB), `{"content":{},"revision":3}`, true)
	if rec.Code != http.StatusNotFound || len(q.saved) != 0 {
		t.Errorf("got %d with %d saves, want 404 and no save", rec.Code, len(q.saved))
	}
}

// LCY-04: a save based on a stale revision is refused, never applied.
func TestSaveContentRevision(t *testing.T) {
	q := newFake()

	rec := do(t, q, "PUT", contentPath(hospitalA), `{"content":{"pages":[]},"revision":3}`, true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"revision":4`) {
		t.Fatalf("current revision: got %d %s, want 200 with revision 4", rec.Code, rec.Body)
	}

	// A second editor still holding revision 3.
	rec = do(t, q, "PUT", contentPath(hospitalA), `{"content":{"pages":[]},"revision":3}`, true)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale revision: got %d, want 409", rec.Code)
	}
	if msg := errorOf(t, rec); !strings.Contains(msg, "Someone else changed this chapter") {
		t.Errorf("stale revision message = %q", msg)
	}
	if len(q.saved) != 1 {
		t.Errorf("saves = %d, want 1", len(q.saved))
	}
}

func TestSaveContentRules(t *testing.T) {
	tests := []struct {
		name  string
		setup func(q *fakeQ)
		body  string
		want  int
	}{
		{"viewer can't edit", func(q *fakeQ) { q.roles[hospitalA] = []string{"viewer"} }, `{"content":{},"revision":3}`, http.StatusForbidden},
		{"only drafts", func(q *fakeQ) { q.chapter.VersionStatus = "published" }, `{"content":{},"revision":3}`, http.StatusConflict},
		{"content not a survey", func(*fakeQ) {}, `{"content":[1,2],"revision":3}`, http.StatusBadRequest},
		{"unreadable body", func(*fakeQ) {}, `{"content":`, http.StatusBadRequest},
		{"unknown chapter", func(q *fakeQ) { q.chapter.ID = uuid.New() }, `{"content":{},"revision":3}`, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := newFake()
			tt.setup(q)
			rec := do(t, q, "PUT", contentPath(hospitalA), tt.body, true)
			if rec.Code != tt.want {
				t.Errorf("got %d %s, want %d", rec.Code, rec.Body, tt.want)
			}
			if errorOf(t, rec) == "" {
				t.Error("error message is empty")
			}
			if len(q.saved) != 0 {
				t.Error("content was saved")
			}
		})
	}
}

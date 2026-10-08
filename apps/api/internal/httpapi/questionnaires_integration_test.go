//go:build integration

package httpapi_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

type questionnaireList struct {
	Items []struct {
		ID     uuid.UUID
		Name   string
		Status string
	}
	Total int
	Page  int
}

func (l questionnaireList) ids() map[uuid.UUID]bool {
	ids := map[uuid.UUID]bool{}
	for _, it := range l.Items {
		ids[it.ID] = true
	}
	return ids
}

func TestListQuestionnaires(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	draft := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	published := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
	inB := w.fx.DraftHQ(t, w.b, w.bea, "basic.json")
	if _, err := w.db.Exec(t.Context(), "UPDATE questionnaires SET description = 'Hip replacement' WHERE id = $1", published.ID); err != nil {
		t.Fatal(err)
	}
	list := func(t *testing.T, c *apitest.Client, path string) questionnaireList {
		t.Helper()
		var l questionnaireList
		r := c.Do("GET", path, nil)
		r.Want(t, http.StatusOK)
		r.Decode(t, &l)
		return l
	}

	t.Run("happy path: the hospital's questionnaires, searchable by name and description", func(t *testing.T) {
		l := list(t, w.author, at(w.a, "/questionnaires"))
		if l.Total != 2 || !l.ids()[draft.ID] || !l.ids()[published.ID] {
			t.Errorf("list = %+v, want the draft and the published one", l)
		}
		if l := list(t, w.author, at(w.a, "/questionnaires?q="+url.QueryEscape(draft.Name))); l.Total != 1 || l.Items[0].ID != draft.ID {
			t.Errorf("search by name = %+v, want only %s", l, draft.Name)
		}
		if l := list(t, w.author, at(w.a, "/questionnaires?q=HIP")); l.Total != 1 || l.Items[0].Status != "published" {
			t.Errorf("search by description = %+v, want only the published one", l)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", at(w.a, "/questionnaires"), nil), http.StatusUnauthorized)
	})
	t.Run("role: a viewer can read", func(t *testing.T) {
		if l := list(t, w.viewer, at(w.a, "/questionnaires")); l.Total != 2 {
			t.Errorf("viewer sees %d, want 2", l.Total)
		}
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, w.other.Do("GET", at(w.a, "/questionnaires"), nil), http.StatusNotFound)
		if l := list(t, w.other, at(w.b, "/questionnaires")); l.Total != 1 || l.Items[0].ID != inB.ID {
			t.Errorf("hospital B's list = %+v, want only its own", l)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("GET", "/api/h/not-a-uuid/questionnaires", nil), http.StatusNotFound)
		for _, page := range []string{"abc", "-2", "0"} {
			if l := list(t, w.author, at(w.a, "/questionnaires?page="+page)); l.Page != 1 || l.Total != 2 {
				t.Errorf("page=%s gave %+v, want page 1", page, l)
			}
		}
		for _, page := range []string{"2", "99999999999"} {
			if l := list(t, w.author, at(w.a, "/questionnaires?page="+page)); len(l.Items) != 0 {
				t.Errorf("page=%s gave %d items, want none", page, len(l.Items))
			}
		}
	})
}

func TestCreateQuestionnaire(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	body := map[string]string{"name": "  Pre-op assessment ", "description": "Before surgery."}
	total := func(t *testing.T) int { return count(t, w.db, "SELECT count(*) FROM questionnaires") }

	t.Run("happy path: the questionnaire and its draft version 1", func(t *testing.T) {
		r := w.author.Do("POST", at(w.a, "/questionnaires"), body)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		var hospital, createdBy uuid.UUID
		var name, status string
		var versionNo int
		err := w.db.QueryRow(t.Context(), `SELECT q.hospital_id, q.name, q.created_by, v.version_no, v.status
			FROM questionnaires q JOIN questionnaire_versions v ON v.questionnaire_id = q.id WHERE q.id = $1`,
			got.ID).Scan(&hospital, &name, &createdBy, &versionNo, &status)
		if err != nil {
			t.Fatal(err)
		}
		if hospital != w.a || name != "Pre-op assessment" || createdBy != w.alex.ID || versionNo != 1 || status != "draft" {
			t.Errorf("row = %v %q %v v%d %s, want hospital A, trimmed name, Alex, v1 draft", hospital, name, createdBy, versionNo, status)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("POST", at(w.a, "/questionnaires"), body), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, w.viewer.Do("POST", at(w.a, "/questionnaires"), body), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, w.other.Do("POST", at(w.a, "/questionnaires"), body), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{
			map[string]string{"name": "  ", "description": "x"},
			map[string]string{"name": "No description"},
			`{"name": `,
		} {
			refused(t, w.author.Do("POST", at(w.a, "/questionnaires"), bad), http.StatusBadRequest)
		}
		// Two drafts in one hospital can't share a name, whatever the case.
		refused(t, w.author.Do("POST", at(w.a, "/questionnaires"),
			map[string]string{"name": "PRE-OP ASSESSMENT", "description": "x"}), http.StatusConflict)
	})
	if n := total(t); n != 1 {
		t.Errorf("questionnaires = %d, want only the happy path's", n)
	}
	t.Run("another hospital can use the same name", func(t *testing.T) {
		w.other.Do("POST", at(w.b, "/questionnaires"), body).Want(t, http.StatusCreated)
	})
}

func TestGetQuestionnaire(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	path := at(w.a, "/questionnaires/"+hq.ID.String())

	t.Run("happy path: the latest version and its Question Sets in order", func(t *testing.T) {
		r := w.author.Do("GET", path, nil)
		r.Want(t, http.StatusOK)
		var got struct {
			Questionnaire struct {
				ID        uuid.UUID
				VersionID uuid.UUID
				Status    string
			}
			Chapters []struct{ Name string }
		}
		r.Decode(t, &got)
		if got.Questionnaire.ID != hq.ID || got.Questionnaire.VersionID != hq.VersionID || got.Questionnaire.Status != "draft" {
			t.Errorf("questionnaire = %+v", got.Questionnaire)
		}
		if len(got.Chapters) != 2 || got.Chapters[0].Name != "About you" || got.Chapters[1].Name != "Nurse checks" {
			t.Errorf("chapters = %+v, want About you, Nurse checks", got.Chapters)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", path, nil), http.StatusUnauthorized)
	})
	t.Run("role: a viewer can read", func(t *testing.T) {
		w.viewer.Do("GET", path, nil).Want(t, http.StatusOK)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, w.other.Do("GET", path, nil), http.StatusNotFound)
		refused(t, w.other.Do("GET", at(w.b, "/questionnaires/"+hq.ID.String()), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("GET", at(w.a, "/questionnaires/abc"), nil), http.StatusNotFound)
		refused(t, w.author.Do("GET", at(w.a, "/questionnaires/"+uuid.NewString()), nil), http.StatusNotFound)
	})
}

func TestDeleteQuestionnaire(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	exists := func(t *testing.T, hq apitest.HQ) bool {
		return count(t, w.db, "SELECT count(*) FROM questionnaires WHERE id = $1", hq.ID) == 1
	}
	path := func(h uuid.UUID, hq apitest.HQ) string { return at(h, "/questionnaires/"+hq.ID.String()) }

	t.Run("happy path: the creator deletes a draft, with its versions and Question Sets", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		w.author.Do("DELETE", path(w.a, hq), nil).Want(t, http.StatusNoContent)
		if exists(t, hq) ||
			count(t, w.db, "SELECT count(*) FROM questionnaire_versions WHERE questionnaire_id = $1", hq.ID) != 0 ||
			count(t, w.db, "SELECT count(*) FROM chapters WHERE version_id = $1", hq.VersionID) != 0 {
			t.Error("rows are left behind")
		}
	})
	t.Run("happy path: a hospital admin deletes someone else's draft", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		admin := w.anon.LoginAs(t, w.fx.User(t, w.a, "hospital_admin"))
		admin.Do("DELETE", path(w.a, hq), nil).Want(t, http.StatusNoContent)
		if exists(t, hq) {
			t.Error("still there")
		}
	})

	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("DELETE", path(w.a, hq), nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: not the creator nor a hospital admin", func(t *testing.T) {
		refused(t, w.viewer.Do("DELETE", path(w.a, hq), nil), http.StatusForbidden)
		otherAuthor := w.anon.LoginAs(t, w.fx.User(t, w.a, "author", "publisher"))
		refused(t, otherAuthor.Do("DELETE", path(w.a, hq), nil), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, w.other.Do("DELETE", path(w.a, hq), nil), http.StatusNotFound)
		refused(t, w.other.Do("DELETE", path(w.b, hq), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("DELETE", at(w.a, "/questionnaires/abc"), nil), http.StatusNotFound)
		published := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		refused(t, w.author.Do("DELETE", path(w.a, published), nil), http.StatusConflict)
		if !exists(t, published) {
			t.Error("a published questionnaire was deleted")
		}
	})
	if !exists(t, hq) {
		t.Error("a refused delete deleted it")
	}
}

// versionStatus is the status of one questionnaire version.
func versionStatus(t *testing.T, db *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var s string
	if err := db.QueryRow(t.Context(), "SELECT status FROM questionnaire_versions WHERE id = $1", id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPublishQuestionnaire(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	publish := func(c *apitest.Client, h uuid.UUID, hq apitest.HQ, versionID uuid.UUID) *apitest.Resp {
		return c.Do("POST", at(h, "/questionnaires/"+hq.ID.String()+"/publish"), map[string]any{"versionId": versionID})
	}

	t.Run("happy path: the draft is published", func(t *testing.T) {
		hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		publish(w.author, w.a, hq, hq.VersionID).Want(t, http.StatusNoContent)
		if s := versionStatus(t, w.db, hq.VersionID); s != "published" {
			t.Errorf("status = %s, want published", s)
		}
	})
	t.Run("happy path: publishing retires the previous version", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		r := w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/versions"), nil)
		r.Want(t, http.StatusCreated)
		var v2 struct{ ID uuid.UUID }
		r.Decode(t, &v2)
		publish(w.author, w.a, hq, v2.ID).Want(t, http.StatusNoContent)
		if v1, v2 := versionStatus(t, w.db, hq.VersionID), versionStatus(t, w.db, v2.ID); v1 != "retired" || v2 != "published" {
			t.Errorf("v1 %s, v2 %s; want retired, published", v1, v2)
		}
	})

	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, publish(w.anon, w.a, hq, hq.VersionID), http.StatusUnauthorized)
	})
	t.Run("wrong role: an author who isn't a publisher, and a viewer", func(t *testing.T) {
		authorOnly := w.anon.LoginAs(t, w.fx.User(t, w.a, "author"))
		refused(t, publish(authorOnly, w.a, hq, hq.VersionID), http.StatusForbidden)
		refused(t, publish(w.viewer, w.a, hq, hq.VersionID), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		bPublisher := w.anon.LoginAs(t, w.fx.User(t, w.b, "author", "publisher"))
		refused(t, publish(bPublisher, w.a, hq, hq.VersionID), http.StatusNotFound)
		refused(t, publish(bPublisher, w.b, hq, hq.VersionID), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, publish(w.author, w.a, hq, uuid.New()), http.StatusConflict) // not the version the author checked
		refused(t, w.author.Do("POST", at(w.a, "/questionnaires/"+hq.ID.String()+"/publish"), `{"versionId":`), http.StatusBadRequest)

		empty := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		if _, err := w.db.Exec(t.Context(), "DELETE FROM chapters WHERE version_id = $1", empty.VersionID); err != nil {
			t.Fatal(err)
		}
		refused(t, publish(w.author, w.a, empty, empty.VersionID), http.StatusBadRequest)
		if s := versionStatus(t, w.db, empty.VersionID); s != "draft" {
			t.Errorf("a draft with no Question Sets is %s, want draft", s)
		}

		published := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		refused(t, publish(w.author, w.a, published, published.VersionID), http.StatusConflict)
	})
	if s := versionStatus(t, w.db, hq.VersionID); s != "draft" {
		t.Errorf("after refusals the status is %s, want draft", s)
	}
}

func TestCreateVersion(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	newVersion := func(c *apitest.Client, h uuid.UUID, hq apitest.HQ) *apitest.Resp {
		return c.Do("POST", at(h, "/questionnaires/"+hq.ID.String()+"/versions"), nil)
	}
	versions := func(t *testing.T, hq apitest.HQ) int {
		return count(t, w.db, "SELECT count(*) FROM questionnaire_versions WHERE questionnaire_id = $1", hq.ID)
	}

	t.Run("happy path: a draft version 2 with the same Question Sets and stable IDs", func(t *testing.T) {
		hq := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		r := newVersion(w.author, w.a, hq)
		r.Want(t, http.StatusCreated)
		var v2 struct{ ID uuid.UUID }
		r.Decode(t, &v2)

		var no int
		var status string
		if err := w.db.QueryRow(t.Context(), "SELECT version_no, status FROM questionnaire_versions WHERE id = $1", v2.ID).Scan(&no, &status); err != nil {
			t.Fatal(err)
		}
		if no != 2 || status != "draft" || versionStatus(t, w.db, hq.VersionID) != "published" {
			t.Errorf("v2 is %d %s, v1 is %s; want 2 draft, and v1 still published", no, status, versionStatus(t, w.db, hq.VersionID))
		}
		// Each Question Set is copied as a new row with the same name, audience and content.
		same := count(t, w.db, `SELECT count(*) FROM chapters old JOIN chapters new
			ON new.position = old.position AND new.name = old.name AND new.audience = old.audience
			   AND new.content = old.content AND new.id <> old.id
			WHERE old.version_id = $1 AND new.version_id = $2`, hq.VersionID, v2.ID)
		if same != 2 || count(t, w.db, "SELECT count(*) FROM chapters WHERE version_id = $1", v2.ID) != 2 {
			t.Errorf("%d of 2 Question Sets copied unchanged", same)
		}
		var copied uuid.UUID
		if err := w.db.QueryRow(t.Context(), "SELECT id FROM chapters WHERE version_id = $1 AND position = 1", v2.ID).Scan(&copied); err != nil {
			t.Fatal(err)
		}
		if q := firstQuestion(t, w, copied); q != "q_smokes" {
			t.Errorf("first question's stable ID = %q, want q_smokes", q)
		}
	})

	hq := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, newVersion(w.anon, w.a, hq), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, newVersion(w.viewer, w.a, hq), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, newVersion(w.other, w.a, hq), http.StatusNotFound)
		refused(t, newVersion(w.other, w.b, hq), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("POST", at(w.a, "/questionnaires/abc/versions"), nil), http.StatusNotFound)
		draft := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
		refused(t, newVersion(w.author, w.a, draft), http.StatusConflict) // it already has a draft
		if versions(t, draft) != 1 {
			t.Error("a second draft was made")
		}
		// Another draft already has this questionnaire's name.
		if _, err := w.db.Exec(t.Context(), "UPDATE questionnaires SET name = $1 WHERE id = $2", hq.Name, draft.ID); err != nil {
			t.Fatal(err)
		}
		refused(t, newVersion(w.author, w.a, hq), http.StatusConflict)
	})
	if n := versions(t, hq); n != 1 {
		t.Errorf("after refusals there are %d versions, want 1", n)
	}
}

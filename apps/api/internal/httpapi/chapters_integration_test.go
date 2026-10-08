//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Content writes to a published version are refused in every test here ("published" cases).

// chapterRow is the part of a chapters row the tests check.
type chapterRow struct {
	Name, Description, Icon, Audience string
	Position, Revision                int
	Content                           string
}

func readChapter(t *testing.T, w *cast, id uuid.UUID) chapterRow {
	t.Helper()
	var c chapterRow
	err := w.db.QueryRow(t.Context(), `SELECT name, description, icon, audience, position, revision, content::text
		FROM chapters WHERE id = $1`, id).Scan(&c.Name, &c.Description, &c.Icon, &c.Audience, &c.Position, &c.Revision, &c.Content)
	if err != nil {
		t.Fatalf("chapter %s: %v", id, err)
	}
	return c
}

func chapterCount(t *testing.T, w *cast, hq apitest.HQ) int {
	t.Helper()
	return count(t, w.db, "SELECT count(*) FROM chapters WHERE version_id = $1", hq.VersionID)
}

func TestAddChapter(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	add := func(c *apitest.Client, h uuid.UUID, hq apitest.HQ, body any) *apitest.Resp {
		return c.Do("POST", at(h, "/questionnaires/"+hq.ID.String()+"/chapters"), body)
	}
	body := map[string]string{"name": " Medicines "}

	t.Run("happy path: added at the end", func(t *testing.T) {
		r := add(w.author, w.a, hq, body)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		c := readChapter(t, w, got.ID)
		if c.Name != "Medicines" || c.Position != 3 || c.Audience != "patient" || c.Revision != 0 {
			t.Errorf("row = %+v, want Medicines at position 3 for patients, revision 0", c)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, add(w.anon, w.a, hq, body), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, add(w.viewer, w.a, hq, body), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, add(w.other, w.a, hq, body), http.StatusNotFound)
		refused(t, add(w.other, w.b, hq, body), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, add(w.author, w.a, hq, map[string]string{"name": "  "}), http.StatusBadRequest)
		refused(t, add(w.author, w.a, hq, `{"name":`), http.StatusBadRequest)
		refused(t, w.author.Do("POST", at(w.a, "/questionnaires/abc/chapters"), body), http.StatusNotFound)
	})
	if n := chapterCount(t, w, hq); n != 3 {
		t.Errorf("chapters = %d, want 2 + the happy path's", n)
	}
	t.Run("published", func(t *testing.T) {
		pub := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		refused(t, add(w.author, w.a, pub, body), http.StatusConflict)
		if n := chapterCount(t, w, pub); n != 2 {
			t.Errorf("published version has %d chapters, want 2", n)
		}
	})
}

func TestReorderChapters(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	first, second := hq.Chapters[0], hq.Chapters[1]
	reorder := func(c *apitest.Client, h uuid.UUID, hq apitest.HQ, body any) *apitest.Resp {
		return c.Do("PUT", at(h, "/questionnaires/"+hq.ID.String()+"/chapter-order"), body)
	}
	positions := func(t *testing.T) [2]int {
		return [2]int{readChapter(t, w, first).Position, readChapter(t, w, second).Position}
	}
	swapped := []uuid.UUID{second, first}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, reorder(w.anon, w.a, hq, swapped), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, reorder(w.viewer, w.a, hq, swapped), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, reorder(w.other, w.a, hq, swapped), http.StatusNotFound)
		refused(t, reorder(w.other, w.b, hq, swapped), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, reorder(w.author, w.a, hq, []uuid.UUID{first}), http.StatusConflict)             // one missing
		refused(t, reorder(w.author, w.a, hq, []uuid.UUID{first, first}), http.StatusConflict)      // one twice
		refused(t, reorder(w.author, w.a, hq, []uuid.UUID{first, uuid.New()}), http.StatusConflict) // not in this version
		refused(t, reorder(w.author, w.a, hq, `["not-a-uuid"]`), http.StatusBadRequest)
	})
	if p := positions(t); p != [2]int{1, 2} {
		t.Fatalf("refusals moved chapters: %v", p)
	}
	t.Run("happy path", func(t *testing.T) {
		reorder(w.author, w.a, hq, swapped).Want(t, http.StatusNoContent)
		if p := positions(t); p != [2]int{2, 1} {
			t.Errorf("positions = %v, want [2 1]", p)
		}
	})
	t.Run("published", func(t *testing.T) {
		pub := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		refused(t, reorder(w.author, w.a, pub, []uuid.UUID{pub.Chapters[1], pub.Chapters[0]}), http.StatusConflict)
		if readChapter(t, w, pub.Chapters[0]).Position != 1 {
			t.Error("a published version's chapters were moved")
		}
	})
}

func TestGetChapter(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	path := func(h uuid.UUID) string { return at(h, "/chapters/"+hq.Chapters[0].String()) }

	t.Run("happy path: the SurveyJS JSON and its revision", func(t *testing.T) {
		r := w.author.Do("GET", path(w.a), nil)
		r.Want(t, http.StatusOK)
		var got struct {
			Name          string
			Revision      int
			VersionStatus string
			Content       struct {
				Pages []struct{ Elements []struct{ Name string } }
			}
		}
		r.Decode(t, &got)
		if got.Name != "About you" || got.Revision != 0 || got.VersionStatus != "draft" ||
			len(got.Content.Pages) != 1 || got.Content.Pages[0].Elements[0].Name != "q_smokes" {
			t.Errorf("chapter = %s", r)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", path(w.a), nil), http.StatusUnauthorized)
	})
	t.Run("role: a viewer can read", func(t *testing.T) {
		w.viewer.Do("GET", path(w.a), nil).Want(t, http.StatusOK)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, w.other.Do("GET", path(w.a), nil), http.StatusNotFound)
		refused(t, w.other.Do("GET", path(w.b), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("GET", at(w.a, "/chapters/abc"), nil), http.StatusNotFound)
		refused(t, w.author.Do("GET", at(w.a, "/chapters/"+uuid.NewString()), nil), http.StatusNotFound)
	})
}

func TestUpdateChapter(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	id := hq.Chapters[0]
	patch := func(c *apitest.Client, h uuid.UUID, id uuid.UUID, body any) *apitest.Resp {
		return c.Do("PATCH", at(h, "/chapters/"+id.String()), body)
	}
	before := readChapter(t, w, id)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, patch(w.anon, w.a, id, map[string]string{"name": "x"}), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, patch(w.viewer, w.a, id, map[string]string{"name": "x"}), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, patch(w.other, w.a, id, map[string]string{"name": "x"}), http.StatusNotFound)
		refused(t, patch(w.other, w.b, id, map[string]string{"name": "x"}), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{
			map[string]string{"name": "  "},
			map[string]string{"icon": "Heart!"},
			map[string]string{"audience": "nurse"},
			`{"name": null}`,
			`{"icon": null}`,
			`{"name":`,
		} {
			refused(t, patch(w.author, w.a, id, bad), http.StatusBadRequest)
		}
		refused(t, w.author.Do("PATCH", at(w.a, "/chapters/abc"), map[string]string{"name": "x"}), http.StatusNotFound)
	})
	if after := readChapter(t, w, id); after != before {
		t.Fatalf("refusals changed the chapter: %+v", after)
	}
	t.Run("happy path: fields left out keep their value", func(t *testing.T) {
		patch(w.author, w.a, id, map[string]string{"name": " Your health ", "icon": "heart", "audience": "clinician"}).
			Want(t, http.StatusNoContent)
		patch(w.author, w.a, id, map[string]string{"description": "About your health"}).Want(t, http.StatusNoContent)
		c := readChapter(t, w, id)
		if c.Name != "Your health" || c.Icon != "heart" || c.Audience != "clinician" || c.Description != "About your health" {
			t.Errorf("row = %+v", c)
		}
		if c.Content != before.Content || c.Revision != before.Revision {
			t.Error("a details change touched the content")
		}
	})
	t.Run("published", func(t *testing.T) {
		pub := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		refused(t, patch(w.author, w.a, pub.Chapters[0], map[string]string{"name": "x"}), http.StatusConflict)
		if readChapter(t, w, pub.Chapters[0]).Name != "About you" {
			t.Error("a published chapter was renamed")
		}
	})
}

func TestDeleteChapter(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	del := func(c *apitest.Client, h uuid.UUID, id uuid.UUID) *apitest.Resp {
		return c.Do("DELETE", at(h, "/chapters/"+id.String()), nil)
	}
	target := hq.Chapters[1]

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, del(w.anon, w.a, target), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, del(w.viewer, w.a, target), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, del(w.other, w.a, target), http.StatusNotFound)
		refused(t, del(w.other, w.b, target), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("DELETE", at(w.a, "/chapters/abc"), nil), http.StatusNotFound)
		refused(t, del(w.author, w.a, uuid.New()), http.StatusNotFound)
	})
	if n := chapterCount(t, w, hq); n != 2 {
		t.Fatalf("refusals deleted chapters: %d left", n)
	}
	t.Run("happy path: only that chapter goes", func(t *testing.T) {
		del(w.author, w.a, target).Want(t, http.StatusNoContent)
		if count(t, w.db, "SELECT count(*) FROM chapters WHERE id = $1", target) != 0 || chapterCount(t, w, hq) != 1 {
			t.Error("want only the other chapter left")
		}
	})
	t.Run("published", func(t *testing.T) {
		pub := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		refused(t, del(w.author, w.a, pub.Chapters[0]), http.StatusConflict)
		if chapterCount(t, w, pub) != 2 {
			t.Error("a published chapter was deleted")
		}
	})
}

func TestSaveChapterContent(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	hq := w.fx.DraftHQ(t, w.a, w.alex, "basic.json")
	id := hq.Chapters[0]
	save := func(c *apitest.Client, h uuid.UUID, id uuid.UUID, body any) *apitest.Resp {
		return c.Do("PUT", at(h, "/chapters/"+id.String()+"/content"), body)
	}
	content := func(question string) json.RawMessage {
		return json.RawMessage(`{"pages": [{"name": "p1", "elements": [{"type": "text", "name": "` + question + `"}]}]}`)
	}
	body := func(question string, revision int) map[string]any {
		return map[string]any{"content": content(question), "revision": revision}
	}
	original := readChapter(t, w, id)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, save(w.anon, w.a, id, body("q_anonym", 0)), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, save(w.viewer, w.a, id, body("q_viewer", 0)), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, save(w.other, w.a, id, body("q_others", 0)), http.StatusNotFound)
		refused(t, save(w.other, w.b, id, body("q_others", 0)), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{
			map[string]any{"content": []int{1, 2}, "revision": 0},
			map[string]any{"content": map[string]any{"pages": "none"}, "revision": 0},
			`{"content":`,
		} {
			refused(t, save(w.author, w.a, id, bad), http.StatusBadRequest)
		}
		refused(t, w.author.Do("PUT", at(w.a, "/chapters/abc/content"), body("q_badids", 0)), http.StatusNotFound)
	})
	if c := readChapter(t, w, id); c != original {
		t.Fatalf("refusals changed the chapter: %+v", c)
	}

	t.Run("happy path: each save adds 1 to the revision", func(t *testing.T) {
		r := save(w.author, w.a, id, body("q_first", 0))
		r.Want(t, http.StatusOK)
		var got struct{ Revision int }
		r.Decode(t, &got)
		if c := readChapter(t, w, id); got.Revision != 1 || c.Revision != 1 || firstQuestion(t, w, id) != "q_first" {
			t.Fatalf("after one save: response revision %d, row %+v", got.Revision, c)
		}
		save(w.author, w.a, id, body("q_second", 1)).Want(t, http.StatusOK)
		if c := readChapter(t, w, id); c.Revision != 2 {
			t.Errorf("after two saves the revision is %d, want 2", c.Revision)
		}
	})
	t.Run("a stale revision: 409 and nothing written", func(t *testing.T) {
		// A second editor still holds revision 1, from before the last save.
		r := save(w.author, w.a, id, body("q_stale", 1))
		refused(t, r, http.StatusConflict)
		if c := readChapter(t, w, id); c.Revision != 2 || firstQuestion(t, w, id) != "q_second" {
			t.Errorf("row = %+v, want revision 2 with q_second", c)
		}
	})
	t.Run("published", func(t *testing.T) {
		pub := w.fx.PublishedHQ(t, w.a, w.alex, "basic.json")
		before := readChapter(t, w, pub.Chapters[0])
		refused(t, save(w.author, w.a, pub.Chapters[0], body("q_publsh", 0)), http.StatusConflict)
		if c := readChapter(t, w, pub.Chapters[0]); c != before {
			t.Error("a published chapter's content changed")
		}
	})
}

// firstQuestion is the stable ID of the chapter's first question.
func firstQuestion(t *testing.T, w *cast, id uuid.UUID) string {
	t.Helper()
	var name string
	if err := w.db.QueryRow(t.Context(), "SELECT content->'pages'->0->'elements'->0->>'name' FROM chapters WHERE id = $1", id).Scan(&name); err != nil {
		t.Fatal(err)
	}
	return name
}

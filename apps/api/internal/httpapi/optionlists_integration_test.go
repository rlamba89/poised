//go:build integration

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

var yesNo = []map[string]any{{"text": "Yes", "score": 1}, {"text": "No", "score": 0}}

// addList inserts an option list straight into the database.
func addList(t *testing.T, w *cast, hospital uuid.UUID, by apitest.User, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := w.db.QueryRow(t.Context(), `INSERT INTO option_lists (hospital_id, name, options, updated_by)
		VALUES ($1, $2, '[{"text": "Yes"}, {"text": "No"}]', $3) RETURNING id`, hospital, name, by.ID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// listRow is a list's name and its options as jsonb text.
func listRow(t *testing.T, w *cast, id uuid.UUID) (name, options string) {
	t.Helper()
	if err := w.db.QueryRow(t.Context(), "SELECT name, options::text FROM option_lists WHERE id = $1", id).Scan(&name, &options); err != nil {
		t.Fatalf("option list %s: %v", id, err)
	}
	return name, options
}

// tooMany is 501 options, one more than a list may have.
func tooMany() []map[string]any {
	options := make([]map[string]any, 501)
	for i := range options {
		options[i] = map[string]any{"text": "Option"}
	}
	return options
}

// badLists are bodies every option-list write refuses with 400.
func badLists() []any {
	return []any{
		map[string]any{"name": "  ", "options": yesNo},
		map[string]any{"name": "Empty", "options": []any{}},
		map[string]any{"name": "No label", "options": []map[string]any{{"score": 1}}},
		map[string]any{"name": "Blank label", "options": []map[string]any{{"text": "  "}}},
		map[string]any{"name": "Not a list", "options": "Yes, No"},
		map[string]any{"name": "Too many", "options": tooMany()},
		`{"name":`,
	}
}

func TestListOptionLists(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	addList(t, w, w.a, w.alex, "zeta")
	addList(t, w, w.a, w.alex, "Alpha")
	addList(t, w, w.b, w.bea, "Beta")
	names := func(t *testing.T, c *apitest.Client, h uuid.UUID) string {
		t.Helper()
		r := c.Do("GET", at(h, "/option-lists"), nil)
		r.Want(t, http.StatusOK)
		var got []struct{ Name string }
		r.Decode(t, &got)
		var out []string
		for _, l := range got {
			out = append(out, l.Name)
		}
		return strings.Join(out, ", ")
	}

	t.Run("happy path: the hospital's lists, by name", func(t *testing.T) {
		if got := names(t, w.author, w.a); got != "Alpha, zeta" {
			t.Errorf("lists = %q, want Alpha, zeta", got)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", at(w.a, "/option-lists"), nil), http.StatusUnauthorized)
	})
	t.Run("role: a viewer can read", func(t *testing.T) {
		if got := names(t, w.viewer, w.a); got != "Alpha, zeta" {
			t.Errorf("lists = %q", got)
		}
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, w.other.Do("GET", at(w.a, "/option-lists"), nil), http.StatusNotFound)
		if got := names(t, w.other, w.b); got != "Beta" {
			t.Errorf("hospital B's lists = %q, want only Beta", got)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("GET", "/api/h/abc/option-lists", nil), http.StatusNotFound)
	})
}

func TestCreateOptionList(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	body := map[string]any{"name": " Yes or no ", "options": yesNo}
	create := func(c *apitest.Client, h uuid.UUID, body any) *apitest.Resp {
		return c.Do("POST", at(h, "/option-lists"), body)
	}
	total := func(t *testing.T) int { return count(t, w.db, "SELECT count(*) FROM option_lists") }

	t.Run("happy path", func(t *testing.T) {
		r := create(w.author, w.a, body)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		var hospital uuid.UUID
		var name string
		var options int
		err := w.db.QueryRow(t.Context(), "SELECT hospital_id, name, jsonb_array_length(options) FROM option_lists WHERE id = $1",
			got.ID).Scan(&hospital, &name, &options)
		if err != nil || hospital != w.a || name != "Yes or no" || options != 2 {
			t.Errorf("row = %v %q %d options (%v), want hospital A, trimmed name, 2 options", hospital, name, options, err)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, create(w.anon, w.a, body), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, create(w.viewer, w.a, body), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, create(w.other, w.a, body), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range badLists() {
			refused(t, create(w.author, w.a, bad), http.StatusBadRequest)
		}
		refused(t, create(w.author, w.a, map[string]any{"name": "Yes or no", "options": yesNo}), http.StatusConflict)
	})
	if n := total(t); n != 1 {
		t.Errorf("lists = %d, want only the happy path's", n)
	}
	t.Run("another hospital can use the same name", func(t *testing.T) {
		create(w.other, w.b, body).Want(t, http.StatusCreated)
	})
}

func TestUpdateOptionList(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	id := addList(t, w, w.a, w.alex, "Yes or no")
	addList(t, w, w.a, w.alex, "Taken")
	update := func(c *apitest.Client, h uuid.UUID, id string, body any) *apitest.Resp {
		return c.Do("PUT", at(h, "/option-lists/"+id), body)
	}
	body := map[string]any{"name": "Yes, no or unsure", "options": append(yesNo, map[string]any{"text": "Unsure"})}
	beforeName, beforeOptions := listRow(t, w, id)

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, update(w.anon, w.a, id.String(), body), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, update(w.viewer, w.a, id.String(), body), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, update(w.other, w.a, id.String(), body), http.StatusNotFound)
		refused(t, update(w.other, w.b, id.String(), body), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range badLists() {
			refused(t, update(w.author, w.a, id.String(), bad), http.StatusBadRequest)
		}
		refused(t, update(w.author, w.a, id.String(), map[string]any{"name": "Taken", "options": yesNo}), http.StatusConflict)
		refused(t, update(w.author, w.a, "abc", body), http.StatusNotFound)
		refused(t, update(w.author, w.a, uuid.NewString(), body), http.StatusNotFound)
	})
	if name, options := listRow(t, w, id); name != beforeName || options != beforeOptions {
		t.Fatalf("refusals changed the list: %q %s", name, options)
	}
	t.Run("happy path", func(t *testing.T) {
		update(w.author, w.a, id.String(), body).Want(t, http.StatusNoContent)
		name, _ := listRow(t, w, id)
		n := count(t, w.db, "SELECT jsonb_array_length(options) FROM option_lists WHERE id = $1", id)
		if name != "Yes, no or unsure" || n != 3 {
			t.Errorf("row = %q with %d options, want the new name and 3 options", name, n)
		}
	})
}

func TestDeleteOptionList(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	id := addList(t, w, w.a, w.alex, "Yes or no")
	del := func(c *apitest.Client, h uuid.UUID, id string) *apitest.Resp {
		return c.Do("DELETE", at(h, "/option-lists/"+id), nil)
	}
	exists := func(t *testing.T) bool {
		return count(t, w.db, "SELECT count(*) FROM option_lists WHERE id = $1", id) == 1
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, del(w.anon, w.a, id.String()), http.StatusUnauthorized)
	})
	t.Run("wrong role: a viewer", func(t *testing.T) {
		refused(t, del(w.viewer, w.a, id.String()), http.StatusForbidden)
	})
	t.Run("another hospital", func(t *testing.T) {
		refused(t, del(w.other, w.a, id.String()), http.StatusNotFound)
		refused(t, del(w.other, w.b, id.String()), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, del(w.author, w.a, "abc"), http.StatusNotFound)
		refused(t, del(w.author, w.a, uuid.NewString()), http.StatusNotFound)
	})
	if !exists(t) {
		t.Fatal("a refused delete deleted the list")
	}
	t.Run("happy path", func(t *testing.T) {
		del(w.author, w.a, id.String()).Want(t, http.StatusNoContent)
		if exists(t) {
			t.Error("still there")
		}
	})
}

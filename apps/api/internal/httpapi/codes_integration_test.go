//go:build integration

package httpapi_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// Codes and categories are reference data shared by every hospital, so any signed-in user
// may read them: there's no role to lack, and no other hospital's copy to reach.

// addCategories inserts categories and returns their ids by name.
func addCategories(t *testing.T, w *cast, names map[string]bool) map[string]uuid.UUID {
	t.Helper()
	ids := map[string]uuid.UUID{}
	for name, noteOnly := range names {
		ids[name] = uuid.New()
		if _, err := w.db.Exec(t.Context(), "INSERT INTO categories (id, name, note_only) VALUES ($1, $2, $3)",
			ids[name], name, noteOnly); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

func TestSearchCodes(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	lifestyle := addCategories(t, w, map[string]bool{"Lifestyle": false})["Lifestyle"]
	for _, c := range []struct {
		set, code, description, status string
	}{
		{"SNOMED", "77176002", "Smoker", "active"},
		{"SNOMED", "266919005", "Never smoked tobacco", "active"},
		{"SNOMED", "8517006", "Ex-smoker", "active"},
		{"SNOMED", "38341003", "Hypertensive disorder", "active"},
		{"SNOMED", "77176999", "Old smoking code", "retired"},
		{"ICD10", "F17.1", "Harmful use of tobacco", "active"},
	} {
		if _, err := w.db.Exec(t.Context(), `INSERT INTO codes (code_set, code, description, full_name, category_id, status)
			VALUES ($1, $2, $3, $3, $4, $5)`, c.set, c.code, c.description, lifestyle, c.status); err != nil {
			t.Fatal(err)
		}
	}
	search := func(t *testing.T, c *apitest.Client, set, q string) []string {
		t.Helper()
		r := c.Do("GET", "/api/codes?set="+set+"&q="+url.QueryEscape(q), nil)
		r.Want(t, http.StatusOK)
		var got []struct{ Code, CategoryName string }
		r.Decode(t, &got)
		var codes []string
		for _, c := range got {
			codes = append(codes, c.Code)
			if c.CategoryName != "Lifestyle" {
				t.Errorf("%s has category %q, want Lifestyle", c.Code, c.CategoryName)
			}
		}
		return codes
	}

	t.Run("happy path: by code prefix and by words in the description", func(t *testing.T) {
		for _, tt := range []struct{ set, q, want string }{
			{"SNOMED", "7717", "77176002"},                   // code prefix; the retired code isn't found
			{"SNOMED", "176002", ""},                         // the middle of a code isn't a prefix
			{"SNOMED", "SMOK", "77176002 266919005 8517006"}, // a word's start, any case, anywhere in the description
			{"SNOMED", "tobacco", "266919005"},               // a word in the middle; the ICD-10 code isn't found
			{"SNOMED", "smoked tobacco", "266919005"},        // several words
			{"ICD10", "tobacco", "F17.1"},                    // the other code set
		} {
			got := strings.Join(search(t, w.author, tt.set, tt.q), " ")
			if !sameWords(got, tt.want) {
				t.Errorf("%s %q found %q, want %q", tt.set, tt.q, got, tt.want)
			}
		}
		// An exact code match comes first.
		if got := search(t, w.author, "SNOMED", "8517006"); len(got) == 0 || got[0] != "8517006" {
			t.Errorf("exact code found %v, want 8517006 first", got)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", "/api/codes?set=SNOMED&q=smok", nil), http.StatusUnauthorized)
	})
	t.Run("role and hospital: any signed-in user gets the same codes", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.viewer, w.other} {
			if got := search(t, c, "SNOMED", "7717"); len(got) != 1 || got[0] != "77176002" {
				t.Errorf("found %v, want 77176002", got)
			}
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.author.Do("GET", "/api/codes?q=smok", nil), http.StatusBadRequest)
		refused(t, w.author.Do("GET", "/api/codes?set=LOINC&q=smok", nil), http.StatusBadRequest)
		if got := search(t, w.author, "SNOMED", "   "); len(got) != 0 {
			t.Errorf("a blank search found %v, want nothing", got)
		}
	})
}

// sameWords reports whether a and b hold the same space-separated words, in any order.
func sameWords(a, b string) bool {
	x, y := strings.Fields(a), strings.Fields(b)
	if len(x) != len(y) {
		return false
	}
	seen := map[string]int{}
	for _, s := range x {
		seen[s]++
	}
	for _, s := range y {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	return true
}

func TestListCategories(t *testing.T) {
	t.Parallel()
	w := newCast(t)
	addCategories(t, w, map[string]bool{"Lifestyle": false, "Cardiac": false, "Unassigned": true})
	names := func(t *testing.T, c *apitest.Client) string {
		t.Helper()
		r := c.Do("GET", "/api/categories", nil)
		r.Want(t, http.StatusOK)
		var got []struct {
			Name     string
			NoteOnly bool
		}
		r.Decode(t, &got)
		var out []string
		for _, c := range got {
			out = append(out, c.Name)
		}
		return strings.Join(out, ", ")
	}

	t.Run("happy path: by name, the note-only category last", func(t *testing.T) {
		if got := names(t, w.author); got != "Cardiac, Lifestyle, Unassigned" {
			t.Errorf("categories = %q", got)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.anon.Do("GET", "/api/categories", nil), http.StatusUnauthorized)
	})
	t.Run("role and hospital: any signed-in user gets the same categories", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.viewer, w.other} {
			if got := names(t, c); got != "Cardiac, Lifestyle, Unassigned" {
				t.Errorf("categories = %q", got)
			}
		}
	})
	// Invalid input: the route takes none.
}

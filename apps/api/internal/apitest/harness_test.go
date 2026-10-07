//go:build integration

package apitest_test

import (
	"net/http"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

func TestHealth(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)

	if r := c.Do(http.MethodGet, "/api/health", nil); r.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", r.Status, r.Body)
	}
}

func TestLoginAsKeepsTheCookie(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	hospital := c.Hospital("Test Hospital")
	user := c.User(hospital, "super_clinician")

	if r := c.Do(http.MethodGet, "/api/me", nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("before sign-in: status = %d, want 401", r.Status)
	}

	c.LoginAs(user)
	r := c.Do(http.MethodGet, "/api/me", nil)
	if r.Status != http.StatusOK {
		t.Fatalf("after sign-in: status = %d, want 200: %s", r.Status, r.Body)
	}
	var me struct {
		User      struct{ ID string }
		Hospitals []struct {
			ID   string
			Role string
		}
	}
	r.Decode(t, &me)
	if me.User.ID != user.String() {
		t.Errorf("user id = %q, want %q", me.User.ID, user)
	}
	if len(me.Hospitals) != 1 || me.Hospitals[0].ID != hospital.String() || me.Hospitals[0].Role != "super_clinician" {
		t.Errorf("hospitals = %+v, want one: %s with role super_clinician", me.Hospitals, hospital)
	}
}

// Each test's database is its own: rows written in one are not seen in another.
func TestDatabasesAreIsolated(t *testing.T) {
	t.Parallel()
	a, b := apitest.New(t), apitest.New(t)
	a.Hospital("Only in A")

	var n int
	if err := b.DB.QueryRow(t.Context(), "SELECT count(*) FROM hospitals").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("hospitals in b = %d, want 0", n)
	}
}

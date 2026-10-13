//go:build integration

package main

import (
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
)

var errBoom = errors.New("boom")

func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }

func count(t *testing.T, c *apitest.Client, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := c.DB.QueryRow(t.Context(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAdd(t *testing.T) {
	c, fx := apitest.New(t)
	cog := &cognito.Recorder{}

	t.Run("a new email: an account, the platform_admins row, and the Cognito user", func(t *testing.T) {
		created, err := add(t.Context(), c.DB, cog, " Rahul@Poised.Example ", "")
		if err != nil || !created {
			t.Fatalf("add = %v, %v", created, err)
		}
		if n := count(t, c, `SELECT count(*) FROM users u JOIN platform_admins p ON p.user_id = u.id
			WHERE u.email = 'rahul@poised.example' AND u.name = 'rahul@poised.example'`); n != 1 {
			t.Errorf("rows %d", n)
		}
		if calls := cog.Calls(); len(calls) != 1 || calls[0] != (cognito.Call{Op: "CreateUser", Email: "rahul@poised.example"}) {
			t.Errorf("cognito %+v", calls)
		}
		// They sign in and are a platform admin, with no trust.
		var id apitest.User
		if err := c.DB.QueryRow(t.Context(), `SELECT id, name FROM users WHERE email = 'rahul@poised.example'`).Scan(&id.ID, &id.Name); err != nil {
			t.Fatal(err)
		}
		r := c.LoginAs(t, id).Do("GET", "/api/admin/orgs", nil)
		r.Want(t, http.StatusOK)
	})
	t.Run("running it again changes nothing", func(t *testing.T) {
		created, err := add(t.Context(), c.DB, cog, "rahul@poised.example", "Rahul")
		if err != nil || created {
			t.Fatalf("add = %v, %v", created, err)
		}
		if count(t, c, `SELECT count(*) FROM platform_admins`) != 1 || len(cog.Calls()) != 1 {
			t.Error("a second row or Cognito call")
		}
	})
	t.Run("an existing account becomes a platform admin; Cognito already has it", func(t *testing.T) {
		u := fx.User(t, apitest.Hospital{}, "")
		var email string
		if err := c.DB.QueryRow(t.Context(), `SELECT email FROM users WHERE id = $1`, u.ID).Scan(&email); err != nil {
			t.Fatal(err)
		}
		if _, err := add(t.Context(), c.DB, cog, email, ""); err != nil {
			t.Fatal(err)
		}
		if count(t, c, `SELECT count(*) FROM platform_admins WHERE user_id = $1`, u.ID) != 1 || len(cog.Calls()) != 1 {
			t.Error("not added, or Cognito was called")
		}
	})
	t.Run("an invalid email is refused", func(t *testing.T) {
		for _, e := range []string{"", "nope", "a@b", "Ann <ann@x.example>"} {
			if _, err := add(t.Context(), c.DB, cog, e, ""); err == nil {
				t.Errorf("%q was accepted", e)
			}
		}
	})
	t.Run("Cognito failing leaves nothing behind", func(t *testing.T) {
		cog.FailWith(errBoom)
		defer cog.FailWith(nil)
		if _, err := add(t.Context(), c.DB, cog, "new@poised.example", ""); err == nil {
			t.Fatal("no error")
		}
		if count(t, c, `SELECT count(*) FROM users WHERE email = 'new@poised.example'`) != 0 {
			t.Error("the account was kept")
		}
	})
}

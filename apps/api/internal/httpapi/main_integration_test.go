//go:build integration

package httpapi_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// TestMain migrates the template database once; each test then gets its own copy.
func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }

func TestHealth(t *testing.T) {
	t.Parallel()
	c, _ := apitest.New(t)

	r := c.Do("GET", "/api/health", nil)
	r.Want(t, http.StatusOK)
	var body struct{ Status string }
	r.Decode(t, &body)
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
}

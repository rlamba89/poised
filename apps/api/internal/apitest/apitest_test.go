//go:build integration

package apitest_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }

// The fixtures' rows are complete enough for the real API to read them.
func TestFixtures(t *testing.T) {
	t.Parallel()
	c, fx := apitest.New(t)
	h := fx.Hospital(t)
	author := fx.TrustUser(t, h.Org, "super_clinician")
	nurse := fx.User(t, h, "clinician")
	a2 := fx.HospitalIn(t, h.Org)
	hq := fx.PublishedHQ(t, h, author, "basic.json")
	e := fx.Episode(t, hq, nurse)

	var n int
	if err := c.DB.QueryRow(t.Context(), "SELECT count(*) FROM hospitals").Scan(&n); err != nil || n != 2 {
		t.Errorf("hospitals = %d (%v), want 2: each test starts from the empty template", n, err)
	}

	var me struct {
		Trusts []struct {
			ID        uuid.UUID
			Role      string
			Hospitals []struct{ ID uuid.UUID }
		}
	}
	signedIn := c.LoginAs(t, author)
	r := signedIn.Do("GET", "/api/me", nil)
	r.Want(t, http.StatusOK)
	r.Decode(t, &me)
	if len(me.Trusts) != 1 || me.Trusts[0].ID != h.Org || me.Trusts[0].Role != "super_clinician" || len(me.Trusts[0].Hospitals) != 2 {
		t.Errorf("me = %+v, want a trust-wide super clinician of %s, with both its hospitals", me, h.Org)
	}

	var got struct {
		Questionnaire struct{ Status string }
		Chapters      []struct{ ID uuid.UUID }
	}
	r = signedIn.Do("GET", a2.Path("/questionnaires/"+hq.ID.String()), nil)
	r.Want(t, http.StatusOK)
	r.Decode(t, &got)
	if got.Questionnaire.Status != "published" || len(got.Chapters) != 2 || got.Chapters[0].ID != hq.Chapters[0] {
		t.Errorf("questionnaire = %s, want published with chapters %v", r, hq.Chapters)
	}

	c.LoginAs(t, nurse).Do("GET", h.Path("/episodes/"+e.ID.String()), nil).Want(t, http.StatusOK)
	c.Do("GET", "/api/p/"+e.Token, nil).Want(t, http.StatusOK)

	// c itself was never signed in.
	c.Do("GET", "/api/me", nil).Want(t, http.StatusUnauthorized)
}

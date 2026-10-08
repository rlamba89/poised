//go:build integration

package apitest_test

import (
	"net/http"
	"os"
	"slices"
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
	author := fx.User(t, h, "author", "publisher")
	nurse := fx.User(t, h, "clinician")
	hq := fx.PublishedHQ(t, h, author, "basic.json")
	e := fx.Episode(t, hq, nurse)

	var n int
	if err := c.DB.QueryRow(t.Context(), "SELECT count(*) FROM hospitals").Scan(&n); err != nil || n != 1 {
		t.Errorf("hospitals = %d (%v), want 1: each test starts from the empty template", n, err)
	}

	var me struct {
		Hospitals []struct {
			ID    uuid.UUID
			Roles []string
		}
	}
	signedIn := c.LoginAs(t, author)
	r := signedIn.Do("GET", "/api/me", nil)
	r.Want(t, http.StatusOK)
	r.Decode(t, &me)
	if len(me.Hospitals) != 1 || me.Hospitals[0].ID != h || !slices.Equal(me.Hospitals[0].Roles, []string{"author", "publisher"}) {
		t.Errorf("me = %+v, want author and publisher in %s", me, h)
	}

	var got struct {
		Questionnaire struct{ Status string }
		Chapters      []struct{ ID uuid.UUID }
	}
	r = signedIn.Do("GET", "/api/h/"+h.String()+"/questionnaires/"+hq.ID.String(), nil)
	r.Want(t, http.StatusOK)
	r.Decode(t, &got)
	if got.Questionnaire.Status != "published" || len(got.Chapters) != 2 || got.Chapters[0].ID != hq.Chapters[0] {
		t.Errorf("questionnaire = %s, want published with chapters %v", r, hq.Chapters)
	}

	c.LoginAs(t, nurse).Do("GET", "/api/h/"+h.String()+"/episodes/"+e.ID.String(), nil).Want(t, http.StatusOK)
	c.Do("GET", "/api/p/"+e.Token, nil).Want(t, http.StatusOK)

	// c itself was never signed in.
	c.Do("GET", "/api/me", nil).Want(t, http.StatusUnauthorized)
}

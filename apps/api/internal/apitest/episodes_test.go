//go:build integration

package apitest_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// An episode needs a patient and a published HQ of the same hospital; it comes with one link.
func TestCreateEpisode(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	h, other := c.Hospital("H"), c.Hospital("Other")
	author := c.User(h, "super_clinician")
	version := c.PublishedHQ(h, author, hqContent)
	patient := c.Patient(h, patientDOB)
	c.LoginAs(c.User(h, "clinician"))
	path := c.HospitalPath(h) + "/episodes"

	r := c.Do(http.MethodPost, path, map[string]any{"patientId": patient, "versionId": version, "procedure": " Knee replacement "})
	if r.Status != http.StatusCreated {
		t.Fatalf("create: status = %d: %s", r.Status, r.Body)
	}
	var created struct{ ID uuid.UUID }
	r.Decode(t, &created)
	var procedure, event string
	var links int
	if err := c.DB.QueryRow(t.Context(), `SELECT e.procedure, (SELECT text FROM episode_events WHERE episode_id = e.id),
		(SELECT count(*) FROM login_links WHERE episode_id = e.id) FROM episodes e WHERE e.id = $1`, created.ID).Scan(&procedure, &event, &links); err != nil {
		t.Fatal(err)
	}
	if procedure != "Knee replacement" || event != "Episode created" || links != 1 {
		t.Errorf("row: procedure %q, event %q, links %d; want trimmed, \"Episode created\", 1", procedure, event, links)
	}

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"unpublished version", map[string]any{"patientId": patient, "versionId": uuid.New()}},
		{"patient of another hospital", map[string]any{"patientId": c.Patient(other, patientDOB), "versionId": version}},
		{"HQ of another hospital", map[string]any{"patientId": patient, "versionId": c.PublishedHQ(other, c.User(other, "super_clinician"), hqContent)}},
	} {
		if r := c.Do(http.MethodPost, path, tc.body); r.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, r.Status)
		}
	}
	var n int
	if err := c.DB.QueryRow(t.Context(), "SELECT count(*) FROM episodes").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("episodes = %d, want only the first", n)
	}
}

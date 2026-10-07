//go:build integration

package apitest_test

import (
	"crypto/sha256"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

const (
	patientDOB = "1974-12-11"
	// One patient question and one clinician-only question, which patients must never get.
	hqContent = `{"pages":[{"name":"p1","elements":[
		{"type":"text","name":"q_smoke","title":"Do you smoke?"},
		{"type":"comment","name":"q_clin","title":"Advice","clinicianOnly":true}]}]}`
)

// episodeWithLink sets up a hospital, a clinician (signed in on c), a patient and an episode,
// and returns the episode id and the patient's link token as the clinician sees it.
func episodeWithLink(t *testing.T, c *apitest.Client) (base string, episode uuid.UUID, token string) {
	t.Helper()
	h := c.Hospital("H")
	version := c.PublishedHQ(h, c.User(h, "super_clinician"), hqContent)
	patient := c.Patient(h, patientDOB)
	c.LoginAs(c.User(h, "clinician"))
	base = c.HospitalPath(h)

	r := c.Do(http.MethodPost, base+"/episodes", map[string]any{"patientId": patient, "versionId": version})
	if r.Status != http.StatusCreated {
		t.Fatalf("create episode: status = %d: %s", r.Status, r.Body)
	}
	var created struct{ ID uuid.UUID }
	r.Decode(t, &created)
	return base, created.ID, linkOf(t, c, base, created.ID)
}

// linkOf is the patient link token the clinician sees on the episode ("" when none can be shown).
func linkOf(t *testing.T, c *apitest.Client, base string, episode uuid.UUID) string {
	t.Helper()
	r := c.Do(http.MethodGet, base+"/episodes/"+episode.String(), nil)
	if r.Status != http.StatusOK {
		t.Fatalf("get episode: status = %d: %s", r.Status, r.Body)
	}
	var got struct{ PatientToken *string }
	r.Decode(t, &got)
	if got.PatientToken == nil {
		return ""
	}
	return *got.PatientToken
}

func dob(p *apitest.Client, token, date string) *apitest.Resp {
	return p.Do(http.MethodPost, "/api/p/links/"+token+"/dob", map[string]string{"dateOfBirth": date})
}

func TestPatientSignsInWithLinkAndDateOfBirth(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	_, episode, token := episodeWithLink(t, c)
	if len(token) != 43 {
		t.Fatalf("token = %q, want 43 characters", token)
	}
	patient := c.NewBrowser()

	// The link alone shows nothing.
	if r := patient.Do(http.MethodGet, "/api/p/hq", nil); r.Status != http.StatusUnauthorized {
		t.Errorf("HQ before sign-in: status = %d, want 401", r.Status)
	}
	if r := patient.Do(http.MethodGet, "/api/p/"+token, nil); r.Status != http.StatusNotFound {
		t.Errorf("the old link-only route: status = %d, want 404", r.Status)
	}
	r := patient.Do(http.MethodPost, "/api/p/links/"+token+"/continue", nil)
	if r.Status != http.StatusOK || !strings.Contains(string(r.Body), `"signedIn":false`) {
		t.Fatalf("continue: %d %s, want 200 signedIn false", r.Status, r.Body)
	}
	if strings.Contains(string(r.Body), "Jo") || strings.Contains(string(r.Body), patientDOB) {
		t.Errorf("continue leaked patient details: %s", r.Body)
	}

	if r := dob(patient, token, "1974-12-12"); r.Status != http.StatusBadRequest {
		t.Errorf("wrong date of birth: status = %d, want 400", r.Status)
	}
	if r := dob(patient, token, patientDOB); r.Status != http.StatusNoContent {
		t.Fatalf("right date of birth: status = %d: %s", r.Status, r.Body)
	}
	if r := patient.Do(http.MethodPost, "/api/p/links/"+token+"/continue", nil); !strings.Contains(string(r.Body), `"signedIn":true`) {
		t.Errorf("continue when signed in: %s, want signedIn true", r.Body)
	}

	// Signed in: the HQ, without clinician-only content, and the usual autosave and submit.
	r = patient.Do(http.MethodGet, "/api/p/hq", nil)
	if r.Status != http.StatusOK {
		t.Fatalf("HQ: status = %d: %s", r.Status, r.Body)
	}
	if strings.Contains(string(r.Body), "q_clin") || strings.Contains(string(r.Body), "Assessment") {
		t.Errorf("HQ has clinician content: %s", r.Body)
	}
	var hq struct {
		Chapters []struct{ ID uuid.UUID }
		Patient  struct{ FirstName string }
	}
	r.Decode(t, &hq)
	if len(hq.Chapters) != 1 || hq.Patient.FirstName != "Jo" {
		t.Fatalf("HQ = %+v, want one patient Question Set for Jo", hq)
	}
	set := hq.Chapters[0].ID.String()
	if r := patient.Do(http.MethodPut, "/api/p/hq/answers/"+set, map[string]any{"data": map[string]any{"q_smoke": "no", "q_clin": "x"}}); r.Status != http.StatusNoContent {
		t.Fatalf("save answers: status = %d: %s", r.Status, r.Body)
	}
	var saved string
	if err := c.DB.QueryRow(t.Context(), "SELECT data::text FROM episode_answers WHERE episode_id = $1 AND actor = 'patient'", episode).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saved, "q_smoke") || strings.Contains(saved, "q_clin") {
		t.Errorf("saved answers = %s, want q_smoke only", saved)
	}
	if r := patient.Do(http.MethodPost, "/api/p/hq/submit", nil); r.Status != http.StatusNoContent {
		t.Fatalf("submit: status = %d: %s", r.Status, r.Body)
	}
	var status string
	if err := c.DB.QueryRow(t.Context(), "SELECT status FROM episodes WHERE id = $1", episode).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "ready_for_review" {
		t.Errorf("episode status = %q, want ready_for_review", status)
	}

	// A patient session is not a staff session.
	if r := patient.Do(http.MethodGet, "/api/me", nil); r.Status != http.StatusUnauthorized {
		t.Errorf("patient on /api/me: status = %d, want 401", r.Status)
	}

	if r := patient.Do(http.MethodPost, "/api/p/logout", nil); r.Status != http.StatusNoContent {
		t.Errorf("logout: status = %d", r.Status)
	}
	if r := patient.Do(http.MethodGet, "/api/p/hq", nil); r.Status != http.StatusUnauthorized {
		t.Errorf("HQ after logout: status = %d, want 401", r.Status)
	}
}

// The database holds the link only as a hash (to find it) and encrypted (to show it again).
func TestPatientLinkIsNotStoredInPlainText(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	_, episode, token := episodeWithLink(t, c)

	var hash, sealed []byte
	if err := c.DB.QueryRow(t.Context(), "SELECT token_hash, token_sealed FROM login_links WHERE episode_id = $1", episode).Scan(&hash, &sealed); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte(token))
	if string(hash) != string(want[:]) {
		t.Error("token_hash is not the SHA-256 of the token")
	}
	if strings.Contains(string(sealed), token) {
		t.Error("token_sealed contains the token in plain text")
	}
}

// Five wrong dates of birth lock the link (PAT-05); then even the right one is refused.
func TestWrongDatesOfBirthLockTheLink(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	base, episode, token := episodeWithLink(t, c)
	patient := c.NewBrowser()

	for i := 1; i <= 4; i++ {
		if r := dob(patient, token, "2000-01-01"); r.Status != http.StatusBadRequest {
			t.Fatalf("wrong try %d: status = %d, want 400", i, r.Status)
		}
	}
	if r := dob(patient, token, "2000-01-01"); r.Status != http.StatusForbidden {
		t.Errorf("fifth wrong try: status = %d, want 403 (locked)", r.Status)
	}
	if r := dob(patient, token, patientDOB); r.Status != http.StatusForbidden {
		t.Errorf("right date after the lock: status = %d, want 403", r.Status)
	}
	if r := patient.Do(http.MethodPost, "/api/p/links/"+token+"/continue", nil); r.Status != http.StatusForbidden {
		t.Errorf("continue on a locked link: status = %d, want 403", r.Status)
	}
	r := c.Do(http.MethodGet, base+"/episodes/"+episode.String(), nil)
	if !strings.Contains(string(r.Body), "locked") {
		t.Errorf("the episode's notes don't mention the lock: %s", r.Body)
	}
}

func TestBadLinksAreNotFound(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	for _, token := range []string{strings.Repeat("A", 43), "short", strings.Repeat("A", 42) + "!"} {
		if r := c.Do(http.MethodPost, "/api/p/links/"+token+"/continue", nil); r.Status != http.StatusNotFound {
			t.Errorf("continue %q: status = %d, want 404", token, r.Status)
		}
		if r := dob(c, token, patientDOB); r.Status != http.StatusNotFound {
			t.Errorf("dob %q: status = %d, want 404", token, r.Status)
		}
	}
	_, _, token := episodeWithLink(t, c)
	for _, date := range []string{"", "11/12/1974", "1974-13-40"} {
		if r := dob(c.NewBrowser(), token, date); r.Status != http.StatusBadRequest {
			t.Errorf("date %q: status = %d, want 400", date, r.Status)
		}
	}
}

// A new link replaces the old one, which stops working, and signs out anyone using it.
func TestNewLinkReplacesTheOld(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	base, episode, old := episodeWithLink(t, c)
	patient := c.NewBrowser()
	if r := dob(patient, old, patientDOB); r.Status != http.StatusNoContent {
		t.Fatalf("sign in with the old link: status = %d", r.Status)
	}

	r := c.Do(http.MethodPost, base+"/episodes/"+episode.String()+"/patient-link", nil)
	if r.Status != http.StatusOK {
		t.Fatalf("new link: status = %d: %s", r.Status, r.Body)
	}
	var got struct{ PatientToken string }
	r.Decode(t, &got)
	if got.PatientToken == "" || got.PatientToken == old || linkOf(t, c, base, episode) != got.PatientToken {
		t.Fatalf("new token %q: want a new one, shown on the episode", got.PatientToken)
	}

	if r := patient.Do(http.MethodGet, "/api/p/hq", nil); r.Status != http.StatusUnauthorized {
		t.Errorf("session from the old link: status = %d, want 401", r.Status)
	}
	if r := patient.Do(http.MethodPost, "/api/p/links/"+old+"/continue", nil); r.Status != http.StatusNotFound {
		t.Errorf("old link: status = %d, want 404", r.Status)
	}
	if r := dob(patient, got.PatientToken, patientDOB); r.Status != http.StatusNoContent {
		t.Errorf("new link: status = %d, want 204", r.Status)
	}
}

// A link made before links were encrypted still works, but can't be shown: the clinician makes a new one.
func TestLinkFromBeforeEncryption(t *testing.T) {
	t.Parallel()
	c := apitest.New(t)
	base, episode, token := episodeWithLink(t, c)
	if _, err := c.DB.Exec(t.Context(), "UPDATE login_links SET token_sealed = NULL WHERE episode_id = $1", episode); err != nil {
		t.Fatal(err)
	}
	if got := linkOf(t, c, base, episode); got != "" {
		t.Errorf("shown token = %q, want none", got)
	}
	if r := dob(c.NewBrowser(), token, patientDOB); r.Status != http.StatusNoContent {
		t.Errorf("sign in: status = %d, want 204", r.Status)
	}
}

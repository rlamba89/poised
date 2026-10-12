// Package httpapi holds the HTTP router, middleware and handlers.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/config"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// Deps is what the routers need. Later plans add the clock, SMS, email, S3, Cognito and
// evaluate; tests pass fakes.
type Deps struct {
	Pool   *pgxpool.Pool
	Config config.Config // the token secret, DEV_LOGIN and the per-country values
}

type server struct {
	pool   *pgxpool.Pool // for transactions; queries inside them use db.New(tx)
	q      db.Querier
	secret []byte
	cfg    config.Config
}

func newServer(d Deps) *server {
	return &server{pool: d.Pool, q: db.New(d.Pool), secret: []byte(d.Config.TokenSecret), cfg: d.Config}
}

// NewStaffRouter serves the staff API: /api/o/…, /api/me, /api/codes, /api/categories,
// /api/logout, and the dev login when DEV_LOGIN=true. On AWS it's the staff-api Lambda (D-4).
func NewStaffRouter(d Deps) http.Handler { return mount(newServer(d), (*server).staffRoutes) }

// NewPatientRouter serves only the patient's link, /api/p/…: the patient-api Lambda (D-4).
func NewPatientRouter(d Deps) http.Handler { return mount(newServer(d), (*server).patientRoutes) }

// NewRouter mounts both on one mux, for the local all-in-one server (cmd/api) and the tests.
func NewRouter(d Deps) http.Handler { return routes(newServer(d)) }

func routes(s *server) http.Handler {
	return mount(s, (*server).staffRoutes, (*server).patientRoutes)
}

// mount builds a mux with the routes every function serves, plus the given groups.
// Paths use Go 1.22+ method and wildcard patterns.
func mount(s *server, groups ...func(*server, *http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/config", s.publicConfig)
	for _, add := range groups {
		add(s, mux)
	}
	return recoverPanics(mux)
}

func (s *server) staffRoutes(mux *http.ServeMux) {
	user := func(h http.HandlerFunc) http.HandlerFunc { return requireUser(s.secret, h) }
	// Every trust's data is under /api/o/{oid}: a hospital's under /api/o/{oid}/h/{hid}.
	hosp := func(h http.HandlerFunc) http.HandlerFunc { return requireUser(s.secret, requireHospital(s.q, h)) }
	org := func(h http.HandlerFunc) http.HandlerFunc { return requireUser(s.secret, requireOrg(s.q, h)) }

	if s.cfg.DevLogin { // the stub sign-in: never in Training or Production (F4)
		mux.HandleFunc("GET /api/dev/users", s.devUsers)
		mux.HandleFunc("POST /api/dev/login", s.devLogin)
	}
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/me", user(s.me))
	mux.HandleFunc("GET /api/codes", user(s.searchCodes))
	mux.HandleFunc("GET /api/categories", user(s.listCategories))

	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/questionnaires", hosp(s.listQuestionnaires))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires", hosp(s.createQuestionnaire))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/questionnaires/{qid}", hosp(s.getQuestionnaire))
	mux.HandleFunc("DELETE /api/o/{oid}/h/{hid}/questionnaires/{qid}", hosp(s.deleteQuestionnaire))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/chapters", hosp(s.addChapter))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/questionnaires/{qid}/chapter-order", hosp(s.reorderChapters))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/publish", hosp(s.publishQuestionnaire))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/versions", hosp(s.createVersion))

	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/chapters/{cid}", hosp(s.getChapter))
	mux.HandleFunc("PATCH /api/o/{oid}/h/{hid}/chapters/{cid}", hosp(s.updateChapter))
	mux.HandleFunc("DELETE /api/o/{oid}/h/{hid}/chapters/{cid}", hosp(s.deleteChapter))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/chapters/{cid}/content", hosp(s.saveContent))

	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/option-lists", hosp(s.listOptionLists))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/option-lists", hosp(s.createOptionList))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/option-lists/{lid}", hosp(s.updateOptionList))
	mux.HandleFunc("DELETE /api/o/{oid}/h/{hid}/option-lists/{lid}", hosp(s.deleteOptionList))

	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/patients", hosp(s.listPatients))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/patients", hosp(s.createPatient))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/published-hqs", hosp(s.listPublishedHQs))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes", hosp(s.listEpisodes))
	mux.HandleFunc("GET /api/o/{oid}/episodes", org(s.listTrustEpisodes)) // "All hospitals" (STF-03)
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes", hosp(s.createEpisode))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}", hosp(s.getEpisode))
	mux.HandleFunc("PATCH /api/o/{oid}/h/{hid}/episodes/{eid}", hosp(s.updateEpisode))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/notes", hosp(s.addEpisodeNote))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}/hq", hosp(s.episodeHQ))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/episodes/{eid}/answers/{cid}", hosp(s.saveClinicianAnswers))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/complete-review", hosp(s.completeReview))
}

// patientRoutes is the patient's link: no sign-in, the token is the key (plan-workflow.md Step 3).
func (s *server) patientRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/p/{token}", s.patientHQ)
	mux.HandleFunc("PUT /api/p/{token}/answers/{cid}", s.savePatientAnswers)
	mux.HandleFunc("POST /api/p/{token}/submit", s.submitPatientHQ)
}

// publicConfig is the deployment's public per-country values (A-20), read once by the web app.
// No sign-in: the sign-in page and the patient's link need it first.
func (s *server) publicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.cfg.Public())
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	if _, err := s.q.Ping(r.Context()); err != nil {
		log.Printf("health: %v", err)
		writeError(w, http.StatusServiceUnavailable, "The database is not reachable.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readJSON decodes the body into v, answering 400 itself when it can't.
func readJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes)).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "The request could not be read.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

// writeError sends the plain-language error shape used by every endpoint (DSG-04).
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

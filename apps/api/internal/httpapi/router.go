// Package httpapi holds the HTTP router, middleware and handlers.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

type server struct {
	pool   *pgxpool.Pool // for transactions; queries inside them use db.New(tx)
	q      db.Querier
	secret []byte
}

// NewRouter wires every route. Paths use Go 1.22+ method and wildcard patterns.
func NewRouter(pool *pgxpool.Pool, secret []byte) http.Handler {
	return routes(&server{pool: pool, q: db.New(pool), secret: secret})
}

func routes(s *server) http.Handler {
	user := func(h http.HandlerFunc) http.HandlerFunc { return requireUser(s.secret, h) }
	hosp := func(h http.HandlerFunc) http.HandlerFunc { return requireUser(s.secret, requireHospital(s.q, h)) }

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)

	mux.HandleFunc("GET /api/dev/users", s.devUsers)
	mux.HandleFunc("POST /api/dev/login", s.devLogin)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/me", user(s.me))
	mux.HandleFunc("GET /api/codes", user(s.searchCodes))
	mux.HandleFunc("GET /api/categories", user(s.listCategories))

	mux.HandleFunc("GET /api/h/{hid}/questionnaires", hosp(s.listQuestionnaires))
	mux.HandleFunc("POST /api/h/{hid}/questionnaires", hosp(s.createQuestionnaire))
	mux.HandleFunc("GET /api/h/{hid}/questionnaires/{qid}", hosp(s.getQuestionnaire))
	mux.HandleFunc("DELETE /api/h/{hid}/questionnaires/{qid}", hosp(s.deleteQuestionnaire))
	mux.HandleFunc("POST /api/h/{hid}/questionnaires/{qid}/chapters", hosp(s.addChapter))
	mux.HandleFunc("PUT /api/h/{hid}/questionnaires/{qid}/chapter-order", hosp(s.reorderChapters))
	mux.HandleFunc("POST /api/h/{hid}/questionnaires/{qid}/publish", hosp(s.publishQuestionnaire))
	mux.HandleFunc("POST /api/h/{hid}/questionnaires/{qid}/versions", hosp(s.createVersion))

	mux.HandleFunc("GET /api/h/{hid}/chapters/{cid}", hosp(s.getChapter))
	mux.HandleFunc("PATCH /api/h/{hid}/chapters/{cid}", hosp(s.updateChapter))
	mux.HandleFunc("DELETE /api/h/{hid}/chapters/{cid}", hosp(s.deleteChapter))
	mux.HandleFunc("PUT /api/h/{hid}/chapters/{cid}/content", hosp(s.saveContent))

	mux.HandleFunc("GET /api/h/{hid}/option-lists", hosp(s.listOptionLists))
	mux.HandleFunc("POST /api/h/{hid}/option-lists", hosp(s.createOptionList))
	mux.HandleFunc("PUT /api/h/{hid}/option-lists/{lid}", hosp(s.updateOptionList))
	mux.HandleFunc("DELETE /api/h/{hid}/option-lists/{lid}", hosp(s.deleteOptionList))

	mux.HandleFunc("GET /api/h/{hid}/patients", hosp(s.listPatients))
	mux.HandleFunc("POST /api/h/{hid}/patients", hosp(s.createPatient))
	mux.HandleFunc("GET /api/h/{hid}/published-hqs", hosp(s.listPublishedHQs))
	mux.HandleFunc("GET /api/h/{hid}/episodes", hosp(s.listEpisodes))
	mux.HandleFunc("POST /api/h/{hid}/episodes", hosp(s.createEpisode))
	mux.HandleFunc("GET /api/h/{hid}/episodes/{eid}", hosp(s.getEpisode))
	mux.HandleFunc("PATCH /api/h/{hid}/episodes/{eid}", hosp(s.updateEpisode))
	mux.HandleFunc("POST /api/h/{hid}/episodes/{eid}/notes", hosp(s.addEpisodeNote))

	// The patient's link: no sign-in (plan-workflow.md Step 3).
	mux.HandleFunc("GET /api/p/{token}", s.patientHQ)
	mux.HandleFunc("PUT /api/p/{token}/answers/{cid}", s.savePatientAnswers)
	mux.HandleFunc("POST /api/p/{token}/submit", s.submitPatientHQ)
	return mux
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

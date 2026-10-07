// Package httpapi holds the HTTP router, middleware and handlers.
package httpapi

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/auth"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

// Config is what the router needs besides the database.
type Config struct {
	// SessionMaxAge and SessionIdle end a session that long after sign-in, or after that long
	// without a request. Zero means auth.DefaultMaxAge and auth.DefaultIdle.
	SessionMaxAge time.Duration
	SessionIdle   time.Duration
	// DevLogin turns on the stub sign-in (GET /api/dev/users, POST /api/dev/login):
	// locally and in QA only, never in Training or Production (f3-lambda-entrypoints.md).
	DevLogin bool
	// SecureCookies marks the session cookie Secure. Off only for plain-HTTP local development.
	SecureCookies bool
	// LinkKey is the 32-byte key that seals patient link tokens (LINK_KEY), so clinicians can
	// see a link again. It is kept outside the database.
	LinkKey []byte
	// SignIn, Directory and Login turn on staff sign-in with Cognito (all three, or none).
	// Directory may be nil alone: sign-in then works but invites answer 503.
	SignIn    StaffSignIn
	Directory StaffDirectory
	Login     *CognitoLogin
	// AppOrigin is the web app's origin, e.g. "http://localhost:3000". Writes from it are trusted
	// even though the API, behind the app's /api proxy, sees a different Host.
	AppOrigin string
}

type server struct {
	pool          *pgxpool.Pool // for transactions; queries inside them use db.New(tx)
	q             db.Querier
	sessionMaxAge time.Duration
	sessionIdle   time.Duration
	devLoginOn    bool
	secureCookies bool
	sealer        *auth.Sealer // nil only in tests that don't reach patient links
	signIn        StaffSignIn  // nil: Cognito sign-in is off
	directory     StaffDirectory
	login         *CognitoLogin
}

// NewRouter wires every route behind the cross-site check. Paths use Go 1.22+ method and
// wildcard patterns. It fails when cfg.LinkKey or cfg.AppOrigin is not valid.
func NewRouter(pool *pgxpool.Pool, cfg Config) (http.Handler, error) {
	sealer, err := auth.NewSealer(cfg.LinkKey)
	if err != nil {
		return nil, err
	}
	s := &server{
		sealer: sealer, signIn: cfg.SignIn, directory: cfg.Directory, login: cfg.Login,
		pool: pool, q: db.New(pool),
		sessionMaxAge: cmp.Or(cfg.SessionMaxAge, auth.DefaultMaxAge),
		sessionIdle:   cmp.Or(cfg.SessionIdle, auth.DefaultIdle),
		devLoginOn:    cfg.DevLogin, secureCookies: cfg.SecureCookies,
	}
	return refuseCrossSite(routes(s), cfg.AppOrigin)
}

func routes(s *server) http.Handler {
	user := func(h http.HandlerFunc) http.HandlerFunc { return requireUser(s.q, s.cookieName(), s.sessionIdle, h) }
	hosp := func(h http.HandlerFunc) http.HandlerFunc {
		return requireUser(s.q, s.cookieName(), s.sessionIdle, requireMembership(s.q, h))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)

	if s.devLoginOn {
		mux.HandleFunc("GET /api/dev/users", s.devUsers)
		mux.HandleFunc("POST /api/dev/login", s.devLogin)
	}
	mux.HandleFunc("GET /api/auth/config", s.authConfig)
	if s.signIn != nil {
		mux.HandleFunc("POST /api/auth/callback", s.cognitoCallback)
	}
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/me", user(s.me))
	mux.HandleFunc("GET /api/codes", user(s.searchCodes))
	mux.HandleFunc("GET /api/categories", user(s.listCategories))

	trustAdmin := func(h http.HandlerFunc) http.HandlerFunc {
		return requireUser(s.q, s.cookieName(), s.sessionIdle, requireTrustAdmin(s.q, h))
	}
	mux.HandleFunc("GET /api/o/{oid}/staff", trustAdmin(s.listStaff))
	mux.HandleFunc("POST /api/o/{oid}/staff/invites", trustAdmin(s.inviteStaff))

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
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes", hosp(s.createEpisode))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}", hosp(s.getEpisode))
	mux.HandleFunc("PATCH /api/o/{oid}/h/{hid}/episodes/{eid}", hosp(s.updateEpisode))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/notes", hosp(s.addEpisodeNote))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}/hq", hosp(s.episodeHQ))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/episodes/{eid}/answers/{cid}", hosp(s.saveClinicianAnswers))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/complete-review", hosp(s.completeReview))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/patient-link", hosp(s.newPatientLink))

	// The patient: their link and date of birth start a session (patient.go), which the HQ needs.
	mux.HandleFunc("POST /api/p/links/{token}/continue", s.continueLink)
	mux.HandleFunc("POST /api/p/links/{token}/dob", s.confirmDateOfBirth)
	mux.HandleFunc("GET /api/p/hq", s.patientHQ)
	mux.HandleFunc("PUT /api/p/hq/answers/{cid}", s.savePatientAnswers)
	mux.HandleFunc("POST /api/p/hq/submit", s.submitPatientHQ)
	mux.HandleFunc("POST /api/p/logout", s.patientLogout)
	return mux
}

// refuseCrossSite rejects writes a browser sends from another site (CSRF), since sign-in is a
// cookie. The standard library reads Sec-Fetch-Site, or compares Origin with Host; appOrigin,
// when set, is trusted as well.
func refuseCrossSite(h http.Handler, appOrigin string) (http.Handler, error) {
	c := http.NewCrossOriginProtection()
	if appOrigin != "" {
		if err := c.AddTrustedOrigin(appOrigin); err != nil {
			return nil, fmt.Errorf("app origin: %w", err)
		}
	}
	c.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, "This request came from another site, so it was refused.")
	}))
	return c.Handler(h), nil
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

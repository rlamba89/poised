// Package httpapi holds the HTTP router, middleware and handlers.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/clock"
	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/config"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/invite"
	"github.com/rlamba89/poised/apps/api/internal/jobs"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
)

// Deps is what the routers need. Later plans add S3 and evaluate; tests pass fakes.
type Deps struct {
	Pool    *pgxpool.Pool
	Config  config.Config  // the token secret, DEV_LOGIN, MESSAGING and the per-country values
	Cognito cognito.Client // staff accounts, their invitations and the sign-in's code exchange (S09, S12)
	// CognitoKeys check the pool's ID tokens (S12). nil: the pool's own published keys (JWKS);
	// tests pass a fixed set.
	CognitoKeys cognito.Keys
	Clock       clock.Clock     // nil: the system clock
	SMS         messaging.SMS   // patients' texts (S14): captured until S17
	Email       messaging.Email // patients' emails (S14): captured until S17
}

type server struct {
	pool    *pgxpool.Pool // for transactions; queries inside them use db.New(tx)
	q       db.Querier
	secret  []byte
	cfg     config.Config
	cognito cognito.Client
	clock   clock.Clock
	sms     messaging.SMS
	email   messaging.Email
	inviter invite.Sender    // the patient's invites and Resend (S14); reminders use it too (S16)
	jobs    jobs.Runner      // the scheduled jobs, for the QA "run jobs as of" route (S16)
	idToken cognito.Verifier // checks the ID token at the Cognito sign-in (S12), when there's a pool
}

func newServer(d Deps) *server {
	if d.Cognito == nil {
		panic("httpapi: Deps.Cognito is not set") // a missing one would lose every invitation
	}
	if d.SMS == nil || d.Email == nil {
		panic("httpapi: Deps.SMS or Deps.Email is not set") // a missing one would lose every invite
	}
	if d.Clock == nil {
		d.Clock = clock.Real{}
	}
	inviter := invite.Sender{Pool: d.Pool, SMS: d.SMS, Email: d.Email, Clock: d.Clock,
		AppURL: d.Config.Country.AppURL, PhoneCountry: d.Config.Country.PhoneCountry}
	s := &server{pool: d.Pool, q: db.New(d.Pool), secret: []byte(d.Config.TokenSecret), cfg: d.Config,
		cognito: d.Cognito, clock: d.Clock, sms: d.SMS, email: d.Email, inviter: inviter,
		jobs: jobs.Runner{Pool: d.Pool, Inviter: inviter, DefaultZone: d.Config.Country.TimeZone}}
	if pool := d.Config.Cognito; pool.Enabled() {
		keys := d.CognitoKeys
		if keys == nil {
			keys = &cognito.JWKS{URL: cognito.JWKSURL(pool.Issuer())}
		}
		s.idToken = cognito.Verifier{Issuer: pool.Issuer(), ClientID: pool.ClientID, Keys: keys}
	}
	return s
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
	if s.cfg.Messaging == config.MessagingCapture { // the captured SMS and email (local and QA only)
		qa := s.qaTool
		mux.HandleFunc("GET /api/qa/messages", qa(s.qaMessages))
		mux.HandleFunc("DELETE /api/qa/rate-limits", qa(s.qaResetRateLimits)) // a tester's fresh start (S15)
		mux.HandleFunc("GET /api/qa/client", qa(noStore(s.qaClient)))         // the address rate limits count (S06)
		// The scheduled jobs (S16): run them as of a later time, or move the local timer's clock.
		mux.HandleFunc("GET /api/qa/jobs", qa(s.qaJobs))
		mux.HandleFunc("POST /api/qa/jobs/run", qa(s.qaRunJobs))
		mux.HandleFunc("PUT /api/qa/jobs/clock", qa(s.qaSetJobsClock))
	}
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

	if s.cfg.DevLogin { // the stub sign-in: local only (config refuses it on AWS, S12)
		mux.HandleFunc("GET /api/dev/users", s.devUsers)
		mux.HandleFunc("POST /api/dev/login", s.devLogin)
	}
	if s.cfg.Cognito.Enabled() { // Cognito's managed login sends the browser back with a code (S12)
		mux.HandleFunc("POST /api/auth/callback", noStore(s.cognitoCallback))
	}
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/me", user(s.me))
	mux.HandleFunc("GET /api/codes", user(s.searchCodes))
	mux.HandleFunc("GET /api/categories", user(s.listCategories))

	// The superadmin console (S10): platform admins only (A-21). No patient data (OPS-06).
	admin := func(h http.HandlerFunc) http.HandlerFunc { return requireUser(s.secret, requirePlatformAdmin(s.q, h)) }
	mux.HandleFunc("GET /api/admin/opt-ins", admin(s.adminOptInDefs))
	mux.HandleFunc("GET /api/admin/orgs", admin(s.adminListOrgs))
	mux.HandleFunc("POST /api/admin/orgs", admin(s.adminCreateOrg))
	mux.HandleFunc("GET /api/admin/orgs/{oid}", admin(s.adminGetOrg))
	mux.HandleFunc("PATCH /api/admin/orgs/{oid}", admin(s.adminUpdateOrg)) // suspend, re-activate, opt-ins
	mux.HandleFunc("POST /api/admin/orgs/{oid}/hospitals", admin(s.adminAddHospital))
	mux.HandleFunc("PATCH /api/admin/orgs/{oid}/hospitals/{hid}", admin(s.adminUpdateHospital))
	mux.HandleFunc("POST /api/admin/orgs/{oid}/admins", admin(s.adminInviteAdmin))

	// The trust's settings and each hospital's (S10): whole-trust admins only.
	mux.HandleFunc("GET /api/o/{oid}/settings", org(s.getTrustSettings))
	mux.HandleFunc("PATCH /api/o/{oid}/settings", org(s.patchTrustSettings))

	// The trust's staff (S09): whole-trust admins only.
	mux.HandleFunc("GET /api/o/{oid}/staff", org(s.listStaff))
	mux.HandleFunc("POST /api/o/{oid}/staff/invites", org(s.inviteStaff))
	mux.HandleFunc("DELETE /api/o/{oid}/staff/{uid}", org(s.removeStaff)) // all their access to the trust
	// One membership: {scope} is "trust" (the whole trust) or a hospital id.
	mux.HandleFunc("PATCH /api/o/{oid}/staff/{uid}/access/{scope}", org(s.updateStaffAccess))
	mux.HandleFunc("DELETE /api/o/{oid}/staff/{uid}/access/{scope}", org(s.removeStaffAccess))

	// The audit log (S11): whole-trust admins only; ?format=csv exports it. Reading it is audited too.
	mux.HandleFunc("GET /api/o/{oid}/audit", org(auditAdmin(s.audited("audit.view", s.listAudit))))

	// The trust's procedure list (S19, PRC-01): everyone in the trust reads it; whole-trust super
	// clinicians and admins change it, and each change is in the audit log.
	mux.HandleFunc("GET /api/o/{oid}/procedures", org(s.listProcedures))
	mux.HandleFunc("POST /api/o/{oid}/procedures", org(s.audited("procedure.create", s.createProcedure)))
	mux.HandleFunc("PATCH /api/o/{oid}/procedures/{prid}", org(s.audited("procedure.update", s.updateProcedure)))

	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/questionnaires", hosp(s.listQuestionnaires))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires", hosp(s.createQuestionnaire))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/questionnaires/{qid}", hosp(s.getQuestionnaire))
	mux.HandleFunc("DELETE /api/o/{oid}/h/{hid}/questionnaires/{qid}", hosp(s.deleteQuestionnaire))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/chapters", hosp(s.addChapter))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/questionnaires/{qid}/chapter-order", hosp(s.reorderChapters))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/publish", hosp(s.publishQuestionnaire))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/versions", hosp(s.createVersion))
	// Where the trust uses each question key (S23): the editor's pre-fill label and option lock (question_keys.go).
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/questionnaires/{qid}/question-keys", hosp(s.questionKeys))

	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/chapters/{cid}", hosp(s.getChapter))
	mux.HandleFunc("PATCH /api/o/{oid}/h/{hid}/chapters/{cid}", hosp(s.updateChapter))
	mux.HandleFunc("DELETE /api/o/{oid}/h/{hid}/chapters/{cid}", hosp(s.deleteChapter))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/chapters/{cid}/content", hosp(s.saveContent))

	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/option-lists", hosp(s.listOptionLists))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/option-lists", hosp(s.createOptionList))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/option-lists/{lid}", hosp(s.updateOptionList))
	mux.HandleFunc("DELETE /api/o/{oid}/h/{hid}/option-lists/{lid}", hosp(s.deleteOptionList))

	// Patient data: each request writes one audit row (audited, S11; STF-06).
	// Patients (S13): this hospital's only. A new patient is created with their first episode.
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/patients", hosp(s.audited("patient.search", s.searchPatients)))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/patients/{pid}", hosp(s.audited("patient.view", s.getPatient)))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/published-hqs", hosp(s.listPublishedHQs))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes", hosp(s.audited("episode.list", s.listEpisodes)))
	mux.HandleFunc("GET /api/o/{oid}/episodes", org(s.audited("episode.list_trust", s.listTrustEpisodes))) // "All hospitals" (STF-03)
	// The worklists' tabs (S21, EPI-02): their labels and counts; the lists are ?worklist= above (worklists.go).
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/worklists", hosp(s.listWorklists))
	mux.HandleFunc("GET /api/o/{oid}/worklists", org(s.listTrustWorklists))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes", hosp(s.audited("episode.create", s.createEpisode)))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}", hosp(s.audited("episode.view", s.getEpisode)))
	mux.HandleFunc("PATCH /api/o/{oid}/h/{hid}/episodes/{eid}", hosp(s.audited("episode.update", s.updateEpisode)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/notes", hosp(s.audited("episode.note", s.addEpisodeNote)))
	// The status (S20): the transition table's moves, cancel, re-open and undo (statuses.go).
	mux.HandleFunc("PATCH /api/o/{oid}/h/{hid}/episodes/{eid}/status", hosp(s.audited("episode.status", s.changeStatus)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/cancel", hosp(s.audited("episode.cancel", s.cancelEpisode)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/reopen", hosp(s.audited("episode.reopen", s.reopenReview)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/undo", hosp(s.audited("episode.undo", s.undoStatus)))
	// Archive and un-archive (S22, EPI-04): a super clinician's; an archived episode is read-only.
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/archive", hosp(s.audited("episode.archive", s.archiveEpisode)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/unarchive", hosp(s.audited("episode.unarchive", s.unarchiveEpisode)))
	// Tasks (S21, TSK-01): anyone at the hospital adds, completes or re-opens one (tasks.go).
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}/tasks", hosp(s.audited("task.list", s.listTasks)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/tasks", hosp(s.audited("task.create", s.createTask)))
	mux.HandleFunc("PATCH /api/o/{oid}/h/{hid}/episodes/{eid}/tasks/{tid}", hosp(s.audited("task.update", s.updateTask)))
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}/hq", hosp(s.audited("answers.view", s.episodeHQ)))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/episodes/{eid}/answers/{cid}", hosp(s.audited("answers.save", s.saveClinicianAnswers)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/complete-review", hosp(s.audited("review.complete", s.completeReview)))
	// The patient's invite (S14): a new link by SMS and email, replacing the old one.
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/resend-invite", hosp(s.audited("invite.resend", s.resendInvite)))
	// Completing the HQ on the patient's behalf (PAT-09): their form, answers marked "entered by".
	mux.HandleFunc("GET /api/o/{oid}/h/{hid}/episodes/{eid}/patient-form", hosp(s.audited("behalf.view", s.staffPatientForm)))
	mux.HandleFunc("PUT /api/o/{oid}/h/{hid}/episodes/{eid}/patient-form/answers/{cid}", hosp(s.audited("behalf.save", s.staffSavePatientForm)))
	mux.HandleFunc("POST /api/o/{oid}/h/{hid}/episodes/{eid}/patient-form/submit", hosp(s.audited("behalf.submit", s.staffSubmitPatientForm)))
}

// patientRoutes are the patient's (S14, A-4): their link's Continue and date of birth, which
// start a session, then their episodes' HQs with that session. There's no GET on a link: the link
// alone returns nothing. Each request is audited with the patient as the actor (S11).
func (s *server) patientRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/p/links/{token}/continue", noStore(s.audited("link.open", s.linkContinue)))
	mux.HandleFunc("POST /api/p/links/{token}/dob", noStore(s.audited("link.dob", s.linkDob)))
	mux.HandleFunc("POST /api/p/logout", noStore(s.patientLogout))
	// A session held in the database (patient_sessions), checked on every request.
	session := func(h http.HandlerFunc) http.HandlerFunc { return noStore(s.requirePatient(h)) }
	mux.HandleFunc("GET /api/p/episodes/{eid}", session(s.audited("link.view", s.patientForm)))
	mux.HandleFunc("PUT /api/p/episodes/{eid}/answers/{cid}", session(s.audited("link.save", s.savePatientForm)))
	mux.HandleFunc("POST /api/p/episodes/{eid}/submit", session(s.audited("link.submit", s.submitPatientForm)))

	// Accounts across trusts (S15): a returning patient's sign-in link (by mobile or email, then
	// Continue and their date of birth, as an invite), their home, and changing a contact. The
	// home and the changes write a row to each trust's log whose records they read or changed.
	mux.HandleFunc("POST /api/p/signin", noStore(s.requestSignin))
	mux.HandleFunc("POST /api/p/signin/{token}/continue", noStore(s.signinContinue))
	mux.HandleFunc("POST /api/p/signin/{token}/dob", noStore(s.audited("account.join", s.signinDob)))
	mux.HandleFunc("GET /api/p/home", session(s.audited("home.view", s.patientHome)))
	mux.HandleFunc("POST /api/p/contact", session(s.requestContactChange))
	mux.HandleFunc("POST /api/p/contact/{token}/confirm", noStore(s.audited("contact.update", s.confirmContact)))
	// Closing the account (S22, EPI-09): the login goes; every trust's records and episodes stay.
	mux.HandleFunc("POST /api/p/account/close", session(s.audited("account.close", s.closeAccount)))
}

// qaTool guards a tester's tool (S12). With the dev login on (local) anyone can sign in as
// anyone, the platform admin included, so a check would add nothing and the tools stay open;
// without it (QA) they need a signed-in platform admin, so nothing on QA works without credentials.
func (s *server) qaTool(h http.HandlerFunc) http.HandlerFunc {
	if s.cfg.DevLogin {
		return h
	}
	return requireUser(s.secret, requirePlatformAdmin(s.q, h))
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

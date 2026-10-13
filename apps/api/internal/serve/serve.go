// Package serve starts one of the API's servers: config, database pool, router, listen.
// cmd/api (every route, for local dev), cmd/staff-api and cmd/patient-api all use it.
//
// The servers keep no state in memory and never touch local disk, so the same router can run
// on AWS Lambda (D-4). S06 adds the Lambda adapter here, used when AWS_LAMBDA_FUNCTION_NAME is set.
package serve

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/config"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
	"github.com/rlamba89/poised/apps/api/internal/jobs"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
)

// Main runs the router built by router until the server fails. name is for the log. With
// jobsTimer (cmd/api only) it also runs the scheduled jobs every minute, unless JOBS_TIMER=false
// (S16): on AWS a schedule runs the jobs Lambda instead (S17).
func Main(name string, router func(httpapi.Deps) http.Handler, jobsTimer bool) {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatalf("%s: %v", name, err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("%s: db: %v", name, err)
	}
	defer pool.Close()

	// MESSAGING=capture is the only mode until S17 (config.Load refuses any other): texts and
	// emails are stored in sent_messages, never sent.
	capture := messaging.Capture{DB: db.New(pool)}
	deps := httpapi.Deps{Pool: pool, Config: cfg, Cognito: cognito.Local{}, SMS: capture, Email: capture}
	srv := &http.Server{Addr: cfg.Addr, Handler: router(deps)}
	devLogin, timer := "off", "off"
	if cfg.DevLogin {
		devLogin = "on"
	}
	if jobsTimer && cfg.JobsTimer {
		timer = "every " + jobs.TimerInterval.String()
		// Testers' clock offset applies only where messages are captured (local and QA).
		go httpapi.NewJobsRunner(deps).Every(context.Background(), jobs.TimerInterval, cfg.Messaging == config.MessagingCapture)
	}
	log.Printf("%s listening on %s (dev login %s, messaging %s, jobs timer %s, locale %s, time zone %s)",
		name, cfg.Addr, devLogin, cfg.Messaging, timer, cfg.Country.Locale, cfg.Country.TimeZone)
	log.Fatal(srv.ListenAndServe())
}

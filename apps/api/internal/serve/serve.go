// Package serve starts one of the API's servers: config, database pool, router, listen.
// cmd/api (every route, for local dev), cmd/staff-api and cmd/patient-api all use it.
//
// The servers keep no state in memory and never touch local disk, so the same router runs on
// AWS Lambda (D-4): when AWS_LAMBDA_FUNCTION_NAME is set, Main hands it to the Lambda adapter
// (lambda.go) instead of listening, after fetching its secrets from AWS (config.Resolve).
package serve

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
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
	ctx := context.Background()
	getenv, err := config.Resolve(ctx, os.Getenv, &awsSecrets{}) // AWS only when a value is named there
	if err != nil {
		log.Fatalf("%s: %v", name, err)
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		log.Fatalf("%s: %v", name, err)
	}
	isLambda := onLambda(os.Getenv)
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("%s: db: the database URL is not valid", name) // never print it: it holds the password
	}
	if isLambda {
		// One request at a time per Lambda instance (plan F3 Step 3). Idle connections are
		// dropped soon, since a frozen instance can't keep them alive; pgxpool pings a
		// connection that has been idle before reusing it.
		poolCfg.MaxConns = 2
		poolCfg.MaxConnIdleTime = time.Minute
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("%s: db: %v", name, err)
	}
	defer pool.Close()

	// MESSAGING=capture is the only mode until S17 (config.Load refuses any other): texts and
	// emails are stored in sent_messages, never sent.
	capture := messaging.Capture{DB: db.New(pool)}
	// The staff user pool (S12): on AWS, and locally only if COGNITO_* are set; otherwise the
	// stand-in that only logs, and staff use the dev login.
	var userPool cognito.Client = cognito.Local{}
	if c := cfg.Cognito; c.Enabled() {
		userPool = &cognito.AWS{PoolID: c.UserPoolID, ClientID: c.ClientID, Domain: c.Domain, RedirectURI: c.RedirectURI}
	}
	deps := httpapi.Deps{Pool: pool, Config: cfg, Cognito: userPool, SMS: capture, Email: capture}
	handler := router(deps)
	devLogin, timer := "off", "off"
	if cfg.DevLogin {
		devLogin = "on"
	}
	if cfg.Cognito.Enabled() {
		devLogin += ", Cognito sign-in on"
	}
	if isLambda {
		log.Printf("%s on Lambda (dev login %s, messaging %s, locale %s, time zone %s, forwarded hops %d)",
			name, devLogin, cfg.Messaging, cfg.Country.Locale, cfg.Country.TimeZone, cfg.ForwardedHops)
		lambda.Start(Lambda(handler, cfg.ForwardedHops))
		return
	}
	srv := &http.Server{Addr: cfg.Addr, Handler: clientAddr(handler, cfg.ForwardedHops)}
	if jobsTimer && cfg.JobsTimer {
		timer = "every " + jobs.TimerInterval.String()
		// Testers' clock offset applies only where messages are captured (local and QA).
		go httpapi.NewJobsRunner(deps).Every(context.Background(), jobs.TimerInterval, cfg.Messaging == config.MessagingCapture)
	}
	log.Printf("%s listening on %s (dev login %s, messaging %s, jobs timer %s, locale %s, time zone %s)",
		name, cfg.Addr, devLogin, cfg.Messaging, timer, cfg.Country.Locale, cfg.Country.TimeZone)
	log.Fatal(srv.ListenAndServe())
}

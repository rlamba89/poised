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
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

// Main runs the router built by router until the server fails. name is for the log.
func Main(name string, router func(httpapi.Deps) http.Handler) {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatalf("%s: %v", name, err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("%s: db: %v", name, err)
	}
	defer pool.Close()

	srv := &http.Server{Addr: cfg.Addr, Handler: router(httpapi.Deps{Pool: pool, Config: cfg, Cognito: cognito.Local{}})}
	devLogin := "off"
	if cfg.DevLogin {
		devLogin = "on"
	}
	log.Printf("%s listening on %s (dev login %s, locale %s, time zone %s)",
		name, cfg.Addr, devLogin, cfg.Country.Locale, cfg.Country.TimeZone)
	log.Fatal(srv.ListenAndServe())
}

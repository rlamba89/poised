// Command api is the Lifebox authoring API: config, database pool, router, listen.
package main

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	linkKey, err := base64.StdEncoding.DecodeString(mustEnv("LINK_KEY"))
	if err != nil {
		log.Fatalf("LINK_KEY is not base64: %v", err)
	}
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	cfg := httpapi.Config{
		SessionMaxAge: envDuration("SESSION_MAX_AGE"),
		SessionIdle:   envDuration("SESSION_IDLE"),
		DevLogin:      os.Getenv("DEV_LOGIN") == "true",
		SecureCookies: os.Getenv("COOKIE_SECURE") != "false", // Secure unless turned off for plain-HTTP local dev
		AppOrigin:     os.Getenv("APP_ORIGIN"),
		LinkKey:       linkKey,
	}
	if poolID := os.Getenv("COGNITO_POOL_ID"); poolID != "" {
		withCognito(&cfg, poolID)
	}
	if cfg.DevLogin {
		log.Print("dev login is ON (DEV_LOGIN=true): local and QA only")
	}
	handler, err := httpapi.NewRouter(pool, cfg)
	if err != nil {
		log.Fatalf("router: %v", err)
	}
	srv := &http.Server{Addr: addr, Handler: handler}
	log.Printf("api listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

// withCognito turns on staff sign-in with the Cognito user pool (A-12). The pool's region is
// the part of its id before "_". Invites call Cognito with the default AWS credentials: locally
// AWS_PROFILE from .env (run aws sso login when it expires), on Lambda the function's role.
func withCognito(cfg *httpapi.Config, poolID string) {
	region, _, _ := strings.Cut(poolID, "_")
	domain := strings.TrimRight(mustEnv("COGNITO_DOMAIN"), "/") // https://<prefix>.auth.<region>.amazoncognito.com
	clientID := mustEnv("COGNITO_CLIENT_ID")
	redirect := mustEnv("COGNITO_REDIRECT_URI")
	issuer := "https://cognito-idp." + region + ".amazonaws.com/" + poolID
	cfg.SignIn = cognito.New(domain, clientID, redirect, issuer, &http.Client{Timeout: 10 * time.Second})
	cfg.Login = &httpapi.CognitoLogin{
		AuthorizeURL: domain + "/oauth2/authorize", LogoutURL: domain + "/logout",
		ClientID: clientID, RedirectURI: redirect, LogoutRedirectURI: mustEnv("COGNITO_LOGOUT_URI"),
	}
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		log.Printf("cognito: invites are off: %v", err)
		return
	}
	cfg.Directory = cognito.NewDirectory(awsCfg, poolID)
	log.Printf("cognito sign-in is ON (pool %s)", poolID)
}

// envDuration reads an optional duration such as "8h" or "30m"; unset means zero (the default).
func envDuration(name string) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return 0
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		log.Fatalf("%s=%q is not a positive duration such as 30m", name, v)
	}
	return d
}

func mustEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is not set (copy .env.example to .env)", name)
	}
	return v
}

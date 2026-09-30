// Command api is the Lifebox authoring API: config, database pool, router, listen.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	secret := []byte(mustEnv("TOKEN_SECRET"))
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	srv := &http.Server{Addr: addr, Handler: httpapi.NewRouter(pool, secret)}
	log.Printf("api listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func mustEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is not set (copy .env.example to .env)", name)
	}
	return v
}

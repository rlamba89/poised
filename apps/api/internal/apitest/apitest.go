//go:build integration

// Package apitest runs API → database integration tests: a real HTTP request goes through
// the real router into a real, migrated Postgres database, one database per test
// (docs/plans/f1-test-foundation.md Step 1). Run them with `make test-integration`.
//
// A test package opts in with
//
//	func TestMain(m *testing.M) { os.Exit(apitest.Main(m)) }
//
// and each test calls apitest.New(t).
package apitest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

const (
	templateDB = "sj_test_template"
	// lockKey is a Postgres advisory lock held while a test package runs. Without it, two
	// packages run in parallel by `go test ./...` would drop each other's template.
	lockKey = 7_020_261_008
)

var (
	admin  *pgxpool.Pool   // TEST_DATABASE_URL: creates and drops the per-test databases
	config *pgxpool.Config // the same server, copied for each test database
	module string          // apps/api, where db/migrations and internal/apitest/testdata are
	secret = []byte("apitest-secret")
)

// Main prepares the template database, runs the tests and returns the exit code.
func Main(m *testing.M) int {
	lock, err := setup(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "apitest: %v\n", err)
		return 1
	}
	defer lock.Close(context.Background()) // releases the advisory lock
	defer admin.Close()
	return m.Run()
}

// setup takes the lock, removes test databases left by killed runs, and (re)creates the
// template with every goose migration. It returns the connection holding the lock.
func setup(ctx context.Context) (*pgx.Conn, error) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		return nil, fmt.Errorf("TEST_DATABASE_URL is not set. Run `make test-integration`, which sets it")
	}
	var err error
	if module, err = moduleDir(); err != nil {
		return nil, err
	}
	if config, err = pgxpool.ParseConfig(url); err != nil {
		return nil, fmt.Errorf("TEST_DATABASE_URL is not a valid Postgres URL: %v", err)
	}
	server := fmt.Sprintf("%s:%d", config.ConnConfig.Host, config.ConnConfig.Port)
	lock, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("can't connect to Postgres at %s: %v\nIs it running? Try `docker compose up -d db`", server, err)
	}
	if _, err := lock.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		lock.Close(ctx)
		return nil, fmt.Errorf("lock: %v", err)
	}
	if err := recreateTemplate(ctx, lock); err != nil {
		lock.Close(ctx)
		return nil, err
	}
	config.MaxConns = 4
	if admin, err = pgxpool.NewWithConfig(ctx, config); err != nil {
		lock.Close(ctx)
		return nil, fmt.Errorf("connect to %s: %v", server, err)
	}
	return lock, nil
}

func recreateTemplate(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx, `SELECT datname FROM pg_database WHERE datname ~ '^sj_test_[0-9a-f]{16}$'`)
	if err != nil {
		return fmt.Errorf("list old test databases: %v", err)
	}
	leftovers, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("list old test databases: %v", err)
	}
	for _, name := range append(leftovers, templateDB) {
		if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			return fmt.Errorf("drop %s: %v", name, err)
		}
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+templateDB); err != nil {
		return fmt.Errorf("create %s: %v", templateDB, err)
	}

	cfg := conn.Config().Copy()
	cfg.Database = templateDB
	sqlDB := stdlib.OpenDB(*cfg) // goose needs database/sql
	defer sqlDB.Close()          // a template must have no open connections when it's copied
	sqlDB.SetMaxOpenConns(1)
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to %s: %v", templateDB, err)
	}
	dir := filepath.Join(module, "db", "migrations")
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, os.DirFS(dir))
	if err != nil {
		return fmt.Errorf("read the migrations in %s: %v", dir, err)
	}
	if _, err := p.Up(ctx); err != nil {
		var partial *goose.PartialError
		if errors.As(err, &partial) {
			err = fmt.Errorf("db/migrations/%s: %v", partial.Failed.Source.Path, partial.Err)
		}
		return fmt.Errorf("a migration failed, so no integration test can run.\n%v", err)
	}
	return nil
}

// moduleDir is apps/api: the nearest folder holding go.mod, above the test's working
// directory (its package folder). It works under -trimpath, unlike the source file's path.
func moduleDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for start := dir; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if dir == filepath.Dir(dir) {
			return "", fmt.Errorf("no go.mod in %s or above it", start)
		}
	}
}

// Client sends requests to a test's own API server. New returns one that isn't signed in;
// LoginAs returns another that is.
type Client struct {
	DB   *pgxpool.Pool // the test's database, for asserting rows with plain SQL
	base string
	http *http.Client
}

// New gives the test its own database, copied from the migrated template, and a real
// router (httpapi.NewRouter) serving it over HTTP. Both are removed when the test ends.
func New(t *testing.T) (*Client, *Fixtures) {
	t.Helper()
	if admin == nil {
		t.Fatal("apitest: the package's TestMain must call apitest.Main")
	}
	ctx := context.Background()
	name := "sj_test_" + randomHex(8)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name+" TEMPLATE "+templateDB); err != nil {
		t.Fatalf("apitest: create %s: %v", name, err)
	}
	cfg := config.Copy()
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("apitest: connect to %s: %v", name, err)
	}
	srv := httptest.NewServer(httpapi.NewRouter(pool, secret))
	t.Cleanup(func() {
		srv.Close()
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("apitest: drop %s: %v", name, err)
		}
	})
	return &Client{DB: pool, base: srv.URL, http: srv.Client()}, &Fixtures{db: pool}
}

// LoginAs signs in through the real POST /api/dev/login and returns a client that keeps
// the session cookie. c itself stays as it was.
func (c *Client) LoginAs(t *testing.T, u User) *Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	signed := &Client{DB: c.DB, base: c.base, http: &http.Client{Transport: c.http.Transport, Jar: jar}}
	if r := signed.Do("POST", "/api/dev/login", map[string]any{"userId": u.ID}); r.Code != http.StatusNoContent {
		t.Fatalf("apitest: sign in as %s: %s", u.Name, r)
	}
	return signed
}

// Do sends a request and returns the response. body is sent as JSON; a string is sent as
// it is, so tests can send broken JSON. A request that gets no response (the server
// dropped the connection) comes back with Code 0 and the error in Body.
func (c *Client) Do(method, path string, body any) *Resp {
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rd = strings.NewReader(b)
	default:
		j, err := json.Marshal(b)
		if err != nil {
			return &Resp{Body: []byte("apitest: encode body: " + err.Error())}
		}
		rd = bytes.NewReader(j)
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	if err != nil {
		return &Resp{Body: []byte("apitest: " + err.Error())}
	}
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return &Resp{Body: []byte("no response: " + err.Error())}
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return &Resp{Code: res.StatusCode, Body: []byte("apitest: read body: " + err.Error())}
	}
	return &Resp{Code: res.StatusCode, Body: b}
}

// Resp is a response's status code and raw body.
type Resp struct {
	Code int
	Body []byte
}

func (r *Resp) String() string { return fmt.Sprintf("%d %s", r.Code, bytes.TrimSpace(r.Body)) }

// Want fails the test now unless the status code is code.
func (r *Resp) Want(t *testing.T, code int) {
	t.Helper()
	if r.Code != code {
		t.Fatalf("got %s, want %d", r, code)
	}
}

// Decode reads the JSON body into v, failing the test if it can't.
func (r *Resp) Decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r, err)
	}
}

// Message is the plain-language {"error": …} message, or "".
func (r *Resp) Message() string {
	var e struct{ Error string }
	_ = json.Unmarshal(r.Body, &e)
	return e.Error
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

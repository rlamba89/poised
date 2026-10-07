//go:build integration

// Package apitest runs the real router against a real, migrated Postgres, for the
// API → database integration tests (docs/plans/f1-test-foundation.md, Step 1).
// Main migrates a template database once; New gives each test its own copy of it.
package apitest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/rlamba89/poised/apps/api/internal/httpapi"
)

// template is the migrated database every test database is copied from.
const template = "sj_test_template"

var (
	admin   *pgxpool.Pool // TEST_DATABASE_URL: a maintenance database, used to create and drop databases
	baseCfg *pgx.ConnConfig
	// createMu serialises CREATE DATABASE: Postgres refuses two copies of one template at once.
	createMu sync.Mutex
)

// Main is called from a package's TestMain. It (re)creates the template database,
// migrates it, runs the tests and exits.
func Main(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		log.Fatal("TEST_DATABASE_URL is not set (run the tests with make test-integration)")
	}
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		log.Fatalf("TEST_DATABASE_URL: %v", err)
	}
	baseCfg = cfg
	if admin, err = pgxpool.New(ctx, url); err != nil {
		log.Fatalf("connect to TEST_DATABASE_URL: %v", err)
	}
	if err := buildTemplate(ctx); err != nil {
		log.Fatalf("build the template database: %v", err)
	}
	code := m.Run()
	admin.Close()
	os.Exit(code)
}

// buildTemplate drops and recreates the template, then runs every goose migration into it.
func buildTemplate(ctx context.Context) error {
	if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+template+" WITH (FORCE)"); err != nil {
		return fmt.Errorf("drop: %w", err)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+template); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	db := stdlib.OpenDB(*dbConfig(template))
	defer db.Close()
	p, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(migrationsDir()))
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// migrationsDir is apps/api/db/migrations, found from this file so any working directory works.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations")
}

// dbConfig is TEST_DATABASE_URL pointed at another database.
func dbConfig(name string) *pgx.ConnConfig {
	cfg := baseCfg.Copy()
	cfg.Database = name
	return cfg
}

// Client is one test's API: the real router on a local HTTP server, a cookie jar, and the database.
type Client struct {
	t    *testing.T
	url  string
	http *http.Client
	// DB is this test's own database, for arranging rows and asserting what was written.
	DB *pgxpool.Pool
}

// New is NewWith the local development config: the dev login on, cookies over plain HTTP.
func New(t *testing.T) *Client {
	t.Helper()
	return NewWith(t, httpapi.Config{DevLogin: true})
}

// NewWith creates a database for this test from the template, starts the real router on it
// with cfg, and removes both when the test ends.
func NewWith(t *testing.T, cfg httpapi.Config) *Client {
	t.Helper()
	if cfg.LinkKey == nil {
		cfg.LinkKey = bytes.Repeat([]byte{7}, 32) // a test key: the sealed links only live in this test's database
	}
	ctx := context.Background()
	name := "t_" + randomHex(t)

	createMu.Lock()
	_, err := admin.Exec(ctx, "CREATE DATABASE "+name+" TEMPLATE "+template)
	createMu.Unlock()
	if err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})

	poolCfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	poolCfg.ConnConfig = dbConfig(name)
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	handler, err := httpapi.NewRouter(pool, cfg)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{t: t, url: srv.URL, http: &http.Client{Jar: jar}, DB: pool}
}

// Resp is a response, read in full. Checking the status is the test's job.
type Resp struct {
	Status int
	Header http.Header
	Body   []byte
}

// Decode unmarshals the JSON body into v, failing the test when it can't.
func (r *Resp) Decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

// Do sends a request with body (if not nil) encoded as JSON, keeping cookies between calls.
func (c *Client) Do(method, path string, body any) *Resp {
	c.t.Helper()
	return c.Send(method, path, body, nil)
}

// Send is Do with extra request headers, e.g. the Sec-Fetch-Site and Origin a browser adds.
func (c *Client) Send(method, path string, body any, header http.Header) *Resp {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("encode body: %v", err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.url+path, rd)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, vs := range header {
		req.Header[k] = vs
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("read %s %s: %v", method, path, err)
	}
	return &Resp{Status: res.StatusCode, Header: res.Header, Body: b}
}

// LoginAs signs in as user through the real dev login and keeps the cookie.
func (c *Client) LoginAs(user uuid.UUID) {
	c.t.Helper()
	if r := c.Do(http.MethodPost, "/api/dev/login", map[string]any{"userId": user}); r.Status != http.StatusNoContent {
		c.t.Fatalf("dev login: status = %d: %s", r.Status, r.Body)
	}
}

// Org inserts an active trust and returns its id.
func (c *Client) Org(name string) uuid.UUID {
	c.t.Helper()
	var id uuid.UUID
	if err := c.DB.QueryRow(context.Background(),
		"INSERT INTO orgs (name, code) VALUES ($1, $2) RETURNING id", name, "T"+randomHex(c.t)).Scan(&id); err != nil {
		c.t.Fatalf("insert org: %v", err)
	}
	return id
}

// HospitalIn inserts a hospital in org and returns its id.
func (c *Client) HospitalIn(org uuid.UUID, name string) uuid.UUID {
	c.t.Helper()
	var id uuid.UUID
	if err := c.DB.QueryRow(context.Background(),
		"INSERT INTO hospitals (org_id, name) VALUES ($1, $2) RETURNING id", org, name).Scan(&id); err != nil {
		c.t.Fatalf("insert hospital: %v", err)
	}
	return id
}

// Hospital inserts a hospital in a new trust of its own and returns its id.
func (c *Client) Hospital(name string) uuid.UUID {
	c.t.Helper()
	return c.HospitalIn(c.Org(name+" Trust"), name)
}

// OrgOf returns the trust a hospital belongs to.
func (c *Client) OrgOf(hospital uuid.UUID) uuid.UUID {
	c.t.Helper()
	var org uuid.UUID
	if err := c.DB.QueryRow(context.Background(), "SELECT org_id FROM hospitals WHERE id = $1", hospital).Scan(&org); err != nil {
		c.t.Fatalf("hospital's org: %v", err)
	}
	return org
}

// User inserts a user with role in one hospital and returns its id.
func (c *Client) User(hospital uuid.UUID, role string) uuid.UUID {
	c.t.Helper()
	id := c.newUser()
	c.Member(id, c.OrgOf(hospital), &hospital, role)
	return id
}

// TrustUser inserts a user with role across the whole of org and returns its id.
func (c *Client) TrustUser(org uuid.UUID, role string) uuid.UUID {
	c.t.Helper()
	id := c.newUser()
	c.Member(id, org, nil, role)
	return id
}

// Member gives user role in org: in one hospital, or the whole trust when hospital is nil.
func (c *Client) Member(user, org uuid.UUID, hospital *uuid.UUID, role string) {
	c.t.Helper()
	if _, err := c.DB.Exec(context.Background(),
		"INSERT INTO memberships (user_id, org_id, hospital_id, role) VALUES ($1, $2, $3, $4)",
		user, org, hospital, role); err != nil {
		c.t.Fatalf("insert membership %q: %v", role, err)
	}
}

func (c *Client) newUser() uuid.UUID {
	c.t.Helper()
	var id uuid.UUID
	email := "user-" + randomHex(c.t) + "@test.example"
	if err := c.DB.QueryRow(context.Background(),
		"INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id", "Test User", email).Scan(&id); err != nil {
		c.t.Fatalf("insert user: %v", err)
	}
	return id
}

// NewBrowser is another browser on the same server and database: its own cookies, no sign-in.
func (c *Client) NewBrowser() *Client {
	c.t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		c.t.Fatal(err)
	}
	return &Client{t: c.t, url: c.url, http: &http.Client{Jar: jar}, DB: c.DB}
}

// PublishedHQ inserts a published questionnaire in hospital, by author, with a patient Question
// Set (content as given) and a clinician one, and returns the version id.
func (c *Client) PublishedHQ(hospital, author uuid.UUID, patientContent string) uuid.UUID {
	c.t.Helper()
	ctx := context.Background()
	var q, v uuid.UUID
	if err := c.DB.QueryRow(ctx, `INSERT INTO questionnaires (hospital_id, name, description, created_by)
		VALUES ($1, $2, 'Test', $3) RETURNING id`, hospital, "HQ "+randomHex(c.t), author).Scan(&q); err != nil {
		c.t.Fatalf("insert questionnaire: %v", err)
	}
	if err := c.DB.QueryRow(ctx, `INSERT INTO questionnaire_versions (questionnaire_id, version_no, status, updated_by)
		VALUES ($1, 1, 'published', $2) RETURNING id`, q, author).Scan(&v); err != nil {
		c.t.Fatalf("insert version: %v", err)
	}
	if _, err := c.DB.Exec(ctx, `INSERT INTO chapters (version_id, position, name, audience, content, updated_by) VALUES
		($1, 1, 'About you', 'patient', $2, $3), ($1, 2, 'Assessment', 'clinician', '{}', $3)`, v, patientContent, author); err != nil {
		c.t.Fatalf("insert chapters: %v", err)
	}
	return v
}

// Patient inserts a made-up patient in hospital with the given date of birth (YYYY-MM-DD).
func (c *Client) Patient(hospital uuid.UUID, dateOfBirth string) uuid.UUID {
	c.t.Helper()
	var id uuid.UUID
	if err := c.DB.QueryRow(context.Background(), `INSERT INTO patients (hospital_id, first_name, last_name, date_of_birth, sex)
		VALUES ($1, 'Jo', 'Test', $2, 'unknown') RETURNING id`, hospital, dateOfBirth).Scan(&id); err != nil {
		c.t.Fatalf("insert patient: %v", err)
	}
	return id
}

// HospitalPath is the API path prefix of one hospital: /api/o/{org}/h/{hospital}.
func (c *Client) HospitalPath(hospital uuid.UUID) string {
	c.t.Helper()
	return "/api/o/" + c.OrgOf(hospital).String() + "/h/" + hospital.String()
}

func randomHex(t *testing.T) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

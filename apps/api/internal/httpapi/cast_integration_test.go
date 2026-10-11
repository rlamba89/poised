//go:build integration

package httpapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// cast is the usual set-up for a route test: hospital A with an author who can also publish
// and a viewer, and hospital B with its own author. Each test gets its own database.
type cast struct {
	fx   *apitest.Fixtures
	db   *pgxpool.Pool
	a, b uuid.UUID // hospitals A and B

	alex, bea apitest.User // author + publisher in A; author in B

	anon   *apitest.Client // not signed in
	author *apitest.Client // Alex
	viewer *apitest.Client // a viewer in A
	other  *apitest.Client // Bea, who isn't a member of A
}

func newCast(t *testing.T) *cast {
	t.Helper()
	c, fx := apitest.New(t)
	w := &cast{fx: fx, db: c.DB, anon: c, a: fx.Hospital(t), b: fx.Hospital(t)}
	w.alex = fx.User(t, w.a, "author", "publisher")
	w.bea = fx.User(t, w.b, "author")
	w.author = c.LoginAs(t, w.alex)
	w.viewer = c.LoginAs(t, fx.User(t, w.a, "viewer"))
	w.other = c.LoginAs(t, w.bea)
	return w
}

// at is a path under /api/h/{hospital}.
func at(hospital uuid.UUID, path string) string {
	return "/api/h/" + hospital.String() + path
}

// count runs a `SELECT count(*) …` on the test's database.
func count(t *testing.T, db *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(t.Context(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%v\n%s", err, sql)
	}
	return n
}

// refused checks a refusal: the status code, and a plain-language message (DSG-04).
func refused(t *testing.T, r *apitest.Resp, code int) {
	t.Helper()
	r.Want(t, code)
	if r.Message() == "" {
		t.Errorf("refusal %s has no error message", r)
	}
}

// meanwhile makes a race happen in a fixed order, to test what a request does when someone else
// changes its rows between its read and its write:
//  1. it runs sql in a transaction that isn't committed yet, so the rows it writes stay locked;
//  2. it sends the request, which reads the old rows and then waits for the lock when it writes;
//  3. once Postgres shows the request waiting, it commits, and returns the request's response.
func meanwhile(t *testing.T, db *pgxpool.Pool, sql string, args []any, request func() *apitest.Resp) *apitest.Resp {
	t.Helper()
	tx, err := db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background()) // releases the lock if the test stops early
	if _, err := tx.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("%v\n%s", err, sql)
	}
	done := make(chan *apitest.Resp, 1)
	go func() { done <- request() }()
	const waiting = `SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`
	for deadline := time.Now().Add(5 * time.Second); count(t, db, waiting) == 0; time.Sleep(5 * time.Millisecond) {
		select {
		case r := <-done:
			t.Fatalf("the request finished without waiting for the lock: %s", r)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("the request never waited for the lock")
		}
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	return <-done
}

//go:build integration

package httpapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// cast is the usual set-up for a route test: hospital A (in trust A) with a trust-wide super
// clinician and a clinician, and hospital B (in trust B) with its own trust-wide super clinician.
// Changing the trust's content needs a whole-trust role (S08 answers, 5). Each test gets its own
// database. "Another trust" is Bea: at A's URL (not a member: 404) and at B's URL with A's ids
// (scoped to B: 404).
type cast struct {
	fx   *apitest.Fixtures
	db   *pgxpool.Pool
	a, b apitest.Hospital // hospitals A and B, each in a trust of its own

	alex, bea apitest.User // super clinicians of the whole of trust A and of trust B

	anon   *apitest.Client // not signed in
	author *apitest.Client // Alex, who can author and publish
	clin   *apitest.Client // a clinician at A: the wrong role for authoring
	other  *apitest.Client // Bea, who isn't a member of trust A
}

func newCast(t *testing.T) *cast {
	t.Helper()
	c, fx := apitest.New(t)
	w := &cast{fx: fx, db: c.DB, anon: c, a: fx.Hospital(t), b: fx.Hospital(t)}
	w.alex = fx.TrustUser(t, w.a.Org, "super_clinician")
	w.bea = fx.TrustUser(t, w.b.Org, "super_clinician")
	w.author = c.LoginAs(t, w.alex)
	w.clin = c.LoginAs(t, fx.User(t, w.a, "clinician"))
	w.other = c.LoginAs(t, w.bea)
	return w
}

// at is a path under /api/o/{trust}/h/{hospital}.
func at(h apitest.Hospital, path string) string {
	return h.Path(path)
}

// badIDs are the hospital's path p with a trust or hospital id that isn't a uuid.
func badIDs(h apitest.Hospital, p string) []string {
	return []string{"/api/o/" + h.Org.String() + "/h/abc" + p, "/api/o/abc/h/" + h.ID.String() + p}
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

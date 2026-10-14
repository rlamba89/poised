// Command qa-users gives the QA environment's demo staff (the seed's accounts) a sign-in in the
// staff user pool with a known password and authenticator secret (S12), so testers and the
// end-to-end journeys can sign in through Cognito's managed login, MFA included. QA only: it
// refuses any other stage.
//
//	qa-users setup            make qa-users ENV=uk-qa (the pipeline runs it after each seed)
//	qa-users code <who>       the authenticator's current 6-digit code (make qa-code WHO=…)
//	qa-users password <who>   the password, for the clipboard (make qa-password WHO=…)
//
// <who> is a name ("Cara Clinician") or an email. The passwords and secrets are kept in one SSM
// SecureString (QA_USERS_PARAM, e.g. /sj/qa/test-users), never in the repo, a log or the output
// of setup. Running setup again keeps them: an account the pool already has, with its secret
// stored, only gets its stored password back; one without a stored secret is made again.
//
// The environment (infra/scripts/qa-users.sh sets it): STAGE, STAFF_POOL_ID,
// STAFF_POOL_CLIENT_ID, QA_USERS_PARAM, and DATABASE_URL for setup.
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
)

// testUser is one demo account's sign-in, as stored in SSM and read by the journeys (e2e/).
type testUser struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	TOTPSecret string `json:"totpSecret"`
}

// person is a demo account in the database.
type person struct{ Name, Email string }

// pool is the part of the staff user pool setup needs (awsPool on AWS; a fake in tests).
type pool interface {
	// Find reports whether the pool has the user, and whether their authenticator is set up.
	Find(ctx context.Context, email string) (found, enrolled bool, err error)
	Create(ctx context.Context, email string) error // with no invitation email
	Delete(ctx context.Context, email string) error
	SetPassword(ctx context.Context, email, password string) error // permanent
	// Enroll signs in with the password and sets up the authenticator (MFA_SETUP), returning its secret.
	Enroll(ctx context.Context, email, password string) (secret string, err error)
}

// store keeps the test users' sign-ins (an SSM SecureString on AWS).
type store interface {
	Load(ctx context.Context) ([]testUser, error) // none yet: nil, nil
	Save(ctx context.Context, users []testUser) error
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
	}
	if stage := os.Getenv("STAGE"); stage != "qa" {
		log.Fatalf("qa-users: test users are for QA only, and this is stage %q", stage)
	}
	ctx := context.Background()
	st := &ssmStore{name: need("QA_USERS_PARAM")}
	switch {
	case os.Args[1] == "setup" && len(os.Args) == 2:
		people, err := demoStaff(ctx, need("DATABASE_URL"))
		if err != nil {
			log.Fatalf("qa-users: %v", err)
		}
		p := &awsPool{poolID: need("STAFF_POOL_ID"), clientID: need("STAFF_POOL_CLIENT_ID")}
		users, err := setup(ctx, people, p, st, newPassword)
		if err != nil {
			log.Fatalf("qa-users: %v", err)
		}
		for _, u := range users {
			fmt.Printf("  %s (%s)\n", u.Name, u.Email)
		}
		fmt.Printf("%d test users can sign in; their passwords and secrets are in SSM %s.\n", len(users), st.name)
	case (os.Args[1] == "code" || os.Args[1] == "password") && len(os.Args) == 3:
		users, err := st.Load(ctx)
		if err != nil {
			log.Fatalf("qa-users: %v", err)
		}
		u, err := find(users, os.Args[2])
		if err != nil {
			log.Fatalf("qa-users: %v", err)
		}
		if os.Args[1] == "password" {
			fmt.Print(u.Password)
			return
		}
		code, err := cognito.TOTP(u.TOTPSecret, time.Now())
		if err != nil {
			log.Fatalf("qa-users: %v", err)
		}
		fmt.Printf("%s (good for %d more seconds)\n", code, 30-time.Now().Unix()%30)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: qa-users setup | qa-users code <name or email> | qa-users password <name or email>")
	os.Exit(2)
}

func need(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("qa-users: %s is not set", name)
	}
	return v
}

// demoStaff are the seed's staff accounts (cmd/seed), known by their fixed ids, so accounts made
// by testers or the journeys (random ids) never get a test sign-in.
func demoStaff(ctx context.Context, url string) ([]person, error) {
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return seededStaff(ctx, db)
}

func seededStaff(ctx context.Context, db *pgxpool.Pool) ([]person, error) {
	rows, err := db.Query(ctx, `SELECT name, lower(email) FROM users
		WHERE id::text LIKE '00000000-0000-4000-8000-%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	people, err := pgx.CollectRows(rows, pgx.RowToStructByPos[person])
	if err == nil && len(people) == 0 {
		err = errors.New("the database has no seeded staff: run the seed first (make aws-db)")
	}
	return people, err
}

// setup gives each person a working test sign-in and saves them all.
func setup(ctx context.Context, people []person, p pool, st store, password func() string) ([]testUser, error) {
	stored, err := st.Load(ctx)
	if err != nil {
		return nil, err
	}
	byEmail := map[string]testUser{}
	for _, u := range stored {
		byEmail[u.Email] = u
	}
	var out []testUser
	for _, who := range people {
		u, err := ensure(ctx, who, byEmail[who.Email], p, password)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", who.Email, err)
		}
		out = append(out, u)
	}
	return out, st.Save(ctx, out)
}

func ensure(ctx context.Context, who person, stored testUser, p pool, password func() string) (testUser, error) {
	found, enrolled, err := p.Find(ctx, who.Email)
	if err != nil {
		return testUser{}, err
	}
	if found && enrolled && stored.TOTPSecret != "" {
		// Keep the secret; put the stored password back in case someone changed it.
		stored.Name = who.Name
		return stored, p.SetPassword(ctx, who.Email, stored.Password)
	}
	if found { // no secret we know: start again
		if err := p.Delete(ctx, who.Email); err != nil {
			return testUser{}, err
		}
	}
	u := testUser{Name: who.Name, Email: who.Email, Password: password()}
	if err := p.Create(ctx, who.Email); err != nil {
		return testUser{}, err
	}
	if err := p.SetPassword(ctx, who.Email, u.Password); err != nil {
		return testUser{}, err
	}
	if u.TOTPSecret, err = p.Enroll(ctx, who.Email, u.Password); err != nil {
		return testUser{}, err
	}
	return u, nil
}

func find(users []testUser, who string) (testUser, error) {
	for _, u := range users {
		if strings.EqualFold(u.Email, who) || strings.EqualFold(u.Name, who) {
			return u, nil
		}
	}
	var names []string
	for _, u := range users {
		names = append(names, u.Name)
	}
	return testUser{}, fmt.Errorf("no test user %q; there are: %s", who, strings.Join(names, ", "))
}

// newPassword is 24 random letters and digits, plus one of each kind the pool's policy asks for.
func newPassword() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	b := []byte("Qa7!")
	for range 24 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(err)
		}
		b = append(b, chars[n.Int64()])
	}
	return string(b)
}

// encode and decode are the stored form: a JSON list.
func encode(users []testUser) (string, error) {
	b, err := json.Marshal(users)
	return string(b), err
}

func decode(s string) ([]testUser, error) {
	var users []testUser
	err := json.Unmarshal([]byte(s), &users)
	return users, err
}

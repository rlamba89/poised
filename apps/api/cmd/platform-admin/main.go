// Command platform-admin adds a superadmin (A-21): only the owner, created by hand once in each
// environment and country, never from a screen.
//
//	platform-admin add <email> [name]     (or: make platform-admin EMAIL=… NAME=…)
//
// It creates the staff account (in Cognito too, for a new email) and the platform_admins row.
// Running it again changes nothing. Locally Cognito is the stand-in that only logs (cognito.Local);
// with STAFF_POOL_ID set it's the real staff pool, whose invitation email (a temporary
// password) goes to the new admin: `make aws-platform-admin ENV=… EMAIL=…` runs it against an AWS
// environment's database and pool (S12).
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/db"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "add" {
		fmt.Fprintln(os.Stderr, "usage: platform-admin add <email> [name]")
		os.Exit(2)
	}
	name := ""
	if len(os.Args) > 3 {
		name = strings.Join(os.Args[3:], " ")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is not set (copy .env.example to .env)")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	var userPool cognito.Client = cognito.Local{}
	if id := os.Getenv("STAFF_POOL_ID"); id != "" {
		userPool = &cognito.AWS{PoolID: id}
	}
	created, err := add(ctx, pool, userPool, os.Args[2], name)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		fmt.Println("Created the account and made it a platform admin.")
	} else {
		fmt.Println("The account exists; it is a platform admin.")
	}
}

// add makes email a platform admin, creating the account first if there's none (created says
// so). The Cognito user is created last, inside the transaction, so a failure leaves nothing.
func add(ctx context.Context, pool *pgxpool.Pool, cog cognito.Client, email, name string) (created bool, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	a, perr := mail.ParseAddress(email)
	if perr != nil || a.Address != email || !strings.Contains(email[strings.LastIndex(email, "@"):], ".") {
		return false, fmt.Errorf("%q is not an email address", email)
	}
	if name = strings.TrimSpace(name); name == "" {
		name = email
	}
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		user, err := q.GetUserByEmail(ctx, email)
		if errors.Is(err, pgx.ErrNoRows) {
			ids, err := q.CreateUserIfNew(ctx, db.CreateUserIfNewParams{Name: name, Email: email})
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				return fmt.Errorf("%s was created by someone else just now; run it again", email)
			}
			user.ID, created = ids[0], true
		} else if err != nil {
			return err
		}
		if err := q.AddPlatformAdmin(ctx, user.ID); err != nil {
			return err
		}
		if created {
			return cog.CreateUser(ctx, email)
		}
		return nil
	})
	return created && err == nil, err
}

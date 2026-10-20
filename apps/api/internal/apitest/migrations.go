//go:build integration

package apitest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrations is an empty database of the test's own, migrated by goose only as far as the test
// says, for testing a migration against rows in the shape before it (New's databases are always
// at the latest migration). It's dropped when the test ends.
type Migrations struct {
	DB *pgxpool.Pool // for inserting and checking rows with plain SQL
	p  *goose.Provider
}

// NewMigrations makes the database and migrates it up to and including migration version `to`.
func NewMigrations(t *testing.T, to int64) *Migrations {
	t.Helper()
	if admin == nil {
		t.Fatal("apitest: the package's TestMain must call apitest.Main")
	}
	ctx := context.Background()
	name := "sj_test_" + randomHex(8)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("apitest: create %s: %v", name, err)
	}
	cfg := dbConfig.Copy()
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("apitest: connect to %s: %v", name, err)
	}
	sqlDB := stdlib.OpenDB(*cfg.ConnConfig.Copy()) // goose needs database/sql
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("apitest: drop %s: %v", name, err)
		}
	})
	m := &Migrations{DB: pool, p: provider(t, sqlDB)}
	m.UpTo(t, to)
	return m
}

func provider(t *testing.T, sqlDB *sql.DB) *goose.Provider {
	t.Helper()
	dir := filepath.Join(module, "db", "migrations")
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, os.DirFS(dir))
	if err != nil {
		t.Fatalf("apitest: read the migrations in %s: %v", dir, err)
	}
	return p
}

// UpTo applies the migrations after the current one, up to and including version v.
func (m *Migrations) UpTo(t *testing.T, v int64) {
	t.Helper()
	if _, err := m.p.UpTo(context.Background(), v); err != nil {
		t.Fatalf("apitest: migrate up to %d: %v", v, migrationErr(err))
	}
}

// Up applies every migration still to run.
func (m *Migrations) Up(t *testing.T) {
	t.Helper()
	if _, err := m.p.Up(context.Background()); err != nil {
		t.Fatalf("apitest: migrate up: %v", migrationErr(err))
	}
}

// DownTo rolls back every migration after version v.
func (m *Migrations) DownTo(t *testing.T, v int64) {
	t.Helper()
	if _, err := m.p.DownTo(context.Background(), v); err != nil {
		t.Fatalf("apitest: migrate down to %d: %v", v, migrationErr(err))
	}
}

func migrationErr(err error) error {
	var partial *goose.PartialError
	if errors.As(err, &partial) {
		return fmt.Errorf("db/migrations/%s: %v", partial.Failed.Source.Path, partial.Err)
	}
	return err
}

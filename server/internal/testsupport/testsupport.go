// Package testsupport provides DB-backed test scaffolding shared by module
// tests. It is a test-only dependency (no production imports).
package testsupport

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"anmo/server/internal/database"
	"anmo/server/internal/shared"
)

// DSN returns the integration-test MySQL DSN or skips the test.
func DSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("ANMO_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("ANMO_TEST_MYSQL_DSN not set; skipping DB test")
	}
	return dsn
}

// findMigrationsDir walks up from the working directory until it finds
// server/migrations (identified by its first migration file).
func findMigrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "migrations")
		if _, err := os.Stat(filepath.Join(candidate, "001_identity.sql")); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("migrations dir not found from working directory")
	return ""
}

// withSchema injects the database name into a DSN of the form
// user:pass@tcp(host:port)/?params (replacing the empty dbname).
func withSchema(dsn, schema string) string {
	i := strings.Index(dsn, "?")
	if i < 0 {
		return strings.TrimRight(dsn, "/") + "/" + schema
	}
	return dsn[:i] + schema + dsn[i:]
}

// NewSchemaDB creates a throwaway database, applies all migrations and returns
// a pool bound to it (database name carried by the DSN so every pooled
// connection lands in the right schema).
func NewSchemaDB(t *testing.T, prefix string) *database.Pool {
	t.Helper()
	rootDSN := DSN(t)
	raw, err := sql.Open("mysql", rootDSN)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	schema := prefix + shared.NewID()
	if _, err := raw.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { raw.Exec("DROP DATABASE `" + schema + "`"); raw.Close() })

	pool, err := database.Open(withSchema(rootDSN, schema))
	if err != nil {
		t.Fatalf("open schema pool: %v", err)
	}
	t.Cleanup(func() { pool.(*database.Pool).DB.Close() })

	migDir := findMigrationsDir(t)
	if _, err := database.Migrate(context.Background(), pool, migDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool.(*database.Pool)
}

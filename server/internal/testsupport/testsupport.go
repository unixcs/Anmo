// Package testsupport provides DB-backed test scaffolding shared by module
// tests. It is a test-only dependency (no production imports).
//
// 2026-09-28 存储层迁移：每个测试获得一个一次性 SQLite 文件库（t.TempDir），
// 迁移后即用即弃——不再依赖外部 MySQL DSN，测试在任何机器上都能跑。
package testsupport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"anmo/server/internal/database"
)

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

// NewSchemaDB creates a throwaway SQLite database in t.TempDir(), applies all
// migrations and returns the pool. Concurrency semantics (WAL + BEGIN
// IMMEDIATE) match production because database.Open applies the same DSN.
func NewSchemaDB(t *testing.T, prefix string) *database.Pool {
	t.Helper()
	path := filepath.Join(t.TempDir(), prefix+".db")

	pool, err := database.Open(path)
	if err != nil {
		t.Fatalf("open sqlite pool: %v", err)
	}
	t.Cleanup(func() { pool.(*database.Pool).DB.Close() })

	migDir := findMigrationsDir(t)
	if _, err := database.Migrate(context.Background(), pool, migDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool.(*database.Pool)
}

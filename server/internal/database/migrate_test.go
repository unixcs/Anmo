package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
)

func randomSchemaName() string {
	return "anmo_migrate_" + strings.ToLower(ulid.MustNew(ulid.Now(), ulid.Monotonic(rand.Reader, 0)).String())
}

// TestMigrateIdempotentAndComplete runs the runner twice against a temp schema
// and asserts idempotency plus the 24-table completeness baseline (AGENTS.md D3).
func TestMigrateIdempotentAndComplete(t *testing.T) {
	dsn := os.Getenv("ANMO_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("ANMO_TEST_MYSQL_DSN not set; skipping DB integration test")
	}
	root0, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	root := &Pool{DB: root0}
	defer root0.Close()

	schema := randomSchemaName()
	if _, err := root.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer root.Exec("DROP DATABASE `" + schema + "`")

	if _, err := root.Exec("USE `" + schema + "`"); err != nil {
		t.Fatalf("use schema: %v", err)
	}
	db := root
	ctx := context.Background()

	migDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := Migrate(ctx, db, migDir); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	applied2, err := Migrate(ctx, db, migDir)
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if len(applied2) != 0 {
		t.Fatalf("second migrate applied %v, want none", applied2)
	}

	want := map[string]bool{
		"identity_user": true, "identity_role": true, "identity_permission": true,
		"member": true, "member_tag": true, "member_tag_rel": true,
		"service_category": true, "service": true,
		"card_template": true, "card_service_rule": true, "member_card": true, "card_transaction": true,
		"appointment": true, "appointment_service": true, "appointment_status_log": true,
		"payment": true, "redemption": true, "redemption_reversal": true,
		"content_page_config": true, "content_banner": true, "content_announcement": true, "content_system_setting": true,
		"ops_operation_log": true, "ops_insight_snapshot": true,
		"sys_sequence": true, // 009: technical counter table
	}
	rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = ?`, schema)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	got := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[strings.ToLower(n)] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("missing table %s", name)
		}
	}
	if len(got) != len(want)+1 { // + schema_migrations
		t.Errorf("table count = %d, want %d (+schema_migrations)", len(got), len(want)+1)
	}
}

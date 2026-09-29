package database

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrateIdempotentAndComplete runs the runner twice against a temp SQLite
// file and asserts idempotency plus the 27-table completeness baseline
// (AGENTS.md D3: 24 业务表 + sys_sequence；+ schema_migrations、新 checklist 对齐).
func TestMigrateIdempotentAndComplete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migrate_test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() {
		if closer, ok := db.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()
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
		"appointment_closure": true, // 011: 闭店日历（D22）
		"payment": true, "redemption": true, "redemption_reversal": true,
		"content_page_config": true, "content_banner": true, "content_announcement": true, "content_system_setting": true,
		"ops_operation_log": true, "ops_insight_snapshot": true,
		"sys_sequence": true, // 009: technical counter table
	}
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
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

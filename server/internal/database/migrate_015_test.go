package database

// migrate_015_test.go — 015（service_tag/service_record 建表 + payment 重建
// member_id → NULL）在有数据的库上按真实升级路径（001-014 + 业务行 → 追加 015）
// 可重复应用，且旧数据无损、唯一约束/生成列/FK/触发器全部保留。D29 验收口径。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrate015RebuildPreservesPayment(t *testing.T) {
	ctx := context.Background()
	realDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	// 阶段一：只应用 001-014（模拟线上旧库）
	oldDir := t.TempDir()
	entries, err := os.ReadDir(realDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasPrefix(name, "015") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(realDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(oldDir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	dbPath := filepath.Join(t.TempDir(), "m015.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.(*Pool).DB.Close()

	if _, err := Migrate(ctx, db, oldDir); err != nil {
		t.Fatalf("migrate 001-014: %v", err)
	}

	// 旧形态业务行：member → appointment → payment（member_id NOT NULL 时代）
	const mid = "01JM0150000000000000000MA"
	const aid = "01JM0150000000000000000AP"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member (id, member_no, name, phone) VALUES (?, 'M202609290001', '老客户', '13800000001')`,
		mid); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO appointment (id, appointment_no, member_id, scheduled_start, scheduled_end)
		 VALUES (?, 'APT202609290001', ?, '2026-09-29 10:00:00', '2026-09-29 11:00:00')`,
		aid, mid); err != nil {
		t.Fatalf("seed appointment: %v", err)
	}
	const payOld = "01JM0150000000000000000P1"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO payment (id, appointment_id, member_id, amount, method, status, remark, idempotency_key)
		 VALUES (?, ?, ?, 12800, 'CARD', 'VALID', '会员卡核销', 'idem-015-old')`,
		payOld, aid, mid); err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	// 旧库直接插 NULL member_id 必须失败（NOT NULL 时代）
	if _, err := db.ExecContext(ctx,
		`INSERT INTO payment (id, member_id, amount, method) VALUES ('01JM0150000000000000000PZ', NULL, 100, 'CASH')`); err == nil {
		t.Fatal("old schema accepted NULL member_id")
	}

	// 阶段二：追加 015 → 重建 payment + 新建 service_tag/service_record
	newDir := t.TempDir()
	b, err := os.ReadFile(filepath.Join(realDir, "015_service_records.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "015_service_records.sql"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	applied, err := Migrate(ctx, db, newDir)
	if err != nil {
		t.Fatalf("migrate 015: %v", err)
	}
	if len(applied) != 1 || applied[0] != "015_service_records.sql" {
		t.Fatalf("applied = %v", applied)
	}

	// 旧 payment 行无损（含 idempotency_key）
	var amount int64
	var method, status, remark, idem string
	var memberID, aptID *string
	if err := db.QueryRowContext(ctx,
		`SELECT member_id, appointment_id, amount, method, status, remark, idempotency_key
		   FROM payment WHERE id = ?`, payOld).
		Scan(&memberID, &aptID, &amount, &method, &status, &remark, &idem); err != nil {
		t.Fatalf("payment row after rebuild: %v", err)
	}
	if memberID == nil || *memberID != mid || aptID == nil || *aptID != aid ||
		amount != 12800 || method != "CARD" || status != "VALID" || remark != "会员卡核销" || idem != "idem-015-old" {
		t.Fatalf("preserved = %v %v %d %s %s %q %q", memberID, aptID, amount, method, status, remark, idem)
	}

	// D29：member_id 现可 NULL（散客不录手机号仅记账）
	if _, err := db.ExecContext(ctx,
		`INSERT INTO payment (id, amount, method) VALUES ('01JM0150000000000000000P2', 5000, 'CASH')`); err != nil {
		t.Fatalf("NULL member_id payment rejected after rebuild: %v", err)
	}

	// valid_lock 唯一仍生效：同一预约第二笔 VALID payment 必须被拒
	if _, err := db.ExecContext(ctx,
		`INSERT INTO payment (id, appointment_id, member_id, amount, method, status)
		 VALUES ('01JM0150000000000000000P3', ?, ?, 100, 'CASH', 'VALID')`, aid, mid); err == nil {
		t.Fatal("uk_payment_valid_lock not enforced after rebuild")
	}
	// VOIDED 行不占用 valid_lock
	if _, err := db.ExecContext(ctx,
		`INSERT INTO payment (id, appointment_id, member_id, amount, method, status)
		 VALUES ('01JM0150000000000000000P4', ?, ?, 100, 'CASH', 'VOIDED')`, aid, mid); err != nil {
		t.Fatalf("VOIDED second payment rejected: %v", err)
	}
	// idempotency 唯一仍生效
	if _, err := db.ExecContext(ctx,
		`INSERT INTO payment (id, member_id, amount, method, idempotency_key)
		 VALUES ('01JM0150000000000000000P5', ?, 100, 'CASH', 'idem-015-old')`, mid); err == nil {
		t.Fatal("uk_payment_idempotency not enforced after rebuild")
	}
	// CHECK 约束仍生效
	if _, err := db.ExecContext(ctx,
		`INSERT INTO payment (id, member_id, amount, method) VALUES ('01JM0150000000000000000P6', ?, 100, 'ALIPAY')`, mid); err == nil {
		t.Fatal("ck_payment_method not enforced after rebuild")
	}
	// 触发器重建成功
	var trig int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = 'trg_payment_updated_at'`).Scan(&trig); err != nil || trig != 1 {
		t.Fatalf("payment trigger missing after rebuild: %d err=%v", trig, err)
	}

	// service_record：FK 强制（孤儿 payment_id 被拒——payment 无被引用关系，改测子表侧）
	if _, err := db.ExecContext(ctx,
		`INSERT INTO service_record (id, payment_id, service_id, service_name)
		 VALUES ('01JM0150000000000000000R1', '01JNOPE0000000000000000XX', '01JSVC0000000000000000NOPE', '肩颈按摩')`); err == nil {
		t.Fatal("orphan service_record.payment_id accepted: FK not enforced")
	}
	// 合法行可写 + CHECK 约束
	if _, err := db.ExecContext(ctx,
		`INSERT INTO service_record (id, member_id, payment_id, service_id, service_name, body_parts, service_method, communicated)
		 VALUES ('01JM0150000000000000000R2', ?, ?, '01JSVC0000000000000000OK01', '肩颈按摩', '["肩颈","腰背"]', '按摩', 1)`,
		mid, payOld); err != nil {
		t.Fatalf("service_record insert: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO service_record (id, payment_id, service_id, service_name, communicated)
		 VALUES ('01JM0150000000000000000R3', ?, '01JSVC0000000000000000OK01', '肩颈按摩', 2)`, payOld); err == nil {
		t.Fatal("ck_rec_communicated not enforced")
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO service_record (id, payment_id, service_id, service_name, status)
		 VALUES ('01JM0150000000000000000R4', ?, '01JSVC0000000000000000OK01', '肩颈按摩', 'DELETED')`, payOld); err == nil {
		t.Fatal("ck_rec_status not enforced")
	}

	// service_tag：组内同名唯一 + 组 CHECK
	if _, err := db.ExecContext(ctx,
		`INSERT INTO service_tag (id, tag_group, name, sort) VALUES ('01JM0150000000000000000T1', 'BODY_PART', '肩颈', 1)`); err != nil {
		t.Fatalf("service_tag insert: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO service_tag (id, tag_group, name, sort) VALUES ('01JM0150000000000000000T2', 'BODY_PART', '肩颈', 2)`); err == nil {
		t.Fatal("uk_tag_group_name not enforced")
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO service_tag (id, tag_group, name, sort) VALUES ('01JM0150000000000000000T3', 'AREA', '肩颈', 3)`); err == nil {
		t.Fatal("ck_tag_group not enforced")
	}

	// 重复应用 015 幂等
	again, err := Migrate(ctx, db, newDir)
	if err != nil || len(again) != 0 {
		t.Fatalf("re-apply 015: %v %v", again, err)
	}
}

package database

// migrate_014_test.go — 014（member 重建：phone 可空 + password_hash）在有数据的
// 库上按真实升级路径（001-013 + 业务行 → 追加 014）可重复应用，且旧数据无损、
// 触发器/索引/FK 子行全部保留。R1 验收口径。

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrate014RebuildPreservesData(t *testing.T) {
	ctx := context.Background()
	realDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	// 阶段一：只应用 001-013（模拟线上旧库）
	oldDir := t.TempDir()
	entries, err := os.ReadDir(realDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasPrefix(name, "014") {
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

	dbPath := filepath.Join(t.TempDir(), "m014.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.(*Pool).DB.Close()

	if _, err := Migrate(ctx, db, oldDir); err != nil {
		t.Fatalf("migrate 001-013: %v", err)
	}

	// 旧形态 member 行：有 phone、有 wx_openid（013），无密码列；FK 子表 member_tag_rel 一行
	const mid = "01JM0140000000000000000XA"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member (id, member_no, name, phone, wx_openid) VALUES (?, 'M202609290001', '老客户', '13999990001', 'o-old-1')`,
		mid); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member_tag_rel (id, member_id, tag_id) VALUES ('01JM0140000000000000000TB', ?, '01J0TAG000000000000000VIP0')`,
		mid); err != nil {
		t.Fatalf("seed tag rel: %v", err)
	}

	// 阶段二：追加 014 → 重建 member 表
	newDir := t.TempDir()
	b, err := os.ReadFile(filepath.Join(realDir, "014_member_password.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "014_member_password.sql"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	applied, err := Migrate(ctx, db, newDir)
	if err != nil {
		t.Fatalf("migrate 014: %v", err)
	}
	if len(applied) != 1 || applied[0] != "014_member_password.sql" {
		t.Fatalf("applied = %v", applied)
	}

	// 旧数据无损：行字段原样保留
	var name, phone, openid string
	var hash sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT name, phone, wx_openid, password_hash FROM member WHERE id = ?`, mid).
		Scan(&name, &phone, &openid, &hash); err != nil {
		t.Fatalf("member row after rebuild: %v", err)
	}
	if name != "老客户" || phone != "13999990001" || openid != "o-old-1" || hash.Valid {
		t.Fatalf("preserved = %q %q %q %v", name, phone, openid, hash)
	}
	// FK 子行保留
	var rel int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM member_tag_rel WHERE member_id = ?`, mid).Scan(&rel); err != nil || rel != 1 {
		t.Fatalf("tag rel = %d err=%v", rel, err)
	}
	// FK 仍生效：重建（DROP+同名 CREATE，非 RENAME）后子表 fk_mtr_member 指向
	// 新 member 表且约束在线——孤儿 member_tag_rel 行必须被拒绝。
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member_tag_rel (id, member_id, tag_id)
		 VALUES ('01JM0140000000000000000TF', '01JNOPE0000000000000000XX', '01J0TAG000000000000000VIP0')`); err == nil {
		t.Fatal("orphan member_tag_rel accepted after rebuild: member FK not enforced")
	}

	// 触发器重建成功：UPDATE 不显式给 updated_at 也能自动刷新。
	// 先把 created_at 压到固定过去值（显式赋值时触发器 WHEN 不成立、不会自触），
	// 再改 name 触发触发器 —— created_at(2000) < updated_at(now) 恒成立，与
	// 时钟精度/负载无关（此前用 sleep 跨秒在满载时偶发同秒失败）。
	var trig int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = 'trg_member_updated_at'`).Scan(&trig); err != nil || trig != 1 {
		t.Fatalf("trigger missing after rebuild: %d err=%v", trig, err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE member SET created_at = '2000-01-01 00:00:00' WHERE id = ?`, mid); err != nil {
		t.Fatalf("pin created_at: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE member SET name = '改名' WHERE id = ?`, mid); err != nil {
		t.Fatalf("update member: %v", err)
	}
	var updated int
	if err := db.QueryRowContext(ctx,
		`SELECT created_at < updated_at FROM member WHERE id = ?`, mid).Scan(&updated); err != nil || updated != 1 {
		t.Fatalf("trigger updated_at = %d err=%v", updated, err)
	}

	// phone 可空：两个纯微信会员（NULL phone）并存
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member (id, member_no, wx_openid) VALUES ('01JM0140000000000000000XB', 'M202609290002', 'o-new-1')`); err != nil {
		t.Fatalf("null phone member 1: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member (id, member_no, wx_openid) VALUES ('01JM0140000000000000000XC', 'M202609290003', 'o-new-2')`); err != nil {
		t.Fatalf("null phone member 2: %v", err)
	}
	// 非空 phone 重复仍被拒
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member (id, member_no, phone) VALUES ('01JM0140000000000000000XD', 'M202609290004', '13999990001')`); err == nil {
		t.Fatal("duplicate phone accepted after rebuild")
	}
	// wx_openid 唯一索引重建成功
	if _, err := db.ExecContext(ctx,
		`INSERT INTO member (id, member_no, wx_openid) VALUES ('01JM0140000000000000000XE', 'M202609290005', 'o-old-1')`); err == nil {
		t.Fatal("duplicate wx_openid accepted after rebuild")
	}
	// 重复应用 014 幂等
	again, err := Migrate(ctx, db, newDir)
	if err != nil || len(again) != 0 {
		t.Fatalf("re-apply 014: %v %v", again, err)
	}
}

package identity

import (
	"context"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/member"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// admin_test.go — 商家自助修改登录手机号/密码（A5.5，热生效）。

func TestUpdateCredentials(t *testing.T) {
	db := testsupport.NewSchemaDB(t, "anmo_idty_")
	cfg := config.Defaults()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Auth.AdminPhone = "13800000001"
	cfg.Auth.AdminPasswordSeed = "old-pass-123"
	cfg.Auth.AdminTokenHours = 1
	mem := member.New(db, cfg)
	p := New(db, cfg, mem, shared.NewNopLogger())
	ctx := context.Background()
	if err := p.EnsureSeed(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// 当前密码错误 → 拒绝
	if err := p.UpdateCredentials(ctx, "", "", "13800000001", ""); err == nil {
		t.Fatal("empty should fail")
	}
	// 找到 owner
	u, err := p.findUserByPhone(ctx, "13800000001")
	if err != nil || u == nil {
		t.Fatalf("owner: %v", err)
	}
	if err := p.UpdateCredentials(ctx, u.id, "wrong-pass", "", ""); err == nil {
		t.Fatal("wrong current password accepted")
	}

	// 改密码：旧密码登录失效，新密码生效（热生效）
	if err := p.UpdateCredentials(ctx, u.id, "old-pass-123", "", "new-pass-456"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if _, _, err := p.AdminLogin(ctx, "13800000001", "old-pass-123"); err == nil {
		t.Fatal("old password still works")
	}
	if _, _, err := p.AdminLogin(ctx, "13800000001", "new-pass-456"); err != nil {
		t.Fatalf("new password login: %v", err)
	}

	// 改手机号：新号登录生效，旧号失效
	if err := p.UpdateCredentials(ctx, u.id, "new-pass-456", "13800000099", ""); err != nil {
		t.Fatalf("change phone: %v", err)
	}
	if _, _, err := p.AdminLogin(ctx, "13800000001", "new-pass-456"); err == nil {
		t.Fatal("old phone still works")
	}
	if _, _, err := p.AdminLogin(ctx, "13800000099", "new-pass-456"); err != nil {
		t.Fatalf("new phone login: %v", err)
	}

	// 同号重复提交 = 幂等无变更（允许）
	if err := p.UpdateCredentials(ctx, u.id, "new-pass-456", "13800000099", ""); err != nil {
		t.Fatalf("idempotent same phone: %v", err)
	}
	// 弱密码/坏手机号格式
	if err := p.UpdateCredentials(ctx, u.id, "new-pass-456", "", "12345"); err == nil {
		t.Fatal("weak password accepted")
	}
	if err := p.UpdateCredentials(ctx, u.id, "new-pass-456", "12345", ""); err == nil {
		t.Fatal("bad phone accepted")
	}
}

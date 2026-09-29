package member

import (
	"context"
	"testing"

	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// ensurebyphonetx_test.go — D29：散客快速结算的手机号归档入口（完整手机号校验口径）。

func TestEnsureByPhoneTx(t *testing.T) {
	p := New(testsupport.NewSchemaDB(t, "anmo_member_"), nil)
	ctx := context.Background()

	var id1 string
	var created1 bool
	var err error
	if err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		id1, created1, err = p.EnsureByPhoneTx(ctx, tx, " 13700001111 ", "散客甲")
		return err
	}); err != nil {
		t.Fatalf("ensure create: %v", err)
	}
	if id1 == "" || !created1 {
		t.Fatalf("first call should create: %q %v", id1, created1)
	}
	var no, name string
	if err := p.db.QueryRowContext(ctx, `SELECT member_no, name FROM member WHERE id = ?`, id1).Scan(&no, &name); err != nil {
		t.Fatalf("query member: %v", err)
	}
	if name != "散客甲" || len(no) != 13 || no[0] != 'M' {
		t.Fatalf("member row = %q %q", no, name)
	}

	// 二次命中：不新建
	var id2 string
	var created2 bool
	if err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		id2, created2, err = p.EnsureByPhoneTx(ctx, tx, "13700001111", "改名无效")
		return err
	}); err != nil {
		t.Fatalf("ensure match: %v", err)
	}
	if id2 != id1 || created2 {
		t.Fatalf("second call should match: %q %v", id2, created2)
	}

	// 非法手机号（0 开头 / 位数不足）→ MEMBER_BAD_PHONE
	bad := []string{"03700001111", "1370000111", "137000011122", "1370000111a"}
	for _, b := range bad {
		err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
			_, _, e := p.EnsureByPhoneTx(ctx, tx, b, "x")
			return e
		})
		if !shared.Is(err, "MEMBER_BAD_PHONE") {
			t.Fatalf("phone %q: want MEMBER_BAD_PHONE, got %v", b, err)
		}
	}
}

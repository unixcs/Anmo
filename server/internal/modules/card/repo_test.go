package card

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"anmo/server/internal/config"
	"anmo/server/internal/database"
	"anmo/server/internal/modules/member"
	svcmodule "anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
)

type cardEnv struct {
	p     *Provider
	mem   *member.Provider
	svc   *svcmodule.Provider
	mbrID string
	tplID string
	svcID string
	card  string
	catID string
}

// mustItem creates a service item under the env category, failing the test on error.
func (e *cardEnv) mustItem(t *testing.T, name string, duration int, price int64) string {
	t.Helper()
	it, err := e.svc.CreateItem(context.Background(), svcmodule.NewItem{CategoryID: e.catID, Name: name, DurationMin: duration, PriceCents: price})
	if err != nil {
		t.Fatalf("create item %s: %v", name, err)
	}
	return it.ID
}

func newCardEnv(t *testing.T) *cardEnv {
	t.Helper()
	dsn := os.Getenv("ANMO_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("ANMO_TEST_MYSQL_DSN not set; skipping DB test")
	}
	raw, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	schema := "anmo_card_" + shared.NewID()
	if _, err := raw.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { raw.Exec("DROP DATABASE `" + schema + "`"); raw.Close() })
	if _, err := raw.Exec("USE `" + schema + "`"); err != nil {
		t.Fatalf("use: %v", err)
	}
	migDir, _ := filepath.Abs("../../../migrations")
	if _, err := database.Migrate(context.Background(), &database.Pool{DB: raw}, migDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db := &database.Pool{DB: raw}
	cfg := &config.Config{}
	e := &cardEnv{p: New(db, cfg), mem: member.New(db, cfg), svc: svcmodule.New(db, cfg)}

	ctx := context.Background()
	err = shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		var er error
		e.mbrID, _, er = e.mem.EnsureByPhone(ctx, tx, "13900000001", "卡测试")
		return er
	})
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	tpl, err := e.p.CreateTemplate(ctx, NewTemplate{Name: "按摩10次卡", Type: "COUNT", TotalCount: 10, PriceCents: 100000})
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	e.tplID = tpl.ID
	cat, err := e.svc.CreateCategory(ctx, "按摩", 1)
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	e.catID = cat.ID
	it, err := e.svc.CreateItem(ctx, svcmodule.NewItem{CategoryID: e.catID, Name: "肩颈按摩", DurationMin: 60, PriceCents: 12800})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	e.svcID = it.ID
	if err := e.p.SetServiceRules(ctx, e.tplID, []string{e.svcID}); err != nil {
		t.Fatalf("rules: %v", err)
	}
	c, err := e.p.IssueCard(ctx, e.mbrID, e.tplID, "op-1")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	e.card = c.ID
	return e
}

func TestIssueCardWritesIssueTransaction(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()

	cards, err := e.p.ListByMember(ctx, e.mbrID)
	if err != nil || len(cards) != 1 || cards[0].RemainingCount != 10 || cards[0].Status != "ACTIVE" {
		t.Fatalf("cards = %+v err=%v", cards, err)
	}
	txn, err := e.p.Transactions(ctx, e.card)
	if err != nil {
		t.Fatalf("txns: %v", err)
	}
	if len(txn) != 1 || txn[0].Type != "ISSUE" || txn[0].Quantity != 10 || txn[0].Before != 0 || txn[0].After != 10 {
		t.Fatalf("issue transaction = %+v", txn)
	}
}

func TestApplyRedeemAndReversal(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()

	var before, after int
	err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		var er error
		before, after, er = e.p.ApplyRedeem(ctx, tx, e.card, e.svcID, 1, "ref-1", "op-1")
		return er
	})
	if err != nil || before != 10 || after != 9 {
		t.Fatalf("redeem: err=%v before=%d after=%d", err, before, after)
	}
	txn, _ := e.p.Transactions(ctx, e.card)
	if len(txn) != 2 || txn[1].Type != "REDEEM" || txn[1].Before != 10 || txn[1].After != 9 {
		t.Fatalf("redeem transaction = %+v", txn)
	}

	err = shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		var er error
		before, after, er = e.p.ApplyReversal(ctx, tx, e.card, 1, "ref-1", "op-1")
		return er
	})
	if err != nil || before != 9 || after != 10 {
		t.Fatalf("reversal: err=%v before=%d after=%d", err, before, after)
	}
	txn, _ = e.p.Transactions(ctx, e.card)
	if txn[2].Type != "REVERSAL" || txn[2].After != 10 {
		t.Fatalf("reversal transaction = %+v", txn[2])
	}
}

func TestRedeemValidationRejects(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()

	// Case 5: service not covered by the card rules
	other := e.mustItem(t, "拔罐", 30, 8800)
	err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		_, _, er := e.p.ApplyRedeem(ctx, tx, e.card, other, 1, "ref-x", "op")
		return er
	})
	if !shared.Is(err, "CARD_SERVICE_NOT_ALLOWED") {
		t.Fatalf("want CARD_SERVICE_NOT_ALLOWED, got %v", err)
	}

	// Case 3: insufficient balance
	err = shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		_, _, er := e.p.ApplyRedeem(ctx, tx, e.card, e.svcID, 11, "ref-y", "op")
		return er
	})
	if !shared.Is(err, "CARD_INSUFFICIENT") {
		t.Fatalf("want CARD_INSUFFICIENT, got %v", err)
	}

	// Case 4: expired card
	yesterday := shared.NowShanghai().AddDate(0, 0, -1).Format("2006-01-02")
	if _, err := e.p.db.ExecContext(ctx, `UPDATE member_card SET valid_until = ? WHERE id = ?`, yesterday, e.card); err != nil {
		t.Fatalf("expire: %v", err)
	}
	err = shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		_, _, er := e.p.ApplyRedeem(ctx, tx, e.card, e.svcID, 1, "ref-z", "op")
		return er
	})
	if !shared.Is(err, "CARD_EXPIRED") {
		t.Fatalf("want CARD_EXPIRED, got %v", err)
	}

	// D12: cancelled card rejected
	if _, err := e.p.db.ExecContext(ctx, `UPDATE member_card SET valid_until = NULL, status = 'CANCELLED' WHERE id = ?`, e.card); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	err = shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		_, _, er := e.p.ApplyRedeem(ctx, tx, e.card, e.svcID, 1, "ref-w", "op")
		return er
	})
	if !shared.Is(err, "CARD_NOT_ACTIVE") {
		t.Fatalf("want CARD_NOT_ACTIVE, got %v", err)
	}
}

func TestAdjustReactivatesUsedUp(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()

	// drain 10 → 0 via real redeems
	for i := 0; i < 10; i++ {
		err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
			_, _, er := e.p.ApplyRedeem(ctx, tx, e.card, e.svcID, 1, "", "op-1")
			return er
		})
		if err != nil {
			t.Fatalf("redeem #%d: %v", i+1, err)
		}
	}
	cards, _ := e.p.ListByMember(ctx, e.mbrID)
	if cards[0].RemainingCount != 0 || cards[0].Status != "USED_UP" {
		t.Fatalf("want USED_UP/0, got %s/%d", cards[0].Status, cards[0].RemainingCount)
	}
	// adjust up reactivates (D13)
	c2, err := e.p.Adjust(ctx, e.card, 2, "补次", "op-1")
	if err != nil {
		t.Fatalf("adjust up: %v", err)
	}
	if c2.RemainingCount != 2 || c2.Status != "ACTIVE" {
		t.Fatalf("after adjust: %s/%d", c2.Status, c2.RemainingCount)
	}
	// below zero rejected
	if _, err := e.p.Adjust(ctx, e.card, -5, "", "op-1"); !shared.Is(err, "CARD_INSUFFICIENT") {
		t.Fatalf("want CARD_INSUFFICIENT, got %v", err)
	}
	// redeem to 0 again → USED_UP again
	err = shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		_, _, er := e.p.ApplyRedeem(ctx, tx, e.card, e.svcID, 2, "", "op-1")
		return er
	})
	if err != nil {
		t.Fatalf("redeem 2: %v", err)
	}
	cards, _ = e.p.ListByMember(ctx, e.mbrID)
	if cards[0].Status != "USED_UP" {
		t.Fatalf("want USED_UP after drain, got %s", cards[0].Status)
	}
}

func TestUsableCardsFiltering(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()

	other := e.mustItem(t, "足疗", 45, 9900)
	cards, err := e.p.UsableCards(ctx, e.mbrID, e.svcID)
	if err != nil || len(cards) != 1 {
		t.Fatalf("usable for allowed service: n=%d err=%v", len(cards), err)
	}
	cards, err = e.p.UsableCards(ctx, e.mbrID, other)
	if err != nil || len(cards) != 0 {
		t.Fatalf("usable for other service: n=%d err=%v", len(cards), err)
	}
}

package card

import (
	"context"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/member"
	svcmodule "anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
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
	db := testsupport.NewSchemaDB(t, "anmo_card_")
	cfg := config.Defaults()
	e := &cardEnv{p: New(db, cfg), mem: member.New(db, cfg), svc: svcmodule.New(db, cfg)}

	ctx := context.Background()
	err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
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

func TestAdjustDrainToZeroMarksUsedUp(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()

	// 调整扣到 0 与核销同口径（D13）：ACTIVE → USED_UP
	c, err := e.p.Adjust(ctx, e.card, -10, "扣完", "op-1")
	if err != nil {
		t.Fatalf("adjust down: %v", err)
	}
	if c.RemainingCount != 0 || c.Status != "USED_UP" {
		t.Fatalf("want USED_UP/0 after adjust, got %s/%d", c.Status, c.RemainingCount)
	}
	// 加回后复活
	c, err = e.p.Adjust(ctx, e.card, 1, "补次", "op-1")
	if err != nil {
		t.Fatalf("adjust up: %v", err)
	}
	if c.Status != "ACTIVE" || c.RemainingCount != 1 {
		t.Fatalf("want ACTIVE/1, got %s/%d", c.Status, c.RemainingCount)
	}
	// EXPIRED 不因调整复活（D18 同口径）
	if _, err := e.p.db.ExecContext(ctx, `UPDATE member_card SET status='EXPIRED' WHERE id=?`, e.card); err != nil {
		t.Fatal(err)
	}
	c, err = e.p.Adjust(ctx, e.card, 1, "过期后补", "op-1")
	if err != nil {
		t.Fatalf("adjust expired: %v", err)
	}
	if c.Status != "EXPIRED" {
		t.Fatalf("expired card must not reactivate, got %s", c.Status)
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

// F8（第九批审查）：未来生效的卡（预售/预发）不可核销、不出现在顾客可用卡列表。
func TestCardNotStartedRejected(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()
	tomorrow := shared.NowShanghai().AddDate(0, 0, 1).Format("2006-01-02")
	nextYear := shared.NowShanghai().AddDate(1, 0, 0).Format("2006-01-02")
	tpl, err := e.p.CreateTemplate(ctx, NewTemplate{
		Name: "预售卡", Type: "COUNT", TotalCount: 5, PriceCents: 50000,
		ValidityType: "FIXED", ValidFrom: &tomorrow, ValidUntil: &nextYear,
	})
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	if err := e.p.SetServiceRules(ctx, tpl.ID, []string{e.svcID}); err != nil {
		t.Fatalf("rules: %v", err)
	}
	c, err := e.p.IssueCard(ctx, e.mbrID, tpl.ID, "op-1")
	if err != nil {
		t.Fatalf("issue future card: %v", err)
	}
	// 锁定后校验：CARD_NOT_STARTED
	err = shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		locked, e2 := e.p.LockForRedeem(ctx, tx, c.ID)
		if e2 != nil {
			return e2
		}
		return e.p.ValidateForRedeem(ctx, tx, locked, e.svcID, 1)
	})
	if !shared.Is(err, "CARD_NOT_STARTED") {
		t.Fatalf("validate future card = %v, want CARD_NOT_STARTED", err)
	}
	// 可用卡列表也不应出现
	cards, err := e.p.UsableCards(ctx, e.mbrID, e.svcID)
	if err != nil {
		t.Fatalf("usable: %v", err)
	}
	for _, uc := range cards {
		if uc.ID == c.ID {
			t.Fatalf("future card must not appear in usable list")
		}
	}
}

// F8（第九批审查）：有效期已结束的模板不允许再发卡。
func TestIssueFromExpiredTemplateRejected(t *testing.T) {
	e := newCardEnv(t)
	ctx := context.Background()
	yesterday := shared.NowShanghai().AddDate(0, 0, -1).Format("2006-01-02")
	lastYear := shared.NowShanghai().AddDate(-1, 0, 0).Format("2006-01-02")
	tpl, err := e.p.CreateTemplate(ctx, NewTemplate{
		Name: "已过气卡", Type: "COUNT", TotalCount: 5, PriceCents: 50000,
		ValidityType: "FIXED", ValidFrom: &lastYear, ValidUntil: &yesterday,
	})
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	_, err = e.p.IssueCard(ctx, e.mbrID, tpl.ID, "op-1")
	if !shared.Is(err, "CARD_TEMPLATE_EXPIRED") {
		t.Fatalf("issue from expired template = %v, want CARD_TEMPLATE_EXPIRED", err)
	}
}

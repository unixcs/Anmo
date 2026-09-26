package transaction

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/modules/card"
	"anmo/server/internal/modules/member"
	svcmodule "anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

type txnEnv struct {
	p     *Provider
	cards *card.Provider
	apt   *appointment.Provider
	mbrID string
	svcID string
	card  string
	apt1  string // confirmed appointment in service
}

func newTxnEnv(t *testing.T) *txnEnv {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_txn_")
	cfg := config.Defaults()
	mem := member.New(db, cfg)
	svc := svcmodule.New(db, cfg)
	cards := card.New(db, cfg)
	apt := appointment.New(db, cfg, svc)
	e := &txnEnv{p: New(db, cfg, cards, apt, mem), cards: cards, apt: apt}

	ctx := context.Background()
	err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
		var er error
		e.mbrID, _, er = mem.EnsureByPhone(ctx, tx, "13900000021", "结算测试")
		return er
	})
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	cat, err := svc.CreateCategory(ctx, "按摩", 1)
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	it, err := svc.CreateItem(ctx, svcmodule.NewItem{CategoryID: cat.ID, Name: "肩颈按摩", DurationMin: 60, PriceCents: 12800})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	e.svcID = it.ID

	tpl, err := cards.CreateTemplate(ctx, card.NewTemplate{Name: "按摩10次卡", Type: "COUNT", TotalCount: 10, PriceCents: 100000})
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	if err := cards.SetServiceRules(ctx, tpl.ID, []string{e.svcID}); err != nil {
		t.Fatalf("rules: %v", err)
	}
	c, err := cards.IssueCard(ctx, e.mbrID, tpl.ID, "op-1")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	e.card = c.ID
	return e
}

// bookInService creates + confirms + starts an appointment (ready to settle).
func (e *txnEnv) bookInService(t *testing.T, day int, hh, mm int) string {
	t.Helper()
	ctx := context.Background()
	d := shared.NowShanghai().AddDate(0, 0, day)
	slot := fmt.Sprintf("%04d-%02d-%02d %02d:%02d", d.Year(), d.Month(), d.Day(), hh, mm)
	a, err := e.apt.Create(ctx, e.mbrID, e.svcID, slot, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := e.apt.Confirm(ctx, a.ID, "op-1"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := e.apt.Start(ctx, a.ID, "op-1"); err != nil {
		t.Fatalf("start: %v", err)
	}
	return a.ID
}

func TestFullRedeemFlow(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	aptID := e.bookInService(t, 1, 10, 0)

	rd, pay, err := e.p.SettleByCard(ctx, aptID, e.card, "op-1", "idem-1")
	if err != nil {
		t.Fatalf("settle by card: %v", err)
	}
	if rd.Status != "SUCCESS" || rd.BeforeCount != 10 || rd.AfterCount != 9 {
		t.Fatalf("redemption = %+v", rd)
	}
	if pay.Method != "CARD" || pay.Status != "VALID" || pay.AmountCents != 12800 {
		t.Fatalf("payment = %+v", pay)
	}
	// card balance & transactions
	cards, _ := e.cards.ListByMember(ctx, e.mbrID)
	if cards[0].RemainingCount != 9 {
		t.Fatalf("remaining = %d", cards[0].RemainingCount)
	}
	txns, _ := e.cards.Transactions(ctx, e.card)
	if len(txns) != 2 || txns[1].Type != "REDEEM" {
		t.Fatalf("card transactions = %+v", txns)
	}
	// appointment completed
	a, _ := e.apt.Get(ctx, aptID)
	if a.Status != appointment.StatusCompleted {
		t.Fatalf("appointment status = %s", a.Status)
	}
	// member last_visit touched
	mem, _ := e.p.members.Get(ctx, e.mbrID)
	if mem.LastVisitAt == nil {
		t.Fatal("last_visit_at not touched")
	}
}

func TestIdempotentReplay(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	aptID := e.bookInService(t, 1, 11, 0)

	rd1, _, err := e.p.SettleByCard(ctx, aptID, e.card, "op-1", "idem-replay")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	// same idem key → same redemption, no double deduction
	rd2, _, err := e.p.SettleByCard(ctx, aptID, e.card, "op-1", "idem-replay")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if rd1.ID != rd2.ID {
		t.Fatalf("replay returned different redemption")
	}
	cards, _ := e.cards.ListByMember(ctx, e.mbrID)
	if cards[0].RemainingCount != 9 {
		t.Fatalf("balance after replay = %d (Case 6)", cards[0].RemainingCount)
	}
}

func TestDifferentKeyOnSameAppointmentBlocked(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	aptID := e.bookInService(t, 1, 12, 0)

	if _, _, err := e.p.SettleByCard(ctx, aptID, e.card, "op-1", "key-a"); err != nil {
		t.Fatalf("first: %v", err)
	}
	// second redeem with different key → active_lock unique blocks
	if _, _, err := e.p.SettleByCard(ctx, aptID, e.card, "op-1", "key-b"); err == nil {
		t.Fatal("second redemption allowed on same appointment")
	}
	cards, _ := e.cards.ListByMember(ctx, e.mbrID)
	if cards[0].RemainingCount != 9 {
		t.Fatalf("balance = %d", cards[0].RemainingCount)
	}
}

func TestReverseRestoresAndSecondReverseFails(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	aptID := e.bookInService(t, 1, 14, 0)

	rd, _, err := e.p.SettleByCard(ctx, aptID, e.card, "op-1", "idem-rev")
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	// Case 7: reversal restores the count
	if err := e.p.ReverseRedemption(ctx, rd.ID, "误核销", "op-1"); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	cards, _ := e.cards.ListByMember(ctx, e.mbrID)
	if cards[0].RemainingCount != 10 || cards[0].Status != "ACTIVE" {
		t.Fatalf("after reverse: %d %s", cards[0].RemainingCount, cards[0].Status)
	}
	// appointment stays COMPLETED (§59)
	a, _ := e.apt.Get(ctx, aptID)
	if a.Status != appointment.StatusCompleted {
		t.Fatalf("appointment after reverse = %s", a.Status)
	}
	// original CARD payment VOIDED (D1)
	pays, _ := e.p.ListPayments(ctx, "VALID")
	for _, py := range pays {
		if py.AppointmentID == aptID {
			t.Fatalf("VALID card payment remains after reversal: %+v", py)
		}
	}
	// Case 8: second reversal fails
	if err := e.p.ReverseRedemption(ctx, rd.ID, "again", "op-1"); !shared.Is(err, "RDM_ALREADY_REVERSED") {
		t.Fatalf("want RDM_ALREADY_REVERSED, got %v", err)
	}
	// can redeem again after reversal (re-settle)
	if _, _, err := e.p.SettleByCard(ctx, aptID, e.card, "op-1", "idem-rev2"); err != nil {
		t.Fatalf("re-settle after reversal: %v", err)
	}
}

func TestSettleByPayNoStatusChangeAndSingleValid(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	aptID := e.bookInService(t, 1, 16, 0)

	pay, err := e.p.SettleByPay(ctx, aptID, "CASH", 12800, "ref-1", "现金", "op-1", "idem-p1")
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	if pay.Status != "VALID" {
		t.Fatalf("payment = %+v", pay)
	}
	a, _ := e.apt.Get(ctx, aptID)
	if a.Status != appointment.StatusInService {
		t.Fatalf("cash payment changed status to %s (D9)", a.Status)
	}
	// second VALID payment blocked (D1)
	if _, err := e.p.SettleByPay(ctx, aptID, "WECHAT_TRANSFER", 12800, "", "", "op-1", "idem-p2"); !shared.Is(err, "PAY_EXISTS") {
		t.Fatalf("want PAY_EXISTS, got %v", err)
	}
}

func TestConcurrentRedeemOneBalance(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	// drain 9 of 10 via sequential settles (day2 4 slots, day3 4 slots, day4 1 slot)
	for i := 0; i < 9; i++ {
		other := e.bookInService(t, 2+i/4, 13+(i%4), 0)
		if _, _, err := e.p.SettleByCard(ctx, other, e.card, "op-1", fmt.Sprintf("drain-%d", i)); err != nil {
			t.Fatalf("drain %d: %v", i, err)
		}
	}
	// now balance = 1: two concurrent redeems on two fresh appointments
	a1 := e.bookInService(t, 3, 10, 0)
	a2 := e.bookInService(t, 3, 11, 0)
	results := make([]error, 2)
	var wg sync.WaitGroup
	for i, id := range []string{a1, a2} {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			_, _, err := e.p.SettleByCard(ctx, id, e.card, "op-1", fmt.Sprintf("race-%d", i))
			results[i] = err
		}(i, id)
	}
	wg.Wait()
	ok := 0
	fail := 0
	for _, err := range results {
		if err == nil {
			ok++
		} else {
			fail++
		}
	}
	if ok != 1 || fail != 1 {
		t.Fatalf("Case 1: %d ok / %d fail, want 1/1 (%v %v)", ok, fail, results[0], results[1])
	}
}

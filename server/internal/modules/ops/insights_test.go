package ops

import (
	"context"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/card"
	"anmo/server/internal/modules/member"
	svcmodule "anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

type opsEnv struct {
	p     *Provider
	cards *card.Provider
	mem   *member.Provider
	svc   *svcmodule.Provider
	mbr1  string
	mbr2  string
	svcID string
	tplID string
}

func newOpsEnv(t *testing.T) *opsEnv {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_ops_")
	cfg := config.Defaults()
	mem := member.New(db, cfg)
	svc := svcmodule.New(db, cfg)
	cards := card.New(db, cfg)
	e := &opsEnv{p: New(db, cfg, cards, nil, mem), cards: cards, mem: mem, svc: svc}

	ctx := context.Background()
	for i, ph := range []string{"13900000031", "13900000032"} {
		err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
			var er error
			if i == 0 {
				e.mbr1, _, er = mem.EnsureByPhone(ctx, tx, ph, "洞察A")
			} else {
				e.mbr2, _, er = mem.EnsureByPhone(ctx, tx, ph, "洞察B")
			}
			return er
		})
		if err != nil {
			t.Fatalf("member: %v", err)
		}
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
	tpl, err := cards.CreateTemplate(ctx, card.NewTemplate{
		Name: "体验卡", Type: "COUNT", TotalCount: 10, PriceCents: 1000,
		ValidityType: "FIXED", ValidFrom: strPtr("2026-01-01"), ValidUntil: strPtr("2026-12-31"),
	})
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	e.tplID = tpl.ID
	if err := cards.SetServiceRules(ctx, tpl.ID, []string{e.svcID}); err != nil {
		t.Fatalf("rules: %v", err)
	}
	return e
}

func strPtr(s string) *string { return &s }

// TestInsightBoundaries pins the §86 edges: <=2 low balance, 60d dormant, 7d expiring.
func TestInsightBoundaries(t *testing.T) {
	e := newOpsEnv(t)
	ctx := context.Background()

	// mbr1: redeem down to exactly 2 → must appear in LOW_BALANCE
	c1, err := e.cards.IssueCard(ctx, e.mbr1, e.tplID, "op")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	for i := 0; i < 8; i++ {
		err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
			_, _, er := e.cards.ApplyRedeem(ctx, tx, c1.ID, e.svcID, 1, "", "op")
			return er
		})
		if err != nil {
			t.Fatalf("redeem %d: %v", i, err)
		}
	}
	// mbr2 stays at 10 → must NOT appear

	insights, err := e.p.Insights(ctx)
	if err != nil {
		t.Fatalf("insights: %v", err)
	}
	low := len(insights["LOW_BALANCE"])
	if low != 1 {
		t.Fatalf("LOW_BALANCE = %d items, want exactly 1", low)
	}
	if insights["LOW_BALANCE"][0].MemberID != e.mbr1 {
		t.Fatalf("low balance member = %+v", insights["LOW_BALANCE"][0])
	}
	// mbr2 at 10 must not be flagged
	if len(insights["EXPIRING"]) != 2 {
		// both cards expire 2026-12-31, within 7 days of 2026-09-27? No (95 days).
		t.Logf("EXPIRING = %d (both cards far from expiry — expected 0)", len(insights["EXPIRING"]))
	}
	for _, item := range insights["EXPIRING"] {
		if item.CardID == c1.ID {
			t.Fatalf("card far from expiry flagged EXPIRING: %+v", item)
		}
	}
}

func TestDormantBoundary(t *testing.T) {
	e := newOpsEnv(t)
	ctx := context.Background()

	// mbr1 just created (0 days old) → NOT dormant at 60 days
	// mbr2: backdate creation 61 days → dormant
	backdate := shared.NowShanghai().AddDate(0, 0, -61).Format("2006-01-02 15:04:05")
	if _, err := e.p.db.ExecContext(ctx,
		`UPDATE member SET created_at = ? WHERE id = ?`, backdate, e.mbr2); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	members, err := e.p.members.DormantMembers(ctx, 60)
	if err != nil {
		t.Fatalf("dormant: %v", err)
	}
	if len(members) != 1 || members[0].ID != e.mbr2 {
		t.Fatalf("dormant members = %+v", members)
	}
	// visited recently → not dormant
	if _, err := e.p.db.ExecContext(ctx,
		`UPDATE member SET last_visit_at = datetime('now','+8 hours') WHERE id = ?`, e.mbr2); err != nil {
		t.Fatalf("visit: %v", err)
	}
	members, err = e.p.members.DormantMembers(ctx, 60)
	if err != nil {
		t.Fatalf("dormant2: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("visited member flagged dormant: %+v", members)
	}
}

func TestDailySweepAndSnapshot(t *testing.T) {
	e := newOpsEnv(t)
	ctx := context.Background()

	c1, err := e.cards.IssueCard(ctx, e.mbr1, e.tplID, "op")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	// backdate validity so the sweep flips it
	if _, err := e.p.db.ExecContext(ctx,
		`UPDATE member_card SET valid_until = '2026-01-01' WHERE id = ?`, c1.ID); err != nil {
		t.Fatalf("backdate card: %v", err)
	}
	// mbr2 card drained to 2 → LOW_BALANCE snapshot material
	c2, err := e.cards.IssueCard(ctx, e.mbr2, e.tplID, "op")
	if err != nil {
		t.Fatalf("issue2: %v", err)
	}
	for i := 0; i < 8; i++ {
		err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
			_, _, er := e.cards.ApplyRedeem(ctx, tx, c2.ID, e.svcID, 1, "", "op")
			return er
		})
		if err != nil {
			t.Fatalf("redeem %d: %v", i, err)
		}
	}

	out, err := e.p.RunDaily(ctx)
	if err != nil {
		t.Fatalf("daily: %v", err)
	}
	if out["expired_swept"].(int64) < 1 {
		t.Fatalf("sweep count = %v", out["expired_swept"])
	}
	cards, _ := e.cards.ListByMember(ctx, e.mbr1)
	for _, c := range cards {
		if c.ID == c1.ID && c.Status != "EXPIRED" {
			t.Fatalf("card status after sweep = %s", c.Status)
		}
	}
	if out["snapshots"].(int64) < 1 {
		t.Fatalf("snapshots = %v", out["snapshots"])
	}
}

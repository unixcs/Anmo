package transaction

import (
	"context"
	"testing"
)

// walkin_test.go — 散客核销（D19）：无预约直接扣次 + 收款 + 撤销链路。

func TestWalkInRedeemAndReverse(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	cardID := e.card

	rd, py, err := e.p.RedeemWalkIn(ctx, cardID, e.svcID, "op-1", "walkin-key-1")
	if err != nil {
		t.Fatalf("walk-in redeem: %v", err)
	}
	if rd.AppointmentID != nil {
		t.Fatalf("walk-in redemption must have NULL appointment, got %s", *rd.AppointmentID)
	}
	if rd.BeforeCount != 10 || rd.AfterCount != 9 {
		t.Fatalf("counts = %d→%d", rd.BeforeCount, rd.AfterCount)
	}
	if py == nil || py.Method != "CARD" || py.Status != "VALID" {
		t.Fatalf("payment = %+v", py)
	}
	if py.AmountCents != 12800 {
		t.Fatalf("payment amount = %d, want service price 12800", py.AmountCents)
	}
	if py.AppointmentID != nil {
		t.Fatalf("walk-in payment must have NULL appointment")
	}

	// idempotent replay returns the original rows
	rd2, py2, err := e.p.RedeemWalkIn(ctx, cardID, e.svcID, "op-1", "walkin-key-1")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if rd2.ID != rd.ID || py2.ID != py.ID {
		t.Fatalf("replay mismatch: %s/%s", rd2.ID, py2.ID)
	}

	// reversal restores the count and voids the payment (D1)
	if err := e.p.ReverseRedemption(ctx, rd.ID, "walkin reverse", "op-1"); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	pays, _ := e.p.ListPayments(ctx, "VALID")
	for _, p := range pays {
		if p.ID == py.ID {
			t.Fatalf("walk-in payment still VALID after reversal")
		}
	}
	// re-redeem after reversal works (active_lock freed)
	if _, _, err := e.p.RedeemWalkIn(ctx, cardID, e.svcID, "op-1", "walkin-key-2"); err != nil {
		t.Fatalf("re-redeem after reversal: %v", err)
	}
}

func TestWalkInRuleEnforcement(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	cardID := e.card

	// unknown service id → rule check fails
	_, _, err := e.p.RedeemWalkIn(ctx, cardID, "01ARZ3NDEKTSV4RRFFQ69G5FAV", "op-1", "walkin-bad-1")
	if err == nil {
		t.Fatalf("walk-in with unknown service must fail")
	}

	// foreign card id → not found / no permission
	_, _, err = e.p.RedeemWalkIn(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAV", e.svcID, "op-1", "walkin-bad-2")
	if err == nil {
		t.Fatalf("walk-in with unknown card must fail")
	}
}

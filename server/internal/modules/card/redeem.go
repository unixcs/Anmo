package card

import (
	"context"
	"database/sql"
	"errors"

	"anmo/server/internal/shared"
)

// redeem.go — the tx-joining primitives the transaction module composes into
// the settlement transaction (AGENTS.md 核销协作; D6).

// LockForRedeem locks the member_card row (SELECT ... FOR UPDATE) inside the
// caller's transaction and returns it.
func (p *Provider) LockForRedeem(ctx context.Context, tx shared.Tx, cardID string) (*MemberCard, error) {
	c, err := scanCard(tx.QueryRowContext(ctx,
		`SELECT `+cardColumns+` FROM member_card WHERE id = ? FOR UPDATE`, cardID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("CARD_NOT_FOUND", "会员卡不存在")
	}
	if err != nil {
		return nil, shared.Server("CARD_LOCK", err)
	}
	return c, nil
}

// ValidateForRedeem checks card status, validity window and service rule
// against an already-locked card (D4/D12/D13). remaining check happens with
// the actual quantity.
func (p *Provider) ValidateForRedeem(ctx context.Context, tx shared.Tx, c *MemberCard, serviceID string, quantity int) error {
	if c.Status != "ACTIVE" {
		return shared.Conflict("CARD_NOT_ACTIVE", "会员卡不可用（状态 "+c.Status+"）")
	}
	if c.ValidUntil != nil {
		today := shared.NowShanghai().Format("2006-01-02")
		if today > *c.ValidUntil {
			return shared.Conflict("CARD_EXPIRED", "会员卡已过期")
		}
	}
	if c.RemainingCount < quantity {
		return shared.Conflict("CARD_INSUFFICIENT", "会员卡余额不足")
	}
	if quantity <= 0 {
		return shared.BadRequest("CARD_BAD_QUANTITY", "核销次数必须大于 0")
	}
	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM card_service_rule WHERE card_template_id = ? AND service_id = ?`,
		c.TemplateID, serviceID).Scan(&count); err != nil {
		return shared.Server("CARD_RULE_QUERY", err)
	}
	if count == 0 {
		return shared.Conflict("CARD_SERVICE_NOT_ALLOWED", "该卡不适用于此服务项目")
	}
	return nil
}

// ApplyRedeem deducts balance and writes the REDEEM transaction. Caller must
// have locked the card (LockForRedeem) and validated (ValidateForRedeem).
func (p *Provider) ApplyRedeem(ctx context.Context, tx shared.Tx, cardID, serviceID string, quantity int, refID, operatorID string) (before, after int, err error) {
	c, err := p.LockForRedeem(ctx, tx, cardID) // re-read inside caller tx
	if err != nil {
		return 0, 0, err
	}
	if err := p.ValidateForRedeem(ctx, tx, c, serviceID, quantity); err != nil {
		return 0, 0, err
	}
	before = c.RemainingCount
	after = before - quantity
	status := "ACTIVE"
	if after == 0 {
		status = "USED_UP" // D13
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE member_card SET remaining_count = ?, status = ? WHERE id = ?`,
		after, status, cardID); err != nil {
		return 0, 0, shared.Server("CARD_REDEEM_UPDATE", err)
	}
	if err := p.writeCardTx(ctx, tx, cardID, c.MemberID, "REDEEM", quantity, before, after,
		"REDEMPTION", operatorID, "核销扣次"); err != nil {
		return 0, 0, err
	}
	return before, after, nil
}

// ApplyReversal restores balance and writes the REVERSAL transaction.
// W4: a CANCELLED card still gets its count restored (ledger correction for a
// mistaken redemption) but stays CANCELLED. W5: an EXPIRED card is never
// resurrected — only USED_UP flips back to ACTIVE.
func (p *Provider) ApplyReversal(ctx context.Context, tx shared.Tx, cardID string, quantity int, refID, operatorID string) (before, after int, err error) {
	c, err := p.LockForRedeem(ctx, tx, cardID)
	if err != nil {
		return 0, 0, err
	}
	before = c.RemainingCount
	after = before + quantity
	status := c.Status
	if status == "USED_UP" {
		status = "ACTIVE"
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE member_card SET remaining_count = ?, status = ? WHERE id = ?`,
		after, status, cardID); err != nil {
		return 0, 0, shared.Server("CARD_REVERSAL_UPDATE", err)
	}
	if err := p.writeCardTx(ctx, tx, cardID, c.MemberID, "REVERSAL", quantity, before, after,
		"REDEMPTION", operatorID, "撤销核销恢复"); err != nil {
		return 0, 0, err
	}
	return before, after, nil
}

// Adjust changes remaining_count by delta (±N) with an ADJUSTMENT transaction.
func (p *Provider) Adjust(ctx context.Context, cardID string, delta int, remark, operatorID string) (*MemberCard, error) {
	var out *MemberCard
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		c, e := p.LockForRedeem(ctx, tx, cardID)
		if e != nil {
			return e
		}
		if c.Status == "CANCELLED" {
			return shared.Conflict("CARD_CANCELLED", "会员卡已作废")
		}
		after := c.RemainingCount + delta
		if after < 0 {
			return shared.Conflict("CARD_INSUFFICIENT", "调整后余额不能小于 0")
		}
		status := c.Status
		if status == "USED_UP" && after > 0 {
			status = "ACTIVE"
		}
		if _, e := tx.ExecContext(ctx,
			`UPDATE member_card SET remaining_count = ?, status = ? WHERE id = ?`,
			after, status, cardID); e != nil {
			return shared.Server("CARD_ADJUST_UPDATE", e)
		}
		if e := p.writeCardTx(ctx, tx, cardID, c.MemberID, "ADJUSTMENT", delta, c.RemainingCount, after,
			"", operatorID, remark); e != nil {
			return e
		}
		out, e = scanCard(tx.QueryRowContext(ctx, `SELECT `+cardColumns+` FROM member_card WHERE id = ?`, cardID))
		return e
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Cancel voids a card: status only, no balance change, no count transaction (D12).
func (p *Provider) Cancel(ctx context.Context, cardID, operatorID string) error {
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE member_card SET status = 'CANCELLED' WHERE id = ? AND status <> 'CANCELLED'`,
			cardID)
		if err != nil {
			return shared.Server("CARD_CANCEL", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return shared.Conflict("CARD_ALREADY_CANCELLED", "会员卡已作废")
		}
		return nil
	})
}

// ListByMember returns a member's cards.
func (p *Provider) ListByMember(ctx context.Context, memberID string) ([]*MemberCard, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+cardColumns+` FROM member_card WHERE member_id = ? ORDER BY issued_at DESC`, memberID)
	if err != nil {
		return nil, shared.Server("CARD_LIST", err)
	}
	defer rows.Close()
	var out []*MemberCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, shared.Server("CARD_SCAN", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// UsableCards returns the member's cards that can redeem the given service
// today (customer-facing candidate list at settlement, §80).
func (p *Provider) UsableCards(ctx context.Context, memberID, serviceID string) ([]*MemberCard, error) {
	today := shared.NowShanghai().Format("2006-01-02")
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+cardColumns+` FROM member_card c
		 WHERE c.member_id = ? AND c.status = 'ACTIVE' AND c.remaining_count > 0
		   AND (c.valid_until IS NULL OR c.valid_until >= ?)
		   AND EXISTS (SELECT 1 FROM card_service_rule r
		               WHERE r.card_template_id = c.card_template_id AND r.service_id = ?)
		 ORDER BY c.issued_at`, memberID, today, serviceID)
	if err != nil {
		return nil, shared.Server("CARD_USABLE", err)
	}
	defer rows.Close()
	var out []*MemberCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, shared.Server("CARD_SCAN", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// Transactions returns a card's balance history.
func (p *Provider) Transactions(ctx context.Context, cardID string) ([]*CardTransaction, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, member_card_id, type, quantity, before_count, after_count, reference_type, remark, created_at
		 FROM card_transaction WHERE member_card_id = ? ORDER BY created_at`, cardID)
	if err != nil {
		return nil, shared.Server("CARD_TXN_LIST", err)
	}
	defer rows.Close()
	var out []*CardTransaction
	for rows.Next() {
		t := &CardTransaction{}
		if err := rows.Scan(&t.ID, &t.CardID, &t.Type, &t.Quantity, &t.Before, &t.After,
			&t.RefType, &t.Remark, &t.CreatedAt); err != nil {
			return nil, shared.Server("CARD_TXN_SCAN", err)
		}
		out = append(out, t)
	}
	return out, nil
}

// LowBalanceCards returns ACTIVE cards with remaining <= threshold (§86).
func (p *Provider) LowBalanceCards(ctx context.Context, threshold int) ([]*MemberCard, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+cardColumns+` FROM member_card WHERE status = 'ACTIVE' AND remaining_count <= ? ORDER BY remaining_count`, threshold)
	if err != nil {
		return nil, shared.Server("CARD_LOW", err)
	}
	defer rows.Close()
	var out []*MemberCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, shared.Server("CARD_SCAN", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// ExpiringCards returns ACTIVE cards expiring within N days (§86).
func (p *Provider) ExpiringCards(ctx context.Context, days int) ([]*MemberCard, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+cardColumns+` FROM member_card
		 WHERE status = 'ACTIVE' AND valid_until IS NOT NULL
		   AND valid_until < CURDATE() + INTERVAL ? DAY`, days)
	if err != nil {
		return nil, shared.Server("CARD_EXP", err)
	}
	defer rows.Close()
	var out []*MemberCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, shared.Server("CARD_SCAN", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// SweepExpired flips ACTIVE cards past valid_until to EXPIRED (W2/D13).
// Lazy validation at redeem time remains the hard constraint; this sweep only
// keeps the displayed status in sync.
func (p *Provider) SweepExpired(ctx context.Context) (int64, error) {
	res, err := p.db.ExecContext(ctx,
		`UPDATE member_card SET status = 'EXPIRED'
		 WHERE status = 'ACTIVE' AND valid_until IS NOT NULL AND valid_until < CURDATE()`)
	if err != nil {
		return 0, shared.Server("CARD_SWEEP", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

package transaction

import (
	"context"

	"anmo/server/internal/shared"
)

// ListPayments returns payments (status filter: VALID/VOIDED/empty=all).
func (p *Provider) ListPayments(ctx context.Context, status string) ([]*Payment, error) {
	where := "1=1"
	args := []any{}
	if status != "" {
		where = "status = ?"
		args = append(args, status)
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+paymentColumns+` FROM payment WHERE `+where+` ORDER BY recorded_at DESC LIMIT 200`, args...)
	if err != nil {
		return nil, shared.Server("PAY_LIST", err)
	}
	defer rows.Close()
	var out []*Payment
	for rows.Next() {
		py, err := scanPayment(rows)
		if err != nil {
			return nil, shared.Server("PAY_SCAN", err)
		}
		out = append(out, py)
	}
	return out, nil
}

// ListRedemptions returns redemptions (status filter: SUCCESS/REVERSED/empty=all).
func (p *Provider) ListRedemptions(ctx context.Context, status string) ([]*Redemption, error) {
	where := "1=1"
	args := []any{}
	if status != "" {
		where = "status = ?"
		args = append(args, status)
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+redemptionColumns+` FROM redemption WHERE `+where+` ORDER BY created_at DESC LIMIT 200`, args...)
	if err != nil {
		return nil, shared.Server("RDM_LIST", err)
	}
	defer rows.Close()
	var out []*Redemption
	for rows.Next() {
		rd, err := scanRedemption(rows)
		if err != nil {
			return nil, shared.Server("RDM_SCAN", err)
		}
		out = append(out, rd)
	}
	return out, nil
}

package transaction

import (
	"context"
	"database/sql"
	"errors"

	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/shared"
)

// Payment — payment row (§46). Only VALID rows count as income (D1).
type Payment struct {
	ID            string       `json:"id"`
	AppointmentID string       `json:"appointment_id"`
	MemberID      string       `json:"member_id"`
	AmountCents   int64        `json:"amount"`
	Method        string       `json:"method"`
	Status        string       `json:"status"`
	ReferenceNo   string       `json:"reference_no"`
	Remark        string       `json:"remark"`
	RecordedAt    sql.NullTime `json:"-"`
}

// Redemption — redemption row (§52).
type Redemption struct {
	ID            string `json:"id"`
	AppointmentID string `json:"appointment_id"`
	MemberID      string `json:"member_id"`
	MemberCardID  string `json:"member_card_id"`
	ServiceID     string `json:"service_id"`
	Quantity      int    `json:"quantity"`
	BeforeCount   int    `json:"before_count"`
	AfterCount    int    `json:"after_count"`
	Status        string `json:"status"` // SUCCESS | REVERSED
	IdemKey       string `json:"idempotency_key"`
}

const (
	redemptionColumns = `id, appointment_id, member_id, member_card_id, service_id, quantity, before_count, after_count, status, idempotency_key`
	paymentColumns    = `id, appointment_id, member_id, amount, method, status, reference_no, remark, recorded_at`
)

func scanRedemption(row interface{ Scan(...any) error }) (*Redemption, error) {
	rd := &Redemption{}
	err := row.Scan(&rd.ID, &rd.AppointmentID, &rd.MemberID, &rd.MemberCardID, &rd.ServiceID,
		&rd.Quantity, &rd.BeforeCount, &rd.AfterCount, &rd.Status, &rd.IdemKey)
	return rd, err
}

// primaryService returns the appointment's first snapshot service (V1: one service).
func (p *Provider) primaryService(ctx context.Context, tx shared.Tx, aptID string) (*appointment.AppointmentService, error) {
	svcs, err := p.appointments.ServicesOfTx(ctx, tx, aptID)
	if err != nil {
		return nil, err
	}
	if len(svcs) == 0 {
		return nil, shared.Conflict("APT_NO_SERVICE", "预约缺少服务项目信息")
	}
	return svcs[0], nil
}

// findRedemptionByIdem returns the existing redemption for a replayed request.
func (p *Provider) findRedemptionByIdem(ctx context.Context, tx shared.Tx, key string) (*Redemption, error) {
	rd, err := scanRedemption(tx.QueryRowContext(ctx,
		`SELECT `+redemptionColumns+` FROM redemption WHERE idempotency_key = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, shared.Server("RDM_QUERY", err)
	}
	return rd, nil
}

// SettleByCard runs the core redemption transaction (§53):
// lock card → validate → deduct → REDEEM txn → redemption → payment(CARD)
// → appointment COMPLETED → status log. Any failure rolls everything back.
// Idempotent on idemKey (replay returns the original result).
func (p *Provider) SettleByCard(ctx context.Context, aptID, cardID, operatorID, idemKey string) (*Redemption, *Payment, error) {
	if idemKey == "" {
		return nil, nil, shared.BadRequest("IDEM_KEY_REQUIRED", "缺少幂等键")
	}
	var redemption *Redemption
	var payment *Payment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		// replay?
		existing, err := p.findRedemptionByIdem(ctx, tx, idemKey)
		if err != nil {
			return err
		}
		if existing != nil {
			redemption = existing
			payment = p.paymentForRedemption(ctx, tx, existing.ID)
			return nil
		}

		apt, err := p.appointments.GetTx(ctx, tx, aptID)
		if err != nil {
			return err
		}
		svc, err := p.primaryService(ctx, tx, aptID)
		if err != nil {
			return err
		}

		// lock + validate + deduct (card module, tx-joining)
		c, err := p.cards.LockForRedeem(ctx, tx, cardID)
		if err != nil {
			return err
		}
		if err := p.cards.ValidateForRedeem(ctx, tx, c, svc.ServiceID, 1); err != nil {
			return err
		}
		before, after, err := p.cards.ApplyRedeem(ctx, tx, cardID, svc.ServiceID, 1, "", operatorID)
		if err != nil {
			return err
		}

		// redemption row (active_lock UNIQUE guards the one-valid-per-apt invariant)
		redemption = &Redemption{
			ID: shared.NewID(), AppointmentID: aptID, MemberID: apt.MemberID,
			MemberCardID: cardID, ServiceID: svc.ServiceID,
			Quantity: 1, BeforeCount: before, AfterCount: after,
			Status: "SUCCESS", IdemKey: idemKey,
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO redemption (id, appointment_id, member_id, member_card_id, service_id, quantity, before_count, after_count, status, idempotency_key)
			 VALUES (?,?,?,?,?,?,?,?,?,?)`,
			redemption.ID, redemption.AppointmentID, redemption.MemberID, redemption.MemberCardID,
			redemption.ServiceID, redemption.Quantity, redemption.BeforeCount, redemption.AfterCount,
			redemption.Status, redemption.IdemKey); err != nil {
			return shared.Server("RDM_INSERT", err)
		}

		// payment CARD (D1)
		payment = &Payment{
			ID: shared.NewID(), AppointmentID: aptID, MemberID: apt.MemberID,
			AmountCents: svc.PriceSnapshot * int64(svc.Quantity), Method: "CARD",
			Status: "VALID", Remark: "会员卡核销",
		}
		if err := insertPayment(ctx, tx, payment, operatorID); err != nil {
			return err
		}

		// appointment COMPLETED + status log (W1)
		if err := p.appointments.MarkCompleted(ctx, tx, aptID, operatorID); err != nil {
			return err
		}

		// last_visit (§12)
		if err := p.members.TouchLastVisit(ctx, tx, apt.MemberID, shared.NowShanghai()); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return redemption, payment, nil
}

// SettleByPay records a non-card payment (cash / wechat transfer / other).
// It writes ONLY the payment row — the appointment status is untouched (D9).
func (p *Provider) SettleByPay(ctx context.Context, aptID, method string, amountCents int64, refNo, remark, operatorID, idemKey string) (*Payment, error) {
	if idemKey == "" {
		return nil, shared.BadRequest("IDEM_KEY_REQUIRED", "缺少幂等键")
	}
	switch method {
	case "WECHAT_TRANSFER", "CASH", "OTHER":
	default:
		return nil, shared.BadRequest("PAY_BAD_METHOD", "结算方式不支持")
	}
	if amountCents < 0 {
		return nil, shared.BadRequest("PAY_BAD_AMOUNT", "金额不能为负")
	}
	var payment *Payment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		// one VALID payment per appointment (D1) — also the double-submit guard
		var n int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM payment WHERE appointment_id = ? AND status = 'VALID'`, aptID).Scan(&n); err != nil {
			return shared.Server("PAY_COUNT", err)
		}
		if n > 0 {
			return shared.Conflict("PAY_EXISTS", "该预约已有一笔有效收款")
		}
		apt, err := p.appointments.GetTx(ctx, tx, aptID)
		if err != nil {
			return err
		}
		if apt.Status != appointment.StatusInService && apt.Status != appointment.StatusCompleted {
			return shared.Conflict("APT_BAD_TRANSITION", "预约需处于服务中或已完成才能收款")
		}
		payment = &Payment{
			ID: shared.NewID(), AppointmentID: aptID, MemberID: apt.MemberID,
			AmountCents: amountCents, Method: method, Status: "VALID",
			ReferenceNo: refNo, Remark: remark,
		}
		return insertPayment(ctx, tx, payment, operatorID)
	})
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// ReverseRedemption undoes a redemption inside one transaction (§57):
// lock card → redemption must be SUCCESS → mark REVERSED → restore count →
// REVERSAL txn → reversal row → original CARD payment VOIDED (D1).
// The appointment stays COMPLETED (§59).
func (p *Provider) ReverseRedemption(ctx context.Context, redemptionID, reason, operatorID string) error {
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		rd, err := scanRedemption(tx.QueryRowContext(ctx,
			`SELECT `+redemptionColumns+` FROM redemption WHERE id = ? FOR UPDATE`, redemptionID))
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("RDM_NOT_FOUND", "核销记录不存在")
		}
		if err != nil {
			return shared.Server("RDM_QUERY", err)
		}
		if rd.Status != "SUCCESS" {
			return shared.Conflict("RDM_ALREADY_REVERSED", "该核销已撤销")
		}
		// idempotency guard: unique reversal row
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO redemption_reversal (id, redemption_id, reason, operator_id) VALUES (?,?,?,?)`,
			shared.NewID(), redemptionID, reason, operatorID); err != nil {
			return shared.Conflict("RDM_ALREADY_REVERSED", "该核销已撤销")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE redemption SET status = 'REVERSED' WHERE id = ?`, redemptionID); err != nil {
			return shared.Server("RDM_REVERSE", err)
		}
		// restore balance (active_lock auto-clears via generated column)
		if _, _, err := p.cards.ApplyReversal(ctx, tx, rd.MemberCardID, rd.Quantity, rd.ID, operatorID); err != nil {
			return err
		}
		// original CARD payment voided (D1)
		if _, err := tx.ExecContext(ctx,
			`UPDATE payment SET status = 'VOIDED', remark = CONCAT(remark, '；核销撤销') 
			 WHERE appointment_id = ? AND method = 'CARD' AND status = 'VALID'`, rd.AppointmentID); err != nil {
			return shared.Server("PAY_VOID", err)
		}
		return nil
	})
}

func (p *Provider) paymentForRedemption(ctx context.Context, tx shared.Tx, redemptionID string) *Payment {
	py, err := scanPayment(tx.QueryRowContext(ctx,
		`SELECT `+paymentColumns+` FROM payment WHERE appointment_id = 
		 (SELECT appointment_id FROM redemption WHERE id = ?) AND method = 'CARD' AND status = 'VALID'`, redemptionID))
	if err != nil {
		return nil
	}
	return py
}

func insertPayment(ctx context.Context, tx shared.Tx, py *Payment, operatorID string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO payment (id, appointment_id, member_id, amount, method, status, reference_no, remark, recorded_by)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		py.ID, py.AppointmentID, py.MemberID, py.AmountCents, py.Method, py.Status,
		py.ReferenceNo, py.Remark, operatorID)
	if err != nil {
		return shared.Server("PAY_INSERT", err)
	}
	return nil
}

func scanPayment(row interface{ Scan(...any) error }) (*Payment, error) {
	py := &Payment{}
	err := row.Scan(&py.ID, &py.AppointmentID, &py.MemberID, &py.AmountCents, &py.Method,
		&py.Status, &py.ReferenceNo, &py.Remark, &py.RecordedAt)
	return py, err
}

package transaction

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"

	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/shared"
)

// Payment — payment row (§46). Only VALID rows count as income (D1).
type Payment struct {
	ID            string       `json:"id"`
	AppointmentID *string      `json:"appointment_id"` // NULL = 散客核销（D19）
	MemberID      string       `json:"member_id"`
	AmountCents   int64        `json:"amount"`
	Method        string       `json:"method"`
	Status        string       `json:"status"`
	ReferenceNo   string       `json:"reference_no"`
	Remark        string       `json:"remark"`
	IdemKey       string       `json:"idempotency_key,omitempty"`
	RecordedAt    sql.NullTime `json:"-"`
}

// Redemption — redemption row (§52).
type Redemption struct {
	ID            string `json:"id"`
	AppointmentID *string `json:"appointment_id"` // NULL = 散客核销（D19）
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
	paymentColumns    = `id, appointment_id, member_id, amount, method, status, reference_no, remark, idempotency_key, recorded_at`
)

func scanRedemption(row interface{ Scan(...any) error }) (*Redemption, error) {
	rd := &Redemption{}
	var aptID sql.NullString
	err := row.Scan(&rd.ID, &aptID, &rd.MemberID, &rd.MemberCardID, &rd.ServiceID,
		&rd.Quantity, &rd.BeforeCount, &rd.AfterCount, &rd.Status, &rd.IdemKey)
	if aptID.Valid {
		v := aptID.String
		rd.AppointmentID = &v
	}
	return rd, err
}

// isDupKey reports a MySQL duplicate-key error (any unique index).
func isDupKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// isDeadlock reports MySQL 1213/1205 lock errors — safe to surface as a
// retryable conflict (verified: no data damage, tx rolled back).
func isDeadlock(err error) bool {
	var me *mysql.MySQLError
	if !errors.As(err, &me) {
		return false
	}
	return me.Number == 1213 || me.Number == 1205
}

func wrapTxErr(code string, err error) error {
	if isDeadlock(err) {
		return shared.Conflict("LOCK_RETRY", "操作繁忙，请重试")
	}
	return shared.Server(code, err)
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
// Idempotent on idemKey (replay returns the original result; replaying a key
// whose redemption was reversed surfaces 409 instead of stale data — W1).
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
			if existing.Status != "SUCCESS" {
				// W1: the settlement this key identifies has been reversed —
				// replay must not resurface it (nor a mismatched payment).
				return shared.Conflict("RDM_REVERSED", "该结算对应的核销已被撤销")
			}
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
		// W3: the card must belong to the member who owns the appointment.
		if c.MemberID != apt.MemberID {
			return shared.NewErr("CARD_NOT_YOURS", "会员卡不属于该预约的顾客", 403)
		}
		if err := p.cards.ValidateForRedeem(ctx, tx, c, svc.ServiceID, 1); err != nil {
			return err
		}
		before, after, err := p.cards.ApplyRedeem(ctx, tx, cardID, svc.ServiceID, 1, "", operatorID)
		if err != nil {
			return err
		}

		// redemption row (active_lock UNIQUE guards the one-valid-per-apt invariant)
		aptIDRef := aptID
		redemption = &Redemption{
			ID: shared.NewID(), AppointmentID: &aptIDRef, MemberID: apt.MemberID,
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
			if isDupKey(err) {
				// lost a race: same key → replay the winner; another valid
				// redemption exists (active_lock) → business conflict (W7)
				if winner, qerr := p.findRedemptionByIdem(ctx, tx, idemKey); qerr == nil && winner != nil {
					if winner.Status == "SUCCESS" {
						redemption = winner
						payment = p.paymentForRedemption(ctx, tx, winner.ID)
						return nil
					}
					return shared.Conflict("RDM_REVERSED", "该结算对应的核销已被撤销")
				}
				return shared.Conflict("RDM_DUPLICATE", "该预约已有一笔有效核销")
			}
			return wrapTxErr("RDM_INSERT", err)
		}

		// payment CARD (D1)
		payment = &Payment{
			ID: shared.NewID(), AppointmentID: &aptIDRef, MemberID: apt.MemberID,
			AmountCents: svc.PriceSnapshot * int64(svc.Quantity), Method: "CARD",
			Status: "VALID", Remark: "会员卡核销", IdemKey: idemKey,
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

// RedeemWalkIn settles a walk-in service with a member card — no appointment
// involved (D19): lock card → validate → deduct → REDEEM txn →
// redemption(appointment NULL) → payment(CARD, amount = service list price)
// → touch last_visit. Idempotent on idemKey; reversal reuses ReverseRedemption.
func (p *Provider) RedeemWalkIn(ctx context.Context, cardID, serviceID, operatorID, idemKey string) (*Redemption, *Payment, error) {
	if idemKey == "" {
		return nil, nil, shared.BadRequest("IDEM_KEY_REQUIRED", "缺少幂等键")
	}
	item, err := p.services.GetItem(ctx, serviceID)
	if err != nil {
		return nil, nil, err
	}
	if item.Status != "ACTIVE" {
		return nil, nil, shared.Conflict("SERVICE_INACTIVE", "服务项目已停用")
	}
	var redemption *Redemption
	var payment *Payment
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		existing, err := p.findRedemptionByIdem(ctx, tx, idemKey)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.Status != "SUCCESS" {
				return shared.Conflict("RDM_REVERSED", "该结算对应的核销已被撤销")
			}
			redemption = existing
			payment = p.paymentForRedemption(ctx, tx, existing.ID)
			return nil
		}

		c, err := p.cards.LockForRedeem(ctx, tx, cardID)
		if err != nil {
			return err
		}
		before, after, err := p.cards.ApplyRedeem(ctx, tx, cardID, serviceID, 1, "", operatorID)
		if err != nil {
			return err
		}

		redemption = &Redemption{
			ID: shared.NewID(), AppointmentID: nil, MemberID: c.MemberID,
			MemberCardID: cardID, ServiceID: serviceID,
			Quantity: 1, BeforeCount: before, AfterCount: after,
			Status: "SUCCESS", IdemKey: idemKey,
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO redemption (id, appointment_id, member_id, member_card_id, service_id, quantity, before_count, after_count, status, idempotency_key)
			 VALUES (?,?,?,?,?,?,?,?,?,?)`,
			redemption.ID, redemption.AppointmentID, redemption.MemberID, redemption.MemberCardID,
			redemption.ServiceID, redemption.Quantity, redemption.BeforeCount, redemption.AfterCount,
			redemption.Status, redemption.IdemKey); err != nil {
			if isDupKey(err) {
				if winner, qerr := p.findRedemptionByIdem(ctx, tx, idemKey); qerr == nil && winner != nil {
					if winner.Status == "SUCCESS" {
						redemption = winner
						payment = p.paymentForRedemption(ctx, tx, winner.ID)
						return nil
					}
					return shared.Conflict("RDM_REVERSED", "该结算对应的核销已被撤销")
				}
				return shared.Conflict("RDM_DUPLICATE", "重复的核销请求")
			}
			return wrapTxErr("RDM_INSERT", err)
		}

		payment = &Payment{
			ID: shared.NewID(), AppointmentID: nil, MemberID: c.MemberID,
			AmountCents: item.PriceCents, Method: "CARD",
			Status: "VALID", Remark: "散客核销", IdemKey: idemKey,
		}
		if err := insertPayment(ctx, tx, payment, operatorID); err != nil {
			return err
		}

		if err := p.members.TouchLastVisit(ctx, tx, c.MemberID, shared.NowShanghai()); err != nil {
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
// B1: the appointment row is locked BEFORE the VALID-count check, and the
// uk_payment_valid_lock unique index backstops concurrent inserters.
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
		// W2: idempotent replay by key.
		existing, err := p.findPaymentByIdem(ctx, tx, idemKey)
		if err != nil {
			return err
		}
		if existing != nil {
			payment = existing
			return nil
		}
		// B1: lock the appointment FIRST so concurrent settlements serialize
		// on the appointment row before the VALID-count check runs.
		apt, err := p.appointments.GetTx(ctx, tx, aptID)
		if err != nil {
			return err
		}
		// one VALID payment per appointment (D1) — belt, suspenders is the
		// uk_payment_valid_lock unique index.
		var n int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM payment WHERE appointment_id = ? AND status = 'VALID'`, aptID).Scan(&n); err != nil {
			return shared.Server("PAY_COUNT", err)
		}
		if n > 0 {
			return shared.Conflict("PAY_EXISTS", "该预约已有一笔有效收款")
		}
		if apt.Status != appointment.StatusInService && apt.Status != appointment.StatusCompleted {
			return shared.Conflict("APT_BAD_TRANSITION", "预约需处于服务中或已完成才能收款")
		}
		payment = &Payment{
			ID: shared.NewID(), AppointmentID: &aptID, MemberID: apt.MemberID,
			AmountCents: amountCents, Method: method, Status: "VALID",
			ReferenceNo: refNo, Remark: remark, IdemKey: idemKey,
		}
		return insertPayment(ctx, tx, payment, operatorID)
	})
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// ReverseRedemption undoes a redemption inside one transaction (§57):
// mark redemption REVERSED → restore count → REVERSAL txn → reversal row →
// original CARD payment VOIDED (D1). The appointment stays COMPLETED (§59).
// W7: the CARD row is locked before the redemption row (same order as the
// settle path) to avoid ABBA deadlocks.
func (p *Provider) ReverseRedemption(ctx context.Context, redemptionID, reason, operatorID string) error {
	if len(reason) > 500 {
		return shared.BadRequest("RDM_REASON_TOO_LONG", "撤销原因不能超过 500 字")
	}
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		// plain read only to locate the card (no lock yet — W7 lock order)
		rd, err := scanRedemption(tx.QueryRowContext(ctx,
			`SELECT `+redemptionColumns+` FROM redemption WHERE id = ?`, redemptionID))
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("RDM_NOT_FOUND", "核销记录不存在")
		}
		if err != nil {
			return shared.Server("RDM_QUERY", err)
		}
		// lock order: card row first (matches SettleByCard)
		if _, err := p.cards.LockForRedeem(ctx, tx, rd.MemberCardID); err != nil {
			return err
		}
		// now take the redemption row lock and re-verify status
		rd, err = scanRedemption(tx.QueryRowContext(ctx,
			`SELECT `+redemptionColumns+` FROM redemption WHERE id = ? FOR UPDATE`, redemptionID))
		if err != nil {
			return shared.Server("RDM_QUERY", err)
		}
		if rd.Status != "SUCCESS" {
			return shared.Conflict("RDM_ALREADY_REVERSED", "该核销已撤销")
		}
		// idempotency guard: unique reversal row (only a true duplicate maps
		// to ALREADY_REVERSED — W6)
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO redemption_reversal (id, redemption_id, reason, operator_id) VALUES (?,?,?,?)`,
			shared.NewID(), redemptionID, reason, operatorID); err != nil {
			if isDupKey(err) {
				return shared.Conflict("RDM_ALREADY_REVERSED", "该核销已撤销")
			}
			return wrapTxErr("RDM_REVERSAL_INSERT", err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE redemption SET status = 'REVERSED' WHERE id = ?`, redemptionID); err != nil {
			return wrapTxErr("RDM_REVERSE", err)
		}
		// restore balance (active_lock auto-clears via generated column);
		// CANCELLED/EXPIRED cards keep their status but the count is still
		// restored so the ledger stays consistent (W4/W5)
		if _, _, err := p.cards.ApplyReversal(ctx, tx, rd.MemberCardID, rd.Quantity, rd.ID, operatorID); err != nil {
			return err
		}
		// original CARD payment voided (D1). Located by the shared
		// idempotency key — appointment_id is NULL for walk-in redemptions.
		if _, err := tx.ExecContext(ctx,
			`UPDATE payment SET status = 'VOIDED', remark = CONCAT(remark, '；核销撤销')
			 WHERE idempotency_key = ? AND method = 'CARD' AND status = 'VALID'`, rd.IdemKey); err != nil {
			return wrapTxErr("PAY_VOID", err)
		}
		return nil
	})
}

// findPaymentByIdem returns a replayed non-card payment (W2).
func (p *Provider) findPaymentByIdem(ctx context.Context, tx shared.Tx, key string) (*Payment, error) {
	py, err := scanPayment(tx.QueryRowContext(ctx,
		`SELECT `+paymentColumns+` FROM payment WHERE idempotency_key = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, shared.Server("PAY_QUERY", err)
	}
	return py, nil
}

func (p *Provider) paymentForRedemption(ctx context.Context, tx shared.Tx, redemptionID string) *Payment {
	py, err := scanPayment(tx.QueryRowContext(ctx,
		`SELECT `+paymentColumns+` FROM payment WHERE idempotency_key =
		 (SELECT idempotency_key FROM redemption WHERE id = ?) AND method = 'CARD'`, redemptionID))
	if err != nil {
		return nil
	}
	return py
}

func insertPayment(ctx context.Context, tx shared.Tx, py *Payment, operatorID string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO payment (id, appointment_id, member_id, amount, method, status, reference_no, remark, idempotency_key, recorded_by)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		py.ID, py.AppointmentID, py.MemberID, py.AmountCents, py.Method, py.Status,
		py.ReferenceNo, py.Remark, py.IdemKey, operatorID)
	if err != nil {
		if isDupKey(err) {
			// valid_lock (one VALID per appointment) or idempotency replay race
			return shared.Conflict("PAY_EXISTS", "该预约已有一笔有效收款")
		}
		return wrapTxErr("PAY_INSERT", err)
	}
	return nil
}

func scanPayment(row interface{ Scan(...any) error }) (*Payment, error) {
	py := &Payment{}
	var aptID sql.NullString
	err := row.Scan(&py.ID, &aptID, &py.MemberID, &py.AmountCents, &py.Method,
		&py.Status, &py.ReferenceNo, &py.Remark, &py.IdemKey, &py.RecordedAt)
	if aptID.Valid {
		v := aptID.String
		py.AppointmentID = &v
	}
	return py, err
}

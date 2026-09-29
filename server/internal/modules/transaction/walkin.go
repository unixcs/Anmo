package transaction

import (
	"context"
	"strings"

	"anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
)

// walkin.go — 散客快速结算（D29，plan §四.4）：无预约无卡，手机号可选，现金/微信/
// 其他收款。单 immediate 事务内完成：服务项校验快照 → 会员归档（可选）→ payment →
// service_record（仅登记会员时落，D29：不录手机号仅记账无记录）。
// D19 的 RDM_WALKIN_BLOCKED（今日有预约会员拦卡核销）不套用于现金/微信散客结算
// （现金无卡无 D9 绕过问题）。

// WalkInInput — POST /admin/walkin/settle 入参。
type WalkInInput struct {
	Phone       string // 选填；非空 → 匹配/创建会员 + 服务记录归档
	Name        string // 选填；新建会员时落 name
	ServiceID   string
	PayMethod   string // CASH | WECHAT_TRANSFER | OTHER（CARD 拒绝）
	AmountCents int64  // ≥0 整数分
	ReferenceNo string
	Remark      string
	Record      RecordFields
	OperatorID  string
	IdemKey     string
}

// WalkInResult — 散客快速结算结果（响应 {payment, record, member_created}）。
type WalkInResult struct {
	Payment       *Payment
	Record        *service.ServiceRecord // nil = 未落记录（未录手机号，仅记账）
	MemberCreated bool
}

// WalkInSettle — 散客快速结算（D29）。幂等回放同 W2（按 uk_payment_idempotency
// 定位原收款并回放其记录）。
func (p *Provider) WalkInSettle(ctx context.Context, in WalkInInput) (*WalkInResult, error) {
	if in.IdemKey == "" {
		return nil, shared.BadRequest("IDEM_KEY_REQUIRED", "缺少幂等键")
	}
	if !in.Record.Communicated {
		return nil, shared.BadRequest("TX_NEED_CONFIRM", "请先勾选「服务前已完成沟通」")
	}
	switch in.PayMethod {
	case "CARD":
		return nil, shared.BadRequest("TX_WALKIN_NO_CARD", "无卡结算不支持卡核销方式")
	case "CASH", "WECHAT_TRANSFER", "OTHER":
	default:
		return nil, shared.BadRequest("PAY_BAD_METHOD", "结算方式不支持")
	}
	if in.AmountCents < 0 {
		return nil, shared.BadRequest("TX_WALKIN_AMOUNT", "金额不能为负")
	}
	phone := strings.TrimSpace(in.Phone)

	res := &WalkInResult{}
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		// W2: idempotent replay by key.
		existing, err := p.findPaymentByIdem(ctx, tx, in.IdemKey)
		if err != nil {
			return err
		}
		if existing != nil {
			res.Payment = existing
			res.Record, err = p.services.RecordByPaymentTx(ctx, tx, existing.ID)
			if err != nil {
				return err
			}
			return nil
		}

		// 服务项校验（事务内）：存在且在架，快照 name + 默认价（TX_WALKIN_SERVICE）
		item, err := p.services.GetItemTx(ctx, tx, in.ServiceID)
		if err != nil {
			// 仅"不存在"映射为 TX_WALKIN_SERVICE；DB 故障等其余错误原样上抛，不吞
			if shared.Is(err, "SERVICE_NOT_FOUND") {
				return shared.NotFound("TX_WALKIN_SERVICE", "服务项目不存在或已下架")
			}
			return err
		}
		if item.Status != "ACTIVE" {
			return shared.Conflict("TX_WALKIN_SERVICE", "服务项目不存在或已下架")
		}

		// 会员归档（可选）：录手机号 → 匹配/创建会员（D29 §四.4）
		var memberID string
		if phone != "" {
			id, created, err := p.members.EnsureByPhoneTx(ctx, tx, phone, in.Name)
			if err != nil {
				return err
			}
			memberID = id
			res.MemberCreated = created
		}

		payment := &Payment{
			ID: shared.NewID(), AppointmentID: nil,
			AmountCents: in.AmountCents, Method: in.PayMethod, Status: "VALID",
			ReferenceNo: in.ReferenceNo, Remark: in.Remark, IdemKey: in.IdemKey,
		}
		if memberID != "" {
			ref := memberID
			payment.MemberID = &ref
		}
		if err := insertPayment(ctx, tx, payment, in.OperatorID); err != nil {
			return err
		}
		res.Payment = payment

		// 服务记录：仅登记会员时落（不录手机号 → 仅记账，D29）
		if memberID != "" {
			if err := p.services.InsertRecordTx(ctx, tx, service.RecordInput{
				MemberID: memberID, PaymentID: payment.ID,
				ServiceID: item.ID, ServiceName: item.Name,
				BodyParts: in.Record.BodyParts, Method: in.Record.ServiceMethod,
				TechNote: in.Record.TechNote,
				Communicated: in.Record.Communicated, OperatorID: in.OperatorID,
			}); err != nil {
				return err
			}
			if res.Record, err = p.services.RecordByPaymentTx(ctx, tx, payment.ID); err != nil {
				return err
			}
		}
		// 到店事实：登记会员时更新最近到店（与其他结算链路一致）
		if memberID != "" {
			if err := p.members.TouchLastVisit(ctx, tx, memberID, shared.NowShanghai()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

package transaction

import (
	"context"
	"testing"

	svcmodule "anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
)

// walkinsettle_test.go — 服务记录接入与散客快速结算（D28/D29）：
// 四入口 communicated 强制；walkin 有/无手机号、CARD 拒绝、金额/服务校验、
// 幂等回放；ReverseRedemption 后记录 REVERSED、重新核销产生新 ACTIVE 记录。

func TestFourEntriesRequireCommunicated(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	aptID := e.bookInService(t, 1, 10, 0)

	// 1) 卡核销（预约结算）
	if _, _, err := e.p.SettleByCard(ctx, aptID, e.card, "", "op-1", "idem-nc-1", RecordFields{}); !shared.Is(err, "TX_NEED_CONFIRM") {
		t.Fatalf("SettleByCard without communicated: err = %v", err)
	}
	// 2) 散客卡核销
	if _, _, err := e.p.RedeemWalkIn(ctx, e.card, e.svcID, "op-1", "idem-nc-2", RecordFields{}); !shared.Is(err, "TX_NEED_CONFIRM") {
		t.Fatalf("RedeemWalkIn without communicated: err = %v", err)
	}
	// 3) 现金/微信结算
	if _, err := e.p.SettleByPay(ctx, aptID, "CASH", 12800, "", "", "op-1", "idem-nc-3", RecordFields{}); !shared.Is(err, "TX_NEED_CONFIRM") {
		t.Fatalf("SettleByPay without communicated: err = %v", err)
	}
	// 4) 散客快速结算
	_, err := e.p.WalkInSettle(ctx, WalkInInput{ServiceID: e.svcID, PayMethod: "CASH", AmountCents: 12800, IdemKey: "idem-nc-4"})
	if !shared.Is(err, "TX_NEED_CONFIRM") {
		t.Fatalf("WalkInSettle without communicated: err = %v", err)
	}

	// 四入口均未产生任何 payment
	pays, _ := e.p.ListPayments(ctx, "")
	if len(pays) != 0 {
		t.Fatalf("no payment should exist after NEED_CONFIRM rejections, got %d", len(pays))
	}
}

func TestWalkInSettleWithPhoneArchivesMemberAndRecord(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()

	res, err := e.p.WalkInSettle(ctx, WalkInInput{
		Phone: "13911112222", Name: "散客张三",
		ServiceID: e.svcID, PayMethod: "WECHAT_TRANSFER", AmountCents: 12800,
		ReferenceNo: "wx-888", Remark: "散客微信",
		Record: RecordFields{
			BodyParts: []string{"肩颈", "背部"}, ServiceMethod: "推拿",
			TechNote: "首次到店，力度中等", Communicated: true,
		},
		OperatorID: "op-1", IdemKey: "walkin-settle-1",
	})
	if err != nil {
		t.Fatalf("walkin settle: %v", err)
	}
	if !res.MemberCreated {
		t.Fatalf("member_created = false, want true (new phone)")
	}
	if res.Payment == nil || res.Payment.MemberID == nil || *res.Payment.MemberID == "" {
		t.Fatalf("payment.member_id must be set, got %+v", res.Payment)
	}
	if res.Record == nil {
		t.Fatalf("record must be written for registered member")
	}
	if res.Record.ServiceName != "肩颈按摩" || res.Record.ServiceID != e.svcID {
		t.Fatalf("record snapshot = %s/%s", res.Record.ServiceName, res.Record.ServiceID)
	}
	if len(res.Record.BodyParts) != 2 || res.Record.BodyParts[0] != "肩颈" {
		t.Fatalf("record body_parts = %v", res.Record.BodyParts)
	}
	if res.Record.ServiceMethod != "推拿" || res.Record.TechNote != "首次到店，力度中等" || !res.Record.Communicated {
		t.Fatalf("record fields = %+v", res.Record)
	}
	if res.Record.AppointmentID != nil || res.Record.RedemptionID != nil {
		t.Fatalf("walkin record must not link appointment/redemption")
	}

	// 幂等回放：同 key 返回原 payment，记录按 payment 重建
	res2, err := e.p.WalkInSettle(ctx, WalkInInput{
		Phone: "13911112222", ServiceID: e.svcID, PayMethod: "CASH", AmountCents: 1,
		Record: RecordFields{Communicated: true}, OperatorID: "op-1", IdemKey: "walkin-settle-1",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if res2.Payment.ID != res.Payment.ID {
		t.Fatalf("replay payment = %s, want %s", res2.Payment.ID, res.Payment.ID)
	}
	if res2.MemberCreated {
		t.Fatalf("replay must not report member_created")
	}
	if res2.Record == nil || res2.Record.ID != res.Record.ID {
		t.Fatalf("replay record = %+v", res2.Record)
	}

	// 再次到店（新 key，同手机号）→ 匹配既有会员，不新建
	res3, err := e.p.WalkInSettle(ctx, WalkInInput{
		Phone: "13911112222", ServiceID: e.svcID, PayMethod: "CASH", AmountCents: 12800,
		Record: RecordFields{Communicated: true}, OperatorID: "op-1", IdemKey: "walkin-settle-2",
	})
	if err != nil {
		t.Fatalf("second visit: %v", err)
	}
	if res3.MemberCreated {
		t.Fatalf("second visit must match existing member, not create")
	}
	// 记录落库两条 ACTIVE
	recs, _, _, err := e.svc.ListMemberRecords(ctx, *res.Payment.MemberID, "", "", "", "")
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("member records = %d, want 2", len(recs))
	}
}

func TestWalkInSettleWithoutPhoneAccountingOnly(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()

	res, err := e.p.WalkInSettle(ctx, WalkInInput{
		ServiceID: e.svcID, PayMethod: "CASH", AmountCents: 9900,
		Record: RecordFields{BodyParts: []string{"肩颈"}, ServiceMethod: "推拿", TechNote: "x", Communicated: true},
		OperatorID: "op-1", IdemKey: "walkin-anon-1",
	})
	if err != nil {
		t.Fatalf("walkin settle: %v", err)
	}
	if res.MemberCreated {
		t.Fatalf("no phone → member_created must be false")
	}
	if res.Payment == nil || res.Payment.MemberID != nil {
		t.Fatalf("no phone → payment.member_id must be NULL, got %+v", res.Payment)
	}
	if res.Record != nil {
		t.Fatalf("no phone → record must be nil (仅记账), got %+v", res.Record)
	}

	// 回放：record 仍为 nil，payment 不变
	res2, err := e.p.WalkInSettle(ctx, WalkInInput{
		ServiceID: e.svcID, PayMethod: "CASH", AmountCents: 9900,
		Record: RecordFields{Communicated: true}, OperatorID: "op-1", IdemKey: "walkin-anon-1",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if res2.Payment.ID != res.Payment.ID || res2.Record != nil || res2.MemberCreated {
		t.Fatalf("replay = %+v", res2)
	}
}

func TestWalkInSettleValidation(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	base := WalkInInput{ServiceID: e.svcID, PayMethod: "CASH", AmountCents: 100,
		Record: RecordFields{Communicated: true}, OperatorID: "op-1"}

	// CARD 拒绝（D29）
	in := base
	in.PayMethod, in.IdemKey = "CARD", "wv-card"
	if _, err := e.p.WalkInSettle(ctx, in); !shared.Is(err, "TX_WALKIN_NO_CARD") {
		t.Fatalf("CARD method: err = %v", err)
	}
	// 非法收款方式
	in = base
	in.PayMethod, in.IdemKey = "ALIPAY", "wv-badmethod"
	if _, err := e.p.WalkInSettle(ctx, in); !shared.Is(err, "PAY_BAD_METHOD") {
		t.Fatalf("ALIPAY method: err = %v", err)
	}
	// 负金额
	in = base
	in.AmountCents, in.IdemKey = -1, "wv-neg"
	if _, err := e.p.WalkInSettle(ctx, in); !shared.Is(err, "TX_WALKIN_AMOUNT") {
		t.Fatalf("negative amount: err = %v", err)
	}
	// 缺幂等键
	in = base
	in.IdemKey = ""
	if _, err := e.p.WalkInSettle(ctx, in); !shared.Is(err, "IDEM_KEY_REQUIRED") {
		t.Fatalf("empty idem key: err = %v", err)
	}
	// 服务不存在
	in = base
	in.ServiceID, in.IdemKey, in.Phone = "01ARZ3NDEKTSV4RRFFQ69G5FAV", "wv-nosvc", "13900009999"
	if _, err := e.p.WalkInSettle(ctx, in); !shared.Is(err, "TX_WALKIN_SERVICE") {
		t.Fatalf("unknown service: err = %v", err)
	}
	// 服务已下架
	it, err := e.svc.CreateItem(ctx, svcmodule.NewItem{CategoryID: e.catID, Name: "下架项目", DurationMin: 30, PriceCents: 5000})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := e.svc.SetItemStatus(ctx, it.ID, "INACTIVE"); err != nil {
		t.Fatalf("disable item: %v", err)
	}
	in = base
	in.ServiceID, in.IdemKey, in.Phone = it.ID, "wv-offsvc", "13900009999"
	if _, err := e.p.WalkInSettle(ctx, in); !shared.Is(err, "TX_WALKIN_SERVICE") {
		t.Fatalf("disabled service: err = %v", err)
	}
	// 校验失败的请求均未落 payment
	pays, _ := e.p.ListPayments(ctx, "")
	if len(pays) != 0 {
		t.Fatalf("no payment should exist after validation failures, got %d", len(pays))
	}
}

func TestReverseRedemptionReversesServiceRecord(t *testing.T) {
	e := newTxnEnv(t)
	ctx := context.Background()
	aptID := e.bookInService(t, 1, 10, 0)

	rd, py, err := e.p.SettleByCard(ctx, aptID, e.card, "", "op-1", "idem-rec-1", RecordFields{
		BodyParts: []string{"肩颈"}, ServiceMethod: "推拿", TechNote: "首次核销", Communicated: true,
	})
	if err != nil {
		t.Fatalf("settle by card: %v", err)
	}

	// 记录已落：关联 appointment + redemption，快照同 redemption 来源
	recs, _, _, err := e.svc.ListMemberRecords(ctx, e.mbrID, "", "", "", "")
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1", len(recs))
	}
	rec := recs[0]
	if rec.AppointmentID == nil || *rec.AppointmentID != aptID ||
		rec.RedemptionID == nil || *rec.RedemptionID != rd.ID || rec.PaymentID != py.ID {
		t.Fatalf("record linkage = %+v", rec)
	}
	if rec.Status != "ACTIVE" || rec.ServiceName != "肩颈按摩" {
		t.Fatalf("record = %+v", rec)
	}

	// 撤销核销 → 记录 REVERSED（联动，非删行）
	if err := e.p.ReverseRedemption(ctx, rd.ID, "误核销", "op-1"); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	var status string
	if err := e.p.db.QueryRowContext(ctx,
		`SELECT status FROM service_record WHERE id = ?`, rec.ID).Scan(&status); err != nil {
		t.Fatalf("query record: %v", err)
	}
	if status != "REVERSED" {
		t.Fatalf("record status = %s, want REVERSED", status)
	}
	// 列表只显示 ACTIVE → 撤销后为空
	recs, _, _, _ = e.svc.ListMemberRecords(ctx, e.mbrID, "", "", "", "")
	if len(recs) != 0 {
		t.Fatalf("after reversal ACTIVE records = %d, want 0", len(recs))
	}

	// 撤销后重新核销（COMPLETED 放行）→ 新记录 ACTIVE
	rd2, py2, err := e.p.SettleByCard(ctx, aptID, e.card, "", "op-1", "idem-rec-2", RecordFields{Communicated: true})
	if err != nil {
		t.Fatalf("re-redeem: %v", err)
	}
	if rd2.ID == rd.ID || py2.ID == py.ID {
		t.Fatalf("re-redeem must create new rows")
	}
	recs, _, _, _ = e.svc.ListMemberRecords(ctx, e.mbrID, "", "", "", "")
	if len(recs) != 1 || recs[0].ID == rec.ID {
		t.Fatalf("re-redeem records = %+v", recs)
	}
}

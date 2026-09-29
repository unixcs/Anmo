package service

import (
	"context"
	"encoding/json"
	"testing"

	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// tags_records_test.go — D28：服务标签 CRUD/used 守卫 + 服务记录事务入口/撤销/追踪。

func newServiceTestEnv(t *testing.T) (*Provider, string, string) {
	t.Helper()
	p := New(testsupport.NewSchemaDB(t, "anmo_svctag_"), nil)
	// 固定 member/payment 夹具（跨模块表，测试内直插）
	ctx := context.Background()
	const mid = "01JTAGT0000000000000000MB1"
	if _, err := p.db.ExecContext(ctx,
		`INSERT INTO member (id, member_no, name, phone) VALUES (?, 'M202609290001', '记录测试', '13811112222')`, mid); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	const pid = "01JTAGT0000000000000000PB1"
	if _, err := p.db.ExecContext(ctx,
		`INSERT INTO payment (id, member_id, amount, method, status) VALUES (?, ?, 12800, 'CASH', 'VALID')`, pid, mid); err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	return p, mid, pid
}

func insertRecord(t *testing.T, p *Provider, in RecordInput) {
	t.Helper()
	if err := shared.RunInTx(context.Background(), p.db, func(tx shared.Tx) error {
		return p.InsertRecordTx(context.Background(), tx, in)
	}); err != nil {
		t.Fatalf("InsertRecordTx: %v", err)
	}
}

func TestServiceTagCRUDAndUsedGuard(t *testing.T) {
	p, mid, pid := newServiceTestEnv(t)
	ctx := context.Background()

	// 添加：sort = 组内 max+1
	bp, err := p.CreateServiceTag(ctx, GroupBodyPart, "肩颈")
	if err != nil {
		t.Fatalf("create body part: %v", err)
	}
	bp2, err := p.CreateServiceTag(ctx, GroupBodyPart, "腰背")
	if err != nil || bp2.Sort != bp.Sort+1 {
		t.Fatalf("sort seq: %+v err=%v", bp2, err)
	}
	if _, err := p.CreateServiceTag(ctx, GroupMethod, "按摩"); err != nil {
		t.Fatalf("create method: %v", err)
	}

	// 同组同名 409
	if _, err := p.CreateServiceTag(ctx, GroupBodyPart, "肩颈"); !shared.Is(err, "TAG_EXISTS") {
		t.Fatalf("want TAG_EXISTS, got %v", err)
	}
	// 非法名 / 非法组
	if _, err := p.CreateServiceTag(ctx, GroupBodyPart, `a"b`); !shared.Is(err, "TAG_BAD_NAME") {
		t.Fatalf("want TAG_BAD_NAME (quote), got %v", err)
	}
	if _, err := p.CreateServiceTag(ctx, GroupBodyPart, "ab_cd"); !shared.Is(err, "TAG_BAD_NAME") {
		t.Fatalf("want TAG_BAD_NAME (underscore), got %v", err)
	}
	if _, err := p.CreateServiceTag(ctx, "AREA", "全身"); !shared.Is(err, "TAG_BAD_GROUP") {
		t.Fatalf("want TAG_BAD_GROUP, got %v", err)
	}

	// 未使用可删
	if err := p.DeleteServiceTag(ctx, bp2.ID); err != nil {
		t.Fatalf("delete unused tag: %v", err)
	}

	// 使用后删除 → TAG_IN_USE；停用后列表仍可见（含 DISABLED）
	insertRecord(t, p, RecordInput{
		MemberID: mid, PaymentID: pid, ServiceID: "01JSVC0000000000000000SV01",
		ServiceName: "肩颈按摩", BodyParts: []string{"肩颈"}, Method: "按摩", Communicated: true,
	})
	if err := p.DeleteServiceTag(ctx, bp.ID); !shared.Is(err, "TAG_IN_USE") {
		t.Fatalf("want TAG_IN_USE, got %v", err)
	}
	if _, err := p.UpdateServiceTag(ctx, bp.ID, nil, nil, strPtr("DISABLED")); err != nil {
		t.Fatalf("disable tag: %v", err)
	}
	all, err := p.ListServiceTags(ctx, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("list = %+v err=%v", all, err)
	}
	for _, tg := range all {
		if tg.Name == "肩颈" && (!tg.Used || tg.Status != "DISABLED") {
			t.Fatalf("used tag state wrong: %+v", tg)
		}
	}
	// 停用标签不出现在录入 chips，但历史筛选仍可见
	parts, methods, err := p.ActiveTagNames(ctx, "")
	if err != nil {
		t.Fatalf("active names: %v", err)
	}
	if len(parts) != 0 || len(methods) != 1 {
		t.Fatalf("active chips = %v %v", parts, methods)
	}
	fp, fm, err := p.AllTagNames(ctx)
	if err != nil || len(fp) != 1 || len(fm) != 1 {
		t.Fatalf("filter names = %v %v err=%v", fp, fm, err)
	}
}

func TestInsertRecordTxValidation(t *testing.T) {
	p, mid, pid := newServiceTestEnv(t)

	run := func(in RecordInput) error {
		err := shared.RunInTx(context.Background(), p.db, func(tx shared.Tx) error {
			return p.InsertRecordTx(context.Background(), tx, in)
		})
		return err
	}
	base := RecordInput{MemberID: mid, PaymentID: pid, ServiceID: "01JSVC0000000000000000SV01", ServiceName: "肩颈按摩", Communicated: true}

	// communicated 未勾选 → 拒绝
	bad := base
	bad.Communicated = false
	if err := run(bad); !shared.Is(err, "SERVICE_NEED_CONFIRM") {
		t.Fatalf("want SERVICE_NEED_CONFIRM, got %v", err)
	}
	// 部位 >3
	bad = base
	bad.BodyParts = []string{"a", "b", "c", "d"}
	if err := run(bad); !shared.Is(err, "REC_BAD_PARTS") {
		t.Fatalf("want REC_BAD_PARTS, got %v", err)
	}
	// 备注超长
	bad = base
	bad.TechNote = string(make([]rune, 201))
	for i := range bad.TechNote {
		([]rune(bad.TechNote))[i] = '备'
	}
	if err := run(bad); !shared.Is(err, "REC_BAD_TECH_NOTE") {
		t.Fatalf("want REC_BAD_TECH_NOTE, got %v", err)
	}
	// 正常落库 + 快照
	if err := run(RecordInput{
		MemberID: mid, PaymentID: pid, ServiceID: "01JSVC0000000000000000SV01",
		ServiceName: "肩颈按摩", BodyParts: []string{" 肩颈 ", "", "腰背"}, Method: " 按摩 ", TechNote: " 力度适中 ",
		Communicated: true, OperatorID: "op-1",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	recs, summary, _, err := p.ListMemberRecords(context.Background(), mid, "", "", "", "")
	if err != nil || len(recs) != 1 {
		t.Fatalf("list = %+v err=%v", recs, err)
	}
	if recs[0].BodyParts[0] != "肩颈" || recs[0].ServiceMethod != "按摩" || recs[0].TechNote != "力度适中" {
		t.Fatalf("snapshot = %+v", recs[0])
	}
	if !recs[0].Communicated || recs[0].Status != "ACTIVE" {
		t.Fatalf("flags = %+v", recs[0])
	}
	if summary.TotalAll != 1 || summary.Total3m != 1 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestRevokeRecordBranches(t *testing.T) {
	p, mid, pid := newServiceTestEnv(t)
	ctx := context.Background()

	// 散客记录（无 redemption）→ 可独立撤销：REVERSED + payment VOIDED
	insertRecord(t, p, RecordInput{MemberID: mid, PaymentID: pid, ServiceID: "01JSVC0000000000000000SV01", ServiceName: "肩颈按摩", Communicated: true})
	recs, _, _, err := p.ListMemberRecords(ctx, mid, "", "", "", "")
	if err != nil || len(recs) != 1 {
		t.Fatalf("list: %v", err)
	}
	recID := recs[0].ID
	if err := p.RevokeRecord(ctx, recID, "op-1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	var status, reversed string
	var payStatus string
	if err := p.db.QueryRowContext(ctx, `SELECT status, COALESCE(reversed_at,''), (SELECT status FROM payment WHERE id = ?) FROM service_record WHERE id = ?`, pid, recID).Scan(&status, &reversed, &payStatus); err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "REVERSED" || reversed == "" || payStatus != "VOIDED" {
		t.Fatalf("after revoke: %s %q %s", status, reversed, payStatus)
	}
	// 撤销后不再进追踪流
	recs, summary, _, _ := p.ListMemberRecords(ctx, mid, "", "", "", "")
	if len(recs) != 0 || summary.TotalAll != 0 {
		t.Fatalf("reversed still counted: %+v %+v", recs, summary)
	}
	// 重复撤销 → 409
	if err := p.RevokeRecord(ctx, recID, "op-1"); !shared.Is(err, "REC_ALREADY_REVERSED") {
		t.Fatalf("want REC_ALREADY_REVERSED, got %v", err)
	}

	// 卡核销记录（redemption_id 非空）→ 拒绝独立撤销
	const rid = "01JTAGT0000000000000000RD1"
	seedCardRedemption(t, p, mid, rid, "idem-rev-1")
	const pid2 = "01JTAGT0000000000000000PB2"
	if _, err := p.db.ExecContext(ctx,
		`INSERT INTO payment (id, member_id, amount, method, status) VALUES (?, ?, 0, 'CARD', 'VALID')`, pid2, mid); err != nil {
		t.Fatalf("seed payment2: %v", err)
	}
	insertRecord(t, p, RecordInput{MemberID: mid, PaymentID: pid2, RedemptionID: rid, ServiceID: "01JSVC0000000000000000SV01", ServiceName: "肩颈按摩", Communicated: true})
	recs2, _, _, _ := p.ListMemberRecords(ctx, mid, "", "", "", "")
	var cardRecID string
	for _, r := range recs2 {
		if r.RedemptionID != nil && *r.RedemptionID == rid {
			cardRecID = r.ID
		}
	}
	if cardRecID == "" {
		t.Fatal("card record not found")
	}
	if err := p.RevokeRecord(ctx, cardRecID, "op-1"); !shared.Is(err, "REC_CARD_REVERSAL") {
		t.Fatalf("want REC_CARD_REVERSAL, got %v", err)
	}
	// 卡核销记录未被撤销、payment 仍 VALID
	var st, pst string
	if err := p.db.QueryRowContext(ctx,
		`SELECT status, (SELECT status FROM payment WHERE id = ?) FROM service_record WHERE id = ?`, pid2, cardRecID).Scan(&st, &pst); err != nil || st != "ACTIVE" || pst != "VALID" {
		t.Fatalf("card record after blocked revoke: %s %s err=%v", st, pst, err)
	}
}

func TestReverseByRedemptionTxIdempotent(t *testing.T) {
	p, mid, pid := newServiceTestEnv(t)
	ctx := context.Background()
	const rid = "01JTAGT0000000000000000RD2"
	seedCardRedemption(t, p, mid, rid, "idem-rev-2")
	insertRecord(t, p, RecordInput{MemberID: mid, PaymentID: pid, RedemptionID: rid, ServiceID: "01JSVC0000000000000000SV01", ServiceName: "肩颈按摩", Communicated: true})

	// 双次执行幂等
	for i := 0; i < 2; i++ {
		if err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
			return p.ReverseByRedemptionTx(ctx, tx, rid)
		}); err != nil {
			t.Fatalf("reverse #%d: %v", i, err)
		}
	}
	var n int
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM service_record WHERE redemption_id = ? AND status = 'REVERSED'`, rid).Scan(&n); err != nil || n != 1 {
		t.Fatalf("reversed rows = %d err=%v", n, err)
	}
	// 未知的 redemptionID 也不报错（幂等）
	if err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		return p.ReverseByRedemptionTx(ctx, tx, "01JNOPE0000000000000000RD9")
	}); err != nil {
		t.Fatalf("reverse unknown: %v", err)
	}
}

func TestTrackFiltersAndSummary(t *testing.T) {
	p, mid, pid := newServiceTestEnv(t)
	ctx := context.Background()

	if _, err := p.CreateServiceTag(ctx, GroupBodyPart, "肩颈"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateServiceTag(ctx, GroupBodyPart, "腰背"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateServiceTag(ctx, GroupMethod, "按摩"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateServiceTag(ctx, GroupMethod, "艾灸"); err != nil {
		t.Fatal(err)
	}

	const pid2 = "01JTAGT0000000000000000PB3"
	if _, err := p.db.ExecContext(ctx,
		`INSERT INTO payment (id, member_id, amount, method, status) VALUES (?, ?, 6800, 'WECHAT_TRANSFER', 'VALID')`, pid2, mid); err != nil {
		t.Fatal(err)
	}
	insertRecord(t, p, RecordInput{MemberID: mid, PaymentID: pid, ServiceID: "01JSVC0000000000000000SV01", ServiceName: "肩颈按摩", BodyParts: []string{"肩颈"}, Method: "按摩", TechNote: "肩颈重点", Communicated: true})
	insertRecord(t, p, RecordInput{MemberID: mid, PaymentID: pid2, ServiceID: "01JSVC0000000000000000SV02", ServiceName: "艾灸调理", BodyParts: []string{"腰背", "肩颈"}, Method: "艾灸", Communicated: true})

	// 倒序：第二条在前
	recs, summary, filters, err := p.ListMemberRecords(ctx, mid, "", "", "", "")
	if err != nil || len(recs) != 2 {
		t.Fatalf("list: %v", err)
	}
	if recs[0].ServiceName != "艾灸调理" || recs[1].ServiceName != "肩颈按摩" {
		t.Fatalf("order wrong: %+v", recs)
	}
	if len(summary.Parts) != 2 || len(summary.Methods) != 2 || summary.TotalAll != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.Parts[0].Name != "肩颈" || summary.Parts[0].Count != 2 {
		t.Fatalf("parts summary = %+v", summary.Parts)
	}
	if len(filters.Parts) != 2 || len(filters.Methods) != 2 {
		t.Fatalf("filters = %+v", filters)
	}
	// payment 装饰
	if recs[0].PaymentMethod != "WECHAT_TRANSFER" || recs[0].PaymentAmount != 6800 {
		t.Fatalf("payment deco = %+v", recs[0])
	}

	// 部位筛选
	if recs, _, _, _ = p.ListMemberRecords(ctx, mid, "腰背", "", "", ""); len(recs) != 1 || recs[0].ServiceMethod != "艾灸" {
		t.Fatalf("part filter = %+v", recs)
	}
	// 方式筛选
	if recs, _, _, _ = p.ListMemberRecords(ctx, mid, "", "按摩", "", ""); len(recs) != 1 || recs[0].ServiceName != "肩颈按摩" {
		t.Fatalf("method filter = %+v", recs)
	}
	// 关键词命中备注
	if recs, _, _, _ = p.ListMemberRecords(ctx, mid, "", "", "", "重点"); len(recs) != 1 {
		t.Fatalf("keyword filter = %+v", recs)
	}
	// 关键词命中服务名
	if recs, _, _, _ = p.ListMemberRecords(ctx, mid, "", "", "", "艾灸"); len(recs) != 1 {
		t.Fatalf("keyword service filter = %+v", recs)
	}
}

func TestMerchantNoteUpdate(t *testing.T) {
	p, mid, pid := newServiceTestEnv(t)
	ctx := context.Background()
	insertRecord(t, p, RecordInput{MemberID: mid, PaymentID: pid, ServiceID: "01JSVC0000000000000000SV01", ServiceName: "肩颈按摩", Communicated: true})
	recs, _, _, _ := p.ListMemberRecords(ctx, mid, "", "", "", "")
	id := recs[0].ID

	rec, err := p.UpdateMerchantNote(ctx, id, "顾客偏好轻力度", "op-9")
	if err != nil {
		t.Fatalf("note: %v", err)
	}
	if rec.MerchantNote != "顾客偏好轻力度" || rec.MerchantNoteByName != "" {
		// op-9 不是真实 identity_user → name 为空
		t.Fatalf("note rec = %+v", rec)
	}
	var by string
	if err := p.db.QueryRowContext(ctx, `SELECT merchant_note_by FROM service_record WHERE id = ?`, id).Scan(&by); err != nil || by != "op-9" {
		t.Fatalf("note by = %q err=%v", by, err)
	}
}

func strPtr(s string) *string { return &s }

// seedCardRedemption — 造一条合法卡核销夹具（card_template → member_card → redemption）。
func seedCardRedemption(t *testing.T, p *Provider, mid, rid, rdmIDem string) {
	t.Helper()
	ctx := context.Background()
	const tplID = "01JTAGT0000000000000000TP01"
	const cardID = "01JTAGT0000000000000000CD01"
	if _, err := p.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO card_template (id, name, type, total_count, price) VALUES (?, '测试卡模板', 'COUNT', 10, 0)`, tplID); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if _, err := p.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO member_card (id, member_id, card_template_id, total_count, remaining_count, valid_from)
		 VALUES (?, ?, ?, 10, 9, '2026-09-29')`, cardID, mid, tplID); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	if _, err := p.db.ExecContext(ctx,
		`INSERT INTO redemption (id, member_id, member_card_id, service_id, quantity, before_count, after_count, status, idempotency_key)
		 VALUES (?, ?, ?, '01JSVC0000000000000000SV01', 1, 10, 9, 'SUCCESS', ?)`, rid, mid, cardID, rdmIDem); err != nil {
		t.Fatalf("seed redemption: %v", err)
	}
}

// jsonOf 便于调试断言（保留给人工排查）
func jsonOf(v any) string { b, _ := json.Marshal(v); return string(b) }

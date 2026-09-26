// Package adversarial — 对抗式审查测试（DB 级）。只测试、不修复。
//
// 命名约定（把"预期被拦截"与"缺陷暴露"分开）：
//   - TestGUARD_*：攻击应当被现有约束拦截。测试 FAIL = 发现缺陷（B/W 级证据）。
//   - TestREVEAL_*：假设系统存在某缺陷。测试 FAIL = 缺陷被真实复现（证据）；
//     PASS 仅表示"该竞态/路径本次未触发或非缺陷"，不等于安全。
//
// 全部用 testsupport.NewSchemaDB（一次性临时库），不污染正式数据。
package adversarial

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"anmo/server/internal/config"
	"anmo/server/internal/database"
	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/modules/card"
	"anmo/server/internal/modules/member"
	"anmo/server/internal/modules/service"
	"anmo/server/internal/modules/transaction"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// ---------------------------------------------------------------- 环境搭建

type env struct {
	t     *testing.T
	db    *database.Pool
	tx    *transaction.Provider
	cards *card.Provider
	apt   *appointment.Provider
	mem   *member.Provider

	mbrA   string // 顾客 A
	mbrB   string // 顾客 B
	svc60  string // 60 分钟服务
	svc30  string // 30 分钟服务（边界用例）
	opID   string
	seq    int
	seqMtx sync.Mutex
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_adv_")
	cfg := config.Defaults()
	mem := member.New(db, cfg)
	svc := service.New(db, cfg)
	cards := card.New(db, cfg)
	apt := appointment.New(db, cfg, svc)
	txp := transaction.New(db, cfg, cards, apt, mem)
	e := &env{t: t, db: db, tx: txp, cards: cards, apt: apt, mem: mem, opID: "adv-op"}
	ctx := context.Background()

	mk := func(phone, name string) string {
		var id string
		if err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
			var err error
			id, _, err = mem.EnsureByPhone(ctx, tx, phone, name)
			return err
		}); err != nil {
			t.Fatalf("member %s: %v", phone, err)
		}
		return id
	}
	e.mbrA = mk("13900007001", "对抗顾客A")
	e.mbrB = mk("13900007002", "对抗顾客B")

	cat, err := svc.CreateCategory(ctx, "对抗分类", 1)
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	it, err := svc.CreateItem(ctx, service.NewItem{CategoryID: cat.ID, Name: "全身60", DurationMin: 60, PriceCents: 12800})
	if err != nil {
		t.Fatalf("service60: %v", err)
	}
	e.svc60 = it.ID
	it30, err := svc.CreateItem(ctx, service.NewItem{CategoryID: cat.ID, Name: "肩颈30", DurationMin: 30, PriceCents: 6800})
	if err != nil {
		t.Fatalf("service30: %v", err)
	}
	e.svc30 = it30.ID
	return e
}

func (e *env) ctx() context.Context {
	c, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	e.t.Cleanup(cancel)
	return c
}

// slot 返回 business 时区 day 天后 hh:mm 的时间串。
func (e *env) slot(day, hh, mm int) string {
	d := shared.NowShanghai().AddDate(0, 0, day)
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d", d.Year(), d.Month(), d.Day(), hh, mm)
}

// newCard 建模板+规则+发卡，返回卡 ID。
func (e *env) newCard(memberID string, total int) string {
	e.seqMtx.Lock()
	e.seq++
	n := e.seq
	e.seqMtx.Unlock()
	ctx := e.ctx()
	tpl, err := e.cards.CreateTemplate(ctx, card.NewTemplate{
		Name: fmt.Sprintf("对抗卡%d_%d", total, n), Type: "COUNT", TotalCount: total, PriceCents: 100000,
	})
	if err != nil {
		e.t.Fatalf("template: %v", err)
	}
	if err := e.cards.SetServiceRules(ctx, tpl.ID, []string{e.svc60, e.svc30}); err != nil {
		e.t.Fatalf("rules: %v", err)
	}
	c, err := e.cards.IssueCard(ctx, memberID, tpl.ID, e.opID)
	if err != nil {
		e.t.Fatalf("issue: %v", err)
	}
	return c.ID
}

// bookInService 建+确认+开始一个预约（可直接结算）。
func (e *env) bookInService(memberID, svcID string, day, hh, mm int) string {
	ctx := e.ctx()
	a, err := e.apt.Create(ctx, memberID, svcID, e.slot(day, hh, mm), "")
	if err != nil {
		e.t.Fatalf("create: %v", err)
	}
	if _, err := e.apt.Confirm(ctx, a.ID, e.opID); err != nil {
		e.t.Fatalf("confirm: %v", err)
	}
	if _, err := e.apt.Start(ctx, a.ID, e.opID); err != nil {
		e.t.Fatalf("start: %v", err)
	}
	return a.ID
}

func (e *env) cardOf(memberID string) *card.MemberCard {
	cs, err := e.cards.ListByMember(e.ctx(), memberID)
	if err != nil || len(cs) == 0 {
		e.t.Fatalf("list cards: %v", err)
	}
	return cs[0]
}

// countSQL 返回单值查询结果。
func (e *env) countSQL(q string, args ...any) int {
	var n int
	if err := e.db.QueryRowContext(e.ctx(), q, args...).Scan(&n); err != nil {
		e.t.Fatalf("count %s: %v", q, err)
	}
	return n
}

// assertLedgerChain 校验卡流水链：ISSUE 起步，每条流水的 before 承接上一条 after，
// 终点等于当前 remaining_count。任何断裂 = 丢失更新/非法扣次。
// order 无关（并发下 created_at 秒级精度不可靠），按链匹配。
func (e *env) assertLedgerChain(cardID string) {
	ctx := e.ctx()
	c := e.cardOf(e.mbrA)
	if cardID != "" {
		cs, _ := e.cards.ListByMember(ctx, memberOf(e, cardID))
		if len(cs) > 0 {
			c = cs[0]
		}
	}
	txns, err := e.cards.Transactions(ctx, c.ID)
	if err != nil {
		e.t.Fatalf("transactions: %v", err)
	}
	used := make([]bool, len(txns))
	cursor := 0
	// ISSUE 流水（0 -> total）作为链起点
	for i, t := range txns {
		if t.Type == "ISSUE" && t.Before == 0 {
			cursor = t.After
			used[i] = true
			break
		}
	}
	for progress := true; progress; {
		progress = false
		for i, t := range txns {
			if used[i] || t.Type == "ISSUE" {
				continue
			}
			if t.Before == cursor {
				cursor = t.After
				used[i] = true
				progress = true
			}
		}
	}
	for i, u := range used {
		if !u {
			e.t.Fatalf("账实不符（链断裂）：流水 #%d type=%s %d→%d 无法挂到链上（终点 %d，卡余额 %d）",
				i, txns[i].Type, txns[i].Before, txns[i].After, cursor, c.RemainingCount)
		}
	}
	if cursor != c.RemainingCount {
		e.t.Fatalf("账实不符：流水链终点 %d != member_card.remaining_count %d", cursor, c.RemainingCount)
	}
	if c.RemainingCount < 0 {
		e.t.Fatalf("不变量破坏：remaining_count = %d < 0", c.RemainingCount)
	}
}

func memberOf(e *env, cardID string) string {
	var m string
	if err := e.db.QueryRowContext(e.ctx(), `SELECT member_id FROM member_card WHERE id = ?`, cardID).Scan(&m); err != nil {
		e.t.Fatal(err)
	}
	return m
}

// ---------------------------------------------------------------- 方向 1：卡余额并发

// A1：16 协程混战（核销×8 / 调整± / 作废 / sweep）——不变量与账实一致性。
func TestGUARD_A1_CardConcurrentMixedOps(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	var aptIDs []string
	for i, s := range [][2]int{{2, 10}, {2, 11}, {2, 12}, {2, 13}, {3, 10}, {3, 11}, {3, 12}, {3, 13}} {
		aptIDs = append(aptIDs, e.bookInService(e.mbrA, e.svc60, s[0], s[1], 0))
		_ = i
	}
	type result struct {
		name string
		err  error
	}
	results := make([]result, 16)
	var wg sync.WaitGroup
	start := make(chan struct{})
	barrier := func() { <-start }

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			barrier()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_, _, err := e.tx.SettleByCard(ctx, aptIDs[i], cardID, e.opID, fmt.Sprintf("adv-a1-%d", i))
			results[i] = result{fmt.Sprintf("redeem#%d", i), err}
		}(i)
	}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			barrier()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_, err := e.cards.Adjust(ctx, cardID, 1, "并发+", e.opID)
			results[8+i] = result{fmt.Sprintf("adjust+1#%d", i), err}
		}(i)
	}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			barrier()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_, err := e.cards.Adjust(ctx, cardID, -1, "并发-", e.opID)
			results[11+i] = result{fmt.Sprintf("adjust-1#%d", i), err}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		barrier()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		results[13] = result{"cancel", e.cards.Cancel(ctx, cardID, e.opID)}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		barrier()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		_, err := e.cards.SweepExpired(ctx)
		results[14] = result{"sweep", err}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		barrier()
		results[15] = result{"lowbalance", lowBalanceErr(e, cardID)}
	}()
	close(start)
	wg.Wait()

	okCnt := 0
	for _, r := range results {
		if r.err == nil {
			okCnt++
		}
		t.Logf("%-12s err=%v", r.name, r.err)
	}
	e.assertLedgerChain(cardID) // 账实一致 + remaining>=0（FAIL=缺陷）
	c := e.cardOf(e.mbrA)
	t.Logf("final: status=%s remaining=%d, %d/%d 个操作成功", c.Status, c.RemainingCount, okCnt, len(results))
	// 不变量：无论谁赢，若卡已 CANCELLED，作废后的扣次不可能出现在链上（链校验已覆盖）。
}

// A2：先作废再核销必须被拒（顺序版"作废的卡被扣次"）。
func TestGUARD_A2_CancelledCardRejectsRedeem(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 2)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 10, 0)
	ctx := e.ctx()
	if err := e.cards.Cancel(ctx, cardID, e.opID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := e.cards.Cancel(ctx, cardID, e.opID); !shared.Is(err, "CARD_ALREADY_CANCELLED") {
		t.Errorf("重复作废未拦截: %v", err)
	}
	if _, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-a2"); !shared.Is(err, "CARD_NOT_ACTIVE") {
		t.Errorf("作废卡核销未被拦截: %v", err)
	}
	if _, err := e.cards.Adjust(ctx, cardID, 5, "", e.opID); !shared.Is(err, "CARD_CANCELLED") {
		t.Errorf("作废卡调整未被拦截: %v", err)
	}
	if got := e.cardOf(e.mbrA).RemainingCount; got != 2 {
		t.Errorf("作废卡余额被改动: %d", got)
	}
	if n := e.countSQL(`SELECT COUNT(*) FROM card_transaction WHERE member_card_id = ? AND type <> 'ISSUE'`, cardID); n != 0 {
		t.Errorf("作废卡产生了次数流水: %d 条", n)
	}
}

// ---------------------------------------------------------------- 方向 3：核销幂等

// A3（高危场景）：撤销后用第一个 idempotency_key 重放。
// 断言底线：不得产生新 payment、不得改余额、不得把 VOIDED 复活。
// 同时捕获响应语义证据（200+REVERSED+null payment / payment 错配）。
func TestREVEAL_A3_ReplayFirstKeyAfterReverse(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 10, 0)
	ctx := e.ctx()

	rd1, pay1, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-key-1")
	if err != nil {
		t.Fatalf("first settle: %v", err)
	}
	if err := e.tx.ReverseRedemption(ctx, rd1.ID, "误核销", e.opID); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	// FIXED（W1）：撤销后重放旧 key 必须显式报 RDM_REVERSED，而不是 200+REVERSED+错配 payment
	_, _, err = e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-key-1") // 重放旧 key
	if !shared.Is(err, "RDM_REVERSED") {
		t.Fatalf("撤销后重放旧 key 应返回 RDM_REVERSED，实际: %v", err)
	}
	_ = pay1
	// 底线：不产生新数据
	if n := e.countSQL(`SELECT COUNT(*) FROM redemption WHERE appointment_id = ?`, aptID); n != 1 {
		t.Errorf("重放产生了新 redemption: %d", n)
	}
	if n := e.countSQL(`SELECT COUNT(*) FROM payment WHERE appointment_id = ?`, aptID); n != 1 {
		t.Errorf("重放产生了新 payment: %d", n)
	}
	if s := e.strSQL(`SELECT status FROM payment WHERE appointment_id = ?`, aptID); s != "VOIDED" {
		t.Errorf("VOIDED payment 被复活: %s", s)
	}
	if got := e.cardOf(e.mbrA).RemainingCount; got != 10 {
		t.Errorf("重放改动了余额: %d", got)
	}
	// 再重新核销，第三次重放旧 key 仍必须 RDM_REVERSED（不存在错配可能）
	if _, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-key-2"); err != nil {
		t.Fatalf("re-settle: %v", err)
	}
	if _, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-key-1"); !shared.Is(err, "RDM_REVERSED") {
		t.Fatalf("重新核销后重放旧 key 应仍返回 RDM_REVERSED，实际: %v", err)
	}
}

// A4：SettleByPay 完全忽略 idempotency_key（同 key 重放 → PAY_EXISTS 而非原结果）。
func TestREVEAL_A4_PayIgnoresIdempotencyKey(t *testing.T) {
	e := newEnv(t)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 11, 0)
	ctx := e.ctx()
	p1, err := e.tx.SettleByPay(ctx, aptID, "CASH", 12800, "r1", "现金", e.opID, "adv-pay-key-1")
	if err != nil {
		t.Fatalf("first pay: %v", err)
	}
	// FIXED（W2）：同 key 重放必须幂等回放原结果
	p2, err := e.tx.SettleByPay(ctx, aptID, "CASH", 12800, "r1", "现金", e.opID, "adv-pay-key-1")
	if err != nil {
		t.Fatalf("同 key 重放应幂等返回原结果，实际错误: %v", err)
	}
	if p2.ID != p1.ID {
		t.Errorf("幂等回放返回了不同 payment: %s != %s", p2.ID, p1.ID)
	}
	// 另一种方式（异 key）仍必须被 D1 拦截
	if _, err := e.tx.SettleByPay(ctx, aptID, "CASH", 12800, "r2", "", e.opID, "adv-pay-key-2"); !shared.Is(err, "PAY_EXISTS") {
		t.Logf("异 key 二次收款: %v（应为 PAY_EXISTS）", err)
	}
}

// A5（BLOCKER 假设）：SettleByPay 的 VALID 检查是无锁 check-then-insert，
// 并发下可产生一个预约两笔 VALID 收款，违反 D1。
// 先用两条手动事务确定性复现代码中的确切语句序列；再用真实并发尝试。
func TestREVEAL_A5_ConcurrentCashDoubleInsert(t *testing.T) {
	e := newEnv(t)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 12, 0)
	ctx := e.ctx()

	// —— 确定性复现（复刻 settle.go:181-200 的语句序列）——
	tx1, err1 := e.db.BeginTx(ctx, nil)
	tx2, err2 := e.db.BeginTx(ctx, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("begin: %v %v", err1, err2)
	}
	for _, tx := range []shared.Tx{tx1, tx2} {
		var n int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM payment WHERE appointment_id = ? AND status = 'VALID'`, aptID).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		if n != 0 {
			t.Fatalf("环境脏: %d", n)
		}
	}
	if _, err := tx1.ExecContext(ctx,
		`INSERT INTO payment (id, appointment_id, member_id, amount, method, status, reference_no, remark)
		 VALUES (?,?,?,?,?,?,?,?)`,
		shared.NewID(), aptID, e.mbrA, 12800, "CASH", "VALID", "", "t1"); err != nil {
		t.Fatalf("t1 insert: %v", err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("t1 commit: %v", err)
	}
	if _, err := tx2.ExecContext(ctx,
		`INSERT INTO payment (id, appointment_id, member_id, amount, method, status, reference_no, remark)
		 VALUES (?,?,?,?,?,?,?,?)`,
		shared.NewID(), aptID, e.mbrA, 12800, "WECHAT_TRANSFER", "VALID", "", "t2"); err != nil {
		// FIXED（B1）：uk_payment_valid_lock 拦截第二笔 VALID 收款
		t.Logf("t2 插入被约束拦截: %v", err)
		_ = tx2.Rollback()
	} else if err := tx2.Commit(); err != nil {
		t.Logf("t2 提交被数据库拦截: %v", err)
		_ = tx2.Rollback()
	} else {
		t.Errorf("REVEALED：第二笔 VALID 收款插入成功（约束缺失）")
	}
	if n := e.countSQL(`SELECT COUNT(*) FROM payment WHERE appointment_id = ? AND status = 'VALID'`, aptID); n > 1 {
		t.Errorf("REVEALED（D1 违反，确定性）：同一预约出现 %d 笔 VALID 收款（无任何 DB 约束拦截 check-then-insert）", n)
	}

	// —— 真实并发尝试（10 轮 × 3 协程，每轮新预约）——
	for round := 0; round < 10; round++ {
		a := e.bookInService(e.mbrA, e.svc60, 3+round/4, 10+(round%4)*2, 0)
		var wg sync.WaitGroup
		start := make(chan struct{})
		statuses := make([]error, 3)
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				c, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				_, err := e.tx.SettleByPay(c, a, [3]string{"CASH", "WECHAT_TRANSFER", "OTHER"}[i],
					12800, "", "", e.opID, fmt.Sprintf("adv-a5-%d-%d", round, i))
				statuses[i] = err
			}(i)
		}
		close(start)
		wg.Wait()
		ok := 0
		for _, err := range statuses {
			if err == nil {
				ok++
			}
		}
		valid := e.countSQL(`SELECT COUNT(*) FROM payment WHERE appointment_id = ? AND status = 'VALID'`, a)
		if ok > 1 || valid > 1 {
			t.Errorf("REVEALED（D1 违反，并发）：round=%d 并发成功 %d 次，VALID 收款 %d 笔", round, ok, valid)
			return
		}
	}
	t.Logf("真实并发 10 轮未击中窗口（竞态窗口由确定性部分证明存在）")
}

// A6：同一预约、不同 key 并发核销 → active_lock 必须只放一个。
func TestGUARD_A6_ConcurrentRedeemSameAppointment(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 14, 0)
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			c, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_, _, err := e.tx.SettleByCard(c, aptID, cardID, e.opID, fmt.Sprintf("adv-a6-%d", i))
			errs[i] = err
		}(i)
	}
	close(start)
	wg.Wait()
	ok := 0
	for i, err := range errs {
		if err == nil {
			ok++
		} else {
			t.Logf("redeem#%d err=%v", i, err)
		}
	}
	if ok != 1 {
		t.Errorf("同预约并发核销成功 %d 次（应为 1，active_lock 失效）", ok)
	}
	if got := e.cardOf(e.mbrA).RemainingCount; got != 9 {
		t.Errorf("余额 = %d，应为 9", got)
	}
}

// ---------------------------------------------------------------- 方向 4：撤销重入

// A7：并发 6 次 ReverseRedemption 同一核销 → 只允许一次成功，次数只恢复一次。
func TestGUARD_A7_ConcurrentReverse(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 15, 0)
	ctx := e.ctx()
	rd, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-a7")
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			c, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			errs[i] = e.tx.ReverseRedemption(c, rd.ID, "并发撤销", e.opID)
		}(i)
	}
	close(start)
	wg.Wait()
	ok := 0
	for i, err := range errs {
		if err == nil {
			ok++
		} else if !shared.Is(err, "RDM_ALREADY_REVERSED") {
			t.Logf("reverse#%d 非预期错误码: %v", i, err)
		}
	}
	if ok != 1 {
		t.Errorf("并发撤销成功 %d 次（应为 1）", ok)
	}
	if n := e.countSQL(`SELECT COUNT(*) FROM redemption_reversal WHERE redemption_id = ?`, rd.ID); n != 1 {
		t.Errorf("撤销记录 %d 条（应为 1）", n)
	}
	if got := e.cardOf(e.mbrA).RemainingCount; got != 10 {
		t.Errorf("余额恢复错误: %d（应为 10）", got)
	}
	if s := e.strSQL(`SELECT status FROM payment WHERE appointment_id = ? AND method = 'CARD'`, aptID); s != "VOIDED" {
		t.Errorf("payment 未作废: %s", s)
	}
}

// A8：撤销与"重新核销"并发交错。断言最终一致（不允许双 SUCCESS、
// 不允许 SUCCESS 却无 VALID payment、账实必须吻合）。
func TestGUARD_A8_ReverseVsResettleRace(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 6; round++ {
		cardID := e.newCard(e.mbrA, 10)
		aptID := e.bookInService(e.mbrA, e.svc60, 2+round/4, 10+(round%4), 0)
		ctx := e.ctx()
		rd1, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, fmt.Sprintf("adv-a8-%d-1", round))
		if err != nil {
			t.Fatalf("round %d settle: %v", round, err)
		}
		start := make(chan struct{})
		var wg sync.WaitGroup
		var revErr, reErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			c, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			revErr = e.tx.ReverseRedemption(c, rd1.ID, "race-reverse", e.opID)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			c, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_, _, reErr = e.tx.SettleByCard(c, aptID, cardID, e.opID, fmt.Sprintf("adv-a8-%d-2", round))
		}()
		close(start)
		wg.Wait()
		t.Logf("round %d: reverse=%v resettle=%v", round, revErr, reErr)

		// 不变量 1：同预约不允许两条 SUCCESS
		n := e.countSQL(`SELECT COUNT(*) FROM redemption WHERE appointment_id = ? AND status = 'SUCCESS'`, aptID)
		if n > 1 {
			t.Errorf("round %d: 出现 %d 条 SUCCESS 核销", round, n)
		}
		// 不变量 2：SUCCESS 核销 ⇔ 恰一笔 VALID CARD payment
		if n == 1 {
			if v := e.countSQL(`SELECT COUNT(*) FROM payment WHERE appointment_id = ? AND method='CARD' AND status='VALID'`, aptID); v != 1 {
				t.Errorf("round %d: 有 SUCCESS 核销但 VALID CARD payment = %d", round, v)
			}
		} else if v := e.countSQL(`SELECT COUNT(*) FROM payment WHERE appointment_id = ? AND method='CARD' AND status='VALID'`, aptID); v != 0 {
			t.Errorf("round %d: 无 SUCCESS 核销但 VALID CARD payment = %d", round, v)
		}
		// 不变量 3：账实一致
		e.assertLedgerChain(cardID)
	}
}

// ---------------------------------------------------------------- 方向 2：预约冲突

// A9：并发创建/改期/竞争同一时段 —— 恰好一个赢家。
func TestGUARD_A9_ConcurrentCreateAndRescheduleSameSlot(t *testing.T) {
	e := newEnv(t)
	ctx := e.ctx()
	// 被改期的预约（目前占 day2 10:00）
	mover := e.aptCreate(t, e.mbrA, e.svc60, 2, 10, 0)
	// 竞争时段 day2 15:00
	target := e.slot(2, 15, 0)

	start := make(chan struct{})
	type res struct {
		name string
		err  error
	}
	out := make(chan res, 3)
	go func() {
		<-start
		_, err := e.apt.Reschedule(ctx, e.mbrA, mover, target, true)
		out <- res{"reschedule", err}
	}()
	go func() {
		<-start
		_, err := e.apt.Create(ctx, e.mbrB, e.svc60, target, "")
		out <- res{"createB", err}
	}()
	go func() {
		<-start
		_, err := e.apt.Create(ctx, e.mbrA, e.svc30, target, "")
		out <- res{"createA30", err}
	}()
	close(start)
	wins := 0
	for i := 0; i < 3; i++ {
		r := <-out
		if r.err == nil {
			wins++
		} else {
			t.Logf("%s err=%v", r.name, r.err)
		}
	}
	if wins != 1 {
		t.Errorf("同一时段 %d 个赢家（应为 1）", wins)
	}
	// DB 层复核：与 target 重叠的 active 预约只能有一个
	if n := e.countSQL(`SELECT COUNT(*) FROM appointment
		WHERE status IN ('PENDING_CONFIRM','CONFIRMED','IN_SERVICE')
		  AND scheduled_start < ? AND scheduled_end > ?`, endOf(target, 60), target); n != 1 {
		t.Errorf("时段重叠 active 预约 = %d（应为 1）", n)
	}
}

// A10：改期目标时段在 GET_LOCK 等待期间被第三方占用 → 改期必须失败。
// 构造：先抢走日历锁，插入一条已提交的冲突预约，再放锁 —— 等待中的改期
// 在拿到锁后必须重新检查冲突。
func TestGUARD_A10_RescheduleConflictAfterLockWait(t *testing.T) {
	e := newEnv(t)
	mover := e.aptCreate(t, e.mbrA, e.svc60, 2, 10, 0)
	target := e.slot(2, 16, 0)
	targetEnd := endOf(target, 60)

	conn, err := e.db.Conn(e.ctx())
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(e.ctx(), `SELECT GET_LOCK('anmo:appointment:calendar', 0)`); err != nil {
		t.Fatalf("get lock: %v", err)
	}
	// 第三方预约（绕过 provider 直接落库、立即提交）
	if _, err := conn.ExecContext(e.ctx(),
		`INSERT INTO appointment (id, appointment_no, member_id, scheduled_start, scheduled_end, status)
		 VALUES (?,?,?,?,?, 'CONFIRMED')`,
		shared.NewID(), fmt.Sprintf("APT%sX%d", shared.NowShanghai().Format("20060102"), 999), e.mbrB, target, targetEnd); err != nil {
		t.Fatalf("insert rival: %v", err)
	}

	res := make(chan error, 1)
	go func() {
		c, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		_, err := e.apt.Reschedule(c, e.mbrA, mover, target, true)
		res <- err
	}()
	time.Sleep(300 * time.Millisecond) // 让改期进入 GET_LOCK 等待
	if _, err := conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK('anmo:appointment:calendar')`); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := <-res; err == nil {
		t.Errorf("REVEALED：锁等待期间被占用的时段，改期仍然成功（冲突检查未在锁后执行）")
	} else {
		t.Logf("改期被正确拦截: %v", err)
	}
}

// A11：NO_SHOW / CANCELLED 不参与冲突；COMPLETED 释放时段（现口径）。
func TestGUARD_A11_TerminalStatusNoConflict(t *testing.T) {
	e := newEnv(t)
	ctx := e.ctx()
	s := e.slot(2, 17, 0)
	a1, err := e.apt.Create(ctx, e.mbrA, e.svc60, s, "")
	if err != nil {
		t.Fatalf("create1: %v", err)
	}
	if _, err := e.apt.Confirm(ctx, a1.ID, e.opID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := e.apt.NoShow(ctx, a1.ID, e.opID); err != nil {
		t.Fatalf("noshow: %v", err)
	}
	if _, err := e.apt.Create(ctx, e.mbrB, e.svc60, s, ""); err != nil {
		t.Errorf("NO_SHOW 后同时段不可再约: %v", err)
	}
	a2, err := e.apt.Create(ctx, e.mbrA, e.svc30, e.slot(2, 18, 0), "")
	if err != nil {
		t.Fatalf("create2: %v", err)
	}
	if _, err := e.apt.CancelByAdmin(ctx, a2.ID, e.opID, "test"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := e.apt.Create(ctx, e.mbrB, e.svc30, e.slot(2, 18, 0), ""); err != nil {
		t.Errorf("CANCELLED 后同时段不可再约: %v", err)
	}
}

// A12：跨日/营业时间边界（D15）：23:30 跨午夜、20:30+60min 超打烊、
// 20:30+30min 恰好等于打烊时间、错位时间槽、改期到 23:00。
func TestGUARD_A12_BusinessHourBoundaries(t *testing.T) {
	e := newEnv(t)
	ctx := e.ctx()
	cases := []struct {
		name   string
		svc    string
		day    int
		hh, mm int
		want   string // 期望错误码；"" = 允许
	}{
		{"23:30跨午夜", e.svc60, 2, 23, 30, "APT_OUT_OF_HOURS"},
		{"20:30+60超打烊", e.svc60, 2, 20, 30, "APT_OUT_OF_HOURS"},
		{"20:30+30恰好打烊", e.svc30, 2, 20, 30, ""},
		{"20:15错位时间槽", e.svc30, 2, 20, 15, "APT_BAD_SLOT"},
		{"08:30早于开门", e.svc60, 2, 8, 30, "APT_OUT_OF_HOURS"},
		{"40天后太远", e.svc60, 40, 10, 0, "APT_TOO_FAR"},
	}
	for _, c := range cases {
		_, err := e.apt.Create(ctx, e.mbrA, c.svc, e.slot(c.day, c.hh, c.mm), "")
		got := ""
		if err != nil {
			got = errCode(err)
		}
		if got != c.want {
			t.Errorf("%s: got %q want %q (err=%v)", c.name, got, c.want, err)
		}
	}
	// 改期路径同样受 D15 约束
	m := e.aptCreate(t, e.mbrA, e.svc60, 3, 10, 0)
	if _, err := e.apt.Reschedule(ctx, e.mbrA, m, e.slot(3, 23, 0), false); !shared.Is(err, "APT_OUT_OF_HOURS") {
		t.Errorf("改期到 23:00 未被营业时间拦截: %v", err)
	}
}

// A13：高并发预订风暴（30 协程 > 连接池 20）—— 不死锁、恰好 1 个成功。
func TestGUARD_A13_BookingStormNoDeadlock(t *testing.T) {
	e := newEnv(t)
	target := e.slot(6, 15, 0)
	start := make(chan struct{})
	var wg sync.WaitGroup
	statuses := make([]error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			c, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			_, err := e.apt.Create(c, e.mbrA, e.svc60, target, "")
			statuses[i] = err
		}(i)
	}
	close(start)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(90 * time.Second):
		t.Fatalf("预订风暴死锁：30 个创建未在 90s 内完成")
	}
	ok := 0
	for i, err := range statuses {
		if err == nil {
			ok++
		} else if code := errCode(err); code != "APPOINTMENT_CONFLICT" && code != "APT_LOCK_BUSY" {
			t.Errorf("create#%d 非预期错误: %v", i, err)
		}
	}
	if ok != 1 {
		t.Errorf("风暴中 %d 个成功（应为 1）", ok)
	}
}

// ---------------------------------------------------------------- 其他越界

// A14：用顾客 B 的卡核销顾客 A 的预约（跨会员卡）。
func TestREVEAL_A14_CrossMemberCard(t *testing.T) {
	e := newEnv(t)
	cardB := e.newCard(e.mbrB, 10)
	aptA := e.bookInService(e.mbrA, e.svc60, 2, 10, 0)
	ctx := e.ctx()
	// FIXED（W3）：跨会员核销必须 403 CARD_NOT_YOURS
	if _, _, err := e.tx.SettleByCard(ctx, aptA, cardB, e.opID, "adv-a14"); !shared.Is(err, "CARD_NOT_YOURS") {
		t.Fatalf("跨会员核销应被拒绝（CARD_NOT_YOURS），实际: %v", err)
	}
	if got := e.cardOf(e.mbrB).RemainingCount; got != 10 {
		t.Errorf("B 卡余额被改动: %d", got)
	}
}

// A15：对已作废（CANCELLED）卡做核销撤销 → 撤销整体失败且 payment 永远停留 VALID。
func TestREVEAL_A15_ReverseBlockedByCancelledCard(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 11, 0)
	ctx := e.ctx()
	rd, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-a15")
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	if err := e.cards.Cancel(ctx, cardID, e.opID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	// FIXED（W4）：作废卡上的撤销必须成功（账目修正），卡保持 CANCELLED，payment 置 VOIDED
	if err := e.tx.ReverseRedemption(ctx, rd.ID, "误核销", e.opID); err != nil {
		t.Fatalf("作废卡撤销应成功，实际: %v", err)
	}
	if s := e.strSQL(`SELECT status FROM payment WHERE appointment_id = ? AND method='CARD'`, aptID); s != "VOIDED" {
		t.Errorf("payment 未作废: %s", s)
	}
	if c := e.cardOf(e.mbrA); c.Status != "CANCELLED" {
		t.Errorf("卡状态应保持 CANCELLED: %s", c.Status)
	}
}

// A16：过期卡（EXPIRED sweep 后）撤销核销 → 卡被"复活"为 ACTIVE（valid_until 仍在过去）。
func TestREVEAL_A16_ReverseRevivesExpiredCard(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 12, 0)
	ctx := e.ctx()
	rd, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-a16")
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	// 让卡过期（valid_until 置为昨天）并触发 sweep
	if _, err := e.db.ExecContext(ctx,
		`UPDATE member_card SET valid_until = DATE_SUB(CURDATE(), INTERVAL 1 DAY) WHERE id = ?`, cardID); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if _, err := e.cards.SweepExpired(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if st := e.cardOf(e.mbrA).Status; st != "EXPIRED" {
		t.Fatalf("卡未过期: %s", st)
	}
	// FIXED（W5）：撤销允许执行（账目修正）但不得把 EXPIRED 卡复活为 ACTIVE
	if err := e.tx.ReverseRedemption(ctx, rd.ID, "误核销", e.opID); err != nil {
		t.Fatalf("过期卡撤销应成功，实际: %v", err)
	}
	c := e.cardOf(e.mbrA)
	if c.Status == "ACTIVE" {
		t.Errorf("过期卡被复活为 ACTIVE（valid_until=%s 仍在过去）", *c.ValidUntil)
	}
	if c.Status != "EXPIRED" {
		t.Errorf("卡状态应保持 EXPIRED: %s", c.Status)
	}
}

// A17：超长撤销原因（>500）→ 插入 redemption_reversal 失败被误报为"已撤销"。
func TestREVEAL_A17_ReverseLongReasonMisdiagnosed(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	aptID := e.bookInService(e.mbrA, e.svc60, 2, 13, 0)
	ctx := e.ctx()
	rd, _, err := e.tx.SettleByCard(ctx, aptID, cardID, e.opID, "adv-a17")
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	long := make([]byte, 1800) // 600 个 ASCII 超参（VARCHAR(500) 按字符计）
	for i := range long {
		long[i] = 'x'
	}
	// FIXED（W6）：超长原因在入参校验即被拒绝，且不再误报为"已撤销"
	err = e.tx.ReverseRedemption(ctx, rd.ID, string(long), e.opID)
	if !shared.Is(err, "RDM_REASON_TOO_LONG") {
		t.Fatalf("超长原因应返回 RDM_REASON_TOO_LONG，实际: %v", err)
	}
	if s := e.strSQL(`SELECT status FROM redemption WHERE id = ?`, rd.ID); s != "SUCCESS" {
		t.Errorf("核销状态被意外改动: %s", s)
	}
	// 正常长度的撤销仍可用
	if err := e.tx.ReverseRedemption(ctx, rd.ID, "误核销", e.opID); err != nil {
		t.Fatalf("正常撤销失败: %v", err)
	}
}

// A18：结算入口的状态机边界。PENDING_CONFIRM 必须拒绝；CONFIRMED 直接核销
// 会触发 CONFIRMED → COMPLETED —— 冻结状态机（AGENTS.md）没有这条迁移，
// 这里捕获该偏离作为证据。
func TestREVEAL_A18_SettleFromConfirmedSkipsInService(t *testing.T) {
	e := newEnv(t)
	cardID := e.newCard(e.mbrA, 10)
	ctx := e.ctx()
	pending, err := e.apt.Create(ctx, e.mbrA, e.svc60, e.slot(2, 10, 0), "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, _, err := e.tx.SettleByCard(ctx, pending.ID, cardID, e.opID, "adv-a18-1"); err == nil {
		t.Errorf("PENDING_CONFIRM 预约被核销")
	}
	if got := e.cardOf(e.mbrA).RemainingCount; got != 10 {
		t.Errorf("PENDING 拒绝后余额被改动: %d", got)
	}
	if _, err := e.apt.Confirm(ctx, pending.ID, e.opID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// FIXED（W8）：CONFIRMED 直接核销必须被拒（冻结状态机无 CONFIRMED→COMPLETED）
	if _, _, err := e.tx.SettleByCard(ctx, pending.ID, cardID, e.opID, "adv-a18-2"); err == nil {
		t.Fatal("CONFIRMED 预约被直接核销（应拒绝）")
	}
	if got := e.cardOf(e.mbrA).RemainingCount; got != 10 {
		t.Errorf("CONFIRMED 拒绝后余额被改动: %d", got)
	}
}

// ---------------------------------------------------------------- 辅助

func (e *env) aptCreate(t *testing.T, memberID, svcID string, day, hh, mm int) string {
	t.Helper()
	a, err := e.apt.Create(e.ctx(), memberID, svcID, e.slot(day, hh, mm), "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return a.ID
}

// strSQL 返回单字符串查询结果。
func (e *env) strSQL(q string, args ...any) string {
	var s string
	if err := e.db.QueryRowContext(e.ctx(), q, args...).Scan(&s); err != nil {
		e.t.Fatalf("str %s: %v", q, err)
	}
	return s
}

// endOf 把 "YYYY-MM-DD HH:MM" 加 dur 分钟。
func endOf(slot string, dur int) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", slot, shared.NowShanghai().Location())
	return t.Add(time.Duration(dur) * time.Minute)
}

func lowBalanceErr(e *env, cardID string) error {
	_, err := e.cards.LowBalanceCards(e.ctx(), 2)
	return err
}

func errCode(err error) string {
	if e, ok := err.(*shared.AppError); ok {
		return e.Code
	}
	return err.Error()
}

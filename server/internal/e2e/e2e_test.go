package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"anmo/server/internal/app"
	"anmo/server/internal/config"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// E2E drives the real HTTP stack (all modules + MySQL) end to end (plan §120).

type client struct {
	t     *testing.T
	base  string
	token string
}

func (c *client) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func (c *client) ok(method, path string, body any) map[string]any {
	c.t.Helper()
	status, out := c.do(method, path, body)
	if status < 200 || status >= 300 {
		c.t.Fatalf("%s %s = %d: %v", method, path, status, out)
	}
	return out
}

func (c *client) data(m map[string]any) map[string]any {
	d, _ := m["data"].(map[string]any)
	return d
}

func str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func num(m map[string]any, key string) float64 {
	v, _ := m[key].(float64)
	return v
}

// slotAt builds a business-hours slot N days ahead (2h lead rule satisfied).
func slotAt(t *testing.T, day int, hh, mm int) string {
	t.Helper()
	d := shared.NowShanghai().AddDate(0, 0, day)
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d", d.Year(), d.Month(), d.Day(), hh, mm)
}

func newServer(t *testing.T) *client {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_e2e_")
	cfg := config.Defaults()
	cfg.Auth.JWTSecret = "e2e-secret"
	cfg.Auth.AdminPhone = "13800000000"
	cfg.Auth.AdminPasswordSeed = "e2e-admin-pass"
	cfg.SMS.Mode = "off" // V2.2 生产口径：短信登录下线（旧端点 410 SMS_DISABLED）

	log := shared.NewNopLogger()
	if err := app.SeedIdentity(db, cfg, log); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := app.Build(db, cfg, log)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return &client{t: t, base: ts.URL}
}

// registerCust — V2.2 顾客建号：H5 手机号+密码注册（成功即登录）。
func registerCust(t *testing.T, c *client, phone, password string) {
	t.Helper()
	res := c.ok("POST", "/api/auth/register", map[string]string{"phone": phone, "password": password})
	c.token = str(res["data"].(map[string]any), "token")
	if c.token == "" {
		t.Fatal("no customer token after register")
	}
}

func TestFullLoop(t *testing.T) {
	admin := newServer(t)

	// ---- 老板登录（§111）----
	res := admin.ok("POST", "/admin/auth/login", map[string]string{
		"phone": "13800000000", "password": "e2e-admin-pass",
	})
	admin.token = str(res["data"].(map[string]any), "token")
	if admin.token == "" {
		t.Fatal("no admin token")
	}

	// ---- 顾客注册登录（V2.2: H5 手机号+密码，§111）----
	cust := &client{t: t, base: admin.base}
	registerCust(t, cust, "13911112222", "cust-pass66")

	// 短信端点已下线（R6）：off 模式两端点 410 SMS_DISABLED
	if status, out := cust.do("POST", "/api/auth/sms/send", map[string]string{"phone": "13911112222"}); status != 410 || str(out, "code") != "SMS_DISABLED" {
		t.Fatalf("legacy sms send = %d %v, want 410 SMS_DISABLED", status, out)
	}
	if status, out := cust.do("POST", "/api/auth/sms/verify", map[string]string{"phone": "13911112222", "code": "123456"}); status != 410 || str(out, "code") != "SMS_DISABLED" {
		t.Fatalf("legacy sms verify = %d %v, want 410 SMS_DISABLED", status, out)
	}
	// H5 密码登录可重入
	res = cust.ok("POST", "/api/auth/login", map[string]string{"phone": "13911112222", "password": "cust-pass66"})
	if str(res["data"].(map[string]any), "token") == "" {
		t.Fatal("h5 login no token")
	}

	// ---- 后台建服务（肩颈按摩 60min 12800, §113/§114）----
	cat := admin.ok("POST", "/admin/service-categories", map[string]any{"name": "按摩", "sort": 1})
	catID := str(cat["data"].(map[string]any), "id")
	svc := admin.ok("POST", "/admin/services", map[string]any{
		"category_id": catID, "name": "肩颈按摩", "duration_minutes": 60, "default_price": 12800,
	})
	svcID := str(svc["data"].(map[string]any), "id")

	// 顾客端可见（§113）
	catalog := cust.ok("GET", "/api/services", nil)
	cats := catalog["data"].(map[string]any)
	svcList := cats["services"].([]any)
	if len(svcList) != 1 {
		t.Fatalf("customer services = %d", len(svcList))
	}

	// ---- 卡模板 + 规则 + 发卡（§114: 10 次卡 ISSUE +10）----
	tpl := admin.ok("POST", "/admin/card-templates", map[string]any{
		"name": "按摩10次卡", "type": "COUNT", "total_count": 10, "price": 100000,
	})
	tplID := str(tpl["data"].(map[string]any), "id")
	admin.ok("PUT", "/admin/card-templates/"+tplID+"/rules", map[string]any{"service_ids": []string{svcID}})

	prof := cust.ok("GET", "/api/me/profile", nil)
	memberID := str(prof["data"].(map[string]any)["member"].(map[string]any), "id")
	issued := admin.ok("POST", "/admin/cards", map[string]any{
		"member_id": memberID, "card_template_id": tplID,
	})
	cardID := str(issued["data"].(map[string]any), "id")
	if got := num(issued["data"].(map[string]any), "remaining_count"); got != 10 {
		t.Fatalf("issued card remaining = %v, want 10", got)
	}
	// ISSUE 流水
	txns := admin.ok("GET", "/admin/cards/"+cardID+"/transactions", nil)
	if len(txns["data"].([]any)) != 1 {
		t.Fatalf("card transactions after issue = %d", len(txns["data"].([]any)))
	}

	// ---- 顾客预约（§120: 选择按摩 → 选择时间 → 预约）----
	slot := slotAt(t, 2, 15, 0)
	booked := cust.ok("POST", "/api/appointments", map[string]any{
		"service_id": svcID, "start_time": slot, "note": "有力度的",
	})
	aptID := str(booked["data"].(map[string]any), "id")
	if str(booked["data"].(map[string]any), "status") != "WAITING" {
		t.Fatalf("created status = %v", booked["data"].(map[string]any)["status"])
	}
	// 2026-09-28 状态机收紧：确认环节已删除（§5），/confirm 端点不复存在
	if status, _ := admin.do("PUT", "/admin/appointments/"+aptID+"/confirm", nil); status != 404 {
		t.Fatalf("legacy confirm endpoint status = %d, want 404", status)
	}

	// 冲突：另一顾客抢同时段 → 409（§28）
	cust2 := &client{t: t, base: admin.base}
	registerCust(t, cust2, "13911113333", "cust2-pass6")
	if status, _ := cust2.do("POST", "/api/appointments", map[string]any{
		"service_id": svcID, "start_time": slot,
	}); status != 409 {
		t.Fatalf("conflicting booking status = %d, want 409", status)
	}

	// 越权：顾客直接访问 admin 路由 → 403
	if status, _ := cust.do("GET", "/admin/appointments", nil); status != 403 {
		t.Fatalf("customer on admin route = %d, want 403", status)
	}

	// ---- 老板：开始服务（创建即 WAITING，无确认环节 §6/§7）----
	admin.ok("PUT", "/admin/appointments/"+aptID+"/start", nil)

	// ---- 取消后时段释放（Case 9, 独立预约）----
	c1 := cust.ok("POST", "/api/appointments", map[string]any{
		"service_id": svcID, "start_time": slotAt(t, 3, 10, 0),
	})
	a1 := str(c1["data"].(map[string]any), "id")
	admin.ok("PUT", "/admin/appointments/"+a1+"/cancel", map[string]string{"reason": "改时间"})
	// 释放后另一顾客可约同一时段
	cust2.ok("POST", "/api/appointments", map[string]any{
		"service_id": svcID, "start_time": slotAt(t, 3, 10, 0),
	})

	// ---- 核销（§120/§53；结算即完成，§8）----
	redeem := admin.ok("POST", "/admin/appointments/"+aptID+"/redeem", map[string]any{
		"card_id": cardID, "communicated": true, "idempotency_key": "e2e-redeem-1",
	})
	rd := redeem["data"].(map[string]any)["redemption"].(map[string]any)
	if str(rd, "status") != "SUCCESS" || num(rd, "after_count") != 9 {
		t.Fatalf("redemption = %v", rd)
	}
	pay := redeem["data"].(map[string]any)["payment"].(map[string]any)
	if str(pay, "method") != "CARD" || str(pay, "status") != "VALID" || num(pay, "amount") != 12800 {
		t.Fatalf("payment = %v", pay)
	}

	// 已完成预约不能再次完成 → 409（§99 Case 10；结算已自动完成）
	if status, _ := admin.do("PUT", "/admin/appointments/"+aptID+"/complete", nil); status != 409 {
		t.Fatalf("re-complete status = %d, want 409", status)
	}

	// 重复核销（同 key 重放）不重复扣（Case 6）
	replay := admin.ok("POST", "/admin/appointments/"+aptID+"/redeem", map[string]any{
		"card_id": cardID, "communicated": true, "idempotency_key": "e2e-redeem-1",
	})
	rd2 := replay["data"].(map[string]any)["redemption"].(map[string]any)
	if str(rd2, "id") != str(rd, "id") {
		t.Fatal("replay returned different redemption")
	}

	// 余额减少（§120: 卡余额减少）
	myCards := cust.ok("GET", "/api/me/cards", nil)
	if got := num(myCards["data"].([]any)[0].(map[string]any), "remaining_count"); got != 9 {
		t.Fatalf("remaining = %v, want 9", got)
	}
	// 交易记录（流水 ISSUE + REDEEM）
	txns = admin.ok("GET", "/admin/cards/"+cardID+"/transactions", nil)
	if got := len(txns["data"].([]any)); got != 2 {
		t.Fatalf("card transactions = %d, want 2", got)
	}

	// ---- 撤销核销 → 次数恢复（Case 7）→ 现金重结不双计（D1）----
	admin.ok("PUT", "/admin/redemptions/"+str(rd, "id")+"/reverse", map[string]string{"reason": "误核销"})
	myCards = cust.ok("GET", "/api/me/cards", nil)
	if got := num(myCards["data"].([]any)[0].(map[string]any), "remaining_count"); got != 10 {
		t.Fatalf("remaining after reverse = %v, want 10", got)
	}
	// 撤销后重新核销（W-E：active_lock 释放）
	redeemed := admin.ok("POST", "/admin/appointments/"+aptID+"/redeem", map[string]any{
		"card_id": cardID, "communicated": true, "idempotency_key": "e2e-redeem-2",
	})
	if str(redeemed["data"].(map[string]any)["redemption"].(map[string]any), "status") != "SUCCESS" {
		t.Fatal("re-settle after reversal failed")
	}
	// 现金收款（另一预约，D9 不改状态）
	cashApt := cust2.ok("POST", "/api/appointments", map[string]any{
		"service_id": svcID, "start_time": slotAt(t, 4, 10, 0),
	})
	cashID := str(cashApt["data"].(map[string]any), "id")
	// 现金结算：WAITING 直接收款即完成（§8 事务一致）
	admin.ok("POST", "/admin/appointments/"+cashID+"/payments", map[string]any{
		"method": "CASH", "amount": 12800, "communicated": true, "idempotency_key": "e2e-cash-1",
	})
	// 第二笔 VALID 收款被拒（D1）
	if status, _ := admin.do("POST", "/admin/appointments/"+cashID+"/payments", map[string]any{
		"method": "WECHAT_TRANSFER", "amount": 12800, "communicated": true, "idempotency_key": "e2e-cash-2",
	}); status != 409 {
		t.Fatalf("second VALID payment status = %d, want 409", status)
	}
	// 收款记录只统计 VALID（§136）
	pays := admin.ok("GET", "/admin/payments?status=VALID", nil)
	validCount := len(pays["data"].([]any))
	if validCount != 2 { // card redeem + cash
		t.Fatalf("VALID payments = %d, want 2", validCount)
	}

	// ---- 今日工作台（§73）----
	wb := admin.ok("GET", "/admin/workbench?date="+slotAt(t, 4, 10, 0)[:10], nil)
	summary := wb["data"].(map[string]any)["summary"].(map[string]any)
	if num(summary, "total") < 1 {
		t.Fatalf("workbench summary = %v", summary)
	}

	// ---- 操作日志（§85）----
	logs := admin.ok("GET", "/admin/logs", nil)
	if len(logs["data"].([]any)) == 0 {
		t.Fatal("operation log empty")
	}

	// ---- 洞察 + 首页/设置（§86/§82/§84）----
	admin.ok("GET", "/admin/insights", nil)
	cust.ok("GET", "/api/home", nil)
	cust.ok("GET", "/api/settings", nil)

	// ---- 顾客改期 + 取消自己的预约（§104/§31）；跨顾客 403（§100）----
	// 新建一个 PENDING 预约用于改期/取消（IN_SERVICE 不可改期，D8）
	pending := cust2.ok("POST", "/api/appointments", map[string]any{
		"service_id": svcID, "start_time": slotAt(t, 5, 14, 0),
	})
	pendingID := str(pending["data"].(map[string]any), "id")
	rescheduled := cust2.ok("PUT", "/api/appointments/"+pendingID+"/reschedule", map[string]string{
		"start_time": slotAt(t, 5, 16, 0),
	})
	if str(rescheduled["data"].(map[string]any), "id") != pendingID {
		t.Fatal("reschedule changed appointment")
	}
	// 顾客 A 不能操作顾客 B 的预约
	if status, _ := cust.do("PUT", "/api/appointments/"+pendingID+"/cancel", nil); status != 403 {
		t.Fatalf("cross-customer cancel = %d, want 403", status)
	}
	// 本人远期预约可取消（2h 限制只拦近期）
	cust2.ok("PUT", "/api/appointments/"+pendingID+"/cancel", nil)

	// 未登录访问 → 401
	if status, _ := (&client{t: t, base: admin.base}).do("GET", "/api/me/profile", nil); status != 401 {
		t.Fatalf("unauthenticated = %d, want 401", status)
	}
}

// TestConcurrentBookingAPI pins Case 2 at the HTTP layer.
func TestConcurrentBookingAPI(t *testing.T) {
	admin := newServer(t)
	res := admin.ok("POST", "/admin/auth/login", map[string]string{
		"phone": "13800000000", "password": "e2e-admin-pass",
	})
	admin.token = str(res["data"].(map[string]any), "token")
	cat := admin.ok("POST", "/admin/service-categories", map[string]any{"name": "按摩", "sort": 1})
	catID := str(cat["data"].(map[string]any), "id")
	svc := admin.ok("POST", "/admin/services", map[string]any{
		"category_id": catID, "name": "全身按摩", "duration_minutes": 60, "default_price": 16800,
	})
	svcID := str(svc["data"].(map[string]any), "id")

	const n = 6
	tokens := make([]string, n)
	for i := 0; i < n; i++ {
		c := &client{t: t, base: admin.base}
		registerCust(t, c, fmt.Sprintf("1392222%04d", i), "conc-pass66")
		tokens[i] = c.token
	}

	slot := slotAt(t, 7, 15, 0)
	statuses := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := &client{t: t, base: admin.base, token: tokens[i]}
			statuses[i], _ = c.do("POST", "/api/appointments", map[string]any{
				"service_id": svcID, "start_time": slot,
			})
		}(i)
	}
	wg.Wait()
	success := 0
	for _, s := range statuses {
		if s == 200 {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("Case 2: %d succeeded, want exactly 1 (statuses=%v)", success, statuses)
	}
}

// TestWxLoginFlow — V2.2 小程序登录全链路（HTTP 层，D25 修订）：wx 首登直建号
// 直发 token → 补手机号撞号（MEMBER_PHONE_TAKEN）→ claim 凭 H5 密码转绑老账号
// 并删空壳 → 二次 login 落在老账号，登录态完成切换。
func TestWxLoginFlow(t *testing.T) {
	c := newServer(t)

	// 1. wx 首登：直建号直发 Token（无 needs_bind/bind_ticket）
	res := c.ok("POST", "/api/auth/wx/login", map[string]string{"code": "wx-code-1"})
	d := res["data"].(map[string]any)
	if d["needs_bind"] != nil || str(d, "bind_ticket") != "" || str(d, "token") == "" {
		t.Fatalf("first wx login = %v", d)
	}
	shell := &client{t: t, base: c.base, token: str(d, "token")}

	// bind_ticket 链路已废除：端点 404
	if status, _ := shell.do("POST", "/api/auth/wx/bind", map[string]string{"bind_ticket": "x"}); status != 404 {
		t.Fatalf("legacy wx/bind status = %d, want 404", status)
	}

	// 2. 老顾客已有 H5 账号（手机号+密码）
	h5 := &client{t: t, base: c.base}
	registerCust(t, h5, "13911114444", "h5-pass666")
	prof := h5.ok("GET", "/api/me/profile", nil)
	h5ID := str(prof["data"].(map[string]any)["member"].(map[string]any), "id")

	// 3. 微信空壳在「我的」补手机号 → 撞号 409 MEMBER_PHONE_TAKEN（弹认领框的触发点）
	status, out := shell.do("PUT", "/api/me/profile", map[string]string{"phone": "13911114444"})
	if status != 409 || str(out, "code") != "MEMBER_PHONE_TAKEN" {
		t.Fatalf("shell set phone = %d %v, want 409 MEMBER_PHONE_TAKEN", status, out)
	}

	// 4. claim：凭 H5 密码转绑 → 返回老账号新 Token（登录态切换）
	res = shell.ok("POST", "/api/auth/wx/claim", map[string]string{"phone": "13911114444", "password": "h5-pass666"})
	d = res["data"].(map[string]any)
	claimed := &client{t: t, base: c.base, token: str(d, "token")}
	if got := str(d, "member_id"); got != h5ID {
		t.Fatalf("claim member_id = %s, want h5 %s", got, h5ID)
	}
	// 老账号身份完整：phone 保留
	prof = claimed.ok("GET", "/api/me/profile", nil)
	if got := str(prof["data"].(map[string]any)["member"].(map[string]any), "phone"); got != "13911114444" {
		t.Fatalf("claimed member phone = %s", got)
	}
	// 空壳已被删除：旧 token 不再可用
	if status, _ := shell.do("GET", "/api/me/profile", nil); status == 200 {
		t.Fatal("deleted shell token still resolves profile")
	}

	// 5. 二次 wx login：直接落在老账号
	res = c.ok("POST", "/api/auth/wx/login", map[string]string{"code": "wx-code-1"})
	d = res["data"].(map[string]any)
	if str(d, "token") == "" || str(d, "member_id") != h5ID {
		t.Fatalf("second wx login = %v, want member_id %s", d, h5ID)
	}

	// 6. 小程序设置 H5 密码（微信身份即凭证，R5）：设完 H5 可直接登录
	claimed.ok("PUT", "/api/me/h5-password", map[string]string{"new_password": "h5-new777"})
	res = c.ok("POST", "/api/auth/login", map[string]string{"phone": "13911114444", "password": "h5-new777"})
	if str(res["data"].(map[string]any), "token") == "" {
		t.Fatal("h5 login with wechat-set password failed")
	}
}

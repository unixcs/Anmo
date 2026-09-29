package e2e

// adversarial_test.go — 对抗式审查测试（HTTP 级）。只测试、不修复。
//
// 命名约定（把"预期被拦截"与"缺陷暴露"分开）：
//   - TestGUARD_*：攻击应当被现有约束拦截；FAIL = 发现缺陷。
//   - TestREVEAL_*：假设存在缺陷；FAIL = 缺陷被真实复现；PASS ≠ 安全。
//
// 复用 e2e_test.go 的 client/newServer/slotAt 工具与真实 HTTP 栈。

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// rawDo goroutine 安全的请求（不在子协程里 t.Fatal）。
func rawDo(base, method, path, token string, body any) (int, map[string]any, error) {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out, nil
}

type fixture struct {
	base      string
	admin     *client
	custA     *client // 顾客 A
	custB     *client // 顾客 B
	memberA   string
	memberB   string
	svcID     string
	tplID     string
	cardA     string
	cardB     string
	aptB      string // B 的预约（越权目标）
	category  string
	jwtSecret string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	admin := newServer(t) // 已含 schema + seed
	res := admin.ok("POST", "/admin/auth/login", map[string]string{
		"phone": "13800000000", "password": "e2e-admin-pass",
	})
	admin.token = str(res["data"].(map[string]any), "token")

	mkCust := func(phone string) (*client, string) {
		c := &client{t: t, base: admin.base}
		r := c.ok("POST", "/api/auth/register", map[string]string{"phone": phone, "password": "adv-pass66"})
		tok := str(r["data"].(map[string]any), "token")
		c.token = tok
		p := c.ok("GET", "/api/me/profile", nil)
		mid := str(p["data"].(map[string]any)["member"].(map[string]any), "id")
		return c, mid
	}
	custA, memberA := mkCust("13933330001")
	custB, memberB := mkCust("13933330002")

	cat := admin.ok("POST", "/admin/service-categories", map[string]any{"name": "对抗分类", "sort": 1})
	categoryID := str(cat["data"].(map[string]any), "id")
	svc := admin.ok("POST", "/admin/services", map[string]any{
		"category_id": categoryID, "name": "对抗按摩", "duration_minutes": 60, "default_price": 12800,
	})
	svcID := str(svc["data"].(map[string]any), "id")
	tpl := admin.ok("POST", "/admin/card-templates", map[string]any{
		"name": "对抗10次卡", "type": "COUNT", "total_count": 10, "price": 100000,
	})
	tplID := str(tpl["data"].(map[string]any), "id")
	admin.ok("PUT", "/admin/card-templates/"+tplID+"/rules", map[string]any{"service_ids": []string{svcID}})
	cardA := str(admin.ok("POST", "/admin/cards", map[string]any{
		"member_id": memberA, "card_template_id": tplID,
	})["data"].(map[string]any), "id")
	cardB := str(admin.ok("POST", "/admin/cards", map[string]any{
		"member_id": memberB, "card_template_id": tplID,
	})["data"].(map[string]any), "id")

	booked := custB.ok("POST", "/api/appointments", map[string]any{
		"service_id": svcID, "start_time": slotAt(t, 2, 10, 0),
	})
	aptB := str(booked["data"].(map[string]any), "id")

	return &fixture{
		base: admin.base, admin: admin, custA: custA, custB: custB,
		memberA: memberA, memberB: memberB, svcID: svcID, tplID: tplID,
		cardA: cardA, cardB: cardB, aptB: aptB, category: categoryID,
		jwtSecret: "e2e-secret",
	}
}

// ---------------------------------------------------------------- 方向 5：权限越界

// H1：顾客 token 扫描全部 /admin 端点；无 token / 垃圾 token 扫描 /admin 与 /api。
func TestGUARD_H1_AdminRouteMatrixAndAnonymous(t *testing.T) {
	f := newFixture(t)

	adminEndpoints := []struct{ method, path string }{
		{"GET", "/admin/appointments"},
		{"GET", "/admin/today"},
		{"PUT", "/admin/appointments/" + f.aptB + "/start"},
		{"PUT", "/admin/appointments/" + f.aptB + "/complete"},
		{"PUT", "/admin/appointments/" + f.aptB + "/cancel"},
		{"PUT", "/admin/appointments/" + f.aptB + "/no-show"},
		{"PUT", "/admin/appointments/" + f.aptB + "/reschedule"},
		{"POST", "/admin/appointments/" + f.aptB + "/redeem"},
		{"POST", "/admin/appointments/" + f.aptB + "/payments"},
		{"PUT", "/admin/redemptions/xxxxxxxx/reverse"},
		{"GET", "/admin/workbench"},
		{"GET", "/admin/payments"},
		{"GET", "/admin/redemptions"},
		{"GET", "/admin/card-templates"},
		{"POST", "/admin/card-templates"},
		{"PUT", "/admin/card-templates/" + f.tplID + "/status"},
		{"PUT", "/admin/card-templates/" + f.tplID + "/rules"},
		{"POST", "/admin/cards"},
		{"GET", "/admin/members/" + f.memberB + "/cards"},
		{"PUT", "/admin/cards/" + f.cardB + "/adjust"},
		{"PUT", "/admin/cards/" + f.cardB + "/cancel"},
		{"GET", "/admin/cards/" + f.cardB + "/transactions"},
		{"GET", "/admin/members"},
		{"POST", "/admin/members"},
		{"GET", "/admin/members/" + f.memberB},
		{"PUT", "/admin/members/" + f.memberB},
		{"PUT", "/admin/members/" + f.memberB + "/password"},
		{"PUT", "/admin/members/" + f.memberB + "/tags"},
		{"GET", "/admin/tags"},
		{"POST", "/admin/tags"},
		{"GET", "/admin/service-categories"},
		{"POST", "/admin/service-categories"},
		{"PUT", "/admin/service-categories/" + f.category + "/status"},
		{"GET", "/admin/services"},
		{"POST", "/admin/services"},
		{"GET", "/admin/content/pages"},
		{"PUT", "/admin/content/pages"},
		{"GET", "/admin/banners"},
		{"POST", "/admin/banners"},
		{"GET", "/admin/announcements"},
		{"POST", "/admin/announcements"},
		{"GET", "/admin/settings"},
		{"PUT", "/admin/settings"},
		{"GET", "/admin/logs"},
		{"GET", "/admin/insights"},
		{"POST", "/admin/ops/daily"},
	}
	bodies := map[string]any{
		"POST /admin/appointments/" + f.aptB + "/redeem":    map[string]any{"card_id": f.cardB, "communicated": true, "idempotency_key": "adv-h1"},
		"POST /admin/appointments/" + f.aptB + "/payments":  map[string]any{"method": "CASH", "amount": 100, "communicated": true},
		"PUT /admin/redemptions/xxxxxxxx/reverse":           map[string]string{"reason": "x"},
		"POST /admin/cards":                                 map[string]any{"member_id": f.memberB, "card_template_id": f.tplID},
		"PUT /admin/cards/" + f.cardB + "/adjust":           map[string]any{"delta": 1},
		"POST /admin/card-templates":                        map[string]any{"name": "x", "type": "COUNT", "total_count": 1},
		"PUT /admin/card-templates/" + f.tplID + "/rules":   map[string]any{"service_ids": []string{f.svcID}},
		"POST /admin/service-categories":                    map[string]any{"name": "x", "sort": 1},
		"POST /admin/services":                              map[string]any{"category_id": f.category, "name": "x", "duration_minutes": 30},
		"POST /admin/members":                               map[string]any{"phone": "13999990000", "name": "x"},
		"POST /admin/tags":                                  map[string]any{"name": "x"},
		"POST /admin/banners":                               map[string]any{"title": "x"},
		"POST /admin/announcements":                         map[string]any{"title": "x"},
		"POST /admin/ops/daily":                             map[string]any{},
		"PUT /admin/content/pages":                          map[string]any{"page": "home", "blocks": []any{}},
		"PUT /admin/settings":                               map[string]any{"key": "x", "value": "y"},
		"PUT /admin/members/" + f.memberB + "/tags":         map[string]any{"tag_ids": []string{}},
		"PUT /admin/members/" + f.memberB + "/password":     map[string]any{"new_password": "adv-reset-1"},
		"PUT /admin/appointments/" + f.aptB + "/reschedule": map[string]string{"start_time": slotAt(t, 2, 14, 0)},
		"PUT /admin/appointments/" + f.aptB + "/cancel":     map[string]string{"reason": "x"},
	}

	for _, ep := range adminEndpoints {
		status, out, err := rawDo(f.base, ep.method, ep.path, f.custA.token, bodies[ep.method+" "+ep.path])
		if err != nil {
			t.Fatalf("%s %s: %v", ep.method, ep.path, err)
		}
		if status != 403 {
			t.Errorf("顾客 token 访问 %s %s = %d %v，应为 403", ep.method, ep.path, status, out["code"])
		}
		// 无 token
		status, _, err = rawDo(f.base, ep.method, ep.path, "", bodies[ep.method+" "+ep.path])
		if err != nil || status != 401 {
			t.Errorf("无 token 访问 %s %s = %d（应为 401，err=%v）", ep.method, ep.path, status, err)
		}
		// 垃圾 token
		status, _, err = rawDo(f.base, ep.method, ep.path, "garbage.token.here", bodies[ep.method+" "+ep.path])
		if err != nil || status != 401 {
			t.Errorf("垃圾 token 访问 %s %s = %d（应为 401，err=%v）", ep.method, ep.path, status, err)
		}
	}

	apiEndpoints := []struct{ method, path string }{
		{"GET", "/api/me/profile"},
		{"PUT", "/api/me/profile"},
		{"PUT", "/api/me/h5-password"},
		{"POST", "/api/auth/wx/claim"},
		{"GET", "/api/me/cards"},
		{"GET", "/api/appointments"},
		{"POST", "/api/appointments"},
		{"GET", "/api/appointments/" + f.aptB},
		{"PUT", "/api/appointments/" + f.aptB + "/cancel"},
		{"PUT", "/api/appointments/" + f.aptB + "/reschedule"},
		{"GET", "/api/services"},
		{"GET", "/api/home"},
		{"GET", "/api/settings"},
	}
	for _, ep := range apiEndpoints {
		status, _, err := rawDo(f.base, ep.method, ep.path, "", nil)
		if err != nil || status != 401 {
			t.Errorf("无 token 访问 %s %s = %d（应为 401，err=%v）", ep.method, ep.path, status, err)
		}
	}

	// 反向：ADMIN token 访问顾客端 → 403
	for _, ep := range apiEndpoints {
		status, out, _ := rawDo(f.base, ep.method, ep.path, f.admin.token, nil)
		if status == 200 {
			t.Errorf("ADMIN token 访问 %s %s 竟然成功: %v", ep.method, ep.path, out)
		}
	}
}

// H2：伪造/篡改 JWT —— 错 secret、过期、alg=none、RS256 混淆、篡改载荷、空 act。
func TestGUARD_H2_ForgedJWT(t *testing.T) {
	f := newFixture(t)
	sign := func(secret string, claims jwt.MapClaims) string {
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
		return tok
	}
	now := time.Now()
	valid := func(act, uid string) jwt.MapClaims {
		return jwt.MapClaims{"act": act, "uid": uid, "iss": "anmo",
			"iat": jwt.NewNumericDate(now), "exp": jwt.NewNumericDate(now.Add(time.Hour))}
	}

	cases := []struct {
		name   string
		token  string
		path   string
		status int
	}{
		{"错secret的ADMIN令牌", sign("wrong-secret", valid("ADMIN", "x1")), "/admin/payments", 401},
		{"过期ADMIN令牌(正确secret)", func() string {
			return sign(f.jwtSecret, jwt.MapClaims{"act": "ADMIN", "uid": "x", "iss": "anmo",
				"iat": jwt.NewNumericDate(now.Add(-2 * time.Hour)), "exp": jwt.NewNumericDate(now.Add(-time.Hour))})
		}(), "/admin/payments", 401},
		{"alg=none", func() string {
			h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
			p := base64.RawURLEncoding.EncodeToString([]byte(`{"act":"ADMIN","uid":"x","exp":9999999999}`))
			return h + "." + p + "."
		}(), "/admin/payments", 401},
		{"RS256头混淆", func() string {
			h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
			p := base64.RawURLEncoding.EncodeToString([]byte(`{"act":"ADMIN","uid":"x","exp":9999999999}`))
			return h + "." + p + ".c2ln"
		}(), "/admin/payments", 401},
		{"篡改载荷", func() string {
			tok := sign(f.jwtSecret, valid("CUSTOMER", f.memberA))
			i := len(tok) - 10
			return tok[:i] + "AAAAAAAAAA"
		}(), "/api/me/profile", 401},
		{"空act但签名正确", sign(f.jwtSecret, valid("", f.memberA)), "/api/me/profile", 403},
		{"空act访问admin", sign(f.jwtSecret, valid("", "x")), "/admin/payments", 403},
		{"CUSTOMER(uid=别人)访问admin", sign(f.jwtSecret, valid("CUSTOMER", f.memberA)), "/admin/payments", 403},
		{"空token头", "", "/admin/payments", 401},
	}
	for _, c := range cases {
		status, out, err := rawDo(f.base, "GET", c.path, c.token, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if status >= 200 && status < 300 || status == 500 {
			t.Errorf("%s: 攻击令牌获得 %d 响应 %v", c.name, status, out)
		}
		if c.status == 401 && status != 401 {
			t.Logf("%s: 返回 %d（%v）——只要非 2xx 即安全", c.name, status, out["code"])
		}
	}
}

// H3：顾客 A 读/改顾客 B 的预约与卡。
func TestGUARD_H3_CrossCustomerIsolation(t *testing.T) {
	f := newFixture(t)

	// 读 B 的预约详情
	status, _, _ := rawDo(f.base, "GET", "/api/appointments/"+f.aptB, f.custA.token, nil)
	if status != 403 {
		t.Errorf("A 读 B 的预约 = %d，应为 403", status)
	}
	// 改 B 的预约（改期）
	status, out, _ := rawDo(f.base, "PUT", "/api/appointments/"+f.aptB+"/reschedule",
		f.custA.token, map[string]string{"start_time": slotAt(t, 2, 15, 0)})
	if status == 200 {
		t.Errorf("A 改期 B 的预约竟然成功: %v", out)
	}
	// 取消 B 的预约
	status, _, _ = rawDo(f.base, "PUT", "/api/appointments/"+f.aptB+"/cancel", f.custA.token, nil)
	if status == 200 {
		t.Errorf("A 取消 B 的预约竟然成功")
	}
	// B 的预约必须原封不动
	status, out, _ = rawDo(f.base, "GET", "/api/appointments/"+f.aptB, f.custB.token, nil)
	if status != 200 {
		t.Fatalf("B 自己读预约失败: %d %v", status, out)
	}
	bApt := out["data"].(map[string]any)["appointment"].(map[string]any)
	if got := str(bApt, "status"); got != "WAITING" {
		t.Errorf("B 的预约状态被改动: %s", got)
	}
	if got := str(bApt, "scheduled_start"); got == slotAt(t, 2, 15, 0)+":00" {
		t.Errorf("B 的预约时间被 A 改动")
	}
	// A 的预约列表不含 B 的预约
	status, out, _ = rawDo(f.base, "GET", "/api/appointments", f.custA.token, nil)
	if status != 200 {
		t.Fatalf("A 列表失败: %d", status)
	}
	if list, ok := out["data"].([]any); ok { // 空列表时 data 为 null
		for _, it := range list {
			if str(it.(map[string]any), "id") == f.aptB {
				t.Errorf("A 的列表泄漏了 B 的预约")
			}
		}
	}
	// A 的卡列表只有 A 的卡
	status, out, _ = rawDo(f.base, "GET", "/api/me/cards", f.custA.token, nil)
	if status != 200 {
		t.Fatalf("A 卡列表失败: %d", status)
	}
	for _, it := range out["data"].([]any) {
		if str(it.(map[string]any), "id") == f.cardB {
			t.Errorf("A 的卡列表泄漏了 B 的卡")
		}
	}
	// 顾客创建预约时 member 归属来自 token（响应 member_id 必须是自己）
	booked := f.custA.ok("POST", "/api/appointments", map[string]any{
		"service_id": f.svcID, "start_time": slotAt(t, 3, 10, 0),
	})
	if got := str(booked["data"].(map[string]any), "member_id"); got != f.memberA {
		t.Errorf("预约归属错误: %s != %s", got, f.memberA)
	}
}

// ---------------------------------------------------------------- 方向 3：HTTP 层并发收款

// H4：同一预约并发 3 笔现金收款 × 8 轮 —— D1（一预约一笔 VALID）。
func TestREVEAL_H4_ConcurrentCashPayments(t *testing.T) {
	f := newFixture(t)
	for round := 0; round < 8; round++ {
		booked := f.custA.ok("POST", "/api/appointments", map[string]any{
			"service_id": f.svcID, "start_time": slotAt(t, 4+round/4, 10+(round%4)*2, 0),
		})
		aptID := str(booked["data"].(map[string]any), "id")
		f.admin.ok("PUT", "/admin/appointments/"+aptID+"/start", nil)

		start := make(chan struct{})
		var wg sync.WaitGroup
		okCount := 0
		var mtx sync.Mutex
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				method := [3]string{"CASH", "WECHAT_TRANSFER", "OTHER"}[i]
				status, _, err := rawDo(f.base, "POST", "/admin/appointments/"+aptID+"/payments", f.admin.token,
					map[string]any{"method": method, "amount": 12800,
						"communicated": true, "idempotency_key": fmt.Sprintf("adv-h4-%d-%d", round, i)})
				if err != nil {
					return
				}
				if status == 200 {
					mtx.Lock()
					okCount++
					mtx.Unlock()
				}
			}(i)
		}
		close(start)
		wg.Wait()
		if okCount > 1 {
			t.Errorf("REVEALED（D1 违反）：round=%d 并发收款成功 %d 次（一个预约多笔 VALID 收款）", round, okCount)
			return
		}
	}
	t.Logf("HTTP 并发收款 8 轮未击中竞态窗口（DB 级已确定性证明，见 adversarial 包 A5）")
}

// H5（高危场景 HTTP 版）：撤销后重放第一个 idempotency_key；再重新核销后重放。
func TestREVEAL_H5_ReplayFirstKeyAfterReverse(t *testing.T) {
	f := newFixture(t)
	booked := f.custA.ok("POST", "/api/appointments", map[string]any{
		"service_id": f.svcID, "start_time": slotAt(t, 2, 11, 0),
	})
	aptID := str(booked["data"].(map[string]any), "id")
	f.admin.ok("PUT", "/admin/appointments/"+aptID+"/start", nil)

	rd1 := f.admin.ok("POST", "/admin/appointments/"+aptID+"/redeem", map[string]any{
		"card_id": f.cardA, "communicated": true, "idempotency_key": "adv-h5-key-1",
	})["data"].(map[string]any)
	rdID := str(rd1["redemption"].(map[string]any), "id")
	f.admin.ok("PUT", "/admin/redemptions/"+rdID+"/reverse", map[string]string{"reason": "误核销"})

	// FIXED（W1）：撤销后重放旧 key 必须 409 RDM_REVERSED（不再 200+错配）
	status, out, _ := rawDo(f.base, "POST", "/admin/appointments/"+aptID+"/redeem", f.admin.token,
		map[string]any{"card_id": f.cardA, "communicated": true, "idempotency_key": "adv-h5-key-1"})
	t.Logf("撤销后重放旧 key: HTTP %d body=%v", status, out)
	if status != 409 {
		t.Fatalf("撤销后重放旧 key 应 409，实际: %d %v", status, out)
	}
	if code, _ := out["code"].(string); code != "RDM_REVERSED" {
		t.Fatalf("错误码应为 RDM_REVERSED: %v", out)
	}
	// 底线：不产生新数据
	pays := f.admin.ok("GET", "/admin/payments", nil)
	cnt := 0
	for _, it := range pays["data"].([]any) {
		if str(it.(map[string]any), "appointment_id") == aptID {
			cnt++
		}
	}
	if cnt != 1 {
		t.Errorf("重放产生了新 payment: %d", cnt)
	}
	cards := f.custA.ok("GET", "/api/me/cards", nil)
	if got := num(cards["data"].([]any)[0].(map[string]any), "remaining_count"); got != 10 {
		t.Errorf("重放改动了余额: %v", got)
	}

	// 重新核销后第三次重放旧 key 仍必须 409
	f.admin.ok("POST", "/admin/appointments/"+aptID+"/redeem", map[string]any{
		"card_id": f.cardA, "communicated": true, "idempotency_key": "adv-h5-key-2",
	})
	status, out, _ = rawDo(f.base, "POST", "/admin/appointments/"+aptID+"/redeem", f.admin.token,
		map[string]any{"card_id": f.cardA, "communicated": true, "idempotency_key": "adv-h5-key-1"})
	if status != 409 {
		t.Fatalf("重新核销后重放旧 key 应仍 409，实际: %d %v", status, out)
	}
}

// H6：HTTP 层跨会员卡核销 —— A 的预约用 B 的卡。
func TestREVEAL_H6_CrossMemberCardRedeem(t *testing.T) {
	f := newFixture(t)
	booked := f.custA.ok("POST", "/api/appointments", map[string]any{
		"service_id": f.svcID, "start_time": slotAt(t, 2, 12, 0),
	})
	aptID := str(booked["data"].(map[string]any), "id")
	f.admin.ok("PUT", "/admin/appointments/"+aptID+"/start", nil)

	// FIXED（W3）：跨会员核销必须 403
	status, out, _ := rawDo(f.base, "POST", "/admin/appointments/"+aptID+"/redeem", f.admin.token,
		map[string]any{"card_id": f.cardB, "communicated": true, "idempotency_key": "adv-h6"})
	if status != 403 {
		t.Fatalf("跨会员核销应 403，实际: %d %v", status, out["code"])
	}
	cards := f.custB.ok("GET", "/api/me/cards", nil)
	if rem := num(cards["data"].([]any)[0].(map[string]any), "remaining_count"); rem != 10 {
		t.Errorf("B 卡余额被改动: %v", rem)
	}
}

// H7：预订风暴 24 并发（> 连接池 20）—— 不死锁、恰好 1 个成功、无 5xx。
func TestGUARD_H7_BookingStormNoDeadlockNo5xx(t *testing.T) {
	f := newFixture(t)
	slot := slotAt(t, 6, 15, 0)
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mtx sync.Mutex
	ok, five := 0, 0
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, _, _ := rawDo(f.base, "POST", "/api/appointments", f.custA.token,
				map[string]any{"service_id": f.svcID, "start_time": slot})
			mtx.Lock()
			defer mtx.Unlock()
			if status == 200 {
				ok++
			}
			if status >= 500 {
				five++
			}
		}()
	}
	close(start)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(90 * time.Second):
		t.Fatalf("预订风暴死锁：24 个请求未在 90s 内完成")
	}
	if ok != 1 {
		t.Errorf("风暴 %d 个成功（应为 1）", ok)
	}
	if five > 0 {
		t.Errorf("风暴产生 %d 个 5xx（并发控制不应以 500 暴露）", five)
	}
}

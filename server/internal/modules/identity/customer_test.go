package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/member"
	"anmo/server/internal/shared"
)

// customer_test.go — V2.2 账号体系矩阵（design §4）：
//   register/login 五类提示逐字校验、claim 全矩阵（无账号/无密码/密码错/
//   已绑微信/有业务数据/成功转绑+空壳删除+登录态切换）、h5-password
//   （无 phone 拒、设置后 H5 可登）、sms off/dev 两态（handler 层）。

func appErr(t *testing.T, err error) *shared.AppError {
	t.Helper()
	ae, ok := err.(*shared.AppError)
	if !ok {
		t.Fatalf("want *shared.AppError, got %v", err)
	}
	return ae
}

// ---- Register / Login 矩阵 ----

func TestRegisterSuccessAndConflicts(t *testing.T) {
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()

	// 注册成功 → 建号+登录态
	token, mid, err := p.Register(ctx, "13911110001", "pass1234")
	if err != nil || token == "" || mid == "" {
		t.Fatalf("register: %v", err)
	}
	m, err := p.members.Get(ctx, mid)
	if err != nil {
		t.Fatal(err)
	}
	if m.Phone != "13911110001" || !m.HasPassword || m.WxBound || m.PasswordHash == "" {
		t.Fatalf("registered member = %+v", m)
	}

	// 同号已设密码 → "该手机号已注册，请直接登录"
	_, _, err = p.Register(ctx, "13911110001", "pass1234")
	ae := appErr(t, err)
	if ae.Code != "AUTH_PHONE_EXISTS" || ae.Status != 409 || ae.Message != "该手机号已注册，请直接登录" {
		t.Fatalf("want AUTH_PHONE_EXISTS, got %+v", ae)
	}

	// 入参校验：手机号 / 弱密码
	if _, _, err := p.Register(ctx, "12345", "pass1234"); !shared.Is(err, "AUTH_BAD_PHONE") {
		t.Fatalf("want AUTH_BAD_PHONE, got %v", err)
	}
	if _, _, err := p.Register(ctx, "13911110002", "12345"); !shared.Is(err, "AUTH_WEAK_PASSWORD") {
		t.Fatalf("want AUTH_WEAK_PASSWORD, got %v", err)
	}
}

// 微信会员补手机号后 H5 注册同号 → "已开通微信账号" 分支。
func TestRegisterPhoneOnWechat(t *testing.T) {
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()

	_, shellID, err := p.WxLogin(ctx, "code-ow")
	if err != nil {
		t.Fatalf("wx login: %v", err)
	}
	// 「我的」补手机号（一次性）
	if err := shared.RunInTx(ctx, p.DB(), func(tx shared.Tx) error {
		return p.members.SetPhoneOnce(ctx, tx, shellID, "13922220002")
	}); err != nil {
		t.Fatalf("set phone: %v", err)
	}
	_, _, err = p.Register(ctx, "13922220002", "pass1234")
	ae := appErr(t, err)
	if ae.Code != "AUTH_PHONE_ON_WECHAT" || ae.Status != 409 ||
		ae.Message != "该手机号已开通微信账号，请前往小程序「我的」设置 H5 密码后登录" {
		t.Fatalf("want AUTH_PHONE_ON_WECHAT, got %+v", ae)
	}
}

// 存量短信时代账号（admin 建号：有 phone、无密码、未绑微信）的登录/注册文案。
func TestLoginLegacyNoPassword(t *testing.T) {
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()

	legacy, err := p.members.Create(ctx, member.NewMember{Name: "老顾客", Phone: "13933330001"})
	if err != nil {
		t.Fatalf("create legacy member: %v", err)
	}

	// 登录 → "尚未设置密码"（含小程序指引）
	_, _, err = p.Login(ctx, "13933330001", "whatever")
	ae := appErr(t, err)
	if ae.Code != "AUTH_NO_PASSWORD" || ae.Status != 401 ||
		ae.Message != "该账号尚未设置密码，请联系商家重置；已使用微信小程序可在「我的」设置 H5 密码" {
		t.Fatalf("want AUTH_NO_PASSWORD, got %+v", ae)
	}
	// H5 注册同号 → "已存在，请联系商家重置密码后登录"
	_, _, err = p.Register(ctx, "13933330001", "pass1234")
	ae = appErr(t, err)
	if ae.Code != "AUTH_PHONE_EXISTS_NO_PWD" || ae.Status != 409 ||
		ae.Message != "该手机号已存在，请联系商家重置密码后登录" {
		t.Fatalf("want AUTH_PHONE_EXISTS_NO_PWD, got %+v", ae)
	}
	// 该会员绑微信后，注册同号切换为微信分支文案
	if err := shared.RunInTx(ctx, p.DB(), func(tx shared.Tx) error {
		return p.members.BindOpenID(ctx, tx, legacy.ID, "dev:legacy")
	}); err != nil {
		t.Fatalf("bind: %v", err)
	}
	_, _, err = p.Register(ctx, "13933330001", "pass1234")
	if !shared.Is(err, "AUTH_PHONE_ON_WECHAT") {
		t.Fatalf("want AUTH_PHONE_ON_WECHAT after bind, got %v", err)
	}
}

func TestLoginMatrix(t *testing.T) {
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()

	token, mid, err := p.Register(ctx, "13911110001", "pass1234")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// 未注册
	_, _, err = p.Login(ctx, "13911119999", "pass1234")
	ae := appErr(t, err)
	if ae.Code != "AUTH_UNREGISTERED" || ae.Status != 401 || ae.Message != "该手机号尚未注册" {
		t.Fatalf("want AUTH_UNREGISTERED, got %+v", ae)
	}
	// 密码错误
	_, _, err = p.Login(ctx, "13911110001", "wrong-pass")
	ae = appErr(t, err)
	if ae.Code != "AUTH_BAD_PASSWORD" || ae.Status != 401 || ae.Message != "密码错误，请重新输入" {
		t.Fatalf("want AUTH_BAD_PASSWORD, got %+v", ae)
	}
	// 空参
	if _, _, err := p.Login(ctx, "", "pass1234"); !shared.Is(err, "AUTH_BAD_REQUEST") {
		t.Fatalf("want AUTH_BAD_REQUEST, got %v", err)
	}
	// 成功：同 member
	token2, mid2, err := p.Login(ctx, "13911110001", "pass1234")
	if err != nil || token2 == "" || mid2 != mid {
		t.Fatalf("login ok: mid2=%s want %s err=%v", mid2, mid, err)
	}
	_ = token
}

// ---- Claim 全矩阵（customer.go 步骤 0→6）----

// claimEnv 组装：H5 老账号（phone+password）+ 微信空壳（wx 首登直建号）。
type claimEnv struct {
	p       *Provider
	h5ID    string
	shellID string
	phone   string
	pass    string
}

func newClaimEnv(t *testing.T) *claimEnv {
	t.Helper()
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()
	e := &claimEnv{p: p, phone: "13944440001", pass: "h5pass66"}
	var err error
	if _, e.h5ID, err = p.Register(ctx, e.phone, e.pass); err != nil {
		t.Fatalf("register h5: %v", err)
	}
	if _, e.shellID, err = p.WxLogin(ctx, "code-claim"); err != nil {
		t.Fatalf("wx login shell: %v", err)
	}
	return e
}

func TestClaimSuccessMovesOpenidAndDeletesShell(t *testing.T) {
	e := newClaimEnv(t)
	ctx := context.Background()

	token, mid, err := e.p.ClaimByPhone(ctx, e.shellID, e.phone, e.pass)
	if err != nil || token == "" || mid != e.h5ID {
		t.Fatalf("claim: mid=%s want %s err=%v", mid, e.h5ID, err)
	}
	// 老账号拿到 openid，手机号/密码不变
	h5, err := e.p.members.Get(ctx, e.h5ID)
	if err != nil {
		t.Fatal(err)
	}
	if !h5.WxBound || h5.Phone != e.phone || !h5.HasPassword {
		t.Fatalf("h5 member after claim = %+v", h5)
	}
	// 空壳同事务删除
	if _, err := e.p.members.Get(ctx, e.shellID); !shared.Is(err, "MEMBER_NOT_FOUND") {
		t.Fatalf("shell should be deleted, got %v", err)
	}
	// openid 唯一不变量：全库该 openid 只落在老账号
	var n int
	if err := e.p.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM member WHERE wx_openid = 'dev:code-claim'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("openid rows = %d err=%v", n, err)
	}
	// 二次 wx 登录 → 直接落在老账号（登录态已切换）
	_, mid2, err := e.p.WxLogin(ctx, "code-claim")
	if err != nil || mid2 != e.h5ID {
		t.Fatalf("wx re-login: mid2=%s want %s err=%v", mid2, e.h5ID, err)
	}
}

func TestClaimRejections(t *testing.T) {
	e := newClaimEnv(t)
	ctx := context.Background()

	// 1. 目标不存在 → 404
	_, _, err := e.p.ClaimByPhone(ctx, e.shellID, "13944449999", e.pass)
	ae := appErr(t, err)
	if ae.Code != "AUTH_CLAIM_NO_ACCOUNT" || ae.Status != 404 {
		t.Fatalf("want AUTH_CLAIM_NO_ACCOUNT, got %+v", ae)
	}

	// 2. 目标无密码（存量账号）→ 401
	if _, err := e.p.members.Create(ctx, member.NewMember{Name: "无密码老号", Phone: "13944440002"}); err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	_, _, err = e.p.ClaimByPhone(ctx, e.shellID, "13944440002", "whatever")
	ae = appErr(t, err)
	if ae.Code != "AUTH_NO_PASSWORD" || ae.Status != 401 || ae.Message != "该账号尚未设置密码，请联系商家重置" {
		t.Fatalf("want AUTH_NO_PASSWORD, got %+v", ae)
	}

	// 3. 密码错误 → 401
	_, _, err = e.p.ClaimByPhone(ctx, e.shellID, e.phone, "wrong!")
	ae = appErr(t, err)
	if ae.Code != "AUTH_BAD_PASSWORD" || ae.Status != 401 || ae.Message != "密码错误，请重新输入" {
		t.Fatalf("want AUTH_BAD_PASSWORD, got %+v", ae)
	}

	// 4. 老账号已绑定其他微信 → 409（plan §二.4）
	if err := shared.RunInTx(ctx, e.p.DB(), func(tx shared.Tx) error {
		return e.p.members.BindOpenID(ctx, tx, e.h5ID, "dev:other-wx")
	}); err != nil {
		t.Fatal(err)
	}
	_, _, err = e.p.ClaimByPhone(ctx, e.shellID, e.phone, e.pass)
	ae = appErr(t, err)
	if ae.Code != "WX_OPENID_BOUND" || ae.Status != 409 ||
		ae.Message != "该手机号已绑定其他微信账号，请联系商家处理" {
		t.Fatalf("want WX_OPENID_BOUND, got %+v", ae)
	}
	// 解绑老账号，让后续步骤 5 可达
	if _, err := e.p.DB().ExecContext(ctx,
		`UPDATE member SET wx_openid = NULL WHERE id = ?`, e.h5ID); err != nil {
		t.Fatal(err)
	}

	// 5. 空壳已有业务数据 → 409，绝不丢数据
	if _, err := e.p.DB().ExecContext(ctx,
		`INSERT INTO appointment (id, appointment_no, member_id, scheduled_start, scheduled_end)
		 VALUES ('01JCLAIM0000000000000000T', 'APT202601010009', ?, '2026-01-01 10:00', '2026-01-01 11:00')`,
		e.shellID); err != nil {
		t.Fatalf("seed appointment: %v", err)
	}
	_, _, err = e.p.ClaimByPhone(ctx, e.shellID, e.phone, e.pass)
	ae = appErr(t, err)
	if ae.Code != "AUTH_CLAIM_HAS_DATA" || ae.Status != 409 ||
		ae.Message != "当前账号已有预约/卡记录，请联系商家处理" {
		t.Fatalf("want AUTH_CLAIM_HAS_DATA, got %+v", ae)
	}
	// 空壳未被删除、openid 未被移走
	if _, err := e.p.members.Get(ctx, e.shellID); err != nil {
		t.Fatalf("shell must survive: %v", err)
	}
	var openid string
	if err := e.p.DB().QueryRowContext(ctx,
		`SELECT wx_openid FROM member WHERE id = ?`, e.shellID).Scan(&openid); err != nil || openid == "" {
		t.Fatalf("shell keeps openid: %q err=%v", openid, err)
	}
}

// 当前会员未绑微信 → WX_NO_OPENID（防御分支）。
func TestClaimNoOpenid(t *testing.T) {
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()
	m, err := p.members.Create(ctx, member.NewMember{Name: "无微信", Phone: "13944440003"})
	if err != nil {
		t.Fatal(err)
	}
	target, err := p.members.Create(ctx, member.NewMember{Name: "目标", Phone: "13944440004"})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.RunInTx(ctx, p.DB(), func(tx shared.Tx) error {
		return p.members.SetPassword(ctx, tx, target.ID, "target66")
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.ClaimByPhone(ctx, m.ID, "13944440004", "target66"); !shared.Is(err, "WX_NO_OPENID") {
		t.Fatalf("want WX_NO_OPENID, got %v", err)
	}
}

// ---- H5 密码设置（R5）----

func TestSetH5PasswordEnablesH5Login(t *testing.T) {
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()

	// 微信会员补手机号 → 设密码 → H5 可登
	_, mid, err := p.WxLogin(ctx, "code-pwd")
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.RunInTx(ctx, p.DB(), func(tx shared.Tx) error {
		return p.members.SetPhoneOnce(ctx, tx, mid, "13955550001")
	}); err != nil {
		t.Fatal(err)
	}
	// 未绑手机号的会员 → MEMBER_PHONE_REQUIRED
	_, mid2, _ := p.WxLogin(ctx, "code-pwd-nophone")
	if err := p.SetH5Password(ctx, mid2, "pass1234"); !shared.Is(err, "MEMBER_PHONE_REQUIRED") {
		t.Fatalf("want MEMBER_PHONE_REQUIRED, got %v", err)
	}
	// 弱密码
	if err := p.SetH5Password(ctx, mid, "12345"); !shared.Is(err, "MEMBER_WEAK_PASSWORD") {
		t.Fatalf("want MEMBER_WEAK_PASSWORD, got %v", err)
	}
	// 设置=重置同路径（plan §二.5）
	if err := p.SetH5Password(ctx, mid, "pass1234"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := p.SetH5Password(ctx, mid, "pass9999"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	// H5 用新密码登录，旧密码失效
	if _, _, err := p.Login(ctx, "13955550001", "pass1234"); err == nil {
		t.Fatal("old password still works")
	}
	if _, _, err := p.Login(ctx, "13955550001", "pass9999"); err != nil {
		t.Fatalf("new password login: %v", err)
	}
}

// ---- SMS off/dev 两态（R6，handler 层）----

func postJSON(t *testing.T, h func(http.ResponseWriter, *http.Request), body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, req)
	return w
}

func TestSMSOffReturns410(t *testing.T) {
	p, _ := newWxEnv(t, func(c *config.Config) { c.SMS.Mode = "off" })

	w := postJSON(t, p.handleSMSSend, map[string]string{"phone": "13966660001"})
	if w.Code != http.StatusGone {
		t.Fatalf("off send status = %d, want 410", w.Code)
	}
	var env struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != "SMS_DISABLED" || env.Msg != "短信登录已下线，请升级小程序至最新版本" {
		t.Fatalf("off send envelope = %+v", env)
	}
	// verify 同样 410（即便 code 正确也不放行）
	w = postJSON(t, p.handleSMSVerify, map[string]string{"phone": "13966660001", "code": "123456"})
	if w.Code != http.StatusGone {
		t.Fatalf("off verify status = %d, want 410", w.Code)
	}
}

func TestSMSDevLegacyFlowStillWorks(t *testing.T) {
	p, _ := newWxEnv(t, nil) // Defaults: sms.mode = dev
	ctx := context.Background()

	// 过渡期 dev：send → verify(123456) → 建号+Token
	w := postJSON(t, p.handleSMSSend, map[string]string{"phone": "13966660002"})
	if w.Code != http.StatusOK {
		t.Fatalf("dev send status = %d body=%s", w.Code, w.Body.String())
	}
	w = postJSON(t, p.handleSMSVerify, map[string]string{"phone": "13966660002", "code": "123456"})
	if w.Code != http.StatusOK {
		t.Fatalf("dev verify status = %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token    string `json:"token"`
			MemberID string `json:"member_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if resp.Data.Token == "" || resp.Data.MemberID == "" {
		t.Fatalf("dev verify data = %+v", resp)
	}
	m, err := p.members.Get(ctx, resp.Data.MemberID)
	if err != nil || m.Phone != "13966660002" {
		t.Fatalf("dev verify member = %+v err=%v", m, err)
	}
}

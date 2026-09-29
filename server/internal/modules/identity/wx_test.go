package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/member"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// wx_test.go — V2 小程序登录链路：code2session → openid 绑定 → 顾客 Token。

func newWxEnv(t *testing.T, mutate func(*config.Config)) (*Provider, *member.Provider) {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_wx_")
	cfg := config.Defaults()
	cfg.Auth.JWTSecret = "wx-test-secret"
	if mutate != nil {
		mutate(cfg)
	}
	mem := member.New(db, cfg)
	p := New(db, cfg, mem, shared.NewNopLogger())
	return p, mem
}

func mustMember(t *testing.T, mem *member.Provider, phone string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := shared.RunInTx(ctx, mem.DB(), func(tx shared.Tx) error {
		var created bool
		var e error
		id, created, e = mem.EnsureByPhone(ctx, tx, phone, "")
		_ = created
		return e
	})
	if err != nil {
		t.Fatalf("member %s: %v", phone, err)
	}
	return id
}

func TestWxLoginDevFallbackAndBind(t *testing.T) {
	p, mem := newWxEnv(t, nil)
	ctx := context.Background()

	// D24 dev 兜底：AppID 为空时 code 即 openid，未绑定 → needs_bind + ticket
	token, mid, ticket, needsBind, err := p.WxLogin(ctx, "code-a")
	if err != nil || !needsBind || token != "" || mid != "" || ticket == "" {
		t.Fatalf("first login: needsBind=%v token=%q mid=%q ticket=%q err=%v", needsBind, token, mid, ticket, err)
	}

	// ticket 无效 → 拒绝绑定
	if err := p.WxBind(ctx, "01J0000000000000000000NOPE", "not-a-ticket"); err == nil {
		t.Fatal("fake ticket accepted")
	}

	// 顾客短信登录后绑定
	m := mustMember(t, mem, "13900000001")
	if err := p.WxBind(ctx, m, ticket); err != nil {
		t.Fatalf("bind: %v", err)
	}

	// 二次 login：直发 Token，同一 member（plan §11 同一个 member_id）
	token, mid, ticket2, needsBind, err := p.WxLogin(ctx, "code-a")
	if err != nil || needsBind || token == "" || mid != m || ticket2 != "" {
		t.Fatalf("second login: needsBind=%v mid=%s err=%v", needsBind, mid, err)
	}

	// 重复绑定同一对 → 幂等 no-op
	if err := p.WxBind(ctx, m, ticket); err != nil {
		t.Fatalf("rebind same pair: %v", err)
	}

	// 空 code → 400
	if _, _, _, _, err := p.WxLogin(ctx, "  "); err == nil {
		t.Fatal("empty code accepted")
	}
}

func TestWxBindConflict(t *testing.T) {
	p, mem := newWxEnv(t, nil)
	ctx := context.Background()

	m1 := mustMember(t, mem, "13900000011")
	m2 := mustMember(t, mem, "13900000012")

	_, _, ticket, needsBind, err := p.WxLogin(ctx, "code-a")
	if err != nil || !needsBind {
		t.Fatalf("login: %v", err)
	}
	if err := p.WxBind(ctx, m1, ticket); err != nil {
		t.Fatalf("bind m1: %v", err)
	}

	// 同一 openid 换 member 绑定 → 409（uk_member_wx_openid）
	ticket2, err := p.tokens.signBindTicket("dev:code-a")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	err = p.WxBind(ctx, m2, ticket2)
	appErr, ok := err.(*shared.AppError)
	if !ok || appErr.Code != "WX_OPENID_BOUND" {
		t.Fatalf("want WX_OPENID_BOUND, got %v", err)
	}
}

func TestWxLoginCode2SessionStub(t *testing.T) {
	var gotReq struct{ appid, secret, jsCode string }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq.appid = r.URL.Query().Get("appid")
		gotReq.secret = r.URL.Query().Get("secret")
		gotReq.jsCode = r.URL.Query().Get("js_code")
		if gotReq.jsCode == "good-code" {
			_, _ = w.Write([]byte(`{"openid":"o-real-1","session_key":"sk"}`))
			return
		}
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid appid"}`))
	}))
	defer srv.Close()

	p, _ := newWxEnv(t, func(c *config.Config) {
		c.Wx.AppID = "wx-app"
		c.Wx.Secret = "wx-secret"
		c.Wx.APIBase = srv.URL
	})
	ctx := context.Background()

	_, _, ticket, needsBind, err := p.WxLogin(ctx, "good-code")
	if err != nil || !needsBind || ticket == "" {
		t.Fatalf("stub login: needsBind=%v err=%v", needsBind, err)
	}
	// openid 来自 stub（真实 code2session），不是 dev: 前缀
	if _, ok, err := p.members.FindByOpenID(ctx, "o-real-1"); err != nil || ok {
		t.Fatalf("openid lookup: ok=%v err=%v", ok, err)
	}
	if gotReq.appid != "wx-app" || gotReq.secret != "wx-secret" || gotReq.jsCode != "good-code" {
		t.Fatalf("code2session params: %+v", gotReq)
	}

	// 微信侧错误码 → 未授权错误
	_, _, _, _, err = p.WxLogin(ctx, "bad-code")
	if err == nil || !strings.Contains(err.Error(), "invalid appid") {
		t.Fatalf("want wechat error surfaced, got %v", err)
	}
}

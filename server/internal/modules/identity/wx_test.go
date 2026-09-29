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

// wx_test.go — V2.2 微信登录链路：code2session → openid 未绑定直建号（D25 修订）
// → 顾客 Token；二登同号；claim 转绑矩阵见 customer_test.go。

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

func TestWxLoginCreatesMemberOnFirstLogin(t *testing.T) {
	p, _ := newWxEnv(t, nil)
	ctx := context.Background()

	// D24 dev 兜底 + D25 修订：首登直建号直发 Token，无 needs_bind/bind_ticket
	token, mid, err := p.WxLogin(ctx, "code-a")
	if err != nil || token == "" || mid == "" {
		t.Fatalf("first login: token=%q mid=%q err=%v", token, mid, err)
	}
	m, err := p.members.Get(ctx, mid)
	if err != nil {
		t.Fatalf("get member: %v", err)
	}
	// 纯微信会员：phone 空、name 空、wx_bound、member_no 已生成
	if m.Phone != "" || m.Name != "" || !m.WxBound || m.HasPassword || m.MemberNo == "" {
		t.Fatalf("first-login member = %+v", m)
	}

	// 二登：同一 member，不重复建号
	token2, mid2, err := p.WxLogin(ctx, "code-a")
	if err != nil || token2 == "" || mid2 != mid {
		t.Fatalf("second login: mid2=%s want %s err=%v", mid2, mid, err)
	}

	// 空 code → 400
	if _, _, err := p.WxLogin(ctx, "  "); err == nil {
		t.Fatal("empty code accepted")
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

	token, mid, err := p.WxLogin(ctx, "good-code")
	if err != nil || token == "" || mid == "" {
		t.Fatalf("stub login: err=%v", err)
	}
	// openid 来自 stub（真实 code2session），不是 dev: 前缀
	if _, ok, err := p.members.FindByOpenID(ctx, p.DB(), "o-real-1"); err != nil || !ok {
		t.Fatalf("openid lookup: ok=%v err=%v", ok, err)
	}
	if gotReq.appid != "wx-app" || gotReq.secret != "wx-secret" || gotReq.jsCode != "good-code" {
		t.Fatalf("code2session params: %+v", gotReq)
	}

	// 微信侧错误码 → 未授权错误
	_, _, err = p.WxLogin(ctx, "bad-code")
	if err == nil || !strings.Contains(err.Error(), "invalid appid") {
		t.Fatalf("want wechat error surfaced, got %v", err)
	}
}

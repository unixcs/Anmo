package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"anmo/server/internal/shared"
)

// wx.go — WeChat mini-program login (V2, plan §11). code2session exchanges a
// wx.login code for the openid; a bound openid signs a customer token directly
// (same member as H5), an unbound one returns a short-lived bind ticket that
// the customer redeems after SMS login (D24/D25).

const wxBindTTL = 10 * time.Minute

type wxSession struct {
	OpenID  string `json:"openid"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// code2session calls the WeChat API. cfg.Wx.APIBase is overridable so tests
// stub the endpoint with an httptest server.
func (p *Provider) code2session(ctx context.Context, code string) (string, error) {
	if p.cfg.Wx.AppID == "" {
		// D24: dev fallback mirrors SMS.Mode=dev (fixed code 123456) — the
		// code IS the openid. Never publish a mini-program against this.
		return "dev:" + code, nil
	}
	q := url.Values{}
	q.Set("appid", p.cfg.Wx.AppID)
	q.Set("secret", p.cfg.Wx.Secret)
	q.Set("js_code", code)
	q.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.cfg.Wx.APIBase+"/sns/jscode2session?"+q.Encode(), nil)
	if err != nil {
		return "", shared.Server("WX_REQUEST", err)
	}
	res, err := p.wxHTTP.Do(req)
	if err != nil {
		return "", shared.Server("WX_REQUEST", err)
	}
	defer res.Body.Close()
	var s wxSession
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		return "", shared.Server("WX_DECODE", err)
	}
	if s.ErrCode != 0 {
		return "", shared.Unauthorized("微信登录失败：" + s.ErrMsg)
	}
	if s.OpenID == "" {
		return "", shared.Server("WX_NO_OPENID", fmt.Errorf("code2session returned empty openid"))
	}
	return s.OpenID, nil
}

// WxLogin exchanges a wx.login code for either a customer token (openid
// already bound) or a bind ticket the client stores until binding completes.
func (p *Provider) WxLogin(ctx context.Context, code string) (token, memberID, bindTicket string, needsBind bool, err error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", "", "", false, shared.BadRequest("WX_BAD_CODE", "缺少 wx.login code")
	}
	openid, err := p.code2session(ctx, code)
	if err != nil {
		return "", "", "", false, err
	}
	id, ok, err := p.members.FindByOpenID(ctx, openid)
	if err != nil {
		return "", "", "", false, err
	}
	if ok {
		token, err := p.tokens.signCustomer(id, time.Duration(p.cfg.Auth.CustomerTokenHours)*time.Hour)
		if err != nil {
			return "", "", "", false, shared.Server("IDENTITY_TOKEN", err)
		}
		return token, id, "", false, nil
	}
	ticket, err := p.tokens.signBindTicket(openid)
	if err != nil {
		return "", "", "", false, shared.Server("IDENTITY_TOKEN", err)
	}
	return "", "", ticket, true, nil
}

// WxBind binds the ticket's openid to the authenticated customer's member.
func (p *Provider) WxBind(ctx context.Context, memberID, bindTicket string) error {
	openid, err := p.tokens.parseBindTicket(bindTicket)
	if err != nil {
		return shared.Unauthorized("绑定凭证无效或已过期，请重新进入小程序")
	}
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		return p.members.BindOpenID(ctx, tx, memberID, openid)
	})
}

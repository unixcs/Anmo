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

// wx.go — WeChat mini-program login (V2, plan §11; V2.2 修订 D25)。code2session
// 交换 openid：已绑定直发顾客 Token；未绑定在同一 immediate 事务内直接建号
// （纯微信会员：phone NULL、name ''）并签发 Token——bind_ticket 流程废除。
// 同一 openid 再次登录返回同一会员；并发首登由 BEGIN IMMEDIATE 串行化 +
// uk_member_wx_openid 兜底（R3）。

type wxSession struct {
	OpenID  string `json:"openid"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// wxErrMsg maps a WeChat errcode to an actionable Chinese message; the errcode
// stays in brackets for support, while the raw English errmsg/rid only goes to
// server logs (V2.2 第六批 review).
func wxErrMsg(errcode int) string {
	switch errcode {
	case 40029, 40163: // code 无效 / 已被使用：wx.login 重新取码即可恢复
		return "登录状态已过期，请重试"
	case 45011: // API 频率限制
		return "操作太频繁，请稍后再试"
	case -1: // 微信系统繁忙，官方口径为可重试
		return "微信服务繁忙，请稍后再试"
	case 40013, 40125, 41002: // appid/secret 配置错误：用户重试无解
		return "微信登录暂不可用，请联系店主"
	default:
		return "微信登录失败，请稍后再试"
	}
}

// code2session calls the WeChat API. cfg.Wx.APIBase is overridable so tests
// stub the endpoint with an httptest server.
func (p *Provider) code2session(ctx context.Context, code string) (string, error) {
	if p.cfg.Wx.AppID == "" {
		// D24: dev fallback mirrors the retired SMS dev code — the code IS the
		// openid. Never publish a mini-program against this.
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
		if p.log != nil {
			p.log.Warn("wx code2session rejected", "errcode", s.ErrCode, "errmsg", s.ErrMsg)
		}
		return "", shared.Unauthorized(fmt.Sprintf("%s[%d]", wxErrMsg(s.ErrCode), s.ErrCode))
	}
	if s.OpenID == "" {
		return "", shared.Server("WX_NO_OPENID", fmt.Errorf("code2session returned empty openid"))
	}
	return s.OpenID, nil
}

// WxLogin exchanges a wx.login code for a customer token. First sight of an
// openid creates the member inside the login transaction (D25 修订：直建号).
func (p *Provider) WxLogin(ctx context.Context, code string) (token, memberID string, err error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", "", shared.BadRequest("WX_BAD_CODE", "缺少 wx.login code")
	}
	openid, err := p.code2session(ctx, code)
	if err != nil {
		return "", "", err
	}
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		id, ok, e := p.members.FindByOpenID(ctx, tx, openid)
		if e != nil {
			return e
		}
		if ok {
			memberID = id
			return nil
		}
		memberID, e = p.members.CreateByOpenID(ctx, tx, openid)
		return e
	})
	if err != nil {
		return "", "", err
	}
	token, err = p.tokens.signCustomer(memberID, time.Duration(p.cfg.Auth.CustomerTokenHours)*time.Hour)
	if err != nil {
		return "", "", shared.Server("IDENTITY_TOKEN", err)
	}
	return token, memberID, nil
}

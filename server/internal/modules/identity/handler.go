package identity

import (
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

// handler.go — identity HTTP endpoints. Auth endpoints are public; claim /
// h5-password / wx-bind(-removed) 类顾客操作走顾客 JWT（api mux）。

type loginReq struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (p *Provider) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	token, user, err := p.AdminLogin(r.Context(), req.Phone, req.Password)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{
		"token": token,
		"user":  map[string]string{"id": user.id, "name": user.name, "role": user.role},
	})
}

// handleRegister — H5 手机号+密码注册（V2.2 R2）：成功即登录（返回顾客 Token）。
func (p *Provider) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	token, memberID, err := p.Register(r.Context(), req.Phone, req.Password)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]string{"token": token, "member_id": memberID})
}

// handleLogin — H5 手机号+密码登录（V2.2 R2）。
func (p *Provider) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	token, memberID, err := p.Login(r.Context(), req.Phone, req.Password)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]string{"token": token, "member_id": memberID})
}

type smsReq struct {
	Phone string `json:"phone"`
}

// smsGone — R6：短信登录已下线（生产 sms.mode=off）。旧版小程序过渡提示。
func smsGone() *shared.AppError {
	return shared.NewErr("SMS_DISABLED", "短信登录已下线，请升级小程序至最新版本", http.StatusGone)
}

// handleUpdateCredentials — 商家后台自助改登录手机号/密码（需管理员 JWT）。
func (p *Provider) handleUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	pr, ok := middleware.PrincipalFrom(r.Context())
	if !ok || pr.ActorType != "ADMIN" {
		shared.Unauthorized("请先登录商家后台").Write(w)
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPhone        string `json:"new_phone"`
		NewPassword     string `json:"new_password"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.UpdateCredentials(r.Context(), pr.ActorID, req.CurrentPassword, req.NewPhone, req.NewPassword); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

func (p *Provider) handleSMSSend(w http.ResponseWriter, r *http.Request) {
	if p.smsDisabled() {
		smsGone().Write(w)
		return
	}
	var req smsReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SendSMS(r.Context(), req.Phone); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"sent": true})
}

type smsVerifyReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

func (p *Provider) handleSMSVerify(w http.ResponseWriter, r *http.Request) {
	if p.smsDisabled() {
		smsGone().Write(w)
		return
	}
	var req smsVerifyReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	token, memberID, err := p.smsDevLogin(r.Context(), req.Phone, req.Code)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]string{"token": token, "member_id": memberID})
}

// handleWxLogin — V2 小程序登录（公开）：code → openid 绑定则直发顾客 Token，
// 未绑定同事务直建号再发 Token（D25 修订）。响应恒为 {token, member_id}。
func (p *Provider) handleWxLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	token, memberID, err := p.WxLogin(r.Context(), req.Code)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]string{"token": token, "member_id": memberID})
}

// handleWxClaim — 手机号撞号认领（顾客鉴权，V2.2 R4）：用该手机号的 H5 密码
// 把当前微信转绑到老账号并切换登录态（返回老账号新 Token）。
func (p *Provider) handleWxClaim(w http.ResponseWriter, r *http.Request) {
	pr, ok := middleware.PrincipalFrom(r.Context())
	if !ok || pr.ActorType != "CUSTOMER" {
		shared.Unauthorized("请先登录").Write(w)
		return
	}
	var req loginReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	token, memberID, err := p.ClaimByPhone(r.Context(), pr.ActorID, req.Phone, req.Password)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]string{"token": token, "member_id": memberID})
}

// handleSetH5Password — 小程序设置/重置 H5 密码（顾客鉴权，V2.2 R5）。
func (p *Provider) handleSetH5Password(w http.ResponseWriter, r *http.Request) {
	pr, ok := middleware.PrincipalFrom(r.Context())
	if !ok || pr.ActorType != "CUSTOMER" {
		shared.Unauthorized("请先登录").Write(w)
		return
	}
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetH5Password(r.Context(), pr.ActorID, req.NewPassword); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

var _ = shared.OK

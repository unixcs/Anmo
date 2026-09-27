package identity

import (
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

// handler.go — identity HTTP endpoints. All auth endpoints are public.

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

type smsReq struct {
	Phone string `json:"phone"`
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
	var req smsVerifyReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	token, memberID, err := p.CustomerLogin(r.Context(), req.Phone, req.Code)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]string{"token": token, "member_id": memberID})
}

var _ = shared.OK

package identity

import (
	"net/http"

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

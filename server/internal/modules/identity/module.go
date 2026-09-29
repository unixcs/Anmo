package identity

import "net/http"

// Mount registers identity routes. Auth endpoints are public; claim 与
// h5-password 是顾客操作，挂在顾客鉴权 api mux 上。
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	root.HandleFunc("POST /admin/auth/login", p.handleAdminLogin)
	admin.HandleFunc("PUT /admin/auth/credentials", p.handleUpdateCredentials)
	// 短信端点保留（过渡期）：off 模式 410 SMS_DISABLED，dev 可用（R6）
	root.HandleFunc("POST /api/auth/sms/send", p.handleSMSSend)
	root.HandleFunc("POST /api/auth/sms/verify", p.handleSMSVerify)
	// H5 手机号+密码（V2.2 R2）
	root.HandleFunc("POST /api/auth/register", p.handleRegister)
	root.HandleFunc("POST /api/auth/login", p.handleLogin)
	// 微信登录：直建号直发 Token（D25 修订）；bind 路由随 bind_ticket 一并删除
	root.HandleFunc("POST /api/auth/wx/login", p.handleWxLogin)
	api.HandleFunc("POST /api/auth/wx/claim", p.handleWxClaim)
	api.HandleFunc("PUT /api/me/h5-password", p.handleSetH5Password)
}

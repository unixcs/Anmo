package identity

import "net/http"

// Mount registers identity routes. Auth endpoints are public (no token);
// wx/bind is customer-guarded and lives on the api mux.
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	root.HandleFunc("POST /admin/auth/login", p.handleAdminLogin)
	admin.HandleFunc("PUT /admin/auth/credentials", p.handleUpdateCredentials)
	root.HandleFunc("POST /api/auth/sms/send", p.handleSMSSend)
	root.HandleFunc("POST /api/auth/sms/verify", p.handleSMSVerify)
	root.HandleFunc("POST /api/auth/wx/login", p.handleWxLogin)
	api.HandleFunc("POST /api/auth/wx/bind", p.handleWxBind)
}

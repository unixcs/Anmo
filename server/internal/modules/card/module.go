package card

import "net/http"

// Mount registers card routes.
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	admin.HandleFunc("GET /admin/card-templates", p.handleListTemplates)
	admin.HandleFunc("POST /admin/card-templates", p.handleCreateTemplate)
	admin.HandleFunc("PUT /admin/card-templates/{id}/status", p.handleSetTemplateStatus)
	admin.HandleFunc("PUT /admin/card-templates/{id}/rules", p.handleSetRules)

	admin.HandleFunc("POST /admin/cards", p.handleIssue)
	admin.HandleFunc("GET /admin/members/{id}/cards", p.handleListByMember)
	admin.HandleFunc("PUT /admin/cards/{id}/adjust", p.handleAdjust)
	admin.HandleFunc("PUT /admin/cards/{id}/cancel", p.handleCancel)
	admin.HandleFunc("GET /admin/cards/{id}/transactions", p.handleTransactions)

	api.HandleFunc("GET /api/me/cards", p.handleMyCards)
	api.HandleFunc("GET /api/me/cards/{id}/transactions", p.handleMyCardTransactions)
}

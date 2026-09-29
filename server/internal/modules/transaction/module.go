package transaction

import "net/http"

// Mount registers settlement routes (admin only).
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	admin.HandleFunc("POST /admin/appointments/{id}/redeem", p.handleSettleCard)
	admin.HandleFunc("POST /admin/cards/{id}/redeem", p.handleRedeemCardDirect)
	admin.HandleFunc("POST /admin/appointments/{id}/payments", p.handleSettlePay)
	admin.HandleFunc("POST /admin/walkin/settle", p.handleWalkInSettle)
	admin.HandleFunc("PUT /admin/redemptions/{id}/reverse", p.handleReverse)
	admin.HandleFunc("GET /admin/workbench", p.handleWorkbench)
	admin.HandleFunc("GET /admin/payments", p.handleListPayments)
	admin.HandleFunc("GET /admin/redemptions", p.handleListRedemptions)
}

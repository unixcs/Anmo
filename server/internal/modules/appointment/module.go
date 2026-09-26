package appointment

import "net/http"

// Mount registers appointment routes (customer + admin).
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	api.HandleFunc("POST /api/appointments", p.handleCustomerCreate)
	api.HandleFunc("GET /api/appointments", p.handleCustomerList)
	api.HandleFunc("GET /api/appointments/{id}", p.handleCustomerGet)
	api.HandleFunc("PUT /api/appointments/{id}/cancel", p.handleCustomerCancel)
	api.HandleFunc("PUT /api/appointments/{id}/reschedule", p.handleCustomerReschedule)

	admin.HandleFunc("GET /admin/appointments", p.handleAdminList)
	admin.HandleFunc("GET /admin/today", p.handleAdminToday)
	admin.HandleFunc("PUT /admin/appointments/{id}/confirm", p.handleAdminConfirm)
	admin.HandleFunc("PUT /admin/appointments/{id}/start", p.handleAdminStart)
	admin.HandleFunc("PUT /admin/appointments/{id}/complete", p.handleAdminComplete)
	admin.HandleFunc("PUT /admin/appointments/{id}/cancel", p.handleAdminCancel)
	admin.HandleFunc("PUT /admin/appointments/{id}/no-show", p.handleAdminNoShow)
	admin.HandleFunc("PUT /admin/appointments/{id}/reschedule", p.handleAdminReschedule)
}

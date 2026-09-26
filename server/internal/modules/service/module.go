package service

import "net/http"

// Mount registers service routes.
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	admin.HandleFunc("GET /admin/service-categories", p.handleListCategories)
	admin.HandleFunc("POST /admin/service-categories", p.handleCreateCategory)
	admin.HandleFunc("PUT /admin/service-categories/{id}/status", p.handleSetCategoryStatus)

	admin.HandleFunc("GET /admin/services", p.handleListItems)
	admin.HandleFunc("POST /admin/services", p.handleCreateItem)
	admin.HandleFunc("PUT /admin/services/{id}", p.handleUpdateItem)
	admin.HandleFunc("PUT /admin/services/{id}/status", p.handleSetItemStatus)

	api.HandleFunc("GET /api/services", p.handleCustomerCatalog)
}

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

	// D28: 服务标签 + 服务记录（商家内部数据，用户端不可见）
	admin.HandleFunc("GET /admin/service-tags", p.handleListServiceTags)
	admin.HandleFunc("POST /admin/service-tags", p.handleCreateServiceTag)
	admin.HandleFunc("PUT /admin/service-tags/{id}", p.handleUpdateServiceTag)
	admin.HandleFunc("DELETE /admin/service-tags/{id}", p.handleDeleteServiceTag)
	admin.HandleFunc("GET /admin/members/{id}/service-records", p.handleListMemberRecords)
	admin.HandleFunc("PUT /admin/service-records/{id}/merchant-note", p.handleMerchantNote)
	admin.HandleFunc("POST /admin/service-records/{id}/revoke", p.handleRevokeRecord)

	api.HandleFunc("GET /api/services", p.handleCustomerCatalog)
}

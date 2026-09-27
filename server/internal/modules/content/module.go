package content

import "net/http"

// Mount registers content routes.
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	api.HandleFunc("GET /api/home", p.handleHome)
	api.HandleFunc("GET /api/settings", p.handlePublicSettings)
	api.HandleFunc("GET /api/store/status", p.HandleStoreStatus)

	admin.HandleFunc("GET /admin/content/pages", p.handleGetPageConfig)
	admin.HandleFunc("PUT /admin/content/pages", p.handleSavePageConfig)
	admin.HandleFunc("GET /admin/banners", p.handleListBanners)
	admin.HandleFunc("POST /admin/banners", p.handleSaveBanner)
	admin.HandleFunc("PUT /admin/banners/{id}/status", p.handleSetBannerStatus)
	admin.HandleFunc("GET /admin/announcements", p.handleListAnnouncements)
	admin.HandleFunc("POST /admin/announcements", p.handleSaveAnnouncement)
	admin.HandleFunc("PUT /admin/announcements/{id}/status", p.handleSetAnnouncementStatus)
	admin.HandleFunc("GET /admin/settings", p.handleSettings)
	admin.HandleFunc("PUT /admin/settings", p.handleSaveSetting)
}

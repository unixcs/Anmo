package member

import "net/http"

// Mount registers member routes: admin management + customer profile.
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	admin.HandleFunc("GET /admin/members", p.handleList)
	admin.HandleFunc("POST /admin/members", p.handleCreate)
	admin.HandleFunc("GET /admin/members/{id}", p.handleGet)
	admin.HandleFunc("PUT /admin/members/{id}", p.handleUpdate)
	admin.HandleFunc("PUT /admin/members/{id}/tags", p.handleSetTags)
	admin.HandleFunc("GET /admin/tags", p.handleListTags)
	admin.HandleFunc("POST /admin/tags", p.handleCreateTag)

	api.HandleFunc("GET /api/me/profile", p.handleMyProfile)
	api.HandleFunc("PUT /api/me/profile", p.handleUpdateMyProfile)
}

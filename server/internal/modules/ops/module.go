package ops

import "net/http"

// Mount registers ops routes (admin).
func (p *Provider) Mount(root, admin, api *http.ServeMux) {
	admin.HandleFunc("GET /admin/logs", p.handleLogs)
	admin.HandleFunc("GET /admin/insights", p.handleInsights)
	admin.HandleFunc("POST /admin/ops/daily", p.handleDaily)
}

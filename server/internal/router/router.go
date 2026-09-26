// Package router assembles the HTTP mux: health, module mounts, middleware.
package router

import (
	"log/slog"
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

// Module is the contract every internal/modules/* package exposes to the
// router (AGENTS.md 模块地图). root=public, admin=JWT(OWNER/OPERATOR),
// api=JWT(customer).
type Module interface {
	Mount(root, admin, api *http.ServeMux)
}

// New builds the root handler. Admin routes live under /admin/, customer
// routes under /api/; both prefixes are auth-guarded here. Public routes
// (login, SMS) are registered by the identity module directly on root.
func New(log *slog.Logger, verify middleware.TokenVerifier, opLog func(r *http.Request, status int), mods ...Module) http.Handler {
	root := http.NewServeMux()
	admin := http.NewServeMux()
	api := http.NewServeMux()

	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, map[string]string{"status": "ok"})
	})

	for _, m := range mods {
		m.Mount(root, admin, api)
	}

	// auth guards (innermost relative to the global chain)
	root.Handle("/admin/", middleware.NewAuth(verify, true)(admin))
	root.Handle("/api/", middleware.NewAuth(verify, false)(api))

	handler := middleware.Recover(log)(root)
	handler = middleware.OperationLog(opLog)(handler)
	handler = middleware.Logging(log)(handler)
	handler = middleware.RequestIDMw(handler)
	return handler
}

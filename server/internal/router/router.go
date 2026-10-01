// Package router assembles the HTTP mux: health, module mounts, middleware.
package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

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
// health is the liveness probe dependency check (nil = process-only): it must
// touch the database, or /healthz reports green while the store is dead (F14).
func New(log *slog.Logger, verify middleware.TokenVerifier, opLog func(r *http.Request, status int, hdr http.Header), health func(ctx context.Context) error, mods ...Module) http.Handler {
	root := http.NewServeMux()
	admin := http.NewServeMux()
	api := http.NewServeMux()

	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if health != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := health(ctx); err != nil {
				shared.Fail(w, shared.NewErr("DB_DOWN", "database unavailable", http.StatusServiceUnavailable))
				return
			}
		}
		shared.OK(w, map[string]string{"status": "ok"})
	})

	// service index for the bare root path (exact match only; unknown paths
	// still fall through to 404)
	root.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, map[string]any{
			"service": "anmo",
			"version": "v1",
			"endpoints": map[string]string{
				"health":   "GET /healthz",
				"customer": "/api/  (JWT 顾客端)",
				"admin":    "/admin/ (JWT OWNER/OPERATOR 管理端)",
			},
		})
	})

	for _, m := range mods {
		m.Mount(root, admin, api)
	}

	// auth guards (innermost relative to the global chain)
	root.Handle("/admin/", middleware.NewAuth(verify, true)(admin))
	root.Handle("/api/", middleware.NewAuth(verify, false)(api))

	handler := middleware.Recover(log)(root)
	handler = middleware.AuthRateLimit(handler) // F3: 公开鉴权端点按 IP 限速
	handler = middleware.OperationLog(opLog)(handler)
	handler = middleware.Logging(log)(handler)
	handler = middleware.BodyLimit(1 << 20)(handler) // F15: 1MiB 请求体上限（防恶意大包）
	handler = middleware.RequestIDMw(handler)
	return handler
}

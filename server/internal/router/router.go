// Package router assembles the HTTP mux: health, module mounts, middleware.
package router

import (
	"log/slog"
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

// Module is the contract every internal/modules/* package exposes to the
// router (AGENTS.md 模块地图).
type Module interface {
	Mount(mux *http.ServeMux)
}

// New builds the root handler.
func New(log *slog.Logger, mods ...Module) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, map[string]string{"status": "ok"})
	})

	for _, m := range mods {
		m.Mount(mux)
	}

	handler := middleware.Recover(log)(mux)
	handler = middleware.Logging(log)(handler)
	handler = middleware.RequestIDMw(handler)
	return handler
}

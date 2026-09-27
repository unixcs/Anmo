// Package app assembles the full application (modules + router + seeds) so
// main.go and the E2E suite share one wiring path.
package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"anmo/server/internal/config"
	"anmo/server/internal/middleware"
	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/modules/card"
	"anmo/server/internal/modules/content"
	"anmo/server/internal/modules/identity"
	"anmo/server/internal/modules/member"
	"anmo/server/internal/modules/ops"
	"anmo/server/internal/modules/service"
	"anmo/server/internal/modules/transaction"
	"anmo/server/internal/router"
	"anmo/server/internal/shared"
)

// Build wires every module and returns the root HTTP handler.
func Build(db shared.DB, cfg *config.Config, log *slog.Logger) http.Handler {
	memberMod := member.New(db, cfg)
	serviceMod := service.New(db, cfg)
	identityMod := identity.New(db, cfg, memberMod, log)
	cardMod := card.New(db, cfg)
	contentMod := content.New(db, cfg)
	// D20: booking rules resolve in content (settings > cfg > default);
	// appointment consumes them through the RulesSource interface.
	bookingRules := appointment.RulesSourceFunc(func(ctx context.Context) (*appointment.BusinessRules, error) {
		r, err := contentMod.BookingRules(ctx)
		if err != nil {
			return nil, err
		}
		return &appointment.BusinessRules{
			OpenTime: r.OpenTime, CloseTime: r.CloseTime, NoonSplit: r.NoonSplit,
			SlotMinutes: r.SlotMinutes, SlotCapacity: r.SlotCapacity,
		}, nil
	})
	appointmentMod := appointment.New(db, cfg, serviceMod, bookingRules)
	transactionMod := transaction.New(db, cfg, cardMod, appointmentMod, memberMod, serviceMod)
	opsMod := ops.New(db, cfg, cardMod, appointmentMod, memberMod)

	opLog := opsMod.NewLogWriter(log)
	logEntry := func(r *http.Request, status int) {
		pr, _ := middleware.PrincipalFrom(r.Context())
		detail, _ := json.Marshal(map[string]any{"method": r.Method, "status": status, "query": r.URL.RawQuery})
		opLog(ops.LogEntry{
			ActorType: pr.ActorType, ActorID: pr.ActorID,
			Action: r.Method + " " + r.URL.Path,
			Detail: string(detail), IP: r.RemoteAddr,
		})
	}
	return router.New(log, identityMod.TokenVerifier(), logEntry,
		identityMod, memberMod, serviceMod, cardMod,
		appointmentMod, transactionMod, contentMod, opsMod,
	)
}

// SeedIdentity runs idempotent identity seeding (OWNER account).
func SeedIdentity(db shared.DB, cfg *config.Config, log *slog.Logger) error {
	memberMod := member.New(db, cfg)
	identityMod := identity.New(db, cfg, memberMod, log)
	return identityMod.EnsureSeed(context.Background())
}

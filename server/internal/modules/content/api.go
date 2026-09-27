// Package content — public API surface of the content module.
//
// Module boundary rule (AGENTS.md): other modules may call ONLY the exported
// Provider methods defined here; importing repo/model internals is forbidden.
package content

import (
	"context"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/shared"
)

// Provider is the handle other modules and main receive.
type Provider struct {
	db  shared.DB
	cfg *config.Config
	// store-status read-only deps (set via Wire, goal §17)
	appts    interface {
		ServingNow(ctx context.Context) (*appointment.ServingSlot, error)
	}
	services interface {
		MinActiveDuration(ctx context.Context) (int, error)
	}
}

// New builds the module Provider. Dependencies are injected by main.
func New(db shared.DB, cfg *config.Config) *Provider {
	return &Provider{db: db, cfg: cfg}
}

// DB exposes the pool to the module's own handler/service files only.
func (p *Provider) DB() shared.DB { return p.db }

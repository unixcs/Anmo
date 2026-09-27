// Package appointment — public API surface of the appointment module.
//
// Module boundary rule (AGENTS.md): other modules may call ONLY the exported
// Provider methods defined here; importing repo/model internals is forbidden.
package appointment

import (
	"anmo/server/internal/config"

	"anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
)

// Provider is the handle other modules and main receive.
type Provider struct {
	db       shared.DB
	cfg      *config.Config
	services *service.Provider
	rules    RulesSource // nil → cfg/defaults (unit tests)
}

// New builds the module Provider. Dependencies are injected by main.
// rules may be nil (tests) — booking rules then come from cfg/defaults.
func New(db shared.DB, cfg *config.Config, services *service.Provider, rules RulesSource) *Provider {
	return &Provider{db: db, cfg: cfg, services: services, rules: rules}
}

// DB exposes the pool to the module's own handler/service files only.
func (p *Provider) DB() shared.DB { return p.db }

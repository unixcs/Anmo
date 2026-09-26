// Package transaction — public API surface of the transaction module.
//
// Module boundary rule (AGENTS.md): other modules may call ONLY the exported
// Provider methods defined here; importing repo/model internals is forbidden.
package transaction

import (
	"anmo/server/internal/config"

	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/modules/card"
	"anmo/server/internal/shared"
)

// Provider is the handle other modules and main receive.
type Provider struct {
	db           shared.DB
	cfg          *config.Config
	cards        *card.Provider
	appointments *appointment.Provider
}

// New builds the module Provider. Dependencies are injected by main.
func New(db shared.DB, cfg *config.Config, cards *card.Provider, appointments *appointment.Provider) *Provider {
	return &Provider{db: db, cfg: cfg, cards: cards, appointments: appointments}
}

// DB exposes the pool to the module's own handler/service files only.
func (p *Provider) DB() shared.DB { return p.db }

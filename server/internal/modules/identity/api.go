// Package identity — public API surface of the identity module.
//
// Module boundary rule (AGENTS.md): other modules may call ONLY the exported
// Provider methods defined here; importing repo/model internals is forbidden.
package identity

import (
	"log/slog"
	"net/http"
	"time"

	"anmo/server/internal/config"

	"anmo/server/internal/middleware"
	"anmo/server/internal/modules/member"
	"anmo/server/internal/shared"
)

// Provider is the handle other modules and main receive.
type Provider struct {
	db      shared.DB
	cfg     *config.Config
	members *member.Provider
	tokens  *tokenService
	sms     *smsStore
	sender  smsSender
	wxHTTP  *http.Client
	log     *slog.Logger
}

// New builds the module Provider. Dependencies are injected by main.
func New(db shared.DB, cfg *config.Config, members *member.Provider, log *slog.Logger) *Provider {
	return &Provider{
		db:      db,
		cfg:     cfg,
		members: members,
		tokens:  &tokenService{secret: []byte(cfg.Auth.JWTSecret)},
		sms:     newSMSStore(),
		sender:  devSender{log: log},
		wxHTTP:  &http.Client{Timeout: 10 * time.Second},
		log:     log,
	}
}

// TokenVerifier exposes JWT verification to the auth middleware.
func (p *Provider) TokenVerifier() middleware.TokenVerifier {
	return p.tokens.Verify
}

// DB exposes the pool to the module's own handler/service files only.
func (p *Provider) DB() shared.DB { return p.db }

package card

import (
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

// handleMyCards returns the signed-in customer's cards. Identity comes from
// the token only (plan §100).
func (p *Provider) handleMyCards(w http.ResponseWriter, r *http.Request) {
	pr, ok := middleware.PrincipalFrom(r.Context())
	if !ok || pr.ActorType != "CUSTOMER" {
		shared.Unauthorized("请先登录").Write(w)
		return
	}
	cards, err := p.ListByMember(r.Context(), pr.ActorID)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, cards)
}

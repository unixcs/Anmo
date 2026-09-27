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

// handleMyCardTransactions — 顾客端卡使用明细（§14：无预约的核销也必须显示）。
// 卡归属只看 token 会员，不信任前端传参。
func (p *Provider) handleMyCardTransactions(w http.ResponseWriter, r *http.Request) {
	pr, ok := middleware.PrincipalFrom(r.Context())
	if !ok || pr.ActorType != "CUSTOMER" {
		shared.Unauthorized("请先登录").Write(w)
		return
	}
	if _, err := p.GetOwned(r.Context(), r.PathValue("id"), pr.ActorID); err != nil {
		shared.Fail(w, err)
		return
	}
	txs, err := p.Transactions(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, txs)
}

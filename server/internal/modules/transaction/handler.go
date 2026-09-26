package transaction

import (
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

func operatorOf(r *http.Request) string {
	if pr, ok := middleware.PrincipalFrom(r.Context()); ok {
		return pr.ActorID
	}
	return ""
}

type settleCardReq struct {
	CardID         string `json:"card_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (p *Provider) handleSettleCard(w http.ResponseWriter, r *http.Request) {
	var req settleCardReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	rd, pay, err := p.SettleByCard(r.Context(), r.PathValue("id"), req.CardID, operatorOf(r), req.IdempotencyKey)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"redemption": rd, "payment": pay})
}

type settlePayReq struct {
	Method         string `json:"method"`
	AmountCents    int64  `json:"amount"`
	ReferenceNo    string `json:"reference_no"`
	Remark         string `json:"remark"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (p *Provider) handleSettlePay(w http.ResponseWriter, r *http.Request) {
	var req settlePayReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	pay, err := p.SettleByPay(r.Context(), r.PathValue("id"), req.Method, req.AmountCents,
		req.ReferenceNo, req.Remark, operatorOf(r), req.IdempotencyKey)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, pay)
}

type reverseReq struct {
	Reason string `json:"reason"`
}

func (p *Provider) handleReverse(w http.ResponseWriter, r *http.Request) {
	var req reverseReq
	_ = shared.DecodeJSON(r, &req)
	if err := p.ReverseRedemption(r.Context(), r.PathValue("id"), req.Reason, operatorOf(r)); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"reversed": true})
}

func (p *Provider) handleListPayments(w http.ResponseWriter, r *http.Request) {
	pays, err := p.ListPayments(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, pays)
}

func (p *Provider) handleListRedemptions(w http.ResponseWriter, r *http.Request) {
	rds, err := p.ListRedemptions(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, rds)
}

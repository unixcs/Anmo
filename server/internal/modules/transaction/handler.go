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

// recordFieldsReq — 结算请求中的服务记录字段（D28 §四.1-3/§四.6，字段名冻结：
// body_parts / service_method / tech_note / communicated）。
type recordFieldsReq struct {
	BodyParts     []string `json:"body_parts"`
	ServiceMethod string   `json:"service_method"`
	TechNote      string   `json:"tech_note"`
	Communicated  bool     `json:"communicated"`
}

func (r recordFieldsReq) fields() RecordFields {
	return RecordFields{
		BodyParts:     r.BodyParts,
		ServiceMethod: r.ServiceMethod,
		TechNote:      r.TechNote,
		Communicated:  r.Communicated,
	}
}

type settleCardReq struct {
	CardID         string `json:"card_id"`
	ServiceID      string `json:"service_id"` // 可选：实际服务（≠预约服务，§11）
	IdempotencyKey string `json:"idempotency_key"`
	recordFieldsReq
}

func (p *Provider) handleSettleCard(w http.ResponseWriter, r *http.Request) {
	var req settleCardReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	rd, pay, err := p.SettleByCard(r.Context(), r.PathValue("id"), req.CardID, req.ServiceID,
		operatorOf(r), req.IdempotencyKey, req.fields())
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
	recordFieldsReq
}

// handleRedeemCardDirect — 散客核销（D19）：无预约、按卡直接扣次。
func (p *Provider) handleRedeemCardDirect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceID      string `json:"service_id"`
		IdempotencyKey string `json:"idempotency_key"`
		recordFieldsReq
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if req.ServiceID == "" || req.IdempotencyKey == "" {
		shared.BadRequest("RDM_BAD_REQ", "缺少 service_id 或 idempotency_key").Write(w)
		return
	}
	rd, py, err := p.RedeemWalkIn(r.Context(), r.PathValue("id"), req.ServiceID,
		operatorOf(r), req.IdempotencyKey, req.fields())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"redemption": rd, "payment": py})
}

func (p *Provider) handleSettlePay(w http.ResponseWriter, r *http.Request) {
	var req settlePayReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	pay, err := p.SettleByPay(r.Context(), r.PathValue("id"), req.Method, req.AmountCents,
		req.ReferenceNo, req.Remark, operatorOf(r), req.IdempotencyKey, req.fields())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, pay)
}

// handleWalkInSettle — 散客快速结算（D29）。
func (p *Provider) handleWalkInSettle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone          string `json:"phone"`
		Name           string `json:"name"`
		ServiceID      string `json:"service_id"`
		PayMethod      string `json:"pay_method"`
		AmountCents    int64  `json:"amount"`
		ReferenceNo    string `json:"reference_no"`
		Remark         string `json:"remark"`
		IdempotencyKey string `json:"idempotency_key"`
		recordFieldsReq
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	res, err := p.WalkInSettle(r.Context(), WalkInInput{
		Phone: req.Phone, Name: req.Name,
		ServiceID: req.ServiceID, PayMethod: req.PayMethod, AmountCents: req.AmountCents,
		ReferenceNo: req.ReferenceNo, Remark: req.Remark,
		Record: req.fields(), OperatorID: operatorOf(r), IdemKey: req.IdempotencyKey,
	})
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{
		"payment":        res.Payment,
		"record":         res.Record,
		"member_created": res.MemberCreated,
		"record_skipped": res.Record == nil,
	})
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

func (p *Provider) handleWorkbench(w http.ResponseWriter, r *http.Request) {
	summary, cards, err := p.Workbench(r.Context(), r.URL.Query().Get("date"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"summary": summary, "cards": cards})
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

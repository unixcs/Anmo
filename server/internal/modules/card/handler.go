package card

import (
	"net/http"

	"anmo/server/internal/shared"
)

func (p *Provider) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var in NewTemplate
	if err := shared.DecodeJSON(r, &in); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	t, err := p.CreateTemplate(r.Context(), in)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, t)
}

func (p *Provider) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	ts, err := p.ListTemplates(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, ts)
}

func (p *Provider) handleSetTemplateStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetTemplateStatus(r.Context(), r.PathValue("id"), req.Status); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

func (p *Provider) handleSetRules(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceIDs []string `json:"service_ids"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetServiceRules(r.Context(), r.PathValue("id"), req.ServiceIDs); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

type issueReq struct {
	MemberID   string `json:"member_id"`
	TemplateID string `json:"card_template_id"`
}

func (p *Provider) handleIssue(w http.ResponseWriter, r *http.Request) {
	var req issueReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	c, err := p.IssueCard(r.Context(), req.MemberID, req.TemplateID, operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, c)
}

func (p *Provider) handleListByMember(w http.ResponseWriter, r *http.Request) {
	cs, err := p.ListByMember(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, cs)
}

type adjustReq struct {
	Delta  int    `json:"delta"`
	Remark string `json:"remark"`
}

func (p *Provider) handleAdjust(w http.ResponseWriter, r *http.Request) {
	var req adjustReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	c, err := p.Adjust(r.Context(), r.PathValue("id"), req.Delta, req.Remark, operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, c)
}

func (p *Provider) handleCancel(w http.ResponseWriter, r *http.Request) {
	if err := p.Cancel(r.Context(), r.PathValue("id"), operatorOf(r)); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"cancelled": true})
}

func (p *Provider) handleTransactions(w http.ResponseWriter, r *http.Request) {
	ts, err := p.Transactions(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, ts)
}

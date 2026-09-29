package service

import (
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

func (p *Provider) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Sort int    `json:"sort"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	c, err := p.CreateCategory(r.Context(), req.Name, req.Sort)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, c)
}

func (p *Provider) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := p.ListCategories(r.Context(), false)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, cats)
}

func (p *Provider) handleSetCategoryStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetCategoryStatus(r.Context(), r.PathValue("id"), req.Status); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

func (p *Provider) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var in NewItem
	if err := shared.DecodeJSON(r, &in); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	it, err := p.CreateItem(r.Context(), in)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, it)
}

func (p *Provider) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	var in NewItem
	if err := shared.DecodeJSON(r, &in); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	it, err := p.UpdateItem(r.Context(), r.PathValue("id"), in)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, it)
}

func (p *Provider) handleSetItemStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetItemStatus(r.Context(), r.PathValue("id"), req.Status); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

func (p *Provider) handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := p.ListItems(r.Context(), false)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, items)
}

// customer: active only
func (p *Provider) handleCustomerCatalog(w http.ResponseWriter, r *http.Request) {
	cats, err := p.ListCategories(r.Context(), true)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	items, err := p.ListItems(r.Context(), true)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"categories": cats, "services": items})
}

// --- D28: 服务标签 / 服务记录 ---

func (p *Provider) handleListServiceTags(w http.ResponseWriter, r *http.Request) {
	tags, err := p.ListServiceTags(r.Context(), r.URL.Query().Get("group"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, tags)
}

func (p *Provider) handleCreateServiceTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Group string `json:"group"`
		Name  string `json:"name"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	t, err := p.CreateServiceTag(r.Context(), req.Group, req.Name)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, t)
}

func (p *Provider) handleUpdateServiceTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   *string `json:"name"`
		Sort   *int    `json:"sort"`
		Status *string `json:"status"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	t, err := p.UpdateServiceTag(r.Context(), r.PathValue("id"), req.Name, req.Sort, req.Status)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, t)
}

func (p *Provider) handleDeleteServiceTag(w http.ResponseWriter, r *http.Request) {
	if err := p.DeleteServiceTag(r.Context(), r.PathValue("id")); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"deleted": true})
}

func (p *Provider) handleListMemberRecords(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, summary, filters, err := p.ListMemberRecords(r.Context(), r.PathValue("id"),
		q.Get("part"), q.Get("method"), q.Get("range"), q.Get("q"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"items": items, "summary": summary, "filters": filters})
}

func (p *Provider) handleMerchantNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Note string `json:"note"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	rec, err := p.UpdateMerchantNote(r.Context(), r.PathValue("id"), req.Note, operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, rec)
}

// handleRevokeRecord — 散客记录独立撤销（D28/D29）：record REVERSED + 原 payment
// VOIDED 同事务；卡核销记录由 RevokeRecord 拒绝并指引导向核销撤销入口。
func (p *Provider) handleRevokeRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := p.RevokeRecord(r.Context(), id, operatorOf(r)); err != nil {
		shared.Fail(w, err)
		return
	}
	rec, err := p.GetRecord(r.Context(), id)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, rec)
}

func operatorOf(r *http.Request) string {
	if pr, ok := middleware.PrincipalFrom(r.Context()); ok {
		return pr.ActorID
	}
	return ""
}

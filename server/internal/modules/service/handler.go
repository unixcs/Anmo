package service

import (
	"net/http"

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

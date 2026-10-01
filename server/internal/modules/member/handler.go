package member

import (
	"net/http"

	"anmo/server/internal/middleware"
	"anmo/server/internal/shared"
)

// handler.go — member HTTP endpoints.

func (p *Provider) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in NewMember
	if err := shared.DecodeJSON(r, &in); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	m, err := p.Create(r.Context(), in)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, m)
}

func (p *Provider) handleList(w http.ResponseWriter, r *http.Request) {
	out, total, err := p.List(r.Context(), ListParams{
		Keyword:  r.URL.Query().Get("keyword"),
		TagID:    r.URL.Query().Get("tag_id"),
		CardType: r.URL.Query().Get("card_type"),
		Page:     shared.PageFromRequest(r),
	})
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.PageOK(w, out, total, shared.PageFromRequest(r))
}

func (p *Provider) handleGet(w http.ResponseWriter, r *http.Request) {
	m, err := p.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	tags, _ := p.TagsOf(r.Context(), m.ID)
	shared.OK(w, map[string]any{"member": m, "tags": tags})
}

func (p *Provider) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var u ProfileUpdate
	if err := shared.DecodeJSON(r, &u); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	m, err := p.UpdateProfile(r.Context(), r.PathValue("id"), u)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, m)
}

// handleAdminSetPassword — 商家后台重置会员 H5 密码（V2.2 R5：忘记密码闭环）。
func (p *Provider) handleAdminSetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.AdminSetPassword(r.Context(), r.PathValue("id"), req.NewPassword); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

type setTagsReq struct {
	TagIDs []string `json:"tag_ids"`
}

func (p *Provider) handleSetTags(w http.ResponseWriter, r *http.Request) {
	var req setTagsReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetTags(r.Context(), r.PathValue("id"), req.TagIDs); err != nil {
		shared.Fail(w, err)
		return
	}
	tags, _ := p.TagsOf(r.Context(), r.PathValue("id"))
	shared.OK(w, tags)
}

func (p *Provider) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := p.ListTags(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, tags)
}

type createTagReq struct {
	Name string `json:"name"`
}

func (p *Provider) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var req createTagReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	t, err := p.CreateTag(r.Context(), req.Name)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, t)
}

func (p *Provider) handleRenameTag(w http.ResponseWriter, r *http.Request) {
	var req createTagReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.RenameTag(r.Context(), r.PathValue("id"), req.Name); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"renamed": true})
}

func (p *Provider) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	if err := p.DeleteTag(r.Context(), r.PathValue("id")); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"deleted": true})
}

// --- customer endpoints: identity always comes from the token (§100) ---

func (p *Provider) handleMyProfile(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	m, err := p.Get(r.Context(), pr.ActorID)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	tags, _ := p.TagsOf(r.Context(), m.ID)
	shared.OK(w, map[string]any{"member": m, "tags": tags})
}

func (p *Provider) handleUpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	var u ProfileUpdate
	if err := shared.DecodeJSON(r, &u); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	m, err := p.UpdateProfile(r.Context(), pr.ActorID, u)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	// F17（第九批审查）：与 GET /api/me/profile 及 H5 端约定一致——
	// 包一层 {member}（裸对象会让前端 res.member 得到 undefined）。
	shared.OK(w, map[string]any{"member": m})
}

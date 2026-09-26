package content

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

func (p *Provider) handleHome(w http.ResponseWriter, r *http.Request) {
	out, err := p.Home(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, out)
}

func (p *Provider) handlePublicSettings(w http.ResponseWriter, r *http.Request) {
	out, err := p.PublicSettings(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, out)
}

func (p *Provider) handleGetPageConfig(w http.ResponseWriter, r *http.Request) {
	blocks, err := p.GetPageConfig(r.Context(), r.URL.Query().Get("page"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, blocks)
}

type savePageReq struct {
	Page   string      `json:"page"`
	Blocks []PageBlock `json:"blocks"`
}

func (p *Provider) handleSavePageConfig(w http.ResponseWriter, r *http.Request) {
	var req savePageReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SavePageConfig(r.Context(), req.Page, req.Blocks, operatorOf(r)); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"saved": true})
}

func (p *Provider) handleListBanners(w http.ResponseWriter, r *http.Request) {
	bs, err := p.listBanners(r.Context(), false)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, bs)
}

func (p *Provider) handleSaveBanner(w http.ResponseWriter, r *http.Request) {
	var b Banner
	if err := shared.DecodeJSON(r, &b); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SaveBanner(r.Context(), &b); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, b)
}

func (p *Provider) handleSetBannerStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetBannerStatus(r.Context(), r.PathValue("id"), req.Status); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

func (p *Provider) handleListAnnouncements(w http.ResponseWriter, r *http.Request) {
	as, err := p.listAnnouncements(r.Context(), false)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, as)
}

func (p *Provider) handleSaveAnnouncement(w http.ResponseWriter, r *http.Request) {
	var a Announcement
	if err := shared.DecodeJSON(r, &a); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SaveAnnouncement(r.Context(), &a); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleSetAnnouncementStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SetAnnouncementStatus(r.Context(), r.PathValue("id"), req.Status); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"updated": true})
}

func (p *Provider) handleSettings(w http.ResponseWriter, r *http.Request) {
	out, err := p.Settings(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, out)
}

type saveSettingReq struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (p *Provider) handleSaveSetting(w http.ResponseWriter, r *http.Request) {
	var req saveSettingReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	if err := p.SaveSetting(r.Context(), req.Key, req.Value, operatorOf(r)); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]bool{"saved": true})
}

package appointment

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

type createReq struct {
	ServiceID string `json:"service_id"`
	StartTime string `json:"start_time"` // YYYY-MM-DD HH:MM
	Note      string `json:"note"`
}

// handleCustomerCreate: member identity from token only (§100).
func (p *Provider) handleCustomerCreate(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	var req createReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	a, err := p.Create(r.Context(), pr.ActorID, req.ServiceID, req.StartTime, req.Note)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleCustomerCancel(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	a, err := p.CancelByCustomer(r.Context(), pr.ActorID, r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleCustomerReschedule(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	var req struct {
		StartTime string `json:"start_time"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	a, err := p.Reschedule(r.Context(), pr.ActorID, r.PathValue("id"), req.StartTime, true)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleCustomerList(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	list, total, err := p.ListMine(r.Context(), pr.ActorID, r.URL.Query().Get("status"), shared.PageFromRequest(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.PageOK(w, list, total, shared.PageFromRequest(r))
}

func (p *Provider) handleCustomerGet(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	a, err := p.GetMine(r.Context(), pr.ActorID, r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	svcs, _ := p.ServicesOf(r.Context(), a.ID)
	shared.OK(w, map[string]any{"appointment": a, "services": svcs})
}

// --- admin ---

func (p *Provider) handleAdminConfirm(w http.ResponseWriter, r *http.Request) {
	a, err := p.Confirm(r.Context(), r.PathValue("id"), operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleAdminStart(w http.ResponseWriter, r *http.Request) {
	a, err := p.Start(r.Context(), r.PathValue("id"), operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleAdminComplete(w http.ResponseWriter, r *http.Request) {
	a, err := p.Complete(r.Context(), r.PathValue("id"), operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleAdminCancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	_ = shared.DecodeJSON(r, &req)
	a, err := p.CancelByAdmin(r.Context(), r.PathValue("id"), operatorOf(r), req.Reason)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleAdminNoShow(w http.ResponseWriter, r *http.Request) {
	a, err := p.NoShow(r.Context(), r.PathValue("id"), operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleAdminReschedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartTime string `json:"start_time"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	a, err := p.Reschedule(r.Context(), "", r.PathValue("id"), req.StartTime, false)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

func (p *Provider) handleAdminList(w http.ResponseWriter, r *http.Request) {
	list, total, err := p.ListAdmin(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("date"), shared.PageFromRequest(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.PageOK(w, list, total, shared.PageFromRequest(r))
}

func (p *Provider) handleAdminToday(w http.ResponseWriter, r *http.Request) {
	summary, details, err := p.Today(r.Context(), r.URL.Query().Get("date"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"summary": summary, "appointments": details})
}

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
	StartTime string `json:"start_time"` // YYYY-MM-DD HH:MM（具体时间，与 date+day_part 二选一）
	Date      string `json:"date"`       // YYYY-MM-DD（模糊预约，配 day_part）
	DayPart   string `json:"day_part"`   // AM | PM（模糊预约）
	Note      string `json:"note"`
}

func (c createReq) bookingReq() BookingReq {
	return BookingReq{StartTime: c.StartTime, Date: c.Date, DayPart: c.DayPart}
}

// handleCustomerCreate: member identity from token only (§100).
func (p *Provider) handleCustomerCreate(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	var req createReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	a, err := p.Create(r.Context(), pr.ActorID, req.ServiceID, req.bookingReq(), req.Note)
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
	var req createReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	a, err := p.Reschedule(r.Context(), pr.ActorID, r.PathValue("id"), req.bookingReq(), true, "")
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, a)
}

// handleCustomerBookingOptions — 可约时段计算（D20 §3.4）。
func (p *Provider) handleCustomerBookingOptions(w http.ResponseWriter, r *http.Request) {
	opts, err := p.BookingOptions(r.Context(), r.URL.Query().Get("date"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, opts)
}

// --- admin ---

func (p *Provider) handleAdminBookingOptions(w http.ResponseWriter, r *http.Request) {
	opts, err := p.BookingOptions(r.Context(), r.URL.Query().Get("date"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, opts)
}

func (p *Provider) handleAdminListClosures(w http.ResponseWriter, r *http.Request) {
	list, err := p.ListClosures(r.Context(), r.URL.Query().Get("from"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, list)
}

func (p *Provider) handleAdminCreateClosure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date    string `json:"date"`
		DayPart string `json:"day_part"` // AM | PM | FULL
		Remark  string `json:"remark"`
	}
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	n, err := p.CreateClosure(r.Context(), req.Date, req.DayPart, req.Remark, operatorOf(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"saved": true, "conflict_count": n})
}

func (p *Provider) handleAdminDeleteClosure(w http.ResponseWriter, r *http.Request) {
	if err := p.DeleteClosure(r.Context(), r.PathValue("id")); err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"deleted": true})
}

func (p *Provider) handleCustomerList(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	list, total, err := p.ListMine(r.Context(), pr.ActorID, r.URL.Query().Get("status"), shared.PageFromRequest(r))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	ids := make([]string, len(list))
	for i, a := range list {
		ids[i] = a.ID
	}
	svcMap, err := p.ServicesOfMany(r.Context(), ids)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	out := make([]*AppointmentWithServices, len(list))
	for i, a := range list {
		// §107：店主内部备注永不下发顾客端
		a.InternalNote = ""
		out[i] = &AppointmentWithServices{Appointment: *a, Services: svcMap[a.ID]}
	}
	shared.PageOK(w, out, total, shared.PageFromRequest(r))
}

func (p *Provider) handleCustomerGet(w http.ResponseWriter, r *http.Request) {
	pr, _ := middleware.PrincipalFrom(r.Context())
	a, err := p.GetMine(r.Context(), pr.ActorID, r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	// §107：店主内部备注永不下发顾客端
	a.InternalNote = ""
	svcs, _ := p.ServicesOf(r.Context(), a.ID)
	shared.OK(w, map[string]any{"appointment": a, "services": svcs})
}

// --- admin ---

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
	var req createReq
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	a, err := p.Reschedule(r.Context(), "", r.PathValue("id"), req.bookingReq(), false, operatorOf(r))
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

func (p *Provider) handleAdminGet(w http.ResponseWriter, r *http.Request) {
	d, err := p.Detail(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, d)
}

func (p *Provider) handleAdminToday(w http.ResponseWriter, r *http.Request) {
	summary, details, err := p.Today(r.Context(), r.URL.Query().Get("date"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, map[string]any{"summary": summary, "appointments": details})
}

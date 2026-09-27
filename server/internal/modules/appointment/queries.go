package appointment

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"anmo/server/internal/shared"
)

// queries.go — read paths for both sides plus the tx-joining helpers the
// transaction module composes (D6).

// Get returns an appointment (plain read).
func (p *Provider) Get(ctx context.Context, id string) (*Appointment, error) {
	a, err := scanAppointment(p.db.QueryRowContext(ctx,
		`SELECT `+aptColumns+` FROM appointment WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("APT_NOT_FOUND", "预约不存在")
	}
	if err != nil {
		return nil, shared.Server("APT_QUERY", err)
	}
	return a, nil
}

// GetTx is the tx-joining read for settlement flows.
func (p *Provider) GetTx(ctx context.Context, tx shared.Tx, id string) (*Appointment, error) {
	a, err := scanAppointment(tx.QueryRowContext(ctx,
		`SELECT `+aptColumns+` FROM appointment WHERE id = ? FOR UPDATE`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("APT_NOT_FOUND", "预约不存在")
	}
	if err != nil {
		return nil, shared.Server("APT_QUERY", err)
	}
	return a, nil
}

// MarkCompleted completes the appointment inside the settlement transaction
// (§53); no-op when already COMPLETED (W1).
func (p *Provider) MarkCompleted(ctx context.Context, tx shared.Tx, id, operatorID string) error {
	a, err := p.GetTx(ctx, tx, id)
	if err != nil {
		return err
	}
	if a.Status == StatusCompleted {
		return nil
	}
	// W8 (对抗审查): frozen state machine has no CONFIRMED→COMPLETED —
	// settle requires the service to have actually started.
	if a.Status != StatusInService {
		return shared.Conflict("APT_BAD_TRANSITION", "预约需处于服务中才能结算")
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE appointment SET status = ?, completed_at = NOW() WHERE id = ? AND status = ?`,
		StatusCompleted, id, a.Status)
	if err != nil {
		return shared.Server("APT_TRANSITION", err)
	}
	return statusLog(ctx, tx, id, a.Status, StatusCompleted, "ADMIN", operatorID, "结算完成")
}

// listFilter — shared WHERE builder.
type listFilter struct {
	MemberID string
	Status   string
	Date     string // YYYY-MM-DD (business tz)
	keyword  string
}

func (p *Provider) queryList(ctx context.Context, f listFilter, limit, offset int) ([]*Appointment, int64, error) {
	where := "1=1"
	args := []any{}
	if f.MemberID != "" {
		where += " AND member_id = ?"
		args = append(args, f.MemberID)
	}
	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.Date != "" {
		where += " AND scheduled_start >= ? AND scheduled_start < ? + INTERVAL 1 DAY"
		args = append(args, f.Date, f.Date)
	}
	var total int64
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM appointment WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, shared.Server("APT_COUNT", err)
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+aptColumns+` FROM appointment WHERE `+where+
			` ORDER BY scheduled_start LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, shared.Server("APT_LIST", err)
	}
	defer rows.Close()
	var out []*Appointment
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			return nil, 0, shared.Server("APT_SCAN", err)
		}
		out = append(out, a)
	}
	p.decorateDayParts(ctx, out)
	return out, total, nil
}

// decorateDayParts fills the display-only DayPart (AM/PM by the configured
// noon split, D20). Only meaningful for fuzzy bookings, but filled for all.
func (p *Provider) decorateDayParts(ctx context.Context, list []*Appointment) {
	if len(list) == 0 {
		return
	}
	rules, err := p.bizRules(ctx)
	if err != nil {
		return
	}
	for _, a := range list {
		w, err := dayBounds(rules, a.ScheduledStart)
		if err != nil {
			continue
		}
		if a.ScheduledStart.Before(w.Noon) {
			a.DayPart = "AM"
		} else {
			a.DayPart = "PM"
		}
	}
}

// ListMine returns a customer's appointments (newest start first).
func (p *Provider) ListMine(ctx context.Context, memberID, status string, page shared.PageParams) ([]*Appointment, int64, error) {
	return p.queryList(ctx, listFilter{MemberID: memberID, Status: status}, page.Limit(), page.Offset())
}

// GetMine returns a customer's own appointment (no internal fields leak is
// handled at the handler layer).
func (p *Provider) GetMine(ctx context.Context, memberID, id string) (*Appointment, error) {
	a, err := p.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.MemberID != memberID {
		return nil, shared.NewErr("APT_NOT_YOURS", "只能查看自己的预约", 403)
	}
	return a, nil
}

// ListAdmin returns appointments with filters for the backoffice.
func (p *Provider) ListAdmin(ctx context.Context, status, date string, page shared.PageParams) ([]*Appointment, int64, error) {
	return p.queryList(ctx, listFilter{Status: status, Date: date}, page.Limit(), page.Offset())
}

// TodaySummary is the workbench header (plan §73).
type TodaySummary struct {
	Date           string `json:"date"`
	Total          int64  `json:"total"`
	PendingConfirm int64  `json:"pending_confirm"`
	Confirmed      int64  `json:"confirmed"`
	InService      int64  `json:"in_service"`
	Completed      int64  `json:"completed"`
	Cancelled      int64  `json:"cancelled"`
	NoShow         int64  `json:"no_show"`
}

// TodayList returns the workbench card list with services and payments.
type TodayList = []AppointmentDetail

// AppointmentDetail bundles the appointment with snapshot and (later) payment info.
type AppointmentDetail struct {
	Appointment
	Service *AppointmentService `json:"service"`
}

// ServicesOf returns the snapshot rows of an appointment.
func (p *Provider) ServicesOf(ctx context.Context, aptID string) ([]*AppointmentService, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, appointment_id, service_id, service_name_snapshot, duration_minutes_snapshot, price_snapshot, quantity
		 FROM appointment_service WHERE appointment_id = ?`, aptID)
	if err != nil {
		return nil, shared.Server("APT_SERVICE_QUERY", err)
	}
	defer rows.Close()
	var out []*AppointmentService
	for rows.Next() {
		s := &AppointmentService{}
		if err := rows.Scan(&s.ID, &s.AppointmentID, &s.ServiceID, &s.NameSnapshot, &s.DurationSnapshot, &s.PriceSnapshot, &s.Quantity); err != nil {
			return nil, shared.Server("APT_SERVICE_SCAN", err)
		}
		out = append(out, s)
	}
	return out, nil
}

// ServicesOfTx is the tx-joining snapshot read for settlement flows (D6).
func (p *Provider) ServicesOfTx(ctx context.Context, tx shared.Tx, aptID string) ([]*AppointmentService, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, appointment_id, service_id, service_name_snapshot, duration_minutes_snapshot, price_snapshot, quantity
		 FROM appointment_service WHERE appointment_id = ?`, aptID)
	if err != nil {
		return nil, shared.Server("APT_SERVICE_QUERY", err)
	}
	defer rows.Close()
	var out []*AppointmentService
	for rows.Next() {
		s := &AppointmentService{}
		if err := rows.Scan(&s.ID, &s.AppointmentID, &s.ServiceID, &s.NameSnapshot, &s.DurationSnapshot, &s.PriceSnapshot, &s.Quantity); err != nil {
			return nil, shared.Server("APT_SERVICE_SCAN", err)
		}
		out = append(out, s)
	}
	return out, nil
}

// Today returns workbench data for a business date (defaults to today).
func (p *Provider) Today(ctx context.Context, date string) (*TodaySummary, []*AppointmentDetail, error) {
	if date == "" {
		date = shared.NowShanghai().Format("2006-01-02")
	}
	where := `scheduled_start >= ? AND scheduled_start < ? + INTERVAL 1 DAY`
	args := []any{date, date}
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+aptColumns+` FROM appointment WHERE `+where+` ORDER BY scheduled_start`, args...)
	if err != nil {
		return nil, nil, shared.Server("APT_TODAY", err)
	}
	defer rows.Close()
	var list []*Appointment
	summary := &TodaySummary{Date: date}
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			return nil, nil, shared.Server("APT_SCAN", err)
		}
		list = append(list, a)
		summary.Total++
		switch a.Status {
		case StatusPendingConfirm:
			summary.PendingConfirm++
		case StatusConfirmed:
			summary.Confirmed++
		case StatusInService:
			summary.InService++
		case StatusCompleted:
			summary.Completed++
		case StatusCancelled:
			summary.Cancelled++
		case StatusNoShow:
			summary.NoShow++
		}
	}
	p.decorateDayParts(ctx, list)
	var details []*AppointmentDetail
	for _, a := range list {
		svcs, err := p.ServicesOf(ctx, a.ID)
		if err != nil {
			return nil, nil, err
		}
		d := &AppointmentDetail{Appointment: *a}
		if len(svcs) > 0 {
			d.Service = svcs[0]
		}
		details = append(details, d)
	}
	return summary, details, nil
}

var _ = time.Now

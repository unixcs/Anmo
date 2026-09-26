package appointment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
)

// Status values (plan §24) — frozen state machine.
const (
	StatusPendingConfirm = "PENDING_CONFIRM"
	StatusConfirmed      = "CONFIRMED"
	StatusInService      = "IN_SERVICE"
	StatusCompleted      = "COMPLETED"
	StatusCancelled      = "CANCELLED"
	StatusNoShow         = "NO_SHOW"
)

// calendarLock serializes conflict checks for the single-therapist calendar (D5).
const calendarLock = "anmo:appointment:calendar"

// Appointment — appointment row.
type Appointment struct {
	ID             string     `json:"id"`
	No             string     `json:"appointment_no"`
	MemberID       string     `json:"member_id"`
	ScheduledStart time.Time  `json:"scheduled_start"`
	ScheduledEnd   time.Time  `json:"scheduled_end"`
	Status         string     `json:"status"`
	CustomerNote   string     `json:"customer_note"`
	InternalNote   string     `json:"internal_note"`
	ConfirmedAt    *time.Time `json:"confirmed_at"`
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	CancelledAt    *time.Time `json:"cancelled_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

const aptColumns = `id, appointment_no, member_id, scheduled_start, scheduled_end, status,
 customer_note, internal_note, confirmed_at, started_at, completed_at, cancelled_at, created_at`

func scanAppointment(row interface{ Scan(...any) error }) (*Appointment, error) {
	a := &Appointment{}
	var confirmed, started, completed, cancelled sql.NullTime
	err := row.Scan(&a.ID, &a.No, &a.MemberID, &a.ScheduledStart, &a.ScheduledEnd,
		&a.Status, &a.CustomerNote, &a.InternalNote,
		&confirmed, &started, &completed, &cancelled, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	setTime := func(dst **time.Time, src sql.NullTime) {
		if src.Valid {
			t := src.Time
			*dst = &t
		}
	}
	setTime(&a.ConfirmedAt, confirmed)
	setTime(&a.StartedAt, started)
	setTime(&a.CompletedAt, completed)
	setTime(&a.CancelledAt, cancelled)
	return a, nil
}

// AppointmentService — snapshot row (§22-23).
type AppointmentService struct {
	ID               string `json:"id"`
	AppointmentID    string `json:"appointment_id"`
	ServiceID        string `json:"service_id"`
	NameSnapshot     string `json:"service_name_snapshot"`
	DurationSnapshot int    `json:"duration_minutes_snapshot"`
	PriceSnapshot    int64  `json:"price_snapshot"`
	Quantity         int    `json:"quantity"`
}

// statusLog appends a transition record.
func statusLog(ctx context.Context, tx shared.Tx, aptID, from, to, operatorType, operatorID, remark string) error {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO appointment_status_log (id, appointment_id, from_status, to_status, operator_type, operator_id, remark)
		 VALUES (?,?,?,?,?,?,?)`,
		shared.NewID(), aptID, from, to, operatorType, operatorID, remark); err != nil {
		return shared.Server("APT_LOG_WRITE", err)
	}
	return nil
}

// parseSlot parses "2006-01-02 15:04" in business timezone.
func parseSlot(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, shared.NowShanghai().Location())
	if err != nil {
		return time.Time{}, shared.BadRequest("APT_BAD_TIME", "时间格式应为 YYYY-MM-DD HH:MM")
	}
	return t, nil
}

// validateWindow enforces business hours and booking-lead rules (§19/§30, D15).
func (p *Provider) validateWindow(start, end time.Time) error {
	b := p.cfg.Business
	loc := shared.NowShanghai().Location()
	day := start.In(loc)
	openT, _ := time.ParseInLocation("15:04", b.OpenTime, loc)
	closeT, _ := time.ParseInLocation("15:04", b.CloseTime, loc)
	open := time.Date(day.Year(), day.Month(), day.Day(), openT.Hour(), openT.Minute(), 0, 0, loc)
	close := time.Date(day.Year(), day.Month(), day.Day(), closeT.Hour(), closeT.Minute(), 0, 0, loc)
	if start.Before(open) || end.After(close) {
		return shared.BadRequest("APT_OUT_OF_HOURS", "预约时间超出营业时间")
	}
	now := shared.NowShanghai()
	if start.Before(now.Add(time.Duration(b.BookMinAheadHours) * time.Hour)) {
		return shared.BadRequest("APT_TOO_SOON", "预约需至少提前 2 小时")
	}
	if start.After(now.AddDate(0, 0, b.BookAheadDays)) {
		return shared.BadRequest("APT_TOO_FAR", fmt.Sprintf("最早可提前 %d 天预约", b.BookAheadDays))
	}
	// slot alignment (§19: fixed 30-minute grid)
	slot := time.Duration(b.SlotMinutes) * time.Minute
	if start.Sub(open)%slot != 0 {
		return shared.BadRequest("APT_BAD_SLOT", "预约时间需对齐时间槽")
	}
	return nil
}

// nextAppointmentNo generates APT+yyyymmdd+seq via the atomic counter table
// (no MAX+1 races; the row lock serializes writers until commit).
func (p *Provider) nextAppointmentNo(ctx context.Context, tx shared.Tx) (string, error) {
	day := shared.NowShanghai().Format("20060102")
	seq, err := shared.NextSeq(ctx, tx, shared.SeqDateName("apt", day))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("APT%s%04d", day, seq), nil
}

// activeExists reports overlapping active appointments (§71), excluding one id.
func activeExists(ctx context.Context, tx shared.Tx, start, end time.Time, excludeID string) (bool, error) {
	q := `SELECT COUNT(*) FROM appointment
	      WHERE status IN ('PENDING_CONFIRM','CONFIRMED','IN_SERVICE')
	        AND scheduled_start < ? AND scheduled_end > ?`
	args := []any{end, start}
	if excludeID != "" {
		q += ` AND id <> ?`
		args = append(args, excludeID)
	}
	var n int
	if err := tx.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return false, shared.Server("APT_CONFLICT_QUERY", err)
	}
	return n > 0, nil
}

// Create books a new appointment for a member: one transaction with the
// calendar lock, conflict check, snapshot insert and status log.
func (p *Provider) Create(ctx context.Context, memberID, serviceID, startStr, note string) (*Appointment, error) {
	start, err := parseSlot(startStr)
	if err != nil {
		return nil, err
	}
	item, err := p.services.GetItem(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if item.Status != "ACTIVE" {
		return nil, shared.Conflict("SERVICE_INACTIVE", "服务项目已停用")
	}
	end := start.Add(time.Duration(item.DurationMin) * time.Minute)
	if err := p.validateWindow(start, end); err != nil {
		return nil, err
	}

	var out *Appointment
	unlock, err := p.acquireCalendar(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		conflict, err := activeExists(ctx, tx, start, end, "")
		if err != nil {
			return err
		}
		if conflict {
			return shared.Conflict("APPOINTMENT_CONFLICT", "该时间刚刚被预约，请重新选择")
		}
		no, err := p.nextAppointmentNo(ctx, tx)
		if err != nil {
			return err
		}
		id := shared.NewID()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO appointment (id, appointment_no, member_id, scheduled_start, scheduled_end, status, customer_note)
			 VALUES (?,?,?,?,?,?,?)`,
			id, no, memberID, start, end, StatusPendingConfirm, note); err != nil {
			return shared.Server("APT_INSERT", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO appointment_service (id, appointment_id, service_id, service_name_snapshot, duration_minutes_snapshot, price_snapshot, quantity)
			 VALUES (?,?,?,?,?,?,1)`,
			shared.NewID(), id, item.ID, item.Name, item.DurationMin, item.PriceCents); err != nil {
			return shared.Server("APT_SERVICE_INSERT", err)
		}
		if err := statusLog(ctx, tx, id, "", StatusPendingConfirm, "CUSTOMER", memberID, "创建预约"); err != nil {
			return err
		}
		out, err = scanAppointment(tx.QueryRowContext(ctx, `SELECT `+aptColumns+` FROM appointment WHERE id = ?`, id))
		if err != nil {
			return shared.Server("APT_QUERY", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// acquireCalendar takes the calendar serialization lock on a dedicated
// connection. The caller defers unlock() so the lock is held through COMMIT —
// the next writer's conflict check then sees committed state (D5).
func (p *Provider) acquireCalendar(ctx context.Context) (shared.LockHandle, error) {
	locker, ok := p.db.(shared.NamedLocker)
	if !ok {
		return nil, shared.Server("LOCK_UNSUPPORTED", nil)
	}
	h, err := locker.NamedLock(ctx, calendarLock, 5)
	if err != nil {
		return nil, shared.Conflict("APT_LOCK_BUSY", "预约繁忙，请重试")
	}
	return shared.LockHandle(h), nil
}

// transition applies a guarded state change (D8) and logs it.
func (p *Provider) transition(ctx context.Context, tx shared.Tx, id, from, to, operatorType, operatorID, remark string) (*Appointment, error) {
	q := `UPDATE appointment SET status = ?`
	switch to {
	case StatusConfirmed:
		q += ", confirmed_at = NOW()"
	case StatusInService:
		q += ", started_at = NOW()"
	case StatusCompleted:
		q += ", completed_at = NOW()"
	case StatusCancelled:
		q += ", cancelled_at = NOW()"
	}
	q += ` WHERE id = ? AND status = ?`
	res, err := tx.ExecContext(ctx, q, to, id, from)
	if err != nil {
		return nil, shared.Server("APT_TRANSITION", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, shared.Conflict("APT_BAD_TRANSITION",
			fmt.Sprintf("预约状态不允许该操作（需要 %s）", from))
	}
	if err := statusLog(ctx, tx, id, from, to, operatorType, operatorID, remark); err != nil {
		return nil, err
	}
	a, err := scanAppointment(tx.QueryRowContext(ctx, `SELECT `+aptColumns+` FROM appointment WHERE id = ?`, id))
	if err != nil {
		return nil, shared.Server("APT_QUERY", err)
	}
	return a, nil
}

// Confirm approves a pending appointment (owner side).
func (p *Provider) Confirm(ctx context.Context, id, operatorID string) (*Appointment, error) {
	var out *Appointment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		out, e = p.transition(ctx, tx, id, StatusPendingConfirm, StatusConfirmed, "ADMIN", operatorID, "确认预约")
		return e
	})
	return out, err
}

// Start begins service.
func (p *Provider) Start(ctx context.Context, id, operatorID string) (*Appointment, error) {
	var out *Appointment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		out, e = p.transition(ctx, tx, id, StatusConfirmed, StatusInService, "ADMIN", operatorID, "开始服务")
		return e
	})
	return out, err
}

// Complete finishes service; idempotent when already completed (W1).
func (p *Provider) Complete(ctx context.Context, id, operatorID string) (*Appointment, error) {
	var out *Appointment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		a, err := scanAppointment(tx.QueryRowContext(ctx, `SELECT `+aptColumns+` FROM appointment WHERE id = ? FOR UPDATE`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("APT_NOT_FOUND", "预约不存在")
		}
		if err != nil {
			return shared.Server("APT_QUERY", err)
		}
		if a.Status == StatusCompleted {
			out = a
			return nil
		}
		out, err = p.transition(ctx, tx, id, StatusInService, StatusCompleted, "ADMIN", operatorID, "完成服务")
		return err
	})
	return out, err
}

// CancelByCustomer cancels with the 2-hour rule (§31).
func (p *Provider) CancelByCustomer(ctx context.Context, memberID, id string) (*Appointment, error) {
	var out *Appointment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		a, err := scanAppointment(tx.QueryRowContext(ctx, `SELECT `+aptColumns+` FROM appointment WHERE id = ? FOR UPDATE`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("APT_NOT_FOUND", "预约不存在")
		}
		if err != nil {
			return shared.Server("APT_QUERY", err)
		}
		if a.MemberID != memberID {
			return shared.NewErr("APT_NOT_YOURS", "只能操作自己的预约", 403)
		}
		if a.Status != StatusPendingConfirm && a.Status != StatusConfirmed {
			return shared.Conflict("APT_BAD_TRANSITION", "当前状态不可取消")
		}
		if a.ScheduledStart.Before(shared.NowShanghai().Add(time.Duration(p.cfg.Business.CancelMinAheadHrs) * time.Hour)) {
			return shared.Conflict("APT_CANCEL_TOO_LATE", "距开始不足 2 小时，请联系店家取消")
		}
		out, err = p.transition(ctx, tx, id, a.Status, StatusCancelled, "CUSTOMER", memberID, "顾客取消")
		return err
	})
	return out, err
}

// CancelByAdmin cancels without the 2-hour restriction.
func (p *Provider) CancelByAdmin(ctx context.Context, id, operatorID, reason string) (*Appointment, error) {
	var out *Appointment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		a, err := scanAppointment(tx.QueryRowContext(ctx, `SELECT `+aptColumns+` FROM appointment WHERE id = ? FOR UPDATE`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("APT_NOT_FOUND", "预约不存在")
		}
		if err != nil {
			return shared.Server("APT_QUERY", err)
		}
		if a.Status != StatusPendingConfirm && a.Status != StatusConfirmed {
			return shared.Conflict("APT_BAD_TRANSITION", "当前状态不可取消")
		}
		out, err = p.transition(ctx, tx, id, a.Status, StatusCancelled, "ADMIN", operatorID, reason)
		return err
	})
	return out, err
}

// NoShow marks a confirmed appointment as no-show (§25: only from CONFIRMED).
func (p *Provider) NoShow(ctx context.Context, id, operatorID string) (*Appointment, error) {
	var out *Appointment
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		out, e = p.transition(ctx, tx, id, StatusConfirmed, StatusNoShow, "ADMIN", operatorID, "爽约")
		return e
	})
	return out, err
}

// Reschedule moves an appointment to a new time on the same appointment
// (§32/D8): cancel semantics not used; conflicts exclude the appointment itself.
func (p *Provider) Reschedule(ctx context.Context, memberID, id, newStartStr string, byCustomer bool) (*Appointment, error) {
	start, err := parseSlot(newStartStr)
	if err != nil {
		return nil, err
	}
	var out *Appointment
	unlock, err := p.acquireCalendar(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		a, err := scanAppointment(tx.QueryRowContext(ctx, `SELECT `+aptColumns+` FROM appointment WHERE id = ? FOR UPDATE`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("APT_NOT_FOUND", "预约不存在")
		}
		if err != nil {
			return shared.Server("APT_QUERY", err)
		}
		if byCustomer && a.MemberID != memberID {
			return shared.NewErr("APT_NOT_YOURS", "只能操作自己的预约", 403)
		}
		if a.Status != StatusPendingConfirm && a.Status != StatusConfirmed {
			return shared.Conflict("APT_BAD_TRANSITION", "当前状态不可改期")
		}
		if byCustomer && a.ScheduledStart.Before(shared.NowShanghai().Add(time.Duration(p.cfg.Business.CancelMinAheadHrs)*time.Hour)) {
			return shared.Conflict("APT_RESCHEDULE_TOO_LATE", "距开始不足 2 小时，请联系店家改期")
		}
		// duration comes from the snapshot
		var dur int
		if err := tx.QueryRowContext(ctx,
			`SELECT duration_minutes_snapshot FROM appointment_service WHERE appointment_id = ? LIMIT 1`, id).Scan(&dur); err != nil {
			return shared.Server("APT_SNAPSHOT_QUERY", err)
		}
		end := start.Add(time.Duration(dur) * time.Minute)
		if err := p.validateWindow(start, end); err != nil {
			return err
		}
		conflict, err := activeExists(ctx, tx, start, end, id)
		if err != nil {
			return err
		}
		if conflict {
			return shared.Conflict("APPOINTMENT_CONFLICT", "新时间已被预约，请重新选择")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE appointment SET scheduled_start = ?, scheduled_end = ? WHERE id = ?`,
			start, end, id); err != nil {
			return shared.Server("APT_RESCHEDULE", err)
		}
		from := a.Status
		if err := statusLog(ctx, tx, id, from, from, "CUSTOMER", memberID, "改期"); err != nil {
			return err
		}
		out, err = scanAppointment(tx.QueryRowContext(ctx, `SELECT `+aptColumns+` FROM appointment WHERE id = ?`, id))
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

var _ = service.Item{}

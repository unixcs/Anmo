package appointment

import (
	"context"
	"fmt"
	"time"

	"anmo/server/internal/shared"
)

// booking.go — V1.x 预约域（D20/D22）：上下午模糊预约、逐槽并发 + 半日名额池、
// 闭店日历、可约选项计算。规则解析在 content 模块（settings > cfg > 默认），
// 本模块只依赖 RulesSource 接口。

// Slot types (D20).
const (
	SlotTypeSpecific = "SPECIFIC"
	SlotTypeHalfDay  = "HALF_DAY"
)

// BusinessRules — resolved booking rules (mirrors content.BusinessRules;
// app.go adapts one into the other).
type BusinessRules struct {
	OpenTime     string
	CloseTime    string
	NoonSplit    string
	SlotMinutes  int
	SlotCapacity int
}

// RulesSource supplies effective booking rules. Nil → cfg/defaults fallback
// (unit tests). app.go adapts the content module's resolver into this.
type RulesSource interface {
	BookingRules(ctx context.Context) (*BusinessRules, error)
}

// RulesSourceFunc adapts a function into a RulesSource.
type RulesSourceFunc func(ctx context.Context) (*BusinessRules, error)

// BookingRules implements RulesSource.
func (f RulesSourceFunc) BookingRules(ctx context.Context) (*BusinessRules, error) { return f(ctx) }

// BookingReq — a create/reschedule target: either an exact slot
// ("2026-09-28 10:00") or a fuzzy half-day (date + AM/PM).
type BookingReq struct {
	StartTime string // "YYYY-MM-DD HH:MM", exclusive with Date+DayPart
	Date      string // "YYYY-MM-DD", with DayPart
	DayPart   string // "AM" | "PM", with Date
}

// window — a resolved booking window.
type window struct {
	Start    time.Time
	End      time.Time
	SlotType string // SPECIFIC | HALF_DAY
	DayPart  string // AM | PM (start side; cross-noon SPECIFIC covers both)
}

// dayWindow — the concrete AM/PM boundaries for one date.
type dayWindow struct {
	Open, Close, Noon time.Time
}

func (w dayWindow) halfStart(part string) time.Time {
	if part == "AM" {
		return w.Open
	}
	return w.Noon
}

func (w dayWindow) halfEnd(part string) time.Time {
	if part == "AM" {
		return w.Noon
	}
	return w.Close
}

// bizRules resolves rules; falls back to cfg/defaults when no source is wired.
func (p *Provider) bizRules(ctx context.Context) (*BusinessRules, error) {
	if p.rules == nil {
		b := p.cfg.Business
		r := &BusinessRules{
			OpenTime:     firstNonEmpty(b.OpenTime, "09:00"),
			CloseTime:    firstNonEmpty(b.CloseTime, "20:00"),
			NoonSplit:    firstNonEmpty("", "12:00"),
			SlotMinutes:  firstPositive(b.SlotMinutes, 30),
			SlotCapacity: firstPositive(b.SlotCapacity, 1),
		}
		if r.SlotMinutes != 30 && r.SlotMinutes != 60 && r.SlotMinutes != 120 {
			r.SlotMinutes = 30
		}
		return r, nil
	}
	return p.rules.BookingRules(ctx)
}

func firstNonEmpty(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

func firstPositive(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

// parseHHMM parses "15:04" in business timezone.
func parseHHMM(s string) (int, int, error) {
	t, err := time.ParseInLocation("15:04", s, shared.NowShanghai().Location())
	if err != nil {
		return 0, 0, shared.BadRequest("APT_BAD_TIME", "时间格式应为 HH:MM")
	}
	return t.Hour(), t.Minute(), nil
}

// dayBounds builds the concrete AM/PM boundaries for the window's date.
func dayBounds(rules *BusinessRules, day time.Time) (dayWindow, error) {
	loc := shared.NowShanghai().Location()
	oh, om, err := parseHHMM(rules.OpenTime)
	if err != nil {
		return dayWindow{}, err
	}
	ch, cm, err := parseHHMM(rules.CloseTime)
	if err != nil {
		return dayWindow{}, err
	}
	nh, nm, err := parseHHMM(rules.NoonSplit)
	if err != nil {
		return dayWindow{}, err
	}
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	w := dayWindow{
		Open:  d.Add(time.Duration(oh)*time.Hour + time.Duration(om)*time.Minute),
		Close: d.Add(time.Duration(ch)*time.Hour + time.Duration(cm)*time.Minute),
		Noon:  d.Add(time.Duration(nh)*time.Hour + time.Duration(nm)*time.Minute),
	}
	if !w.Open.Before(w.Noon) || !w.Noon.Before(w.Close) {
		return dayWindow{}, shared.BadRequest("APT_BAD_WINDOW", "营业时间配置无效（需 开门 < 上下午分界 < 关门）")
	}
	return w, nil
}

// planWindow resolves a BookingReq into a concrete window, enforcing hours,
// slot alignment, lead time and horizon (§19/§30/D15/D20).
func (p *Provider) planWindow(ctx context.Context, rules *BusinessRules, req BookingReq, durationMin int) (window, error) {
	loc := shared.NowShanghai().Location()
	now := shared.NowShanghai()

	if req.DayPart != "" || req.Date != "" {
		if req.StartTime != "" {
			return window{}, shared.BadRequest("APT_BAD_TARGET", "具体时间与上午/下午二选一")
		}
		if req.DayPart != "AM" && req.DayPart != "PM" {
			return window{}, shared.BadRequest("APT_BAD_DAYPART", "day_part 需为 AM 或 PM")
		}
		day, err := time.ParseInLocation("2006-01-02", req.Date, loc)
		if err != nil {
			return window{}, shared.BadRequest("APT_BAD_TIME", "时间格式应为 YYYY-MM-DD")
		}
		w, err := dayBounds(rules, day)
		if err != nil {
			return window{}, err
		}
		start, end := w.halfStart(req.DayPart), w.halfEnd(req.DayPart)
		// 模糊预约不设 2h 提前量（由店主安排），只要求该半天尚未结束 + 预约视野内。
		if !end.After(now) {
			return window{}, shared.BadRequest("APT_TOO_SOON", "该半天已结束")
		}
		if start.After(now.AddDate(0, 0, p.cfg.Business.BookAheadDays)) {
			return window{}, shared.BadRequest("APT_TOO_FAR", fmt.Sprintf("最早可提前 %d 天预约", p.cfg.Business.BookAheadDays))
		}
		return window{Start: start, End: end, SlotType: SlotTypeHalfDay, DayPart: req.DayPart}, nil
	}

	start, err := parseSlot(req.StartTime)
	if err != nil {
		return window{}, err
	}
	end := start.Add(time.Duration(durationMin) * time.Minute)
	w, err := dayBounds(rules, start)
	if err != nil {
		return window{}, err
	}
	if start.Before(w.Open) || end.After(w.Close) {
		return window{}, shared.BadRequest("APT_OUT_OF_HOURS", "预约时间超出营业时间")
	}
	if start.Before(now.Add(time.Duration(p.cfg.Business.BookMinAheadHours) * time.Hour)) {
		return window{}, shared.BadRequest("APT_TOO_SOON", "预约需至少提前 2 小时")
	}
	if start.After(now.AddDate(0, 0, p.cfg.Business.BookAheadDays)) {
		return window{}, shared.BadRequest("APT_TOO_FAR", fmt.Sprintf("最早可提前 %d 天预约", p.cfg.Business.BookAheadDays))
	}
	slot := time.Duration(rules.SlotMinutes) * time.Minute
	if start.Sub(w.Open)%slot != 0 {
		return window{}, shared.BadRequest("APT_BAD_SLOT", "预约时间需对齐时间槽")
	}
	dp := "AM"
	if !start.Before(w.Noon) {
		dp = "PM"
	}
	return window{Start: start, End: end, SlotType: SlotTypeSpecific, DayPart: dp}, nil
}

// checkClosures rejects windows hitting a closure day-part (D22). A window
// covering both half-days is rejected when either is closed.
func checkClosures(ctx context.Context, tx shared.Tx, w dayWindow, win window) error {
	parts := []string{}
	if win.Start.Before(w.Noon) && win.End.After(w.Open) {
		parts = append(parts, "AM")
	}
	if win.End.After(w.Noon) && win.Start.Before(w.Close) {
		parts = append(parts, "PM")
	}
	for _, part := range parts {
		var n int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM appointment_closure WHERE closure_date = ? AND day_part = ?`,
			win.Start.Format("2006-01-02"), part).Scan(&n); err != nil {
			return shared.Server("APT_CLOSURE_QUERY", err)
		}
		if n > 0 {
			label := "上午"
			if part == "PM" {
				label = "下午"
			}
			return shared.Conflict("APT_CLOSED", fmt.Sprintf("%s店铺休息，无法预约", label))
		}
	}
	return nil
}

// activeCount counts active bookings overlapping [ws,we), excluding excludeID.
// withSpecific=true restricts to SPECIFIC rows (per-slot concurrency);
// withSpecific=false counts every active row (half-day pool).
func activeCount(ctx context.Context, tx shared.Tx, ws, we time.Time, excludeID string, specificOnly bool) (int, error) {
	q := `SELECT COUNT(*) FROM appointment
	      WHERE status IN ('PENDING_CONFIRM','CONFIRMED','IN_SERVICE')
	        AND scheduled_start < ? AND scheduled_end > ?`
	args := []any{we, ws}
	if specificOnly {
		q += ` AND slot_type = 'SPECIFIC'`
	}
	if excludeID != "" {
		q += ` AND id <> ?`
		args = append(args, excludeID)
	}
	var n int
	if err := tx.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return 0, shared.Server("APT_CONFLICT_QUERY", err)
	}
	return n, nil
}

// checkCapacity enforces per-slot concurrency (SPECIFIC only) and the
// half-day pool (all active rows) over the window (D20).
func checkCapacity(ctx context.Context, tx shared.Tx, rules *BusinessRules, w dayWindow, win window, excludeID string) error {
	slot := time.Duration(rules.SlotMinutes) * time.Minute
	// per-slot concurrency for pinned-time bookings
	if win.SlotType == SlotTypeSpecific {
		for s := w.Open; s.Before(w.Close); s = s.Add(slot) {
			if !s.Before(win.End) {
				break
			}
			e := s.Add(slot)
			if win.Start.Before(e) && win.End.After(s) {
				n, err := activeCount(ctx, tx, s, e, excludeID, true)
				if err != nil {
					return err
				}
				if n >= rules.SlotCapacity {
					return shared.Conflict("APT_SLOT_FULL", "这个时间刚被约满，换个时间试试")
				}
			}
		}
	}
	// half-day pool: every half-day the window touches
	parts := []struct{ ws, we time.Time }{}
	if win.Start.Before(w.Noon) && win.End.After(w.Open) {
		parts = append(parts, struct{ ws, we time.Time }{w.Open, w.Noon})
	}
	if win.End.After(w.Noon) && win.Start.Before(w.Close) {
		parts = append(parts, struct{ ws, we time.Time }{w.Noon, w.Close})
	}
	for _, hp := range parts {
		slots := int(hp.we.Sub(hp.ws) / slot)
		pool := slots * rules.SlotCapacity
		n, err := activeCount(ctx, tx, hp.ws, hp.we, excludeID, false)
		if err != nil {
			return err
		}
		if n >= pool {
			return shared.Conflict("APT_HALFDAY_FULL", "该时段名额已约满，请联系店家")
		}
	}
	return nil
}

// --- closures (D22) ---

// Closure — one closed half-day.
type Closure struct {
	ID        string `json:"id"`
	Date      string `json:"closure_date"`
	DayPart   string `json:"day_part"` // AM | PM
	Remark    string `json:"remark"`
	CreatedAt string `json:"created_at"`
}

const closureColumns = `id, DATE_FORMAT(closure_date, '%Y-%m-%d'), day_part, remark,
 DATE_FORMAT(created_at, '%Y-%m-%d %H:%i')`

// CreateClosure closes half-day(s) inside the calendar lock and reports how
// many active bookings the merchant is taking on (D22 informed consent).
func (p *Provider) CreateClosure(ctx context.Context, date, dayPart, remark, operatorID string) (int, error) {
	if dayPart != "AM" && dayPart != "PM" && dayPart != "FULL" {
		return 0, shared.BadRequest("CLOSURE_BAD_PART", "闭店范围需为 AM / PM / FULL")
	}
	if _, err := time.ParseInLocation("2006-01-02", date, shared.NowShanghai().Location()); err != nil {
		return 0, shared.BadRequest("CLOSURE_BAD_DATE", "日期格式应为 YYYY-MM-DD")
	}
	parts := []string{dayPart}
	if dayPart == "FULL" {
		parts = []string{"AM", "PM"}
	}
	rules, err := p.bizRules(ctx)
	if err != nil {
		return 0, err
	}
	day, _ := time.ParseInLocation("2006-01-02", date, shared.NowShanghai().Location())
	w, err := dayBounds(rules, day)
	if err != nil {
		return 0, err
	}
	conflict := 0
	unlock, err := p.acquireCalendar(ctx)
	if err != nil {
		return 0, err
	}
	defer unlock()
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		for _, part := range parts {
			var n int
			if err := tx.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM appointment_closure WHERE closure_date = ? AND day_part = ?`,
				date, part).Scan(&n); err != nil {
				return shared.Server("CLOSURE_QUERY", err)
			}
			if n > 0 {
				return shared.Conflict("CLOSURE_EXISTS", "该时段已设置闭店")
			}
		}
		for _, part := range parts {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO appointment_closure (id, closure_date, day_part, remark) VALUES (?,?,?,?)`,
				shared.NewID(), date, part, remark); err != nil {
				return shared.Server("CLOSURE_INSERT", err)
			}
			// informed-consent count: active bookings in that half-day
			n, err := activeCount(ctx, tx, w.halfStart(part), w.halfEnd(part), "", false)
			if err != nil {
				return err
			}
			conflict += n
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return conflict, nil
}

// ListClosures returns closures on/after from (defaults today).
func (p *Provider) ListClosures(ctx context.Context, from string) ([]*Closure, error) {
	if from == "" {
		from = shared.NowShanghai().Format("2006-01-02")
	}
	if _, err := time.ParseInLocation("2006-01-02", from, shared.NowShanghai().Location()); err != nil {
		return nil, shared.BadRequest("CLOSURE_BAD_DATE", "日期格式应为 YYYY-MM-DD")
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+closureColumns+` FROM appointment_closure WHERE closure_date >= ? ORDER BY closure_date, day_part`, from)
	if err != nil {
		return nil, shared.Server("CLOSURE_LIST", err)
	}
	defer rows.Close()
	var out []*Closure
	for rows.Next() {
		c := &Closure{}
		if err := rows.Scan(&c.ID, &c.Date, &c.DayPart, &c.Remark, &c.CreatedAt); err != nil {
			return nil, shared.Server("CLOSURE_SCAN", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// DeleteClosure removes one closure row (undo a mistaken closure).
func (p *Provider) DeleteClosure(ctx context.Context, id string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM appointment_closure WHERE id = ?`, id)
	if err != nil {
		return shared.Server("CLOSURE_DELETE", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.NotFound("CLOSURE_NOT_FOUND", "闭店记录不存在")
	}
	return nil
}

// --- booking options ---

// SlotOption — one bookable slot with remaining concurrency.
type SlotOption struct {
	Time      string `json:"time"`
	Remaining int    `json:"remaining"`
}

// HalfDayOptions — one half-day's availability.
type HalfDayOptions struct {
	Closed    bool         `json:"closed"`
	Total     int          `json:"total"`
	Remaining int          `json:"remaining"`
	Slots     []*SlotOption `json:"slots"`
}

// BookingOptions — full-day availability for the customer/merchant pickers.
type BookingOptions struct {
	Date string        `json:"date"`
	Open bool          `json:"open"`
	AM   *HalfDayOptions `json:"am"`
	PM   *HalfDayOptions `json:"pm"`
}

// BookingOptions computes availability for one date from rules, closures and
// active bookings (D20 §3.4).
func (p *Provider) BookingOptions(ctx context.Context, date string) (*BookingOptions, error) {
	day, err := time.ParseInLocation("2006-01-02", date, shared.NowShanghai().Location())
	if err != nil {
		return nil, shared.BadRequest("APT_BAD_TIME", "时间格式应为 YYYY-MM-DD")
	}
	rules, err := p.bizRules(ctx)
	if err != nil {
		return nil, err
	}
	w, err := dayBounds(rules, day)
	if err != nil {
		return nil, err
	}
	slot := time.Duration(rules.SlotMinutes) * time.Minute

	var closedAM, closedPM int
	if err := p.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(day_part='AM'),0), COALESCE(SUM(day_part='PM'),0) FROM appointment_closure WHERE closure_date = ?`,
		date).Scan(&closedAM, &closedPM); err != nil {
		return nil, shared.Server("APT_CLOSURE_QUERY", err)
	}

	// load active bookings of the day once; compute overlaps in memory.
	rows, err := p.db.QueryContext(ctx,
		`SELECT scheduled_start, scheduled_end, slot_type FROM appointment
		 WHERE status IN ('PENDING_CONFIRM','CONFIRMED','IN_SERVICE')
		   AND scheduled_start < ? AND scheduled_end > ?`,
		w.Close, w.Open)
	if err != nil {
		return nil, shared.Server("APT_LIST", err)
	}
	defer rows.Close()
	type act struct {
		start, end time.Time
		specific   bool
	}
	var acts []act
	for rows.Next() {
		var st, en time.Time
		var stype string
		if err := rows.Scan(&st, &en, &stype); err != nil {
			return nil, shared.Server("APT_SCAN", err)
		}
		acts = append(acts, act{st, en, stype == SlotTypeSpecific})
	}

	half := func(part string) *HalfDayOptions {
		ws, we := w.halfStart(part), w.halfEnd(part)
		h := &HalfDayOptions{Closed: (part == "AM" && closedAM > 0) || (part == "PM" && closedPM > 0)}
		h.Total = int(we.Sub(ws) / slot) * rules.SlotCapacity
		pool := 0
		for _, a := range acts {
			if a.start.Before(we) && a.end.After(ws) {
				pool++
			}
		}
		h.Remaining = h.Total - pool
		if h.Remaining < 0 {
			h.Remaining = 0
		}
		for s := ws; s.Before(we); s = s.Add(slot) {
			e := s.Add(slot)
			rem := rules.SlotCapacity
			for _, a := range acts {
				if a.specific && a.start.Before(e) && a.end.After(s) {
					rem--
				}
			}
			if rem < 0 {
				rem = 0
			}
			h.Slots = append(h.Slots, &SlotOption{Time: s.Format("15:04"), Remaining: rem})
		}
		if h.Closed {
			h.Slots = nil
		}
		return h
	}

	am, pm := half("AM"), half("PM")
	open := (!am.Closed && am.Remaining > 0) || (!pm.Closed && pm.Remaining > 0)
	return &BookingOptions{Date: date, Open: open, AM: am, PM: pm}, nil
}

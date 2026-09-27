package appointment

import (
	"context"
	"testing"

	"anmo/server/internal/shared"
)

// booking_test.go — V1.x 模糊预约 / 名额池 / 闭店（D20/D22）。

// halfDay books a fuzzy half-day and returns the row.
func halfDay(t *testing.T, e *aptEnv, mbr, part string, day int) *Appointment {
	t.Helper()
	d := shared.NowShanghai().AddDate(0, 0, day)
	a, err := e.p.Create(context.Background(), mbr, e.svcID, BookingReq{
		Date: d.Format("2006-01-02"), DayPart: part,
	}, "")
	if err != nil {
		t.Fatalf("half-day create: %v", err)
	}
	return a
}

func TestHalfDayWindowAndPool(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()

	a := halfDay(t, e, e.mbr1, "AM", 2)
	if a.SlotType != SlotTypeHalfDay || a.DayPart != "AM" {
		t.Fatalf("slot_type/day_part = %s/%s", a.SlotType, a.DayPart)
	}
	// HALF_DAY window must equal the AM boundaries (09:00–12:00 defaults)
	if a.ScheduledStart.Format("15:04") != "09:00" || a.ScheduledEnd.Format("15:04") != "12:00" {
		t.Fatalf("half-day window = %s~%s", a.ScheduledStart.Format("15:04"), a.ScheduledEnd.Format("15:04"))
	}

	// pool = 6 slots × capacity 1 → five more AM fuzzies fit, the 7th must fail
	for i := 0; i < 5; i++ {
		halfDay(t, e, e.mbr2, "AM", 2)
	}
	_, err := e.p.Create(ctx, e.mbr1, e.svcID, BookingReq{Date: dateAt(t, 2), DayPart: "AM"}, "")
	if !shared.Is(err, "APT_HALFDAY_FULL") {
		t.Fatalf("want APT_HALFDAY_FULL for 7th AM fuzzy, got %v", err)
	}
	// PM pool untouched
	halfDay(t, e, e.mbr1, "PM", 2)
}

// 一个模糊预约不占用逐槽并发：具体时间仍可与之共存（用户公式），但名额池已消耗。
func TestSpecificCoexistsWithFuzzy(t *testing.T) {
	e := newAptEnv(t, nil)
	halfDay(t, e, e.mbr1, "AM", 3)
	// slot 10:00 free on the per-slot axis → allowed even though a fuzzy exists
	if _, err := e.p.Create(context.Background(), e.mbr2, e.svcID, BookingReq{StartTime: slotAt(t, 3, 10, 0)}, ""); err != nil {
		t.Fatalf("specific should coexist with fuzzy: %v", err)
	}
}

func TestFuzzyToSpecificReschedule(t *testing.T) {
	e := newAptEnv(t, nil)
	a := halfDay(t, e, e.mbr1, "PM", 4)
	moved, err := e.p.Reschedule(context.Background(), e.mbr1, a.ID, BookingReq{StartTime: slotAt(t, 4, 14, 0)}, false)
	if err != nil {
		t.Fatalf("fuzzy→specific reschedule: %v", err)
	}
	if moved.SlotType != SlotTypeSpecific || moved.ScheduledStart.Format("15:04") != "14:00" {
		t.Fatalf("rescheduled = %s %s", moved.SlotType, moved.ScheduledStart.Format("15:04"))
	}
	// and back to fuzzy
	back, err := e.p.Reschedule(context.Background(), e.mbr1, a.ID, BookingReq{Date: dateAt(t, 4), DayPart: "PM"}, true)
	if err != nil {
		t.Fatalf("specific→fuzzy reschedule: %v", err)
	}
	if back.SlotType != SlotTypeHalfDay || back.DayPart != "PM" {
		t.Fatalf("rescheduled back = %s/%s", back.SlotType, back.DayPart)
	}
}

func TestClosureBlocksAndReportsConflicts(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()
	day := dateAt(t, 5)

	halfDay(t, e, e.mbr1, "AM", 5)
	if _, err := e.p.Create(ctx, e.mbr2, e.svcID, BookingReq{StartTime: slotAt(t, 5, 10, 0)}, ""); err != nil {
		t.Fatalf("setup specific: %v", err)
	}

	// closing AM reports 2 active bookings (1 fuzzy + 1 specific)
	n, err := e.p.CreateClosure(ctx, day, "AM", "外出", "op-1")
	if err != nil {
		t.Fatalf("closure: %v", err)
	}
	if n != 2 {
		t.Fatalf("conflict_count = %d, want 2", n)
	}

	// AM rejected, PM fine
	if _, err := e.p.Create(ctx, e.mbr1, e.svcID, BookingReq{Date: day, DayPart: "AM"}, ""); !shared.Is(err, "APT_CLOSED") {
		t.Fatalf("want APT_CLOSED for AM, got %v", err)
	}
	if _, err := e.p.Create(ctx, e.mbr1, e.svcID, BookingReq{StartTime: slotAt(t, 5, 10, 30)}, ""); !shared.Is(err, "APT_CLOSED") {
		t.Fatalf("want APT_CLOSED for specific in AM, got %v", err)
	}
	if _, err := e.p.Create(ctx, e.mbr1, e.svcID, BookingReq{StartTime: slotAt(t, 5, 14, 0)}, ""); err != nil {
		t.Fatalf("PM should be bookable: %v", err)
	}

	// delete reopens the half-day
	list, err := e.p.ListClosures(ctx, day)
	if err != nil || len(list) != 1 {
		t.Fatalf("closures: %v %d", err, len(list))
	}
	if err := e.p.DeleteClosure(ctx, list[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := e.p.Create(ctx, e.mbr1, e.svcID, BookingReq{Date: day, DayPart: "AM"}, ""); err != nil {
		t.Fatalf("AM reopened: %v", err)
	}
}

func dateAt(t *testing.T, day int) string {
	t.Helper()
	return shared.NowShanghai().AddDate(0, 0, day).Format("2006-01-02")
}

func TestBookingOptionsShape(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()
	day := dateAt(t, 6)

	halfDay(t, e, e.mbr1, "AM", 6)
	opts, err := e.p.BookingOptions(ctx, day)
	if err != nil {
		t.Fatalf("options: %v", err)
	}
	if !opts.Open || opts.AM.Closed || opts.PM.Closed {
		t.Fatalf("open/closed = %v %v %v", opts.Open, opts.AM.Closed, opts.PM.Closed)
	}
	// AM pool: 6 slots × 1 - 1 fuzzy = 5
	if opts.AM.Total != 6 || opts.AM.Remaining != 5 {
		t.Fatalf("am total/remaining = %d/%d", opts.AM.Total, opts.AM.Remaining)
	}
	if len(opts.AM.Slots) != 6 || opts.AM.Slots[0].Remaining != 1 {
		t.Fatalf("am slots = %d first rem %d", len(opts.AM.Slots), opts.AM.Slots[0].Remaining)
	}
	// PM untouched
	if opts.PM.Total != 16 || opts.PM.Remaining != 16 {
		t.Fatalf("pm total/remaining = %d/%d", opts.PM.Total, opts.PM.Remaining)
	}
	// closure hides the half-day
	if _, err := e.p.CreateClosure(ctx, day, "PM", "t", "op"); err != nil {
		t.Fatalf("closure: %v", err)
	}
	opts, err = e.p.BookingOptions(ctx, day)
	if err != nil {
		t.Fatalf("options2: %v", err)
	}
	if !opts.PM.Closed || len(opts.PM.Slots) != 0 || !opts.Open {
		t.Fatalf("pm closed = %v slots %d open %v", opts.PM.Closed, len(opts.PM.Slots), opts.Open)
	}
}

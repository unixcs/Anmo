package appointment

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/member"
	svcmodule "anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

type aptEnv struct {
	p     *Provider
	mem   *member.Provider
	svc   *svcmodule.Provider
	mbr1  string
	mbr2  string
	svcID string
	cfg   *config.Config
}

func newAptEnv(t *testing.T, mutate func(*config.Config)) *aptEnv {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_apt_")
	cfg := config.Defaults()
	if mutate != nil {
		mutate(cfg)
	}
	e := &aptEnv{p: New(db, cfg, svcmodule.New(db, cfg)), mem: member.New(db, cfg), svc: svcmodule.New(db, cfg), cfg: cfg}

	ctx := context.Background()
	for i, ph := range []string{"13900000011", "13900000012"} {
		err := shared.RunInTx(ctx, e.p.db, func(tx shared.Tx) error {
			var er error
			if i == 0 {
				e.mbr1, _, er = e.mem.EnsureByPhone(ctx, tx, ph, "顾客A")
			} else {
				e.mbr2, _, er = e.mem.EnsureByPhone(ctx, tx, ph, "顾客B")
			}
			return er
		})
		if err != nil {
			t.Fatalf("member: %v", err)
		}
	}
	cat, err := e.svc.CreateCategory(ctx, "按摩", 1)
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	it, err := e.svc.CreateItem(ctx, svcmodule.NewItem{CategoryID: cat.ID, Name: "肩颈按摩", DurationMin: 60, PriceCents: 12800})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	e.svcID = it.ID
	return e
}

// slotAt builds a business-hours slot string relative to tomorrow at the given
// hour:minute so it always satisfies the 2h lead rule.
func slotAt(t *testing.T, day int, hh, mm int) string {
	t.Helper()
	d := shared.NowShanghai().AddDate(0, 0, day)
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d", d.Year(), d.Month(), d.Day(), hh, mm)
}

func TestCreateAndConflictDetection(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()

	// base booking 15:00-16:00 (2 days out to satisfy lead rules)
	a, err := e.p.Create(ctx, e.mbr1, e.svcID, slotAt(t, 2, 15, 0), "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Status != StatusPendingConfirm || a.No == "" {
		t.Fatalf("appointment = %+v", a)
	}

	// §28: 15:30-16:30 overlaps → conflict
	if _, err := e.p.Create(ctx, e.mbr2, e.svcID, slotAt(t, 2, 15, 30), ""); !shared.Is(err, "APPOINTMENT_CONFLICT") {
		t.Fatalf("want conflict for overlap, got %v", err)
	}
	// §28: 16:00-17:00 adjacent → allowed
	if _, err := e.p.Create(ctx, e.mbr2, e.svcID, slotAt(t, 2, 16, 0), ""); err != nil {
		t.Fatalf("adjacent slot rejected: %v", err)
	}
	// same member cannot double-book either
	if _, err := e.p.Create(ctx, e.mbr1, e.svcID, slotAt(t, 2, 15, 30), ""); !shared.Is(err, "APPOINTMENT_CONFLICT") {
		t.Fatalf("want conflict for same member, got %v", err)
	}
}

func TestSlotAndWindowValidation(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()

	// off-grid start (not aligned to 30min)
	if _, err := e.p.Create(ctx, e.mbr1, e.svcID, slotAt(t, 2, 15, 10), ""); !shared.Is(err, "APT_BAD_SLOT") {
		t.Fatalf("want APT_BAD_SLOT, got %v", err)
	}
	// too soon: with a 240h lead requirement, any in-hours slot is too soon
	e2 := newAptEnv(t, func(c *config.Config) { c.Business.BookMinAheadHours = 240 })
	if _, err := e2.p.Create(ctx, e2.mbr1, e2.svcID, slotAt(t, 2, 15, 0), ""); !shared.Is(err, "APT_TOO_SOON") {
		t.Fatalf("want APT_TOO_SOON, got %v", err)
	}
	// beyond closing: 20:30 + 60min = 21:30 > 21:00 (D15)
	if _, err := e.p.Create(ctx, e.mbr1, e.svcID, slotAt(t, 2, 20, 30), ""); !shared.Is(err, "APT_OUT_OF_HOURS") {
		t.Fatalf("want APT_OUT_OF_HOURS, got %v", err)
	}
}

func TestStateGuardsAndCancelReleasesSlot(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()

	// Case 9: create 18:00, cancel, re-book same slot by another member
	a, err := e.p.Create(ctx, e.mbr1, e.svcID, slotAt(t, 3, 18, 0), "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := e.p.Start(ctx, a.ID, "op"); !shared.Is(err, "APT_BAD_TRANSITION") {
		t.Fatalf("start before confirm must fail, got %v", err)
	}
	if _, err := e.p.Confirm(ctx, a.ID, "op"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// double confirm rejected
	if _, err := e.p.Confirm(ctx, a.ID, "op"); !shared.Is(err, "APT_BAD_TRANSITION") {
		t.Fatalf("double confirm must fail, got %v", err)
	}
	// customer cancel inside 2h → rejected (slot is 3 days out so bypass: cancel by admin semantics on customer path use now; instead verify release via admin cancel)
	if _, err := e.p.CancelByAdmin(ctx, a.ID, "op", "释放时段"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	// Case 10: completed guard on cancelled appointment
	if _, err := e.p.Complete(ctx, a.ID, "op"); !shared.Is(err, "APT_BAD_TRANSITION") {
		t.Fatalf("complete after cancel must fail, got %v", err)
	}
	// cancelled slot no longer blocks (Case 9)
	if _, err := e.p.Create(ctx, e.mbr2, e.svcID, slotAt(t, 3, 18, 0), ""); err != nil {
		t.Fatalf("slot not released after cancel: %v", err)
	}
}

func TestCompleteIdempotentAndNoShowGuard(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()

	a, _ := e.p.Create(ctx, e.mbr1, e.svcID, slotAt(t, 4, 10, 0), "")
	if _, err := e.p.Confirm(ctx, a.ID, "op"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// no-show only from CONFIRMED
	b, _ := e.p.Create(ctx, e.mbr2, e.svcID, slotAt(t, 4, 11, 0), "")
	if _, err := e.p.Confirm(ctx, b.ID, "op"); err != nil {
		t.Fatalf("confirm b: %v", err)
	}
	if _, err := e.p.NoShow(ctx, b.ID, "op"); err != nil {
		t.Fatalf("no-show from confirmed: %v", err)
	}
	if _, err := e.p.NoShow(ctx, b.ID, "op"); !shared.Is(err, "APT_BAD_TRANSITION") {
		t.Fatalf("double no-show must fail, got %v", err)
	}
	if _, err := e.p.Start(ctx, a.ID, "op"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := e.p.Complete(ctx, a.ID, "op"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	// Case 10: 重复完成必须拒绝（§99）
	if _, err := e.p.Complete(ctx, a.ID, "op"); !shared.Is(err, "APT_BAD_TRANSITION") {
		t.Fatalf("second complete must fail, got %v", err)
	}
}

func TestRescheduleExcludesSelf(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()

	a, _ := e.p.Create(ctx, e.mbr1, e.svcID, slotAt(t, 5, 14, 0), "")
	if _, err := e.p.Confirm(ctx, a.ID, "op"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// b occupies 15:00
	if _, err := e.p.Create(ctx, e.mbr2, e.svcID, slotAt(t, 5, 15, 0), ""); err != nil {
		t.Fatalf("create b: %v", err)
	}
	// a reschedules onto its own current time is a no-op conflict-wise but
	// moving onto 15:00 (b's slot) must fail
	if _, err := e.p.Reschedule(ctx, e.mbr1, a.ID, slotAt(t, 5, 15, 0), true); !shared.Is(err, "APPOINTMENT_CONFLICT") {
		t.Fatalf("want conflict, got %v", err)
	}
	// moving to a free slot works
	moved, err := e.p.Reschedule(ctx, e.mbr1, a.ID, slotAt(t, 5, 16, 0), true)
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if !moved.ScheduledStart.Equal(timeMustParse(slotAt(t, 5, 16, 0))) {
		t.Fatalf("moved to %v", moved.ScheduledStart)
	}
}

func TestConcurrentBookingSameSlot(t *testing.T) {
	e := newAptEnv(t, nil)
	ctx := context.Background()

	slot := slotAt(t, 6, 15, 0)
	const n = 8
	results := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m := e.mbr1
			if i%2 == 1 {
				m = e.mbr2
			}
			_, err := e.p.Create(ctx, m, e.svcID, slot, "")
			results[i] = err
		}(i)
	}
	wg.Wait()
	successes := 0
	for i, err := range results {
		if err == nil {
			successes++
		} else if !shared.Is(err, "APPOINTMENT_CONFLICT") && !shared.Is(err, "APT_LOCK_BUSY") {
			t.Fatalf("unexpected error from goroutine %d: %v", i, err)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent bookings: %d succeeded, want exactly 1", successes)
	}
}

func timeMustParse(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, shared.NowShanghai().Location())
	return t
}

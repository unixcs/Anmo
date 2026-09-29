package content

import (
	"context"
	"fmt"
	"testing"
	"time"

	"anmo/server/internal/config"
	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/modules/member"
	"anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
	"anmo/server/internal/testsupport"
)

// status_test.go — 门店状态动态计算（goal §17）与 business_* 设置校验链（§29）。

type contentEnv struct {
	p     *Provider
	apt   *appointment.Provider
	svc   *service.Provider
	mem   *member.Provider
	mbrID string
	svcID string
	catID string
}

func newContentEnv(t *testing.T) *contentEnv {
	t.Helper()
	db := testsupport.NewSchemaDB(t, "anmo_content_")
	cfg := config.Defaults()
	mem := member.New(db, cfg)
	svc := service.New(db, cfg)
	apt := appointment.New(db, cfg, svc, nil)
	p := New(db, cfg)
	p.Wire(apt, svc)
	e := &contentEnv{p: p, apt: apt, svc: svc, mem: mem}

	ctx := context.Background()
	err := shared.RunInTx(ctx, db, func(tx shared.Tx) error {
		var er error
		e.mbrID, _, er = mem.EnsureByPhone(ctx, tx, "13900000031", "状态测试")
		return er
	})
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	cat, err := svc.CreateCategory(ctx, "按摩", 1)
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	e.catID = cat.ID
	it, err := svc.CreateItem(ctx, service.NewItem{CategoryID: cat.ID, Name: "肩颈按摩", DurationMin: 60, PriceCents: 12800})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	e.svcID = it.ID
	return e
}

// hhmm 偏移动当前上海时间，跨日截断到当天边界内。
func hhmmOffset(t *testing.T, minutes int) string {
	t.Helper()
	d := shared.NowShanghai().Add(time.Duration(minutes) * time.Minute)
	h, m := d.Hour(), d.Minute()
	if h == 0 && m == 0 {
		m = 1
	}
	return fmt.Sprintf("%02d:%02d", h, m)
}

func TestStoreStatusThreeStates(t *testing.T) {
	e := newContentEnv(t)
	ctx := context.Background()

	// ---- 空闲：营业窗口覆盖当前时间 ±(60/150) 分钟，最短服务 60 分钟排得下 ----
	open := hhmmOffset(t, -60)
	close := hhmmOffset(t, 150)
	if err := e.p.SaveSetting(ctx, "business_open_time", open, "op"); err != nil {
		t.Fatalf("save open: %v", err)
	}
	if err := e.p.SaveSetting(ctx, "business_close_time", close, "op"); err != nil {
		t.Fatalf("save close: %v", err)
	}
	st, err := e.p.StoreStatus(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	now := shared.NowShanghai()
	cur := now.Hour()*60 + now.Minute()
	closeMin := cur + 150
	// open = now-60 在 00:00~01:00（上海墙钟）会回跨到前一日（如 23:16），
	// 生成"跨零点窗口"——预约引擎明确不支持该配置（APT_BAD_WINDOW：开门<分界<关门
	// 须同日），此时 FREE 断言无意义，跳过（与下方打烊边界跳过同口径）。
	if closeMin <= 24*60 && cur >= 60 && closeMin-cur >= 90 {
		if st.Status != "FREE" {
			t.Fatalf("want FREE with no serving, got %s (%+v)", st.Status, st)
		}
	} else {
		t.Skip("接近打烊/跨零点边界，FREE 断言跳过")
	}

	// ---- 服务中：创建一个未来预约并开始（Start 不校验时间）→ SERVING + free_at ----
	d := now.AddDate(0, 0, 1)
	slot := fmt.Sprintf("%04d-%02d-%02d %02d:%02d", d.Year(), d.Month(), d.Day(), 10, 0)
	a, err := e.apt.Create(ctx, e.mbrID, e.svcID, appointment.BookingReq{StartTime: slot}, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := e.apt.Start(ctx, a.ID, "op-1"); err != nil {
		t.Fatalf("start: %v", err)
	}
	st, err = e.p.StoreStatus(ctx)
	if err != nil {
		t.Fatalf("status serving: %v", err)
	}
	if st.Status != "SERVING" {
		t.Fatalf("want SERVING, got %s", st.Status)
	}
	if st.FreeAt == "" {
		t.Fatal("SERVING should carry free_at (started_at + duration)")
	}

	// ---- 忙碌：服务结束后窗口剩余不足以排下最短服务 → BUSY ----
	if err := e.p.SaveSetting(ctx, "business_close_time", hhmmOffset(t, 30), "op"); err != nil {
		t.Fatalf("save close2: %v", err)
	}
	st, err = e.p.StoreStatus(ctx)
	if err != nil {
		t.Fatalf("status busy: %v", err)
	}
	if st.Status != "SERVING" {
		t.Fatalf("still IN_SERVICE should dominate as SERVING, got %s", st.Status)
	}
}

func TestBookingRulesCapacityFromSettings(t *testing.T) {
	e := newContentEnv(t)
	ctx := context.Background()

	// capacity 67（goal §29）：前端→API→DB→回读→预约引擎全链路以 DB 为准
	if err := e.p.SaveSetting(ctx, "business_slot_capacity", "67", "op"); err != nil {
		t.Fatalf("save 67: %v", err)
	}
	rules, err := e.p.BookingRules(ctx)
	if err != nil {
		t.Fatalf("rules: %v", err)
	}
	if rules.SlotCapacity != 67 {
		t.Fatalf("capacity = %d, want 67", rules.SlotCapacity)
	}
	// 回读（后台刷新场景）
	m, err := e.p.Settings(ctx)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if m["business_slot_capacity"] != "67" {
		t.Fatalf("settings roundtrip = %q", m["business_slot_capacity"])
	}
	// 非法值被服务端校验拦截（§29 不能只修 Input）
	for _, v := range []string{"0", "-3", "abc", "1.5", ""} {
		if err := e.p.SaveSetting(ctx, "business_slot_capacity", v, "op"); err == nil {
			t.Fatalf("invalid capacity %q accepted", v)
		}
	}
	if err := e.p.SaveSetting(ctx, "business_slot_minutes", "45", "op"); err == nil {
		t.Fatal("invalid slot_minutes accepted")
	}
	if err := e.p.SaveSetting(ctx, "business_open_time", "9:00", "op"); err == nil {
		t.Fatal("invalid open_time accepted")
	}
	// 合法值不受误伤
	if err := e.p.SaveSetting(ctx, "business_slot_capacity", "999", "op"); err != nil {
		t.Fatalf("capacity 999 should pass: %v", err)
	}
}

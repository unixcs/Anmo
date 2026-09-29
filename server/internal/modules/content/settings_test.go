package content

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"anmo/server/internal/config"
	"anmo/server/internal/testsupport"
)

// settings_test.go — 首页服务推荐 settings 键（V2.1）：校验矩阵 + PublicSettings 下发矩阵。

func TestValidateHomeServiceLimit(t *testing.T) {
	e := newContentEnv(t)
	ctx := context.Background()
	// 合法：空 = 客户端默认 6；2/4/6/8
	for _, v := range []string{"", "2", "4", "6", "8"} {
		if err := e.p.SaveSetting(ctx, "home_service_limit", v, "op"); err != nil {
			t.Fatalf("limit %q should pass: %v", v, err)
		}
	}
	// 非法：越界数字 / 补零 / 非数字
	for _, v := range []string{"1", "3", "5", "7", "9", "02", "66", "abc", "-2", "2.0"} {
		if err := e.p.SaveSetting(ctx, "home_service_limit", v, "op"); err == nil {
			t.Fatalf("invalid limit %q accepted", v)
		}
	}
}

func TestValidateHomeServiceIds(t *testing.T) {
	e := newContentEnv(t)
	ctx := context.Background()
	// 合法：空 / "[]"（后台清空勾选时保存）/ 单项 / 恰好 50 项
	fifty := make([]string, 50)
	for i := range fifty {
		fifty[i] = fmt.Sprintf("id%02d", i)
	}
	fiftyJSON, _ := json.Marshal(fifty)
	for _, v := range []string{"", "[]", `["abc"]`, string(fiftyJSON)} {
		if err := e.p.SaveSetting(ctx, "home_service_ids", v, "op"); err != nil {
			t.Fatalf("ids %q should pass: %v", v, err)
		}
	}
	// 非法：非 JSON / 非字符串元素 / 空元素 / null 元素 / 51 项 / 对象
	fiftyOne := make([]string, 51)
	for i := range fiftyOne {
		fiftyOne[i] = fmt.Sprintf("id%02d", i)
	}
	fiftyOneJSON, _ := json.Marshal(fiftyOne)
	for _, v := range []string{"abc", `["a",1]`, `["", "a"]`, `["a",null]`, string(fiftyOneJSON), `{"a":1}`} {
		if err := e.p.SaveSetting(ctx, "home_service_ids", v, "op"); err == nil {
			t.Fatalf("invalid ids %q accepted", v)
		}
	}
}

func TestPublicSettingsHomeServiceKeys(t *testing.T) {
	db := testsupport.NewSchemaDB(t, "anmo_homecfg_")
	p := New(db, config.Defaults())
	ctx := context.Background()

	// 未配置：下发空串（客户端回落默认 6 / 全量）
	out, err := p.PublicSettings(ctx)
	if err != nil {
		t.Fatalf("public settings: %v", err)
	}
	if out["home_service_limit"] != "" || out["home_service_ids"] != "" {
		t.Fatalf("unset keys should downlink empty, got limit=%q ids=%q", out["home_service_limit"], out["home_service_ids"])
	}

	// 合法配置：原样下发
	if err := p.SaveSetting(ctx, "home_service_limit", "4", "op"); err != nil {
		t.Fatalf("save limit: %v", err)
	}
	if err := p.SaveSetting(ctx, "home_service_ids", `["s1","s2"]`, "op"); err != nil {
		t.Fatalf("save ids: %v", err)
	}
	out, err = p.PublicSettings(ctx)
	if err != nil {
		t.Fatalf("public settings: %v", err)
	}
	if out["home_service_limit"] != "4" {
		t.Fatalf("limit downlink = %q, want 4", out["home_service_limit"])
	}
	if out["home_service_ids"] != `["s1","s2"]` {
		t.Fatalf("ids downlink = %q", out["home_service_ids"])
	}

	// 历史脏数据（绕过校验直写 DB）：limit 非法时下发空串，ids 原样（客户端解析失败回落全量）
	if _, err := db.ExecContext(ctx,
		`UPDATE content_system_setting SET setting_value='9' WHERE setting_key='home_service_limit'`); err != nil {
		t.Fatalf("corrupt limit: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE content_system_setting SET setting_value='not-json' WHERE setting_key='home_service_ids'`); err != nil {
		t.Fatalf("corrupt ids: %v", err)
	}
	out, err = p.PublicSettings(ctx)
	if err != nil {
		t.Fatalf("public settings: %v", err)
	}
	if out["home_service_limit"] != "" {
		t.Fatalf("corrupt limit should downlink empty, got %q", out["home_service_limit"])
	}
	if out["home_service_ids"] != "not-json" {
		t.Fatalf("corrupt ids should pass through, got %q", out["home_service_ids"])
	}
}

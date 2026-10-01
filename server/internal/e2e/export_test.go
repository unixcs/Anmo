package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"anmo/server/internal/middleware"
	"anmo/server/internal/router"
)

// rawDo — 二进制响应通道：导出端点返回文件而非 JSON 包络（D30）。
func (c *client) rawDo(method, path string, body any) (int, http.Header, []byte) {
	c.t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(b))
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	return res.StatusCode, res.Header, data
}

func mustExport(t *testing.T, c *client, domain string, body map[string]any) (http.Header, []byte) {
	t.Helper()
	status, hdr, data := c.rawDo("POST", "/admin/export/"+domain, body)
	if status != 200 {
		t.Fatalf("export %s = %d: %s", domain, status, data)
	}
	return hdr, data
}

// TestExportDomains — D30 P1 三域导出：xlsx 魔数 / TXT 打码与结构 / 权限与格式校验。
// 数据侧复用 D29 散客结算：一次带手机号的散客收款同时产出 member + payment + service_record。
func TestExportDomains(t *testing.T) {
	admin := newServer(t)
	res := admin.ok("POST", "/admin/auth/login", map[string]string{
		"phone": "13800000000", "password": "e2e-admin-pass",
	})
	admin.token = str(res["data"].(map[string]any), "token")

	cat := admin.ok("POST", "/admin/service-categories", map[string]any{"name": "导出测试类", "sort": 1})
	catID := str(cat["data"].(map[string]any), "id")
	svc := admin.ok("POST", "/admin/services", map[string]any{
		"category_id": catID, "name": "精油开背", "duration_minutes": 60,
		"default_price": int64(19800),
	})
	svcID := str(svc["data"].(map[string]any), "id")

	// 带手机号散客结算：建档 + payment(现金) + 服务记录（部位快照）
	walk := admin.ok("POST", "/admin/walkin/settle", map[string]any{
		"phone": "13912345678", "name": "王五", "service_id": svcID,
		"pay_method": "CASH", "amount": int64(19800),
		"body_parts": []string{"肩颈", "背部"}, "service_method": "精油",
		"communicated": true, "idempotency_key": "e6000000-0000-0000-0000-0000000000e1",
	})
	if walk["data"] == nil {
		t.Fatalf("walkin settle failed: %v", walk)
	}

	// 未录手机号的散客 payment（member_id NULL → LEFT JOIN 留白，v3 验收 9）
	admin.ok("POST", "/admin/walkin/settle", map[string]any{
		"service_id": svcID, "pay_method": "OTHER", "amount": int64(100),
		"communicated": true, "idempotency_key": "e6000000-0000-0000-0000-0000000000e2",
	})

	// ---- 会员总表 ----
	hdr, xlsx := mustExport(t, admin, "members", map[string]any{"format": "xlsx"})
	if !bytes.HasPrefix(xlsx, []byte("PK")) {
		t.Fatal("members xlsx not zip")
	}
	if hdr.Get("X-Export-Rows") == "" || hdr.Get("X-Export-Rows") == "0" {
		t.Fatalf("X-Export-Rows = %q", hdr.Get("X-Export-Rows"))
	}
	_, txt := mustExport(t, admin, "members", map[string]any{"format": "txt"})
	body := string(txt)
	if !strings.Contains(body, "王五") || !strings.Contains(body, "139****5678") {
		t.Fatalf("members txt missing masked member:\n%s", body)
	}
	if strings.Contains(body, "13912345678") {
		t.Fatalf("members txt leaked full phone:\n%s", body)
	}
	if !strings.Contains(body, "¥198.00") { // 累计消费 分→元
		t.Fatalf("members txt missing money:\n%s", body)
	}
	// 筛选传参
	_, ftxt := mustExport(t, admin, "members", map[string]any{"format": "txt", "keyword": "王五"})
	if !strings.Contains(string(ftxt), "王五") {
		t.Fatalf("keyword filter failed:\n%s", string(ftxt))
	}
	_, ftxt2 := mustExport(t, admin, "members", map[string]any{"format": "txt", "keyword": "不存在"})
	if strings.Contains(string(ftxt2), "王五") {
		t.Fatalf("keyword filter not applied:\n%s", string(ftxt2))
	}

	// ---- 收款流水 ----
	_, payTxt := mustExport(t, admin, "payments", map[string]any{"format": "txt"})
	pb := string(payTxt)
	if !strings.Contains(pb, "微信转账") && !strings.Contains(pb, "现金") {
		t.Fatalf("payments txt missing method labels:\n%s", pb)
	}
	if strings.Contains(pb, "13912345678") {
		t.Fatalf("payments txt leaked full phone:\n%s", pb)
	}
	// TXT 只出 5 列（v3）：会员编号/手机号/预约号不出现
	if strings.Contains(pb, "会员编号") || strings.Contains(pb, "预约号") {
		t.Fatalf("payments txt should be 5-col:\n%s", pb)
	}
	// 散客 NULL member 行在 xlsx（全列）中出现：xlsx 含 8 列表头
	_, payXlsx := mustExport(t, admin, "payments", map[string]any{"format": "xlsx"})
	if !bytes.HasPrefix(payXlsx, []byte("PK")) {
		t.Fatal("payments xlsx not zip")
	}
	// status 筛选
	_, voided := mustExport(t, admin, "payments", map[string]any{"format": "txt", "status": "VOIDED"})
	if strings.Contains(string(voided), "现金") {
		t.Fatalf("status=VOIDED filter failed:\n%s", string(voided))
	}

	// ---- 服务记录 ----
	_, recTxt := mustExport(t, admin, "service-records", map[string]any{"format": "txt"})
	rb := string(recTxt)
	if !strings.Contains(rb, "精油开背") || !strings.Contains(rb, "肩颈、背部") || !strings.Contains(rb, "精油") {
		t.Fatalf("service-records txt wrong:\n%s", rb)
	}

	// ---- 校验类 ----
	status, _, data := admin.rawDo("POST", "/admin/export/members", map[string]any{"format": "csv"})
	if status != 400 || !strings.Contains(string(data), "EXPORT_BAD_FORMAT") {
		t.Fatalf("bad format = %d %s", status, data)
	}
	anon := &client{t: t, base: admin.base}
	status, _, data = anon.rawDo("POST", "/admin/export/members", map[string]any{"format": "txt"})
	if status != 401 {
		t.Fatalf("unauthenticated export = %d", status)
	}

	// ---- 审计：POST /admin/export/* 必须落操作日志且含行数（D30 §4）----
	logs := admin.ok("GET", "/admin/logs", nil)
	logList, _ := logs["data"].([]any)
	found := false
	for _, it := range logList {
		m, _ := it.(map[string]any)
		action, _ := m["action"].(string)
		detail, _ := m["detail"].(string)
		if strings.HasPrefix(action, "POST /admin/export/") {
			if strings.Contains(detail, `"rows"`) {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("audit log missing export rows: %v", logs)
	}
}

// TestOperationLogCapturesExportHeader — 中间件层：handler 设置的 X-Export-Rows
// 必须出现在审计回调的 hdr 里（D30 §4 机制）。
func TestOperationLogCapturesExportHeader(t *testing.T) {
	got := make(chan http.Header, 1)
	h := router.New(slog.Default(), verifyStub, func(r *http.Request, status int, hdr http.Header) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/admin/") {
			got <- hdr
		}
	}, nil, exportModuleHolder{})
	req := httptest.NewRequest("POST", "/admin/export/members", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if h2 := <-got; h2.Get("X-Export-Rows") != "7" {
		t.Fatalf("X-Export-Rows = %q", h2.Get("X-Export-Rows"))
	}
}

func verifyStub(token string) (middleware.Principal, error) {
	return middleware.Principal{ActorType: "ADMIN", Role: "OWNER"}, nil
}

type exportModuleHolder struct{}

func (exportModuleHolder) Mount(root, admin, api *http.ServeMux) {
	admin.HandleFunc("POST /admin/export/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Export-Rows", "7")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("data"))
	})
}

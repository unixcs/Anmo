package export

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func sampleTable(layout string) Table {
	return Table{
		Sheet:        "会员",
		FilenameBase: "anmo-会员总表",
		Title:        "会员总表",
		MDLayout:     layout,
		CardTitleCol: 1,
		Cols: []Col{
			{Header: "手机号", Kind: KindPhone, InTXT: true},
			{Header: "姓名", Kind: KindText, InTXT: true},
			{Header: "累计消费", Kind: KindMoney, InTXT: true},
			{Header: "内部备注", Kind: KindText, InTXT: false},
		},
		Rows: [][]any{
			{"13800001234", "张三", int64(358000), "=cmd|' /c calc'!A0"},
			{"13800001111", "李四", int64(0), "-SUM(A1)"},
			{"", "散客", int64(12800), "@"},
		},
	}
}

func TestMaskPhoneBoundaries(t *testing.T) {
	cases := map[string]string{
		"13800001234": "138****1234",
		"138000012":   "13****12",  // 8~10 位：头尾各 2
		"1234567":     "***",       // <8 位
		"":            "",
	}
	for in, want := range cases {
		if got := MaskPhone(in); got != want {
			t.Fatalf("MaskPhone(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRenderTXTTableMasksAndDropsCols(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := RenderTXT(rec, sampleTable(LayoutTable)); err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	h := rec.Header()
	if got := h.Get("X-Export-Rows"); got != "3" {
		t.Fatalf("X-Export-Rows=%q", got)
	}
	if !strings.Contains(h.Get("Content-Disposition"), "filename*") {
		t.Fatalf("missing RFC5987 filename*: %q", h.Get("Content-Disposition"))
	}
	if !strings.Contains(body, "138****1234") || strings.Contains(body, "13800001234") {
		t.Fatalf("phone not masked in TXT")
	}
	// InTXT=false 列必须整体消失（含注入串）
	if strings.Contains(body, "内部备注") || strings.Contains(body, "cmd") {
		t.Fatalf("InTXT=false column leaked:\n%s", body)
	}
	if !strings.Contains(body, "| 手机号 | 姓名 | 累计消费 |") {
		t.Fatalf("table header wrong:\n%s", body)
	}
	if !strings.Contains(body, "¥3580.00") || !strings.Contains(body, "¥0.00") || !strings.Contains(body, "¥128.00") {
		t.Fatalf("money render wrong:\n%s", body)
	}
}

func TestRenderTXTCards(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := RenderTXT(rec, sampleTable(LayoutCards)); err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	// 卡片标题 = CardTitleCol=1（姓名）
	if !strings.Contains(body, "### 张三\n") || !strings.Contains(body, "### 散客\n") {
		t.Fatalf("cards headings wrong:\n%s", body)
	}
	if !strings.Contains(body, "- 手机号：138****1234") {
		t.Fatalf("card field wrong:\n%s", body)
	}
	if strings.Contains(body, "内部备注") {
		t.Fatalf("InTXT=false column leaked in cards")
	}
}

func TestRenderXLSXBasics(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := RenderXLSX(rec, sampleTable(LayoutTable)); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("PK")) {
		t.Fatalf("not an xlsx (zip) payload")
	}
	if got := rec.Header().Get("X-Export-Rows"); got != "3" {
		t.Fatalf("X-Export-Rows=%q", got)
	}
	if rec.Header().Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("content type %q", rec.Header().Get("Content-Type"))
	}
}

func TestRenderXLSXReadBack(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := RenderXLSX(rec, sampleTable(LayoutTable)); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()
	rows, err := f.GetRows("会员")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 { // header + 3
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[1][0] != "13800001234" { // xlsx 全量手机号
		t.Fatalf("phone cell = %q", rows[1][0])
	}
	if rows[1][2] != "3580.00" { // money: 358000 分 → 3580.00（0.00 数字格式）
		t.Fatalf("money cell = %q", rows[1][2])
	}
	if rows[3][0] != "" { // 空手机号
		t.Fatalf("empty phone cell = %q", rows[3][0])
	}
}

func TestRowCap(t *testing.T) {
	tbl := sampleTable(LayoutTable)
	rows := make([][]any, RowCap+1)
	for i := range rows {
		rows[i] = []any{"x", "y", int64(1), "z"}
	}
	tbl.Rows = rows
	rec := httptest.NewRecorder()
	err := RenderTXT(rec, tbl)
	if err == nil || !strings.Contains(err.Error(), "EXPORT_TOO_LARGE") {
		t.Fatalf("want EXPORT_TOO_LARGE, got %v", err)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("build-then-write violated: bytes written before error")
	}
}

// Package export renders admin data exports (D30): XLSX for bookkeeping and
// TXT (Markdown content) for phone viewing / AI consumption. It is the single
// place that writes export HTTP responses — build fully in memory first
// (build-then-write), so any render error happens before the first byte and
// can still surface as a normal JSON error via shared.Fail.
package export

import (
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"anmo/server/internal/shared"
)

// RowCap bounds any export; exceeded → 400 EXPORT_TOO_LARGE (提案 §4).
const RowCap = 50_000

// ValidateFormat normalizes the requested export format: "" → "xlsx",
// otherwise only "xlsx" | "txt" (D30).
func ValidateFormat(f string) (string, error) {
	switch f {
	case "", "xlsx":
		return "xlsx", nil
	case "txt":
		return "txt", nil
	default:
		return "", shared.BadRequest("EXPORT_BAD_FORMAT", "导出格式仅支持 xlsx / txt")
	}
}

// Kind describes how a column value is rendered per format.
type Kind int

const (
	KindText Kind = iota // raw string; formula-injection guarded in xlsx
	KindMoney            // int64 cents → 元 with 2 decimals (number cell in xlsx)
	KindPhone            // full in xlsx; masked in TXT (§3.1)
)

// Col is one export column. InTXT=false columns are dropped from the TXT
// rendering to keep it phone-narrow (v3: 收款流水 5 列), xlsx always keeps all.
type Col struct {
	Header string
	Kind   Kind
	InTXT  bool
}

// Layout chooses the TXT rendering: "table" (≤ narrow domains) or "cards"
// (wide domains, one block per row with `### ` title).
const (
	LayoutTable = "table"
	LayoutCards = "cards"
)

// Table is a fully materialized export (rows already queried).
type Table struct {
	Sheet        string   // xlsx sheet name
	FilenameBase string   // e.g. "anmo-会员总表" (date+ext appended)
	Title        string   // TXT H1
	MDLayout     string   // LayoutTable | LayoutCards
	CardTitleCol int      // column index used as `### ` heading in cards layout
	Cols         []Col
	Rows         [][]any  // values must match Cols; nil → empty cell
}

// RenderXLSX builds the workbook and writes it as an attachment. On success it
// sets X-Export-Rows before WriteHeader so the audit middleware can capture it.
func RenderXLSX(w http.ResponseWriter, t Table) error {
	if len(t.Rows) > RowCap {
		return shared.BadRequest("EXPORT_TOO_LARGE", fmt.Sprintf("导出行数超过上限 %d，请分日期段导出", RowCap))
	}
	f := excelize.NewFile()
	defer f.Close()
	sheet := t.Sheet
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return err
	}
	// header row, bold
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	money, err := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("0.00")})
	if err != nil {
		return err
	}
	header := make([]any, len(t.Cols))
	for i, c := range t.Cols {
		header[i] = c.Header
	}
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := f.SetSheetRow(sheet, cell, &header); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, cell, cell, bold); err != nil {
		return err
	}
	for r, row := range t.Rows {
		vals := make([]any, len(t.Cols))
		for i, c := range t.Cols {
			vals[i] = xlsxCell(c, rowVal(row, i))
		}
		cell, _ := excelize.CoordinatesToCellName(1, r+2)
		if err := f.SetSheetRow(sheet, cell, &vals); err != nil {
			return err
		}
	}
	// apply money style column-wise (SetSheetRow overwrote nothing else)
	for i, c := range t.Cols {
		if c.Kind != KindMoney {
			continue
		}
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetCellStyle(sheet, fmt.Sprintf("%s2", col), fmt.Sprintf("%s%d", col, len(t.Rows)+1), money); err != nil {
			return err
		}
	}
	// width: CJK header ≈ 2 units per rune, min 8 max 40
	for i, c := range t.Cols {
		w := runeWidth(c.Header)
		for _, row := range t.Rows {
			if v := fmt.Sprintf("%v", rowVal(row, i)); runeWidth(v) > w {
				w = runeWidth(v)
			}
		}
		if w < 8 {
			w = 8
		}
		if w > 40 {
			w = 40
		}
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, col, col, float64(w)+2)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	writeHeaders(w, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		t.FilenameBase+".xlsx", len(t.Rows))
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

// RenderTXT writes the Markdown rendering as a .txt attachment.
func RenderTXT(w http.ResponseWriter, t Table) error {
	rows := t.Rows
	if len(rows) > RowCap {
		return shared.BadRequest("EXPORT_TOO_LARGE", fmt.Sprintf("导出行数超过上限 %d，请分日期段导出", RowCap))
	}
	var b strings.Builder
	b.WriteString("# " + t.Title + "\n\n")
	b.WriteString("> 导出时间 " + time.Now().Format("2006-01-02 15:04") + " · 共 " + fmt.Sprint(len(rows)) + " 条 · 手机号已打码\n\n")
	tcols := txtCols(t)
	if t.MDLayout == LayoutCards {
		for _, row := range rows {
			b.WriteString("### " + txtCell(t.Cols[t.CardTitleCol], rowVal(row, t.CardTitleCol)) + "\n")
			for _, ci := range tcols {
				if ci == t.CardTitleCol {
					continue
				}
				b.WriteString("- " + t.Cols[ci].Header + "：" + txtCell(t.Cols[ci], rowVal(row, ci)) + "\n")
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("| " + strings.Join(txtHeaders(t), " | ") + " |\n")
		b.WriteString("|" + strings.Repeat(" --- |", len(tcols)) + "\n")
		for _, row := range rows {
			cells := make([]string, len(tcols))
			for j, ci := range tcols {
				cells[j] = txtCell(t.Cols[ci], rowVal(row, ci))
			}
			b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		}
	}
	writeHeaders(w, "text/plain; charset=utf-8", t.FilenameBase+".txt", len(rows))
	_, err := w.Write([]byte(b.String()))
	return err
}

// DateRange validates optional YYYY-MM-DD wall-clock date filters shared by
// the payments / service-records domains (提案 §3 筛选 schema).
func DateRange(from, to string) (string, string, error) {
	parse := func(s string) (string, error) {
		if s == "" {
			return "", nil
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return "", shared.BadRequest("EXPORT_BAD_DATE", "日期格式应为 YYYY-MM-DD")
		}
		return s, nil
	}
	f, err := parse(from)
	if err != nil {
		return "", "", err
	}
	t, err := parse(to)
	if err != nil {
		return "", "", err
	}
	if f != "" && t != "" && f > t {
		return "", "", shared.BadRequest("EXPORT_BAD_DATE", "开始日期不能晚于结束日期")
	}
	return f, t, nil
}

func txtCols(t Table) []int {
	out := make([]int, 0, len(t.Cols))
	for i, c := range t.Cols {
		if c.InTXT {
			out = append(out, i)
		}
	}
	return out
}

func txtHeaders(t Table) []string {
	hs := make([]string, 0, len(t.Cols))
	for _, ci := range txtCols(t) {
		hs = append(hs, t.Cols[ci].Header)
	}
	return hs
}

func rowVal(row []any, i int) any {
	if i < len(row) {
		return row[i]
	}
	return nil
}

// xlsxCell maps a typed value to a spreadsheet cell value.
func xlsxCell(c Col, v any) any {
	switch c.Kind {
	case KindMoney:
		if n, ok := v.(int64); ok {
			return float64(n) / 100
		}
		return nil
	case KindPhone:
		s, _ := v.(string)
		return sanitize(s)
	default:
		s, _ := v.(string)
		return sanitize(s)
	}
}

// txtCell renders one value for Markdown; phones are masked (§3.1).
func txtCell(c Col, v any) string {
	switch c.Kind {
	case KindMoney:
		if n, ok := v.(int64); ok {
			yuan := float64(n) / 100
			return fmt.Sprintf("¥%.2f", yuan)
		}
		return ""
	case KindPhone:
		s, _ := v.(string)
		return MaskPhone(s)
	default:
		s, _ := v.(string)
		return s
	}
}

// MaskPhone: 11 digits → 前3+****+后4; 8~10 → 头尾各 2; <8 → ***; "" → "".
func MaskPhone(p string) string {
	n := len([]rune(p))
	switch {
	case p == "":
		return ""
	case n >= 11:
		r := []rune(p)
		return string(r[:3]) + "****" + string(r[n-4:])
	case n >= 8:
		r := []rune(p)
		return string(r[:2]) + "****" + string(r[n-2:])
	default:
		return "***"
	}
}

// sanitize neutralizes CSV/Excel formula injection for text cells (=+-@).
func sanitize(s string) string {
	if s == "" {
		return ""
	}
	switch s[0] {
	case '=', '+', '-', '@':
		return "'" + s
	}
	return s
}

func runeWidth(s string) int {
	w := 0
	for _, r := range s {
		if r > 0x2E80 { // CJK & fullwidth
			w += 2
		} else {
			w++
		}
	}
	return w
}

func strPtr(s string) *string { return &s }

// writeHeaders sets attachment headers. filename*= via mime.FormatMediaType
// gives the RFC 5987 UTF-8 form; mime adds a quoted ASCII fallback itself.
func writeHeaders(w http.ResponseWriter, contentType, filename string, rows int) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Export-Rows", fmt.Sprint(rows))
	w.Header().Set("Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
}

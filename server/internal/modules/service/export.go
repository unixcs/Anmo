package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"anmo/server/internal/shared"
	"anmo/server/internal/shared/export"
)

// export.go — 服务记录导出（D30，提案 §3 域 3）。body_parts 存标签名 JSON 数组
// 快照（D28），导出解析为顿号列表；REVERSED 行照导、状态列标注。

type recordExportIn struct {
	Format   string `json:"format"`
	DateFrom string `json:"date_from"` // YYYY-MM-DD，可空
	DateTo   string `json:"date_to"`
}

// handleExportServiceRecords — POST /admin/export/service-records
func (p *Provider) handleExportServiceRecords(w http.ResponseWriter, r *http.Request) {
	var in recordExportIn
	if err := shared.DecodeJSON(r, &in); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	format, err := export.ValidateFormat(in.Format)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	from, to, err := export.DateRange(in.DateFrom, in.DateTo)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	rows, err := p.exportRecordRows(r.Context(), from, to)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	tbl := export.Table{
		Sheet:        "服务记录",
		FilenameBase: "anmo-服务记录",
		Title:        "服务记录",
		MDLayout:     export.LayoutCards, // 10 列宽域：卡片块
		CardTitleCol: 4,
		Cols: []export.Col{
			{Header: "时间", Kind: export.KindText, InTXT: true},
			{Header: "会员编号", Kind: export.KindText, InTXT: true},
			{Header: "姓名", Kind: export.KindText, InTXT: true},
			{Header: "手机号", Kind: export.KindPhone, InTXT: true},
			{Header: "服务名称", Kind: export.KindText, InTXT: true},
			{Header: "部位", Kind: export.KindText, InTXT: true},
			{Header: "手法", Kind: export.KindText, InTXT: true},
			{Header: "商家备注", Kind: export.KindText, InTXT: true},
			{Header: "状态", Kind: export.KindText, InTXT: true},
			{Header: "预约号", Kind: export.KindText, InTXT: true},
		},
		Rows: rows,
	}
	if format == "txt" {
		err = export.RenderTXT(w, tbl)
	} else {
		err = export.RenderXLSX(w, tbl)
	}
	if err != nil {
		shared.Fail(w, err)
	}
}

func (p *Provider) exportRecordRows(ctx context.Context, from, to string) ([][]any, error) {
	where := "1=1"
	args := []any{}
	if from != "" {
		where += " AND r.created_at >= ?"
		args = append(args, from+" 00:00:00")
	}
	if to != "" {
		where += " AND r.created_at <= ?"
		args = append(args, to+" 23:59:59")
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT r.created_at, COALESCE(m.member_no,''), COALESCE(m.name,''), COALESCE(m.phone,''),
		        r.service_name, r.body_parts, r.service_method, r.merchant_note, r.status,
		        COALESCE(a.appointment_no,'')
		 FROM service_record r
		 LEFT JOIN member m ON m.id = r.member_id
		 LEFT JOIN appointment a ON a.id = r.appointment_id
		 WHERE `+where+` ORDER BY r.created_at DESC`, args...)
	if err != nil {
		return nil, shared.Server("EXPORT_REC_Q", err)
	}
	defer rows.Close()
	var out [][]any
	for rows.Next() {
		var createdAt, memberNo, name, phone, svcName, bodyParts, method, note, status, apptNo string
		if err := rows.Scan(&createdAt, &memberNo, &name, &phone, &svcName,
			&bodyParts, &method, &note, &status, &apptNo); err != nil {
			return nil, shared.Server("EXPORT_REC_S", err)
		}
		statusText := "正常"
		if status == "REVERSED" {
			statusText = "已撤销"
		}
		out = append(out, []any{
			createdAt, memberNo, name, phone, svcName,
			partsText(bodyParts), method, note, statusText, apptNo,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, shared.Server("EXPORT_REC_E", err)
	}
	return out, nil
}

// partsText turns the stored JSON name-snapshot array into a 、-joined list.
func partsText(s string) string {
	var arr []string
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return s
	}
	return strings.Join(arr, "、")
}

package transaction

import (
	"context"
	"net/http"

	"anmo/server/internal/shared"
	"anmo/server/internal/shared/export"
)

// export.go — 收款流水导出（D30，提案 §3 域 2）。含 VOIDED 行带状态列供老板筛选；
// payment.member_id 自 015 起可空（D29 散客），LEFT JOIN 留白呈现（v3 审查 P1-1）。

type paymentExportIn struct {
	Format   string `json:"format"`
	Status   string `json:"status"`    // "" | VALID | VOIDED
	DateFrom string `json:"date_from"` // YYYY-MM-DD，可空
	DateTo   string `json:"date_to"`
}

var payMethodText = map[string]string{
	"CARD":            "次卡核销",
	"WECHAT_TRANSFER": "微信转账",
	"CASH":            "现金",
	"OTHER":           "其他",
}

var payStatusText = map[string]string{"VALID": "有效", "VOIDED": "已作废"}

// handleExportPayments — POST /admin/export/payments
func (p *Provider) handleExportPayments(w http.ResponseWriter, r *http.Request) {
	var in paymentExportIn
	if err := shared.DecodeJSON(r, &in); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	format, err := export.ValidateFormat(in.Format)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	if in.Status != "" && in.Status != "VALID" && in.Status != "VOIDED" {
		shared.BadRequest("EXPORT_BAD_STATUS", "status 仅支持 VALID / VOIDED").Write(w)
		return
	}
	from, to, err := export.DateRange(in.DateFrom, in.DateTo)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	rows, err := p.exportPaymentRows(r.Context(), in.Status, from, to)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	tbl := export.Table{
		Sheet:        "收款流水",
		FilenameBase: "anmo-收款流水",
		Title:        "收款流水",
		MDLayout:     export.LayoutTable,
		Cols: []export.Col{
			{Header: "收款时间", Kind: export.KindText, InTXT: true},
			{Header: "会员编号", Kind: export.KindText, InTXT: false},
			{Header: "姓名", Kind: export.KindText, InTXT: true},
			{Header: "手机号", Kind: export.KindPhone, InTXT: false},
			{Header: "预约号", Kind: export.KindText, InTXT: false},
			{Header: "支付方式", Kind: export.KindText, InTXT: true},
			{Header: "金额", Kind: export.KindMoney, InTXT: true},
			{Header: "状态", Kind: export.KindText, InTXT: true},
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

func (p *Provider) exportPaymentRows(ctx context.Context, status, from, to string) ([][]any, error) {
	where := "1=1"
	args := []any{}
	if status != "" {
		where += " AND p.status = ?"
		args = append(args, status)
	}
	if from != "" {
		where += " AND p.recorded_at >= ?"
		args = append(args, from+" 00:00:00")
	}
	if to != "" {
		where += " AND p.recorded_at <= ?"
		args = append(args, to+" 23:59:59")
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT p.recorded_at, COALESCE(m.member_no,''), COALESCE(m.name,''), COALESCE(m.phone,''),
		        COALESCE(a.appointment_no,''), p.method, p.amount, p.status
		 FROM payment p
		 LEFT JOIN member m ON m.id = p.member_id
		 LEFT JOIN appointment a ON a.id = p.appointment_id
		 WHERE `+where+` ORDER BY p.recorded_at DESC`, args...)
	if err != nil {
		return nil, shared.Server("EXPORT_PAY_Q", err)
	}
	defer rows.Close()
	var out [][]any
	for rows.Next() {
		var recorded, memberNo, name, phone, apptNo, method string
		var amount int64
		var status string
		if err := rows.Scan(&recorded, &memberNo, &name, &phone, &apptNo, &method, &amount, &status); err != nil {
			return nil, shared.Server("EXPORT_PAY_S", err)
		}
		out = append(out, []any{
			recorded, memberNo, name, phone, apptNo,
			payMethodText[method], amount, payStatusText[status],
		})
	}
	if err := rows.Err(); err != nil {
		return nil, shared.Server("EXPORT_PAY_E", err)
	}
	return out, nil
}

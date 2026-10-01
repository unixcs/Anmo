package member

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"anmo/server/internal/shared"
	"anmo/server/internal/shared/export"
)

// export.go — 会员总表导出（D30，提案 §3 域 1）。聚合直接在模块内 GROUP BY
// JOIN payment/member_card（先例 repo.go:170），不经跨模块 api 注入——
// transaction.Api import 了 member 包，反向注入即 import cycle（v3 审查 P0-2）。

type memberExportIn struct {
	Format   string `json:"format"`
	Keyword  string `json:"keyword"`
	TagID    string `json:"tag_id"`
	CardType string `json:"card_type"`
}

// handleExportMembers — POST /admin/export/members
// 筛选与 GET /admin/members 列表语义逐字同名（keyword/tag_id/card_type）。
func (p *Provider) handleExportMembers(w http.ResponseWriter, r *http.Request) {
	var in memberExportIn
	if err := shared.DecodeJSON(r, &in); err != nil {
		shared.BadRequest("BAD_JSON", "请求格式错误").Write(w)
		return
	}
	format, err := export.ValidateFormat(in.Format)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	rows, err := p.exportRows(r.Context(), in)
	if err != nil {
		shared.Fail(w, err)
		return
	}
	tbl := export.Table{
		Sheet:        "会员总表",
		FilenameBase: "anmo-会员总表",
		Title:        "会员总表",
		MDLayout:     export.LayoutCards, // 14 列宽域：卡片块（提案 §3.1）
		CardTitleCol: 0,
		Cols: []export.Col{
			{Header: "会员编号", Kind: export.KindText, InTXT: true},
			{Header: "姓名", Kind: export.KindText, InTXT: true},
			{Header: "手机号", Kind: export.KindPhone, InTXT: true},
			{Header: "性别", Kind: export.KindText, InTXT: true},
			{Header: "生日", Kind: export.KindText, InTXT: true},
			{Header: "微信绑定", Kind: export.KindText, InTXT: true},
			{Header: "标签", Kind: export.KindText, InTXT: true},
			{Header: "注册时间", Kind: export.KindText, InTXT: true},
			{Header: "最近到店", Kind: export.KindText, InTXT: true},
			{Header: "累计消费", Kind: export.KindMoney, InTXT: true},
			{Header: "消费次数", Kind: export.KindText, InTXT: true},
			{Header: "有效卡数", Kind: export.KindText, InTXT: true},
			{Header: "剩余总次数", Kind: export.KindText, InTXT: true},
			{Header: "备注", Kind: export.KindText, InTXT: true},
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

func (p *Provider) exportRows(ctx context.Context, in memberExportIn) ([][]any, error) {
	// 主表：筛选语义与 List 一致（repo.go List），DATETIME 一律按库内墙上字符串原样导出
	where := "1=1"
	args := []any{}
	if kw := strings.TrimSpace(in.Keyword); kw != "" {
		where += " AND (name LIKE ? OR phone LIKE ?)"
		like := "%" + kw + "%"
		args = append(args, like, like)
	}
	if in.TagID != "" {
		where += " AND EXISTS (SELECT 1 FROM member_tag_rel r WHERE r.member_id = member.id AND r.tag_id = ?)"
		args = append(args, in.TagID)
	}
	if in.CardType != "" {
		where += " AND EXISTS (SELECT 1 FROM member_card mc JOIN card_template ct ON ct.id = mc.card_template_id WHERE mc.member_id = member.id AND ct.type = ?)"
		args = append(args, in.CardType)
	}
	type row struct {
		id, memberNo, name, phone, gender, birthday string
		wxBound                                     bool
		remark, lastVisit, createdAt                string
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, member_no, name, COALESCE(phone,''), gender, COALESCE(birthday,''),
		        (wx_openid IS NOT NULL AND wx_openid <> ''), COALESCE(remark,''),
		        COALESCE(last_visit_at,''), created_at
		 FROM member WHERE `+where+` ORDER BY member_no`, args...)
	if err != nil {
		return nil, shared.Server("EXPORT_MEMBER_Q", err)
	}
	defer rows.Close()
	var ms []row
	ids := []string{}
	for rows.Next() {
		var m row
		if err := rows.Scan(&m.id, &m.memberNo, &m.name, &m.phone, &m.gender, &m.birthday,
			&m.wxBound, &m.remark, &m.lastVisit, &m.createdAt); err != nil {
			return nil, shared.Server("EXPORT_MEMBER_S", err)
		}
		ms = append(ms, m)
		ids = append(ids, m.id)
	}
	if err := rows.Err(); err != nil {
		return nil, shared.Server("EXPORT_MEMBER_E", err)
	}

	// 聚合 1：累计消费/消费次数 = VALID payment（散客 NULL member 行天然不入组）
	paySum := map[string]int64{}
	payCnt := map[string]int{}
	payRows, err := p.db.QueryContext(ctx,
		`SELECT member_id, SUM(amount), COUNT(*) FROM payment
		 WHERE status = 'VALID' AND member_id IS NOT NULL GROUP BY member_id`)
	if err != nil {
		return nil, shared.Server("EXPORT_MEMBER_PAY", err)
	}
	defer payRows.Close()
	for payRows.Next() {
		var id string
		var sum, cnt int64
		if err := payRows.Scan(&id, &sum, &cnt); err != nil {
			return nil, shared.Server("EXPORT_MEMBER_PAYS", err)
		}
		paySum[id], payCnt[id] = sum, int(cnt)
	}
	if err := payRows.Err(); err != nil {
		return nil, shared.Server("EXPORT_MEMBER_PAYE", err)
	}

	// 聚合 2：有效卡数/剩余总次数（仅 ACTIVE）
	cardCnt := map[string]int{}
	cardRem := map[string]int{}
	cardRows, err := p.db.QueryContext(ctx,
		`SELECT member_id, COUNT(*), COALESCE(SUM(remaining_count),0) FROM member_card
		 WHERE status = 'ACTIVE' GROUP BY member_id`)
	if err != nil {
		return nil, shared.Server("EXPORT_MEMBER_CARD", err)
	}
	defer cardRows.Close()
	for cardRows.Next() {
		var id string
		var cnt, rem int
		if err := cardRows.Scan(&id, &cnt, &rem); err != nil {
			return nil, shared.Server("EXPORT_MEMBER_CARDS", err)
		}
		cardCnt[id], cardRem[id] = cnt, rem
	}
	if err := cardRows.Err(); err != nil {
		return nil, shared.Server("EXPORT_MEMBER_CARDE", err)
	}

	// 标签：批量一条 JOIN（禁逐会员 N+1，先例 decorateTags）
	tagMap := map[string][]string{}
	if len(ids) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		qargs := make([]any, len(ids))
		for i, id := range ids {
			qargs[i] = id
		}
		tagRows, err := p.db.QueryContext(ctx,
			`SELECT r.member_id, t.name FROM member_tag_rel r JOIN member_tag t ON t.id = r.tag_id
			 WHERE r.member_id IN (`+ph+`) ORDER BY t.name`, qargs...)
		if err != nil {
			return nil, shared.Server("EXPORT_MEMBER_TAG", err)
		}
		defer tagRows.Close()
		for tagRows.Next() {
			var id, name string
			if err := tagRows.Scan(&id, &name); err != nil {
				return nil, shared.Server("EXPORT_MEMBER_TAGS", err)
			}
			tagMap[id] = append(tagMap[id], name)
		}
		if err := tagRows.Err(); err != nil {
			return nil, shared.Server("EXPORT_MEMBER_TAGE", err)
		}
	}

	out := make([][]any, len(ms))
	for i, m := range ms {
		wx := "否"
		if m.wxBound {
			wx = "是"
		}
		out[i] = []any{
			m.memberNo, m.name, m.phone, m.gender, m.birthday, wx,
			strings.Join(tagMap[m.id], "、"), m.createdAt, m.lastVisit,
			paySum[m.id], strconv.Itoa(payCnt[m.id]), strconv.Itoa(cardCnt[m.id]), strconv.Itoa(cardRem[m.id]),
			m.remark,
		}
	}
	return out, nil
}


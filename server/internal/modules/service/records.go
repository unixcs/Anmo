package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"anmo/server/internal/shared"
)

// records.go — 服务记录（D28，plan §四/§六）。核销与结算事务内落库（D6：调用方
// transaction 模块开事务传入 Tx）；历史存名称快照（body_parts/service_method 均为
// 标签名快照，不受标签后续增删/停用影响）；撤销只置 REVERSED 不删行。

// ServiceRecord — service_record 的 JSON 投影（追踪列表 / 散客结算响应共用）。
type ServiceRecord struct {
	ID            string   `json:"id"`
	AppointmentID *string  `json:"appointment_id"`
	PaymentID     string   `json:"payment_id"`
	RedemptionID  *string  `json:"redemption_id"`
	ServiceID     string   `json:"service_id"`
	ServiceName   string   `json:"service_name"`
	BodyParts     []string `json:"body_parts"`
	ServiceMethod string   `json:"service_method"`
	TechNote      string   `json:"tech_note"`
	MerchantNote  string   `json:"merchant_note"`
	Communicated  bool     `json:"communicated"`
	Status        string   `json:"status"` // ACTIVE | REVERSED
	CreatedAt     string   `json:"created_at"`
	ReversedAt    *string  `json:"reversed_at"`

	// 装饰字段（追踪列表 / 商家备注回显）
	MerchantNoteByName string  `json:"merchant_note_by_name,omitempty"`
	MerchantNoteAt     *string `json:"merchant_note_at,omitempty"`
	MemberID           *string `json:"member_id,omitempty"`
	// payment 装饰（追踪列表金额/方式）
	PaymentMethod string `json:"payment_method,omitempty"`
	PaymentAmount int64  `json:"payment_amount,omitempty"`
	PaymentStatus string `json:"payment_status,omitempty"`
}

// RecordInput — 结算事务内的记录入参（D6 跨模块入口）。
type RecordInput struct {
	MemberID      string   // "" = NULL（防御；散客无手机号不落记录，见 D29）
	AppointmentID string   // "" = NULL
	PaymentID     string   // 必填：每条记录对应一笔 VALID payment
	RedemptionID  string   // "" = NULL（现金/微信/散客）
	ServiceID     string
	ServiceName   string // 快照，与 redemption/payment 同源
	BodyParts     []string
	Method        string
	TechNote      string
	Communicated  bool
	OperatorID    string
}

// insertRecordTx 校验阈值（design §2.1）。
const (
	maxRecordParts   = 3
	maxPartRunes     = 32
	maxMethodRunes   = 64
	maxTechNoteRunes = 200
)

// InsertRecordTx — 在调用方事务内落一条服务记录（payment/redemption 落库之后调用）。
// 校验：部位 ≤3 且每项 1..32 字、方式 ≤64、技师备注 ≤200；communicated=false 兜底拒绝
// （transaction 层已提前校验 TX_NEED_CONFIRM）。
func (p *Provider) InsertRecordTx(ctx context.Context, tx shared.Tx, in RecordInput) error {
	if !in.Communicated {
		return shared.BadRequest("SERVICE_NEED_CONFIRM", "请先勾选「服务前已完成沟通」")
	}
	if strings.TrimSpace(in.PaymentID) == "" || strings.TrimSpace(in.ServiceID) == "" || strings.TrimSpace(in.ServiceName) == "" {
		return shared.BadRequest("REC_BAD_INPUT", "服务记录缺少收款/服务信息")
	}
	parts := make([]string, 0, len(in.BodyParts))
	for _, raw := range in.BodyParts {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		n := runeLen(s)
		if n < 1 || n > maxPartRunes {
			return shared.BadRequest("REC_BAD_PARTS", "调理部位名称需 1~32 个字")
		}
		parts = append(parts, s)
	}
	if len(parts) > maxRecordParts {
		return shared.BadRequest("REC_BAD_PARTS", "调理部位最多选择 3 项")
	}
	method := strings.TrimSpace(in.Method)
	if runeLen(method) > maxMethodRunes {
		return shared.BadRequest("REC_BAD_METHOD", "服务方式过长")
	}
	techNote := strings.TrimSpace(in.TechNote)
	if runeLen(techNote) > maxTechNoteRunes {
		return shared.BadRequest("REC_BAD_TECH_NOTE", "技师备注不能超过 200 字")
	}
	partsJSON, err := json.Marshal(parts)
	if err != nil {
		return shared.Server("REC_ENCODE", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO service_record (id, member_id, appointment_id, payment_id, redemption_id,
		 service_id, service_name, body_parts, service_method, tech_note, communicated, created_by)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		shared.NewID(), nullIfEmpty(in.MemberID), nullIfEmpty(in.AppointmentID), in.PaymentID,
		nullIfEmpty(in.RedemptionID), in.ServiceID, in.ServiceName, string(partsJSON), method,
		techNote, boolInt(in.Communicated), nullIfEmpty(in.OperatorID)); err != nil {
		return shared.Server("REC_INSERT", err)
	}
	return nil
}

// ReverseByRedemptionTx — 核销撤销联动（§四.5）：该核销下的 ACTIVE 记录同事务置
// REVERSED。幂等：无行/已 REVERSED 均不报错。
func (p *Provider) ReverseByRedemptionTx(ctx context.Context, tx shared.Tx, redemptionID string) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE service_record SET status = 'REVERSED', reversed_at = ?
		  WHERE redemption_id = ? AND status = 'ACTIVE'`,
		shared.NowShanghai(), redemptionID); err != nil {
		return shared.Server("REC_REVERSE", err)
	}
	return nil
}

// RevokeRecord — 散客记录独立撤销（非卡，D28）：单事务 record → REVERSED +
// 原 payment → VOIDED。卡核销记录（redemption_id 非空）必须走核销撤销入口以还次数。
func (p *Provider) RevokeRecord(ctx context.Context, recordID, operatorID string) error {
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var paymentID, status string
		var redemptionID sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT payment_id, redemption_id, status FROM service_record WHERE id = ?`, recordID).
			Scan(&paymentID, &redemptionID, &status)
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("REC_NOT_FOUND", "服务记录不存在")
		}
		if err != nil {
			return shared.Server("REC_QUERY", err)
		}
		if redemptionID.Valid && redemptionID.String != "" {
			return shared.BadRequest("REC_CARD_REVERSAL", "卡核销记录请通过核销撤销入口撤销，以恢复卡次数")
		}
		if status != "ACTIVE" {
			return shared.Conflict("REC_ALREADY_REVERSED", "该服务记录已撤销")
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE payment SET status = 'VOIDED', remark = remark || '；记录撤销'
			  WHERE id = ? AND status = 'VALID'`, paymentID)
		if err != nil {
			return shared.Server("PAY_VOID", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return shared.Conflict("PAY_ALREADY_VOIDED", "对应收款已作废，记录可能已撤销")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE service_record SET status = 'REVERSED', reversed_at = ?
			  WHERE id = ? AND status = 'ACTIVE'`, shared.NowShanghai(), recordID); err != nil {
			return shared.Server("REC_REVERSE", err)
		}
		return nil
	})
}

// RecordByPaymentTx — 按 payment 定位记录（散客结算幂等回放 / 响应装配用）；
// 无记录（仅记账）返回 (nil, nil)。
func (p *Provider) RecordByPaymentTx(ctx context.Context, tx shared.Tx, paymentID string) (*ServiceRecord, error) {
	rows, err := tx.QueryContext(ctx, recordTrackQuery+` WHERE r.payment_id = ? ORDER BY r.created_at DESC, r.rowid DESC`, paymentID)
	if err != nil {
		return nil, shared.Server("REC_QUERY", err)
	}
	defer rows.Close()
	for rows.Next() {
		rec, err := scanTrackRecord(rows)
		if err != nil {
			return nil, shared.Server("REC_SCAN", err)
		}
		return rec, nil
	}
	return nil, rows.Err()
}

// --- 商家备注 ---

// UpdateMerchantNote — 追加/修改商家备注（独立于技师备注），记录操作人与时间。
func (p *Provider) UpdateMerchantNote(ctx context.Context, recordID, note, operatorID string) (*ServiceRecord, error) {
	note = strings.TrimSpace(note)
	if len([]rune(note)) > 500 {
		return nil, shared.BadRequest("REC_BAD_NOTE", "商家备注不能超过 500 字")
	}
	res, err := p.db.ExecContext(ctx,
		`UPDATE service_record SET merchant_note = ?, merchant_note_by = ?, merchant_note_at = ?
		  WHERE id = ?`, note, nullIfEmpty(operatorID), shared.NowShanghai(), recordID)
	if err != nil {
		return nil, shared.Server("REC_NOTE_UPDATE", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, shared.NotFound("REC_NOT_FOUND", "服务记录不存在")
	}
	return p.GetRecord(ctx, recordID)
}

// GetRecord — 单条记录（商家备注/撤销回显用）。
func (p *Provider) GetRecord(ctx context.Context, recordID string) (*ServiceRecord, error) {
	rows, err := p.db.QueryContext(ctx, recordTrackQuery+` WHERE r.id = ?`, recordID)
	if err != nil {
		return nil, shared.Server("REC_QUERY", err)
	}
	defer rows.Close()
	if rows.Next() {
		return scanTrackRecord(rows)
	}
	return nil, shared.NotFound("REC_NOT_FOUND", "服务记录不存在")
}

// --- 服务追踪（§六） ---

// TrackSummary — 汇总（全库排除 REVERSED）。
type TrackSummary struct {
	TotalAll int           `json:"total_all"`
	Total3m  int           `json:"total_3m"`
	Parts    []NameCount   `json:"parts"`
	Methods  []NameCount   `json:"methods"`
}

// NameCount — 汇合计数项。
type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// TrackFilters — 历史筛选器数据源（§五：停用标签在历史筛选仍可见）。
type TrackFilters struct {
	Parts   []string `json:"parts"`
	Methods []string `json:"methods"`
}

const recordTrackQuery = `SELECT r.id, r.appointment_id, r.payment_id, r.redemption_id,
 r.service_id, r.service_name, r.body_parts, r.service_method, r.tech_note, r.merchant_note,
 r.communicated, r.status, r.created_at, r.reversed_at,
 u.name, r.merchant_note_at, r.member_id, p.method, p.amount, p.status
 FROM service_record r
 LEFT JOIN identity_user u ON u.id = r.merchant_note_by
 LEFT JOIN payment p ON p.id = r.payment_id`

func scanTrackRecord(row interface{ Scan(...any) error }) (*ServiceRecord, error) {
	r := &ServiceRecord{}
	var aptID, rdmID, memberID, reversedAt, noteByName, noteAt sql.NullString
	var partsJSON string
	var communicated int
	var payMethod sql.NullString
	var payAmount sql.NullInt64
	var payStatus sql.NullString
	if err := row.Scan(&r.ID, &aptID, &r.PaymentID, &rdmID, &r.ServiceID, &r.ServiceName,
		&partsJSON, &r.ServiceMethod, &r.TechNote, &r.MerchantNote, &communicated, &r.Status,
		&r.CreatedAt, &reversedAt, &noteByName, &noteAt, &memberID, &payMethod, &payAmount, &payStatus); err != nil {
		return nil, err
	}
	r.AppointmentID = nullStrPtr(aptID)
	r.RedemptionID = nullStrPtr(rdmID)
	r.MemberID = nullStrPtr(memberID)
	r.ReversedAt = nullStrPtr(reversedAt)
	r.MerchantNoteAt = nullStrPtr(noteAt)
	r.MerchantNoteByName = noteByName.String
	r.Communicated = communicated == 1
	r.BodyParts = []string{}
	if err := json.Unmarshal([]byte(partsJSON), &r.BodyParts); err != nil {
		r.BodyParts = []string{}
	}
	r.PaymentMethod = payMethod.String
	r.PaymentAmount = payAmount.Int64
	r.PaymentStatus = payStatus.String
	return r, nil
}

// ListMemberRecords — 服务追踪（§六）：倒序 + 部位/方式/时间筛选 + 关键词搜备注。
// REVERSED 不出现在追踪流；range ∈ {"",1m,3m}；q 命中技师备注/商家备注/服务名。
func (p *Provider) ListMemberRecords(ctx context.Context, memberID, part, method, rng, q string) ([]*ServiceRecord, *TrackSummary, *TrackFilters, error) {
	where := ` WHERE r.member_id = ? AND r.status = 'ACTIVE'`
	args := []any{memberID}
	if part != "" {
		where += ` AND r.body_parts LIKE ?`
		args = append(args, `%"`+part+`"%`)
	}
	if method != "" {
		where += ` AND r.service_method = ?`
		args = append(args, method)
	}
	if since := rangeStart(rng); since != "" {
		where += ` AND r.created_at >= ?`
		args = append(args, since)
	}
	if kw := strings.TrimSpace(q); kw != "" {
		where += ` AND (r.tech_note LIKE ? OR r.merchant_note LIKE ? OR r.service_name LIKE ?)`
		like := `%` + kw + `%`
		args = append(args, like, like, like)
	}
	rows, err := p.db.QueryContext(ctx, recordTrackQuery+where+` ORDER BY r.created_at DESC, r.rowid DESC`, args...)
	if err != nil {
		return nil, nil, nil, shared.Server("REC_LIST", err)
	}
	defer rows.Close()
	var out []*ServiceRecord
	for rows.Next() {
		rec, err := scanTrackRecord(rows)
		if err != nil {
			return nil, nil, nil, shared.Server("REC_SCAN", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, shared.Server("REC_LIST", err)
	}

	summary, err := p.trackSummary(ctx, memberID)
	if err != nil {
		return nil, nil, nil, err
	}
	parts, methods, err := p.AllTagNames(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return out, summary, &TrackFilters{Parts: parts, Methods: methods}, nil
}

// trackSummary — total_all / total_3m / 部位与方式计数（全部时间，排除 REVERSED）。
func (p *Provider) trackSummary(ctx context.Context, memberID string) (*TrackSummary, error) {
	s := &TrackSummary{Parts: []NameCount{}, Methods: []NameCount{}}
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM service_record WHERE member_id = ? AND status = 'ACTIVE'`, memberID).
		Scan(&s.TotalAll); err != nil {
		return nil, shared.Server("REC_SUMMARY", err)
	}
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM service_record WHERE member_id = ? AND status = 'ACTIVE' AND created_at >= ?`,
		memberID, rangeStart("3m")).Scan(&s.Total3m); err != nil {
		return nil, shared.Server("REC_SUMMARY", err)
	}
	// 方式计数
	rows, err := p.db.QueryContext(ctx,
		`SELECT service_method, COUNT(*) FROM service_record
		  WHERE member_id = ? AND status = 'ACTIVE' AND service_method != ''
		  GROUP BY service_method ORDER BY COUNT(*) DESC, service_method ASC LIMIT 20`, memberID)
	if err != nil {
		return nil, shared.Server("REC_SUMMARY", err)
	}
	defer rows.Close()
	for rows.Next() {
		var nc NameCount
		if err := rows.Scan(&nc.Name, &nc.Count); err != nil {
			return nil, shared.Server("REC_SUMMARY", err)
		}
		s.Methods = append(s.Methods, nc)
	}
	// 部位计数：body_parts 为 JSON 数组，取出行内展开（记录量级小，Go 侧聚合）
	rows2, err := p.db.QueryContext(ctx,
		`SELECT body_parts FROM service_record WHERE member_id = ? AND status = 'ACTIVE' AND body_parts != '[]'`, memberID)
	if err != nil {
		return nil, shared.Server("REC_SUMMARY", err)
	}
	defer rows2.Close()
	counts := map[string]int{}
	for rows2.Next() {
		var raw string
		if err := rows2.Scan(&raw); err != nil {
			return nil, shared.Server("REC_SUMMARY", err)
		}
		var names []string
		if json.Unmarshal([]byte(raw), &names) != nil {
			continue
		}
		for _, n := range names {
			counts[n]++
		}
	}
	for name, count := range counts {
		s.Parts = append(s.Parts, NameCount{Name: name, Count: count})
	}
	sortNameCounts(s.Parts)
	if len(s.Parts) > 20 {
		s.Parts = s.Parts[:20]
	}
	return s, nil
}

// AllTagNames — 全量标签名（含 DISABLED，§五 历史筛选仍可见），按 sort, created_at。
func (p *Provider) AllTagNames(ctx context.Context) (parts, methods []string, err error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT tag_group, name FROM service_tag ORDER BY sort, created_at`)
	if err != nil {
		return nil, nil, shared.Server("TAG_LIST", err)
	}
	defer rows.Close()
	for rows.Next() {
		var g, name string
		if err := rows.Scan(&g, &name); err != nil {
			return nil, nil, shared.Server("TAG_SCAN", err)
		}
		if g == GroupBodyPart {
			parts = append(parts, name)
		} else {
			methods = append(methods, name)
		}
	}
	return parts, methods, rows.Err()
}

// rangeStart — 近1月/近3月起点（Asia/Shanghai 墙上时间字符串，字典序=时间序）。
func rangeStart(rng string) string {
	switch rng {
	case "1m":
		return shared.NowShanghai().AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	case "3m":
		return shared.NowShanghai().AddDate(0, -3, 0).Format("2006-01-02 15:04:05")
	default:
		return ""
	}
}

func sortNameCounts(list []NameCount) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0; j-- {
			a, b := list[j-1], list[j]
			if a.Count < b.Count || (a.Count == b.Count && a.Name > b.Name) {
				list[j-1], list[j] = list[j], list[j-1]
			} else {
				break
			}
		}
	}
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullStrPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// runeLen — 字符数（rune），用于中文友好的长度上限校验。
func runeLen(s string) int { return len([]rune(s)) }

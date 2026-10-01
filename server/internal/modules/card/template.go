package card

import (
	"context"
	"database/sql"
	"errors"
	"time"


	"anmo/server/internal/shared"
)

// Template — card_template row (§35). type carries no behavior (D11).
type Template struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"` // COUNT | ACTIVITY
	TotalCount   int      `json:"total_count"`
	ValidityType string   `json:"validity_type"` // PERMANENT | FIXED
	ValidFrom    *string  `json:"valid_from"`
	ValidUntil   *string  `json:"valid_until"`
	PriceCents   int64    `json:"price"`
	Status       string   `json:"status"`
	ServiceIDs   []string `json:"service_ids,omitempty"` // 可核销服务（D4，列表回显）
}

type NewTemplate struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	TotalCount   int     `json:"total_count"`
	ValidityType string  `json:"validity_type"`
	ValidFrom    *string `json:"valid_from"`
	ValidUntil   *string `json:"valid_until"`
	PriceCents   int64   `json:"price"`
}

func (p *Provider) CreateTemplate(ctx context.Context, in NewTemplate) (*Template, error) {
	if in.Name == "" {
		return nil, shared.BadRequest("CARD_TEMPLATE_NO_NAME", "模板名称必填")
	}
	if in.Type != "COUNT" && in.Type != "ACTIVITY" {
		in.Type = "COUNT"
	}
	if in.TotalCount <= 0 {
		return nil, shared.BadRequest("CARD_TEMPLATE_BAD_COUNT", "次数必须大于 0")
	}
	if in.ValidityType != "PERMANENT" && in.ValidityType != "FIXED" {
		in.ValidityType = "PERMANENT"
	}
	if in.ValidityType == "FIXED" && (in.ValidFrom == nil || in.ValidUntil == nil) {
		return nil, shared.BadRequest("CARD_TEMPLATE_BAD_VALIDITY", "固定有效期模板必须提供起止日期")
	}
	t := &Template{
		ID: shared.NewID(), Name: in.Name, Type: in.Type, TotalCount: in.TotalCount,
		ValidityType: in.ValidityType, ValidFrom: in.ValidFrom, ValidUntil: in.ValidUntil,
		PriceCents: in.PriceCents, Status: "ACTIVE",
	}
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO card_template (id, name, type, total_count, validity_type, valid_from, valid_until, price)
		 VALUES (?,?,?,?,?,?,?,?)`,
		t.ID, t.Name, t.Type, t.TotalCount, t.ValidityType, t.ValidFrom, t.ValidUntil, t.PriceCents)
	if err != nil {
		if shared.IsDupKey(err) {
			return nil, shared.Conflict("CARD_TEMPLATE_NAME_EXISTS", "模板名称已存在")
		}
		return nil, shared.Server("CARD_TEMPLATE_INSERT", err)
	}
	return t, nil
}

func (p *Provider) ListTemplates(ctx context.Context) ([]*Template, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, name, type, total_count, validity_type, valid_from, valid_until, price, status
		 FROM card_template ORDER BY created_at DESC`)
	if err != nil {
		return nil, shared.Server("CARD_TEMPLATE_LIST", err)
	}
	defer rows.Close()
	var out []*Template
	for rows.Next() {
		t := &Template{}
		var vf, vu sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.TotalCount, &t.ValidityType, &vf, &vu, &t.PriceCents, &t.Status); err != nil {
			return nil, shared.Server("CARD_TEMPLATE_SCAN", err)
		}
		if vf.Valid {
			t.ValidFrom = strPtr(vf.String)
		}
		if vu.Valid {
			t.ValidUntil = strPtr(vu.String)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, shared.Server("CARD_TEMPLATE_SCAN", err)
	}
	if len(out) > 0 {
		ruleRows, err := p.db.QueryContext(ctx,
			`SELECT card_template_id, service_id FROM card_service_rule ORDER BY created_at`)
		if err != nil {
			return nil, shared.Server("CARD_TEMPLATE_LIST", err)
		}
		defer ruleRows.Close()
		rules := make(map[string][]string, len(out))
		for ruleRows.Next() {
			var tid, sid string
			if err := ruleRows.Scan(&tid, &sid); err != nil {
				return nil, shared.Server("CARD_TEMPLATE_SCAN", err)
			}
			rules[tid] = append(rules[tid], sid)
		}
		if err := ruleRows.Err(); err != nil {
			return nil, shared.Server("CARD_TEMPLATE_SCAN", err)
		}
		for _, t := range out {
			t.ServiceIDs = rules[t.ID]
		}
	}
	return out, nil
}

func (p *Provider) SetTemplateStatus(ctx context.Context, id, status string) error {
	if status != "ACTIVE" && status != "INACTIVE" {
		return shared.BadRequest("CARD_TEMPLATE_BAD_STATUS", "状态仅支持 ACTIVE/INACTIVE")
	}
	_, err := p.db.ExecContext(ctx, `UPDATE card_template SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return shared.Server("CARD_TEMPLATE_UPDATE", err)
	}
	return nil
}

// SetServiceRules replaces the template's redeemable service list (D4).
func (p *Provider) SetServiceRules(ctx context.Context, templateID string, serviceIDs []string) error {
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM card_service_rule WHERE card_template_id = ?`, templateID); err != nil {
			return shared.Server("CARD_RULE_CLEAR", err)
		}
		for _, sid := range serviceIDs {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO card_service_rule (id, card_template_id, service_id) VALUES (?,?,?)`,
				shared.NewID(), templateID, sid); err != nil {
				return shared.Server("CARD_RULE_SET", err)
			}
		}
		return nil
	})
}

func strPtr(s string) *string { return &s }

// MemberCard — member_card row.
type MemberCard struct {
	ID             string    `json:"id"`
	MemberID       string    `json:"member_id"`
	TemplateID     string    `json:"card_template_id"`
	TotalCount     int       `json:"total_count"`
	RemainingCount int       `json:"remaining_count"`
	ValidFrom      string    `json:"valid_from"`
	ValidUntil     *string   `json:"valid_until"`
	Status         string    `json:"status"`
	IssuedAt       time.Time `json:"issued_at"`

	// CardName 来自 card_template.name（动态，不落 member_card 表，§15）。
	CardName string `json:"card_name"`
}

const cardColumns = `id, member_id, card_template_id, total_count, remaining_count, valid_from, valid_until, status, issued_at`

func scanCard(row interface{ Scan(...any) error }) (*MemberCard, error) {
	c := &MemberCard{}
	var vu sql.NullString
	var issued sql.NullTime
	err := row.Scan(&c.ID, &c.MemberID, &c.TemplateID, &c.TotalCount, &c.RemainingCount,
		&c.ValidFrom, &vu, &c.Status, &issued)
	if err != nil {
		return nil, err
	}
	if vu.Valid {
		c.ValidUntil = strPtr(vu.String)
	}
	if issued.Valid {
		c.IssuedAt = issued.Time
	}
	c.ValidFrom = shortDate(c.ValidFrom)
	if c.ValidUntil != nil {
		v := shortDate(*c.ValidUntil)
		c.ValidUntil = &v
	}
	return c, nil
}

// shortDate 把驱动扫成 time.Time 的 DATE 值（convertAssign 产出 RFC3339）
// 归一为 YYYY-MM-DD，杜绝 "T00:00:00+08:00" 泄漏到前端（§16）。
func shortDate(s string) string {
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		return s[:10]
	}
	return s
}

// IssueCard creates a member_card plus its ISSUE transaction in ONE database
// transaction (§127). Re-newal (D10) is simply issuing another card.
func (p *Provider) IssueCard(ctx context.Context, memberID, templateID, operatorID string) (*MemberCard, error) {
	var out *MemberCard
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		out, e = p.IssueCardTx(ctx, tx, memberID, templateID, operatorID)
		return e
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IssueCardTx is the tx-joining variant (D6) used by other modules/tests.
func (p *Provider) IssueCardTx(ctx context.Context, tx shared.Tx, memberID, templateID, operatorID string) (*MemberCard, error) {
	var total int
	var vf, vu sql.NullString
	err := tx.QueryRowContext(ctx,
		`SELECT total_count, valid_from, valid_until FROM card_template WHERE id = ? AND status = 'ACTIVE'`,
		templateID).Scan(&total, &vf, &vu)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("CARD_TEMPLATE_NOT_FOUND", "卡模板不存在或未上架")
	}
	if err != nil {
		return nil, shared.Server("CARD_TEMPLATE_QUERY", err)
	}
	// F8: 模板有效期已结束的卡不允许再发——发出来的卡 ACTIVE 却立即过期，
	// 顾客端展示与核销校验直接打架。
	if vu.Valid {
		today := shared.NowShanghai().Format("2006-01-02")
		if vu.String < today {
			return nil, shared.Conflict("CARD_TEMPLATE_EXPIRED", "该卡模板有效期已结束，无法发卡")
		}
	}

	cardID := shared.NewID()
	validFrom := shared.NowShanghai().Format("2006-01-02")
	if vf.Valid {
		validFrom = vf.String
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO member_card (id, member_id, card_template_id, total_count, remaining_count, valid_from, valid_until, status, issued_by)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		cardID, memberID, templateID, total, total, validFrom, vu, "ACTIVE", operatorID); err != nil {
		return nil, shared.Server("CARD_INSERT", err)
	}
	if err := p.writeCardTx(ctx, tx, cardID, memberID, "ISSUE", total, 0, total, "", operatorID, "发卡"); err != nil {
		return nil, err
	}
	c, err := scanCard(tx.QueryRowContext(ctx, `SELECT `+cardColumns+` FROM member_card WHERE id = ?`, cardID))
	if err != nil {
		return nil, shared.Server("CARD_QUERY", err)
	}
	return c, nil
}

// writeCardTx appends a card_transaction row (the balance history, §125).
// card_name snapshot comes from the template at write time (§15: 改模板名不改历史).
func (p *Provider) writeCardTx(ctx context.Context, tx shared.Tx, cardID, memberID, typ string, quantity, before, after int, refType, operatorID, remark string) error {
	var cardName string
	_ = tx.QueryRowContext(ctx,
		`SELECT t.name FROM member_card c JOIN card_template t ON t.id = c.card_template_id WHERE c.id = ?`,
		cardID).Scan(&cardName)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO card_transaction (id, member_card_id, member_id, type, quantity, before_count, after_count, reference_type, remark, operator_id, card_name)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		shared.NewID(), cardID, memberID, typ, quantity, before, after, refType, remark, operatorID, cardName); err != nil {
		return shared.Server("CARD_TXN_WRITE", err)
	}
	return nil
}

// CardTransaction — card_transaction row.
type CardTransaction struct {
	ID        string    `json:"id"`
	CardID    string    `json:"member_card_id"`
	CardName  string    `json:"card_name"`
	Type      string    `json:"type"`
	Quantity  int       `json:"quantity"`
	Before    int       `json:"before_count"`
	After     int       `json:"after_count"`
	RefType   string    `json:"reference_type"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

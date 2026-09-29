package member

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"anmo/server/internal/shared"
)

// Model — member row shape (mirrors 014_member_password.sql; 002 原表 014 重建).
type Member struct {
	ID          string     `json:"id"`
	MemberNo    string     `json:"member_no"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"` // 014 起可空："" = 未设置（纯微信会员）
	Gender      string     `json:"gender"`
	Birthday    *string    `json:"birthday"` // YYYY-MM-DD
	Status      string     `json:"status"`
	Remark      string     `json:"remark"`
	LastVisitAt *time.Time `json:"last_visit_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// password_hash 永不出现在任何 JSON 响应/日志（json:"-"）；HasPassword/WxBound
	// 为派生布尔，供 admin 列表/详情与顾客端判断使用。
	PasswordHash string `json:"-"`
	HasPassword  bool   `json:"has_password"`
	WxBound      bool   `json:"wx_bound"`
	// Tags 仅列表/详情展示用（§32 组合搜索的展示面），不落 member 表
	Tags []string `json:"tags,omitempty"`
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// 014 重建后列序变化（password_hash 插在 remark 段之后）：全代码禁 SELECT *，
// 一律按名列取（memberColumns 常量为唯一列清单）。
const memberColumns = `id, member_no, name, phone, gender, birthday, status, remark, last_visit_at, password_hash, wx_openid, created_at, updated_at`

func scanMember(row interface{ Scan(...any) error }) (*Member, error) {
	m := &Member{}
	var birthday sql.NullTime
	var lastVisit sql.NullTime
	var phone, passwordHash, wxOpenid sql.NullString
	err := row.Scan(&m.ID, &m.MemberNo, &m.Name, &phone, &m.Gender, &birthday,
		&m.Status, &m.Remark, &lastVisit, &passwordHash, &wxOpenid, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if phone.Valid {
		m.Phone = phone.String
	}
	if birthday.Valid {
		b := birthday.Time.Format("2006-01-02")
		m.Birthday = &b
	}
	if lastVisit.Valid {
		m.LastVisitAt = &lastVisit.Time
	}
	if passwordHash.Valid {
		m.PasswordHash = passwordHash.String
		m.HasPassword = passwordHash.String != ""
	}
	m.WxBound = wxOpenid.Valid && wxOpenid.String != ""
	return m, nil
}

// Create inserts a member with a generated member_no; phone must be unique.
func (p *Provider) Create(ctx context.Context, in NewMember) (*Member, error) {
	var out *Member
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		phone := strings.TrimSpace(in.Phone)
		if len(phone) != 11 {
			return shared.BadRequest("MEMBER_BAD_PHONE", "手机号格式不正确")
		}
		var exists string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM member WHERE phone = ?`, phone).Scan(&exists); err == nil {
			return shared.Conflict("MEMBER_PHONE_EXISTS", "该手机号已存在会员")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return shared.Server("MEMBER_QUERY", err)
		}
		no, err := p.nextMemberNo(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO member (id, member_no, name, phone, gender, birthday, remark) VALUES (?,?,?,?,?,?,?)`,
			shared.NewID(), no, strings.TrimSpace(in.Name), phone, in.Gender, in.Birthday, in.Remark); err != nil {
			return shared.Server("MEMBER_INSERT", err)
		}
		row := tx.QueryRowContext(ctx, `SELECT `+memberColumns+` FROM member WHERE phone = ?`, phone)
		out, err = scanMember(row)
		if err != nil {
			return shared.Server("MEMBER_QUERY", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type NewMember struct {
	Name     string  `json:"name"`
	Phone    string  `json:"phone"`
	Gender   string  `json:"gender"`
	Birthday *string `json:"birthday"`
	Remark   string  `json:"remark"`
}

// Get returns one member.
func (p *Provider) Get(ctx context.Context, id string) (*Member, error) {
	m, err := scanMember(p.db.QueryRowContext(ctx,
		`SELECT `+memberColumns+` FROM member WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
	}
	if err != nil {
		return nil, shared.Server("MEMBER_QUERY", err)
	}
	return m, nil
}

// GetByPhone returns one member by phone（未注册 → MEMBER_NOT_FOUND）。
// q 可传连接池（登录）也可传事务（注册/认领需 immediate 事务内查建）。
func (p *Provider) GetByPhone(ctx context.Context, q shared.Querier, phone string) (*Member, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil, shared.BadRequest("MEMBER_BAD_PHONE", "手机号格式不正确")
	}
	m, err := scanMember(q.QueryRowContext(ctx,
		`SELECT `+memberColumns+` FROM member WHERE phone = ?`, phone))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
	}
	if err != nil {
		return nil, shared.Server("MEMBER_QUERY", err)
	}
	return m, nil
}

// ListParams filters the admin member list (goal §32: keyword + tag + card
// type combinable).
type ListParams struct {
	Keyword  string // matches name or phone
	TagID    string // member carries this tag
	CardType string // member holds a card of this template type (COUNT/ACTIVITY)
	Page     shared.PageParams
}

// List returns a page of members.
func (p *Provider) List(ctx context.Context, q ListParams) ([]*Member, int64, error) {
	where := "1=1"
	args := []any{}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		where += " AND (name LIKE ? OR phone LIKE ?)"
		like := "%" + kw + "%"
		args = append(args, like, like)
	}
	if q.TagID != "" {
		where += " AND EXISTS (SELECT 1 FROM member_tag_rel r WHERE r.member_id = member.id AND r.tag_id = ?)"
		args = append(args, q.TagID)
	}
	if q.CardType != "" {
		where += " AND EXISTS (SELECT 1 FROM member_card mc JOIN card_template ct ON ct.id = mc.card_template_id WHERE mc.member_id = member.id AND ct.type = ?)"
		args = append(args, q.CardType)
	}
	var total int64
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM member WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, shared.Server("MEMBER_COUNT", err)
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+memberColumns+` FROM member WHERE `+where+
			` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		append(args, q.Page.Limit(), q.Page.Offset())...)
	if err != nil {
		return nil, 0, shared.Server("MEMBER_LIST", err)
	}
	defer rows.Close()
	var out []*Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, 0, shared.Server("MEMBER_SCAN", err)
		}
		out = append(out, m)
	}
	if err := p.decorateTags(ctx, out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// decorateTags fills each member's Tags for list display (one query per page).
func (p *Provider) decorateTags(ctx context.Context, members []*Member) error {
	if len(members) == 0 {
		return nil
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(members)), ",")
	args := make([]any, len(members))
	nameByID := map[string][]string{}
	for i, m := range members {
		args[i] = m.ID
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT r.member_id, t.name FROM member_tag_rel r JOIN member_tag t ON t.id = r.tag_id
		 WHERE r.member_id IN (`+ph+`) ORDER BY t.name`, args...)
	if err != nil {
		return shared.Server("TAG_PAGE", err)
	}
	defer rows.Close()
	for rows.Next() {
		var mid, name string
		if err := rows.Scan(&mid, &name); err != nil {
			return shared.Server("TAG_PAGE_SCAN", err)
		}
		nameByID[mid] = append(nameByID[mid], name)
	}
	for _, m := range members {
		m.Tags = nameByID[m.ID]
	}
	return nil
}

// UpdateProfile edits editable fields. Empty pointer = leave unchanged.
// Phone 特殊：仅 SetPhoneOnce 语义（一次性设置，已有手机号不可改，撞号 409）——
// 顾客端完善资料与 admin 端共用本入口；admin UI 不开放改手机号。
type ProfileUpdate struct {
	Name     *string `json:"name"`
	Phone    *string `json:"phone"`
	Gender   *string `json:"gender"`
	Birthday *string `json:"birthday"`
	Remark   *string `json:"remark"`
}

func (p *Provider) UpdateProfile(ctx context.Context, id string, u ProfileUpdate) (*Member, error) {
	if u.Gender != nil && *u.Gender != "" && *u.Gender != "男" && *u.Gender != "女" {
		return nil, shared.BadRequest("MEMBER_BAD_GENDER", "性别仅支持 男/女")
	}
	if u.Birthday != nil && *u.Birthday != "" {
		if _, err := time.Parse("2006-01-02", *u.Birthday); err != nil {
			return nil, shared.BadRequest("MEMBER_BAD_BIRTHDAY", "生日格式应为 YYYY-MM-DD")
		}
	}
	err := shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		// 手机号一次性设置（空串视为未传，沿用"空指针=不变"口径）
		if u.Phone != nil && *u.Phone != "" {
			if err := p.SetPhoneOnce(ctx, tx, id, *u.Phone); err != nil {
				return err
			}
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE member SET
			   name = COALESCE(?, name),
			   gender = COALESCE(?, gender),
			   birthday = COALESCE(?, birthday),
			   remark = COALESCE(?, remark)
			 WHERE id = ?`,
			u.Name, u.Gender, u.Birthday, u.Remark, id)
		if err != nil {
			return shared.Server("MEMBER_UPDATE", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			if _, err := p.Get(ctx, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p.Get(ctx, id)
}

// DormantMembers returns members with no visit in N days (§86 沉睡客户).
// A member who never visited counts once they are older than N days.
// 截止时间在 Go 侧计算（等价 MySQL NOW() - INTERVAL ? DAY，Asia/Shanghai 墙上时间）。
func (p *Provider) DormantMembers(ctx context.Context, days int) ([]*Member, error) {
	cutoff := shared.NowShanghai().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+memberColumns+` FROM member
		 WHERE status = 'ACTIVE'
		   AND (last_visit_at IS NULL AND created_at < ?
		        OR last_visit_at < ?)
		 ORDER BY created_at`, cutoff, cutoff)
	if err != nil {
		return nil, shared.Server("MEMBER_DORMANT", err)
	}
	defer rows.Close()
	var out []*Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, shared.Server("MEMBER_SCAN", err)
		}
		out = append(out, m)
	}
	return out, nil
}

// TouchLastVisit sets last_visit_at (called by transaction module after redeem).
func (p *Provider) TouchLastVisit(ctx context.Context, tx shared.Tx, memberID string, at time.Time) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE member SET last_visit_at = ? WHERE id = ?`, at, memberID); err != nil {
		return shared.Server("MEMBER_TOUCH", err)
	}
	return nil
}

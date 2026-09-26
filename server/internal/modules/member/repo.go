package member

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"anmo/server/internal/shared"
)

// Model — member row shape (mirrors 002_member.sql).
type Member struct {
	ID          string     `json:"id"`
	MemberNo    string     `json:"member_no"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	Gender      string     `json:"gender"`
	Birthday    *string    `json:"birthday"` // YYYY-MM-DD
	Status      string     `json:"status"`
	Remark      string     `json:"remark"`
	LastVisitAt *time.Time `json:"last_visit_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const memberColumns = `id, member_no, name, phone, gender, birthday, status, remark, last_visit_at, created_at, updated_at`

func scanMember(row interface{ Scan(...any) error }) (*Member, error) {
	m := &Member{}
	var birthday sql.NullTime
	var lastVisit sql.NullTime
	err := row.Scan(&m.ID, &m.MemberNo, &m.Name, &m.Phone, &m.Gender, &birthday,
		&m.Status, &m.Remark, &lastVisit, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if birthday.Valid {
		b := birthday.Time.Format("2006-01-02")
		m.Birthday = &b
	}
	if lastVisit.Valid {
		m.LastVisitAt = &lastVisit.Time
	}
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

// ListParams filters the admin member list.
type ListParams struct {
	Keyword string // matches name or phone
	Page    shared.PageParams
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
	return out, total, nil
}

// UpdateProfile edits editable fields. Empty pointer = leave unchanged.
type ProfileUpdate struct {
	Name     *string `json:"name"`
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
	res, err := p.db.ExecContext(ctx,
		`UPDATE member SET
		   name = COALESCE(?, name),
		   gender = COALESCE(?, gender),
		   birthday = COALESCE(?, birthday),
		   remark = COALESCE(?, remark)
		 WHERE id = ?`,
		u.Name, u.Gender, u.Birthday, u.Remark, id)
	if err != nil {
		return nil, shared.Server("MEMBER_UPDATE", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := p.Get(ctx, id); err != nil {
			return nil, err
		}
	}
	return p.Get(ctx, id)
}

// TouchLastVisit sets last_visit_at (called by transaction module after redeem).
func (p *Provider) TouchLastVisit(ctx context.Context, tx shared.Tx, memberID string, at time.Time) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE member SET last_visit_at = ? WHERE id = ?`, at, memberID); err != nil {
		return shared.Server("MEMBER_TOUCH", err)
	}
	return nil
}

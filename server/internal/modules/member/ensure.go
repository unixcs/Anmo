package member

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"anmo/server/internal/shared"
)

// EnsureByPhone returns the member for phone, creating one on first sight
// (plan §10: first login creates/binds the member). Runs inside the caller's
// transaction (D6). The daily member_no sequence is serialized with a named
// lock so concurrent first logins cannot collide.
func (p *Provider) EnsureByPhone(ctx context.Context, tx shared.Tx, phone, name string) (id string, created bool, err error) {
	phone = strings.TrimSpace(phone)
	if len(phone) != 11 {
		return "", false, shared.BadRequest("MEMBER_BAD_PHONE", "手机号格式不正确")
	}
	err = tx.QueryRowContext(ctx, `SELECT id FROM member WHERE phone = ?`, phone).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, shared.Server("MEMBER_QUERY", err)
	}

	// create: serialize member_no generation per day
	no, err := p.nextMemberNo(ctx, tx)
	if err != nil {
		return "", false, err
	}
	id = shared.NewID()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO member (id, member_no, name, phone) VALUES (?, ?, ?, ?)`,
		id, no, name, phone); err != nil {
		return "", false, shared.Server("MEMBER_INSERT", err)
	}
	return id, true, nil
}

// nextMemberNo generates M + yyyymmdd + 4-digit sequence under GET_LOCK.
func (p *Provider) nextMemberNo(ctx context.Context, tx shared.Tx) (string, error) {
	day := shared.NowShanghai().Format("20060102")
	lock := "anmo:member:no:" + day
	res := tx.QueryRowContext(ctx, `SELECT GET_LOCK(?, 5)`, lock)
	var got sql.NullInt64
	if err := res.Scan(&got); err != nil || !got.Valid || got.Int64 != 1 {
		return "", shared.Conflict("MEMBER_NO_LOCK", "会员号生成繁忙，请重试")
	}
	defer tx.ExecContext(context.Background(), `SELECT RELEASE_LOCK(?)`, lock)

	var maxNo sql.NullString
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(member_no) FROM member WHERE member_no LIKE CONCAT('M', ?, '%')`,
		day).Scan(&maxNo); err != nil {
		return "", shared.Server("MEMBER_NO_QUERY", err)
	}
	seq := 1
	if maxNo.Valid && len(maxNo.String) >= 9 {
		if n := atoi4(maxNo.String[len(maxNo.String)-4:]); n > 0 {
			seq = n + 1
		}
	}
	return sprintf("M%s%04d", day, seq), nil
}

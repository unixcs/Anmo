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

// nextMemberNo generates M+yyyymmdd+seq via the atomic counter table.
func (p *Provider) nextMemberNo(ctx context.Context, tx shared.Tx) (string, error) {
	day := shared.NowShanghai().Format("20060102")
	seq, err := shared.NextSeq(ctx, tx, shared.SeqDateName("member", day))
	if err != nil {
		return "", err
	}
	return sprintf("M%s%04d", day, seq), nil
}

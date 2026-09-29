package member

import (
	"context"
	"database/sql"
	"errors"
	"strings"


	"anmo/server/internal/shared"
)

// openid.go — WeChat openid ↔ member binding (V2, plan §11: 微信登录→绑定手机号→
// 同一个 member_id)。H5 与小程序最终落在同一个 member 行上。

// FindByOpenID resolves a member by bound WeChat openid. ok=false when the
// openid has no member yet (caller then issues a bind ticket).
func (p *Provider) FindByOpenID(ctx context.Context, openid string) (id string, ok bool, err error) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return "", false, shared.BadRequest("MEMBER_BAD_OPENID", "openid 为空")
	}
	err = p.db.QueryRowContext(ctx, `SELECT id FROM member WHERE wx_openid = ?`, openid).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return "", false, shared.Server("MEMBER_QUERY", err)
}

// BindOpenID links a WeChat openid to the member inside the caller's
// transaction (D6). Rebinding the same pair is a no-op; the unique key
// uk_member_wx_openid backstops races and maps to WX_OPENID_BOUND.
func (p *Provider) BindOpenID(ctx context.Context, tx shared.Tx, memberID, openid string) error {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return shared.BadRequest("MEMBER_BAD_OPENID", "openid 为空")
	}
	var current sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT wx_openid FROM member WHERE id = ?`, memberID).Scan(&current)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
		}
		return shared.Server("MEMBER_QUERY", err)
	}
	if current.Valid && current.String == openid {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE member SET wx_openid = ? WHERE id = ?`, openid, memberID); err != nil {
		if shared.IsDupKey(err) {
			return shared.Conflict("WX_OPENID_BOUND", "该微信已绑定其他会员")
		}
		return shared.Server("MEMBER_UPDATE", err)
	}
	return nil
}

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
// openid has no member yet. q 可传连接池也可传事务（WxLogin 需在 immediate 事务内
// 查建，避免同 openid 并发首登双双 miss）。
func (p *Provider) FindByOpenID(ctx context.Context, q shared.Querier, openid string) (id string, ok bool, err error) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return "", false, shared.BadRequest("MEMBER_BAD_OPENID", "openid 为空")
	}
	err = q.QueryRowContext(ctx, `SELECT id FROM member WHERE wx_openid = ?`, openid).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return "", false, shared.Server("MEMBER_QUERY", err)
}

// OpenIDOf returns the member's bound openid, ok=false when unbound
//（claim 场景：登录态即 openid 持有者，identity 在事务内读取）。
func (p *Provider) OpenIDOf(ctx context.Context, q shared.Querier, memberID string) (openid string, ok bool, err error) {
	var cur sql.NullString
	err = q.QueryRowContext(ctx, `SELECT wx_openid FROM member WHERE id = ?`, memberID).Scan(&cur)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
		}
		return "", false, shared.Server("MEMBER_QUERY", err)
	}
	if !cur.Valid || cur.String == "" {
		return "", false, nil
	}
	return cur.String, true, nil
}

// ClearOpenID unbinds the member's WeChat openid (claim 事务内先解绑空壳，
// 才能把同一 openid 转绑到老账号——uk_member_wx_openid 不允许两行同值）。
func (p *Provider) ClearOpenID(ctx context.Context, tx shared.Tx, memberID string) error {
	res, err := tx.ExecContext(ctx, `UPDATE member SET wx_openid = NULL WHERE id = ?`, memberID)
	if err != nil {
		return shared.Server("MEMBER_UPDATE", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
	}
	return nil
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

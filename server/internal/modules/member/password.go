package member

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"anmo/server/internal/shared"
)

// password.go — 顾客密码体系（V2.2 第二批，plan §二）。所有写操作都在调用方的
// `_txlock=immediate` 事务内（D5/D6），手机号/openid 唯一性由 uk 约束兜底。
// password_hash 只落库，永不进 JSON 响应/日志（Member.PasswordHash json:"-"）。

// validatePhone — 11 位手机号，与 identity 的 `^1\d{10}$` 同口径
// （EnsureByPhone 的 len==11 是短信时代 dev 捷径，仅限过渡期 dev 登录）。
func validatePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if len(phone) != 11 || phone[0] != '1' {
		return "", shared.BadRequest("MEMBER_BAD_PHONE", "手机号格式不正确")
	}
	for i := 1; i < len(phone); i++ {
		if phone[i] < '0' || phone[i] > '9' {
			return "", shared.BadRequest("MEMBER_BAD_PHONE", "手机号格式不正确")
		}
	}
	return phone, nil
}

// validatePassword — 6~64 位（字符数），同时限制字节数在 bcrypt 72 字节内。
func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < 6 || n > 64 || len(password) > 72 {
		return shared.BadRequest("MEMBER_WEAK_PASSWORD", "密码需 6~64 位")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", shared.Server("MEMBER_HASH", err)
	}
	return string(b), nil
}

// CreateWithPhone — H5 注册建号：手机号+密码（phone NOT NULL 语义，bcrypt 落库）。
// 手机号已存在的分支由 identity.Register 在同一事务内先行判定（三种 409 文案），
// 这里只做 uk 撞键兜底。运行在调用方事务内（D6）。
func (p *Provider) CreateWithPhone(ctx context.Context, tx shared.Tx, phone, password string) (string, error) {
	phone, err := validatePhone(phone)
	if err != nil {
		return "", err
	}
	if err := validatePassword(password); err != nil {
		return "", err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return "", err
	}
	no, err := p.nextMemberNo(ctx, tx)
	if err != nil {
		return "", err
	}
	id := shared.NewID()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO member (id, member_no, name, phone, password_hash) VALUES (?,?,?,?,?)`,
		id, no, "", phone, hash); err != nil {
		if shared.IsDupKey(err) {
			return "", shared.Conflict("MEMBER_PHONE_EXISTS", "该手机号已存在会员")
		}
		return "", shared.Server("MEMBER_INSERT", err)
	}
	return id, nil
}

// SetPassword — 顾客自设/重置 H5 密码（微信身份即凭证，plan §二.5）。
// 前提：本会员已绑手机号（未绑 → MEMBER_PHONE_REQUIRED）。幂等覆盖（设置=重置）。
func (p *Provider) SetPassword(ctx context.Context, tx shared.Tx, memberID, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	var phone sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT phone FROM member WHERE id = ?`, memberID).Scan(&phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
		}
		return shared.Server("MEMBER_QUERY", err)
	}
	if !phone.Valid || phone.String == "" {
		return shared.BadRequest("MEMBER_PHONE_REQUIRED", "请先完善手机号")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE member SET password_hash = ? WHERE id = ?`, hash, memberID); err != nil {
		return shared.Server("MEMBER_UPDATE", err)
	}
	return nil
}

// AdminSetPassword — 商家后台重置密码（"忘记密码联系商家"闭环最后一段）。
// 无 phone 也可设（仅存储，登录仍需手机号）；复用 EnsureSeed 同款 bcrypt。
func (p *Provider) AdminSetPassword(ctx context.Context, memberID, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	res, err := p.db.ExecContext(ctx,
		`UPDATE member SET password_hash = ? WHERE id = ?`, hash, memberID)
	if err != nil {
		return shared.Server("MEMBER_UPDATE", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
	}
	return nil
}

// SetPhoneOnce — 手机号一次性设置：当前必须为空（MEMBER_PHONE_SET），
// 撞号 → MEMBER_PHONE_TAKEN 409（小程序端据此弹出 H5 密码认领框）。
func (p *Provider) SetPhoneOnce(ctx context.Context, tx shared.Tx, memberID, phone string) error {
	phone, err := validatePhone(phone)
	if err != nil {
		return err
	}
	var cur sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT phone FROM member WHERE id = ?`, memberID).Scan(&cur)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shared.NotFound("MEMBER_NOT_FOUND", "会员不存在")
		}
		return shared.Server("MEMBER_QUERY", err)
	}
	if cur.Valid && cur.String != "" {
		return shared.Conflict("MEMBER_PHONE_SET", "手机号已设置，不可修改")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE member SET phone = ? WHERE id = ?`, phone, memberID); err != nil {
		if shared.IsDupKey(err) {
			return shared.Conflict("MEMBER_PHONE_TAKEN", "该手机号已被其他账号使用")
		}
		return shared.Server("MEMBER_UPDATE", err)
	}
	return nil
}

// HasBusinessData — 会员名下是否存在预约/会员卡/核销流水任一（认领保护：
// 空壳才允许转绑删除，绝不丢业务数据）。
func (p *Provider) HasBusinessData(ctx context.Context, tx shared.Tx, memberID string) (bool, error) {
	var has bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM appointment WHERE member_id = ?)
		      OR EXISTS(SELECT 1 FROM member_card WHERE member_id = ?)
		      OR EXISTS(SELECT 1 FROM redemption WHERE member_id = ?)`,
		memberID, memberID, memberID).Scan(&has)
	if err != nil {
		return false, shared.Server("MEMBER_QUERY", err)
	}
	return has, nil
}

// DeleteShell — 物理删除空壳会员（仅 claim 事务内、HasBusinessData=false 时使用；
// 核心历史数据禁止物理删除的规则不适用于从未产生业务数据的自动建号空壳）。
// member_tag_rel 是 member 的 FK 子表，先清再删。
func (p *Provider) DeleteShell(ctx context.Context, tx shared.Tx, memberID string) error {
	has, err := p.HasBusinessData(ctx, tx, memberID)
	if err != nil {
		return err
	}
	if has {
		return shared.Conflict("MEMBER_HAS_DATA", "会员已有业务数据，禁止删除")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM member_tag_rel WHERE member_id = ?`, memberID); err != nil {
		return shared.Server("MEMBER_DELETE", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM member WHERE id = ?`, memberID); err != nil {
		return shared.Server("MEMBER_DELETE", err)
	}
	return nil
}

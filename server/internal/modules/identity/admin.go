package identity

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"anmo/server/internal/shared"
)

// ensureOwner seeds the OWNER account from config on first run. Password is
// bcrypt-hashed here (never in SQL seeds).
func (p *Provider) ensureOwner(ctx context.Context) error {
	phone := strings.TrimSpace(p.cfg.Auth.AdminPhone)
	if phone == "" {
		return nil
	}
	var count int
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM identity_user WHERE phone = ?`, phone).Scan(&count); err != nil {
		return shared.Server("IDENTITY_SEED", err)
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(p.cfg.Auth.AdminPasswordSeed), bcrypt.DefaultCost)
	if err != nil {
		return shared.Server("IDENTITY_SEED", err)
	}
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO identity_user (id, phone, password_hash, name, role) VALUES (?, ?, ?, ?, 'OWNER')`,
		shared.NewID(), phone, string(hash), "老板")
	if err != nil {
		return shared.Server("IDENTITY_SEED", err)
	}
	return nil
}

type adminUser struct {
	id           string
	phone        string
	passwordHash string
	name         string
	role         string
	status       string
}

func (p *Provider) findUserByPhone(ctx context.Context, phone string) (*adminUser, error) {
	u := &adminUser{}
	err := p.db.QueryRowContext(ctx,
		`SELECT id, phone, password_hash, name, role, status FROM identity_user WHERE phone = ?`,
		phone).Scan(&u.id, &u.phone, &u.passwordHash, &u.name, &u.role, &u.status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, shared.Server("IDENTITY_QUERY", err)
	}
	return u, nil
}

// AdminLogin verifies phone+password and issues an admin JWT.
func (p *Provider) AdminLogin(ctx context.Context, phone, password string) (string, *adminUser, error) {
	if phone == "" || password == "" {
		return "", nil, shared.BadRequest("IDENTITY_BAD_REQUEST", "手机号和密码必填")
	}
	u, err := p.findUserByPhone(ctx, phone)
	if err != nil {
		return "", nil, err
	}
	if u == nil || u.status != "ACTIVE" ||
		bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password)) != nil {
		return "", nil, shared.Unauthorized("手机号或密码错误")
	}
	token, err := p.tokens.signAdmin(u.id, u.role, time.Duration(p.cfg.Auth.AdminTokenHours)*time.Hour)
	if err != nil {
		return "", nil, shared.Server("IDENTITY_TOKEN", err)
	}
	return token, u, nil
}

// EnsureSeed runs idempotent identity seeding (OWNER account).
func (p *Provider) EnsureSeed(ctx context.Context) error {
	return p.ensureOwner(ctx)
}

// UpdateCredentials lets the merchant change their own login phone and/or
// password from the admin UI (goal §A5.5). Hot-effective: login reads the DB
// row on every attempt, and issued JWTs are keyed by user id so the current
// session survives the change. Current password must be re-entered.
func (p *Provider) UpdateCredentials(ctx context.Context, userID, currentPassword, newPhone, newPassword string) error {
	if currentPassword == "" {
		return shared.BadRequest("IDENTITY_PWD_REQUIRED", "请输入当前密码")
	}
	newPhone = strings.TrimSpace(newPhone)
	if newPhone == "" && newPassword == "" {
		return shared.BadRequest("IDENTITY_NOTHING", "新手机号和新密码至少填一项")
	}
	if newPhone != "" && !regexp.MustCompile(`^1\d{10}$`).MatchString(newPhone) {
		return shared.BadRequest("IDENTITY_BAD_PHONE", "新手机号格式不正确")
	}
	if newPassword != "" && len(newPassword) < 6 {
		return shared.BadRequest("IDENTITY_WEAK_PWD", "新密码至少 6 位")
	}

	u := &adminUser{}
	err := p.db.QueryRowContext(ctx,
		`SELECT id, phone, password_hash, name, role, status FROM identity_user WHERE id = ?`,
		userID).Scan(&u.id, &u.phone, &u.passwordHash, &u.name, &u.role, &u.status)
	if errors.Is(err, sql.ErrNoRows) {
		return shared.NotFound("IDENTITY_NOT_FOUND", "账号不存在")
	}
	if err != nil {
		return shared.Server("IDENTITY_QUERY", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(currentPassword)) != nil {
		return shared.Unauthorized("当前密码错误")
	}

	if newPhone != "" && newPhone != u.phone {
		if existing, err := p.findUserByPhone(ctx, newPhone); err != nil {
			return err
		} else if existing != nil && existing.id != u.id {
			return shared.Conflict("IDENTITY_PHONE_TAKEN", "该手机号已被使用")
		}
	}

	hash := u.passwordHash
	if newPassword != "" {
		b, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return shared.Server("IDENTITY_HASH", err)
		}
		hash = string(b)
	}
	phone := u.phone
	if newPhone != "" {
		phone = newPhone
	}
	if _, err := p.db.ExecContext(ctx,
		`UPDATE identity_user SET phone = ?, password_hash = ? WHERE id = ?`,
		phone, hash, u.id); err != nil {
		return shared.Server("IDENTITY_UPDATE", err)
	}
	return nil
}

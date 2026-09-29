package identity

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"anmo/server/internal/shared"
)

// customer.go — 顾客账号面（V2.2 第二批，plan §二）：
//   - H5 手机号+密码 注册/登录（短信验证码登录已下线）
//   - 微信首登直建号（bind_ticket 流程废除，D25 修订）
//   - 手机号撞号认领（原 H5 密码确认后转绑，plan §二.3/二.4）
//   - 小程序设置/重置 H5 密码（微信身份即凭证，plan §二.5）
//
// 所有建号/转绑都在 `_txlock=immediate` 事务内（D5），手机号/openid 唯一性由
// uk 约束兜底。bcrypt 与商家后台同款（DefaultCost）。密码明文永不落日志。

// validatePassword — 6~64 位（字符数），字节上限同时防 bcrypt 72 字节截断。
func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < 6 || n > 64 || len(password) > 72 {
		return shared.BadRequest("AUTH_WEAK_PASSWORD", "密码需 6~64 位")
	}
	return nil
}

// phoneRe — 11 位手机号（与 admin 凭证更新同口径）。
var phoneRe = regexp.MustCompile(`^1\d{10}$`)

// Register — H5 注册：手机号不存在 → 建号+登录；存在则按账号形态三分支拒绝
// （plan §二.4：不自动创建重复账号）。整个判定与建号在同一 immediate 事务内。
func (p *Provider) Register(ctx context.Context, phone, password string) (token, memberID string, err error) {
	phone = strings.TrimSpace(phone)
	if !phoneRe.MatchString(phone) {
		return "", "", shared.BadRequest("AUTH_BAD_PHONE", "手机号格式不正确")
	}
	if err := validatePassword(password); err != nil {
		return "", "", err
	}
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		m, e := p.members.GetByPhone(ctx, tx, phone)
		if e != nil && !shared.Is(e, "MEMBER_NOT_FOUND") {
			return e
		}
		if m != nil {
			switch {
			case m.HasPassword:
				return shared.Conflict("AUTH_PHONE_EXISTS", "该手机号已注册，请直接登录")
			case m.WxBound:
				return shared.Conflict("AUTH_PHONE_ON_WECHAT", "该手机号已开通微信账号，请前往小程序「我的」设置 H5 密码后登录")
			default:
				return shared.Conflict("AUTH_PHONE_EXISTS_NO_PWD", "该手机号已存在，请联系商家重置密码后登录")
			}
		}
		memberID, e = p.members.CreateWithPhone(ctx, tx, phone, password)
		return e
	})
	if err != nil {
		// uk 兜底（并发注册同名手机号，理论不可达：immediate 串行化）→ 统一文案
		if shared.Is(err, "MEMBER_PHONE_EXISTS") {
			return "", "", shared.Conflict("AUTH_PHONE_EXISTS", "该手机号已注册，请直接登录")
		}
		return "", "", err
	}
	token, err = p.tokens.signCustomer(memberID, time.Duration(p.cfg.Auth.CustomerTokenHours)*time.Hour)
	if err != nil {
		return "", "", shared.Server("IDENTITY_TOKEN", err)
	}
	return token, memberID, nil
}

// Login — H5 手机号+密码登录。四类提示逐字采用 plan/design 口径：
// 未注册 / 无密码（存量短信时代账号）/ 密码错误 / 成功。
func (p *Provider) Login(ctx context.Context, phone, password string) (token, memberID string, err error) {
	phone = strings.TrimSpace(phone)
	if phone == "" || password == "" {
		return "", "", shared.BadRequest("AUTH_BAD_REQUEST", "手机号和密码必填")
	}
	m, err := p.members.GetByPhone(ctx, p.db, phone)
	if err != nil {
		if shared.Is(err, "MEMBER_NOT_FOUND") {
			return "", "", shared.NewErr("AUTH_UNREGISTERED", "该手机号尚未注册", 401)
		}
		return "", "", err
	}
	if !m.HasPassword {
		return "", "", shared.NewErr("AUTH_NO_PASSWORD", "该账号尚未设置密码，请联系商家重置；已使用微信小程序可在「我的」设置 H5 密码", 401)
	}
	if bcrypt.CompareHashAndPassword([]byte(m.PasswordHash), []byte(password)) != nil {
		return "", "", shared.NewErr("AUTH_BAD_PASSWORD", "密码错误，请重新输入", 401)
	}
	token, err = p.tokens.signCustomer(m.ID, time.Duration(p.cfg.Auth.CustomerTokenHours)*time.Hour)
	if err != nil {
		return "", "", shared.Server("IDENTITY_TOKEN", err)
	}
	return token, m.ID, nil
}

// SetH5Password — 小程序「我的」设置/重置 H5 密码（设置=重置同路径，plan §二.5；
// 微信身份即凭证：能进到这一步的必然是当前微信登录态本人）。前提已绑手机号。
func (p *Provider) SetH5Password(ctx context.Context, memberID, newPassword string) error {
	return shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		return p.members.SetPassword(ctx, tx, memberID, newPassword)
	})
}

// ClaimByPhone — 手机号撞号认领（`POST /api/auth/wx/claim`，顾客 JWT）：
// 当前微信空壳会员用「该手机号的 H5 密码」把 openid 转绑到老 H5 账号
// （plan §二.3），同事务删除当前空壳并签发老账号的新 Token（登录态切换）。
// 冲突保护（plan §二.4，不自动认定谁是正确账号）：
//   - 老账号已绑定其他微信 → WX_OPENID_BOUND
//   - 当前空壳已有业务数据 → AUTH_CLAIM_HAS_DATA（绝不丢数据）
func (p *Provider) ClaimByPhone(ctx context.Context, currentMemberID, phone, password string) (token, memberID string, err error) {
	phone = strings.TrimSpace(phone)
	if phone == "" || password == "" {
		return "", "", shared.BadRequest("AUTH_BAD_REQUEST", "手机号和密码必填")
	}
	var targetID string
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		// 0. openid 从当前 member 行取（登录态即 openid 持有者）
		openid, ok, e := p.members.OpenIDOf(ctx, tx, currentMemberID)
		if e != nil {
			return e
		}
		if !ok {
			return shared.BadRequest("WX_NO_OPENID", "当前账号未绑定微信")
		}
		// 1. 目标账号必须存在（正常流程先 SetPhoneOnce 撞号才会进 claim）
		target, e := p.members.GetByPhone(ctx, tx, phone)
		if e != nil {
			if shared.Is(e, "MEMBER_NOT_FOUND") {
				return shared.NotFound("AUTH_CLAIM_NO_ACCOUNT", "该手机号尚未注册")
			}
			return e
		}
		// 2/3. 目标必须设过 H5 密码且密码正确
		if !target.HasPassword {
			return shared.NewErr("AUTH_NO_PASSWORD", "该账号尚未设置密码，请联系商家重置", 401)
		}
		if bcrypt.CompareHashAndPassword([]byte(target.PasswordHash), []byte(password)) != nil {
			return shared.NewErr("AUTH_BAD_PASSWORD", "密码错误，请重新输入", 401)
		}
		// 4. 目标已绑定其他微信 → 拒绝（plan §二.4）
		if target.WxBound {
			return shared.Conflict("WX_OPENID_BOUND", "该手机号已绑定其他微信账号，请联系商家处理")
		}
		// 5. 当前空壳已有业务数据 → 拒绝转绑，绝不丢数据
		hasData, e := p.members.HasBusinessData(ctx, tx, currentMemberID)
		if e != nil {
			return e
		}
		if hasData {
			return shared.Conflict("AUTH_CLAIM_HAS_DATA", "当前账号已有预约/卡记录，请联系商家处理")
		}
		// 6a. openid 转绑到目标：先解绑当前空壳（uk 不允许两行同值），
		// 再绑定到老账号；DupKey 兜底同 409，文案按 claim 场景
		if e := p.members.ClearOpenID(ctx, tx, currentMemberID); e != nil {
			return e
		}
		if e := p.members.BindOpenID(ctx, tx, target.ID, openid); e != nil {
			if shared.Is(e, "WX_OPENID_BOUND") {
				return shared.Conflict("WX_OPENID_BOUND", "该手机号已绑定其他微信账号，请联系商家处理")
			}
			return e
		}
		// 6b. 同事务删除当前空壳（DeleteShell 内部自带 HasBusinessData 守卫）
		targetID = target.ID
		return p.members.DeleteShell(ctx, tx, currentMemberID)
	})
	if err != nil {
		return "", "", err
	}
	token, err = p.tokens.signCustomer(targetID, time.Duration(p.cfg.Auth.CustomerTokenHours)*time.Hour)
	if err != nil {
		return "", "", shared.Server("IDENTITY_TOKEN", err)
	}
	return token, targetID, nil
}

// SendSMS delivers a login code to the phone. Dev mode sends the fixed code
// 123456 (W3); off 模式在 handler 层已拦截（410 SMS_DISABLED）。生产模式未实现
// 真实发送方——仅 dev 与 off 两个合法值（config.validate 保证）。
func (p *Provider) SendSMS(ctx context.Context, phone string) error {
	if len(phone) != 11 {
		return shared.BadRequest("SMS_BAD_PHONE", "手机号格式不正确")
	}
	fixed := ""
	if p.cfg.SMS.Mode == "dev" {
		fixed = "123456"
	}
	return p.sms.send(ctx, p.sender, phone,
		time.Duration(p.cfg.SMS.CodeTTL)*time.Second, fixed)
}

// smsDisabled — 短信登录是否已下线。只有 dev 模式可用（R6：生产必须 off；
// 未显式配置时的默认值 dev 属本地开发形态，runbook 强调生产改 off）。
func (p *Provider) smsDisabled() bool {
	return p.cfg.SMS.Mode != "dev"
}

// smsDevLogin — 过渡期 dev 短信登录（未升级小程序在本地开发环境仍可跑通）。
// off 模式在 handler 已 410 拒绝。首登建号复用 member.EnsureByPhone（D3）。
func (p *Provider) smsDevLogin(ctx context.Context, phone, code string) (token, memberID string, err error) {
	if len(phone) != 11 {
		return "", "", shared.BadRequest("SMS_BAD_PHONE", "手机号格式不正确")
	}
	if !p.sms.verify(phone, code) {
		return "", "", shared.Unauthorized("验证码错误或已过期")
	}
	err = shared.RunInTx(ctx, p.db, func(tx shared.Tx) error {
		var e error
		memberID, _, e = p.members.EnsureByPhone(ctx, tx, phone, "")
		return e
	})
	if err != nil {
		return "", "", err
	}
	token, err = p.tokens.signCustomer(memberID, time.Duration(p.cfg.Auth.CustomerTokenHours)*time.Hour)
	if err != nil {
		return "", "", shared.Server("IDENTITY_TOKEN", err)
	}
	return token, memberID, nil
}

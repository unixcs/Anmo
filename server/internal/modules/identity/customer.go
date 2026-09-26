package identity

import (
	"context"
	"time"

	"anmo/server/internal/shared"
)

// CustomerLogin verifies the SMS code and issues a customer JWT. First login
// creates the member (plan §10); later logins bind to the same member (D3).
func (p *Provider) CustomerLogin(ctx context.Context, phone, code string) (token, memberID string, err error) {
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

// SendSMS delivers a login code to the phone. Dev mode sends the fixed code
// 123456 (W3); production modes would use a random code with a real sender.
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

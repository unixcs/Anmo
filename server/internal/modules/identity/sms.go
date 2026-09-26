package identity

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"anmo/server/internal/shared"
)

// smsService handles verification-code delivery. V1 has no real SMS provider
// (W3): dev mode uses a fixed code and logs it. The sender interface is the
// seam for a production provider.
type smsSender interface {
	Send(ctx context.Context, phone, code string) error
}

type devSender struct {
	log *slog.Logger
}

func (d devSender) Send(_ context.Context, phone, code string) error {
	d.log.Info("sms(dev)", "phone", phone, "code", code)
	return nil
}

type smsStore struct {
	mu     sync.Mutex
	limits map[string]time.Time // last send time per phone (rate limit)
	codes  map[string]smsCode
}

type smsCode struct {
	code      string
	expiresAt time.Time
}

func newSMSStore() *smsStore {
	return &smsStore{limits: map[string]time.Time{}, codes: map[string]smsCode{}}
}

func randomCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "123456"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// send stores and delivers a code; enforces a 60s per-phone rate limit.
// fixedCode is used in dev mode (W3: fixed "123456"); empty means random.
func (s *smsStore) send(ctx context.Context, sender smsSender, phone string, ttl time.Duration, fixedCode string) error {
	s.mu.Lock()
	if last, ok := s.limits[phone]; ok && time.Since(last) < 60*time.Second {
		s.mu.Unlock()
		return shared.Conflict("SMS_RATE_LIMITED", "发送太频繁，请稍后再试")
	}
	code := fixedCode
	if code == "" {
		code = randomCode()
	}
	s.limits[phone] = time.Now()
	s.codes[phone] = smsCode{code: code, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
	return sender.Send(ctx, phone, code)
}

// verify checks and consumes the code.
func (s *smsStore) verify(phone, code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[phone]
	if !ok || time.Now().After(c.expiresAt) || c.code != code {
		return false
	}
	delete(s.codes, phone)
	return true
}

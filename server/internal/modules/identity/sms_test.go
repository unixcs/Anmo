package identity

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestSMSStoreVerifyConsume(t *testing.T) {
	s := newSMSStore()
	ctx := context.Background()
	sender := devSender{log: slog.Default()}

	if err := s.send(ctx, sender, "13911112222", 5*time.Minute, "123456"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !s.verify("13911112222", "123456") {
		t.Fatal("correct code rejected")
	}
	if s.verify("13911112222", "123456") {
		t.Fatal("code reused after consume")
	}
}

func TestSMSStoreRateLimitAndExpiry(t *testing.T) {
	s := newSMSStore()
	ctx := context.Background()
	sender := devSender{log: slog.Default()}

	if err := s.send(ctx, sender, "13911112222", 5*time.Minute, "111111"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := s.send(ctx, sender, "13911112222", 5*time.Minute, "111111"); err == nil {
		t.Fatal("second immediate send allowed (rate limit missing)")
	}

	// expiry
	s.codes["13900003333"] = smsCode{code: "222222", expiresAt: time.Now().Add(-time.Second)}
	if s.verify("13900003333", "222222") {
		t.Fatal("expired code accepted")
	}

	// wrong code
	if err := s.send(ctx, sender, "13900004444", 5*time.Minute, "333333"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if s.verify("13900004444", "999999") {
		t.Fatal("wrong code accepted")
	}
}

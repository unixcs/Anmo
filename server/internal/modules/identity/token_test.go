package identity

import (
	"testing"
	"time"

	"anmo/server/internal/middleware"
)

func TestTokenRoundtrip(t *testing.T) {
	s := &tokenService{secret: []byte("test-secret")}
	tok, err := s.signAdmin("user-1", "OWNER", time.Hour)
	if err != nil {
		t.Fatalf("signAdmin: %v", err)
	}
	p, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if p.ActorType != "ADMIN" || p.ActorID != "user-1" || p.Role != "OWNER" {
		t.Fatalf("principal = %+v", p)
	}

	cust, err := s.signCustomer("member-9", 24*time.Hour)
	if err != nil {
		t.Fatalf("signCustomer: %v", err)
	}
	pc, err := s.Verify(cust)
	if err != nil {
		t.Fatalf("Verify customer: %v", err)
	}
	if pc.ActorType != "CUSTOMER" || pc.ActorID != "member-9" {
		t.Fatalf("principal = %+v", pc)
	}
}

func TestTokenRejectsTamperedAndWrongSecret(t *testing.T) {
	s := &tokenService{secret: []byte("secret-a")}
	tok, _ := s.signAdmin("u", "OWNER", time.Hour)

	if _, err := s.Verify(tok + "x"); err == nil {
		t.Fatal("tampered token accepted")
	}
	other := &tokenService{secret: []byte("secret-b")}
	if _, err := other.Verify(tok); err == nil {
		t.Fatal("wrong-secret token accepted")
	}

	expired, _ := s.signAdmin("u", "OWNER", -time.Minute)
	if _, err := s.Verify(expired); err == nil {
		t.Fatal("expired token accepted")
	}
}

var _ = middleware.Principal{}

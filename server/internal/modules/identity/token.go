package identity

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"anmo/server/internal/middleware"
)

// tokenService signs and verifies the two token kinds: admin (identity_user)
// and customer (member).
type tokenService struct {
	secret []byte
}

type claims struct {
	ActorType string `json:"act"` // ADMIN | CUSTOMER
	ActorID   string `json:"uid"`
	Role      string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

func (s *tokenService) signAdmin(userID, role string, ttl time.Duration) (string, error) {
	return s.sign("ADMIN", userID, role, ttl)
}

func (s *tokenService) signCustomer(memberID string, ttl time.Duration) (string, error) {
	return s.sign("CUSTOMER", memberID, "", ttl)
}

// actWxBind marks a bind-ticket token: it carries an openid, not an actor id.
const actWxBind = "WXBIND"

// signBindTicket issues the short-lived ticket returned by /api/auth/wx/login
// for unbound openids (D25). Redeemed by POST /api/auth/wx/bind with a
// customer token.
func (s *tokenService) signBindTicket(openid string) (string, error) {
	now := time.Now()
	c := claims{
		ActorType: actWxBind,
		ActorID:   openid,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(wxBindTTL)),
			Issuer:    "anmo",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
}

// parseBindTicket validates a bind ticket and returns its openid.
func (s *tokenService) parseBindTicket(token string) (string, error) {
	c, err := s.parse(token)
	if err != nil || c.ActorType != actWxBind || c.ActorID == "" {
		return "", errors.New("invalid bind ticket")
	}
	return c.ActorID, nil
}

func (s *tokenService) sign(actorType, actorID, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		ActorType: actorType,
		ActorID:   actorID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "anmo",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
}

// parse validates the signature/expiry and returns the claims.
func (s *tokenService) parse(token string) (*claims, error) {
	var c claims
	t, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	return &c, nil
}

// Verify implements the middleware.TokenVerifier contract.
func (s *tokenService) Verify(token string) (middleware.Principal, error) {
	c, err := s.parse(token)
	if err != nil {
		return middleware.Principal{}, err
	}
	return middleware.Principal{ActorType: c.ActorType, ActorID: c.ActorID, Role: c.Role}, nil
}

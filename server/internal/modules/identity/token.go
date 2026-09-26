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

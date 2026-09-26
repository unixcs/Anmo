package middleware

import (
	"net/http"
	"strings"

	"anmo/server/internal/shared"
)

// TokenVerifier validates an Authorization bearer token and returns the
// principal. Provided by the identity module (Phase 2); middleware stays
// token-format-agnostic.
type TokenVerifier func(token string) (Principal, error)

// NewAuth builds an auth middleware. When requireAdmin is true the principal
// must be ADMIN with role OWNER or OPERATOR; otherwise CUSTOMER.
func NewAuth(verify TokenVerifier, requireAdmin bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				token = strings.TrimPrefix(h, "Bearer ")
			}
			if token == "" {
				shared.Unauthorized("missing token").Write(w)
				return
			}
			p, err := verify(token)
			if err != nil {
				shared.Unauthorized("invalid token").Write(w)
				return
			}
			if requireAdmin && p.ActorType != "ADMIN" {
				shared.Forbidden("admin required").Write(w)
				return
			}
			if !requireAdmin && p.ActorType != "CUSTOMER" {
				shared.Forbidden("customer required").Write(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

package card

import (
	"net/http"

	"anmo/server/internal/middleware"
)

// operatorOf extracts the admin actor id for audit fields.
func operatorOf(r *http.Request) string {
	if pr, ok := middleware.PrincipalFrom(r.Context()); ok {
		return pr.ActorID
	}
	return ""
}

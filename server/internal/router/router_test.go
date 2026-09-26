package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"anmo/server/internal/middleware"
)

func noopVerify(token string) (middleware.Principal, error) {
	return middleware.Principal{}, nil
}

func TestHealthz(t *testing.T) {
	h := New(slog.Default(), noopVerify, noopModule{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != 200 {
		t.Fatalf("healthz status = %d", rec.Code)
	}
	if want := `"status":"ok"`; !contains(rec.Body.String(), want) {
		t.Fatalf("healthz body = %s", rec.Body.String())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

type noopModule struct{}

func (noopModule) Mount(root, admin, api *http.ServeMux) {}

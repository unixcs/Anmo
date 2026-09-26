package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type noopModule struct{}

func (noopModule) Mount(mux *http.ServeMux) {}

func TestHealthz(t *testing.T) {
	h := New(slog.Default(), noopModule{})
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

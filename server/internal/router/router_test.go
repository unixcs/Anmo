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
	h := New(slog.Default(), noopVerify, nil, noopModule{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != 200 {
		t.Fatalf("healthz status = %d", rec.Code)
	}
	if want := `"status":"ok"`; !contains(rec.Body.String(), want) {
		t.Fatalf("healthz body = %s", rec.Body.String())
	}
}

func TestRootIndex(t *testing.T) {
	h := New(slog.Default(), noopVerify, nil, noopModule{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("root status = %d", rec.Code)
	}
	for _, want := range []string{`"service":"anmo"`, `"health"`, `"customer"`, `"admin"`} {
		if !contains(rec.Body.String(), want) {
			t.Fatalf("root body = %s, missing %s", rec.Body.String(), want)
		}
	}

	// exact-match only: unknown paths must keep their 404
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/nope", nil))
	if rec.Code != 404 {
		t.Fatalf("unknown path status = %d, want 404", rec.Code)
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

package router

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"anmo/server/internal/middleware"
)

func noopVerify(token string) (middleware.Principal, error) {
	return middleware.Principal{}, nil
}

func noopHealth(context.Context) error { return nil }

func TestHealthz(t *testing.T) {
	h := New(slog.Default(), noopVerify, nil, noopHealth, noopModule{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != 200 {
		t.Fatalf("healthz status = %d", rec.Code)
	}
	if want := `"status":"ok"`; !contains(rec.Body.String(), want) {
		t.Fatalf("healthz body = %s", rec.Body.String())
	}
}

// F14: 数据库探活失败时 /healthz 必须 503，而不是进程活着就报绿。
func TestHealthzDBDown(t *testing.T) {
	h := New(slog.Default(), noopVerify, nil,
		func(context.Context) error { return errors.New("disk gone") }, noopModule{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("healthz(db down) status = %d, want 503", rec.Code)
	}
	if want := `"code":"DB_DOWN"`; !contains(rec.Body.String(), want) {
		t.Fatalf("healthz(db down) body = %s", rec.Body.String())
	}
}

func TestRootIndex(t *testing.T) {
	h := New(slog.Default(), noopVerify, nil, noopHealth, noopModule{})

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

// F3: 公开鉴权端点按 IP 限速，第 21 次（>20/min）必须 429；非鉴权路径不受影响。
func TestAuthRateLimit(t *testing.T) {
	root := http.NewServeMux()
	root.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		shared_OK(w)
	})
	root.HandleFunc("POST /other", func(w http.ResponseWriter, r *http.Request) { shared_OK(w) })
	h := New(slog.Default(), noopVerify, nil, noopHealth, pubModule{root: root})

	for i := 0; i < 20; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/auth/login", nil))
		if rec.Code != 200 {
			t.Fatalf("req %d status = %d, want 200", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/auth/login", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("req 21 status = %d, want 429", rec.Code)
	}
	// 其他路径不受限速影响
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/other", nil))
	if rec.Code != 200 {
		t.Fatalf("non-auth path status = %d, want 200", rec.Code)
	}
}

func shared_OK(w http.ResponseWriter) {
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{}`))
}

type pubModule struct{ root *http.ServeMux }

func (m pubModule) Mount(root, _, _ *http.ServeMux) {
	root.Handle("/api/auth/", m.root)
	root.Handle("/other", m.root)
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

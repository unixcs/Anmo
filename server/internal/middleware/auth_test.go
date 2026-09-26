package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func verifyAdmin(token string) (Principal, error) {
	if token == "admin-tok" {
		return Principal{ActorType: "ADMIN", ActorID: "u1", Role: "OWNER"}, nil
	}
	if token == "cust-tok" {
		return Principal{ActorType: "CUSTOMER", ActorID: "m1"}, nil
	}
	return Principal{}, errors.New("invalid")
}

func TestAuthAdminRequired(t *testing.T) {
	h := NewAuth(verifyAdmin, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		_, _ = w.Write([]byte(p.ActorID))
	}))

	// no token → 401
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != 401 {
		t.Fatalf("no token: %d", rec.Code)
	}

	// customer token → 403
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer cust-tok")
	h.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("customer on admin: %d", rec.Code)
	}

	// admin token → 200 + principal in context
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer admin-tok")
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "u1" {
		t.Fatalf("admin: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAuthCustomerRequired(t *testing.T) {
	h := NewAuth(verifyAdmin, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		_, _ = w.Write([]byte(p.ActorID))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer cust-tok")
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "m1" {
		t.Fatalf("customer: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer admin-tok")
	h.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("admin on customer route: %d", rec.Code)
	}
}

package shared

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailAppError(t *testing.T) {
	rec := httptest.NewRecorder()
	Fail(rec, Conflict("APPOINTMENT_CONFLICT", "该时间已被预约"))
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"APPOINTMENT_CONFLICT"`) {
		t.Fatalf("body missing code: %s", body)
	}
}

func TestFailUnknownBecomes500(t *testing.T) {
	rec := httptest.NewRecorder()
	Fail(rec, errors.New("boom"))
	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"INTERNAL"`) {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestPageParams(t *testing.T) {
	p := PageParams{Page: 3, PerPage: 50}
	if p.Limit() != 50 || p.Offset() != 100 {
		t.Fatalf("limit/offset = %d/%d", p.Limit(), p.Offset())
	}
	zeros := PageParams{}
	if zeros.Limit() != 20 || zeros.Offset() != 0 {
		t.Fatalf("defaults: %d/%d", zeros.Limit(), zeros.Offset())
	}
	huge := PageParams{PerPage: 500}
	if huge.Limit() != 100 {
		t.Fatalf("cap: %d", huge.Limit())
	}
}

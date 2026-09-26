// Package shared holds cross-module primitives: app errors, id generation,
// transaction runners and response helpers. Nothing here may import modules.
package shared

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the single error shape crossing module boundaries and handlers.
type AppError struct {
	Code    string // stable machine code, e.g. APPOINTMENT_CONFLICT
	Message string // human readable, safe to expose
	Status  int    // http status
	Err     error  // wrapped cause, never exposed
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func NewErr(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

func WrapErr(code, message string, status int, err error) *AppError {
	return &AppError{Code: code, Message: message, Status: status, Err: err}
}

// Common constructors used across modules.
func BadRequest(code, msg string) *AppError { return NewErr(code, msg, http.StatusBadRequest) }
func Unauthorized(msg string) *AppError     { return NewErr("UNAUTHORIZED", msg, http.StatusUnauthorized) }
func Forbidden(msg string) *AppError        { return NewErr("FORBIDDEN", msg, http.StatusForbidden) }
func NotFound(code, msg string) *AppError   { return NewErr(code, msg, http.StatusNotFound) }
func Conflict(code, msg string) *AppError   { return NewErr(code, msg, http.StatusConflict) }
func Server(code string, err error) *AppError {
	return WrapErr(code, "internal error", http.StatusInternalServerError, err)
}

// Is reports whether err is an *AppError with the given code.
func Is(err error, code string) bool {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code == code
	}
	return false
}

// Write responds with this error using the standard envelope.
func (e *AppError) Write(w http.ResponseWriter) {
	writeJSON(w, e.Status, envelope{Code: e.Code, Msg: e.Message})
}

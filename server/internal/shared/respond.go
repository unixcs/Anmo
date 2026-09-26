package shared

import (
	"encoding/json"
	"errors"
	"net/http"
)

// respond.go — the one place that writes HTTP response bodies.

type envelope struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data any    `json:"data,omitempty"`
}

type pageEnvelope struct {
	Code    string `json:"code,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Data    any    `json:"data"`
	Total   int64  `json:"total"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
}

// OK writes a 200 JSON body {"data": ...}.
func OK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{Data: data})
}

// PageOK writes a paginated 200 JSON body.
func PageOK(w http.ResponseWriter, data any, total int64, p PageParams) {
	writeJSON(w, http.StatusOK, pageEnvelope{Data: data, Total: total, Page: p.Page, PerPage: p.Limit()})
}

// Fail maps an error to the standard error body. *AppError keeps its code,
// message and status; anything else becomes a 500 INTERNAL.
func Fail(w http.ResponseWriter, err error) {
	var ae *AppError
	if errors.As(err, &ae) {
		writeJSON(w, ae.Status, envelope{Code: ae.Code, Msg: ae.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, envelope{Code: "INTERNAL", Msg: "internal error"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// DecodeJSON decodes a request body into dst, rejecting unknown fields.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

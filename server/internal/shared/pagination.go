package shared

import (
	"net/http"
	"strconv"
)

// PageParams is the shared pagination shape for list endpoints.
type PageParams struct {
	Page    int // 1-based
	PerPage int // default 20, max 100
}

func (p PageParams) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	return (p.Page - 1) * p.Limit()
}

func (p PageParams) Limit() int {
	if p.PerPage < 1 {
		return 20
	}
	if p.PerPage > 100 {
		return 100
	}
	return p.PerPage
}

func PageFromRequest(r *http.Request) PageParams {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	per, _ := strconv.Atoi(q.Get("per_page"))
	return PageParams{Page: page, PerPage: per}
}

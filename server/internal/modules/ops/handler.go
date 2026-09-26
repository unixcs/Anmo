package ops

import (
	"net/http"

	"anmo/server/internal/shared"
)

func (p *Provider) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := p.Logs(r.Context(), r.URL.Query().Get("action"))
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, logs)
}

func (p *Provider) handleInsights(w http.ResponseWriter, r *http.Request) {
	out, err := p.Insights(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, out)
}

func (p *Provider) handleDaily(w http.ResponseWriter, r *http.Request) {
	out, err := p.RunDaily(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, out)
}

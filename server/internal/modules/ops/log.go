package ops

import (
	"context"
	"encoding/json"
	"log/slog"

	"anmo/server/internal/shared"
)

// log.go — operation log writing (D7: business modules never import ops; the
// HTTP middleware calls the writer returned here).

// LogEntry — one ops_operation_log row.
type LogEntry struct {
	ActorType  string `json:"-"`
	ActorID    string `json:"-"`
	Action     string
	TargetType string
	TargetID   string
	Detail     string
	IP         string
}

// LogWriter is installed into the HTTP middleware chain by main.
type LogWriter func(entry LogEntry)

// NewLogWriter builds the callback the middleware uses. Failures only log —
// an audit write must never break the request.
func (p *Provider) NewLogWriter(log *slog.Logger) LogWriter {
	return func(entry LogEntry) {
		if entry.Detail == "" {
			entry.Detail = "{}"
		}
		if _, err := p.db.ExecContext(context.Background(),
			`INSERT INTO ops_operation_log (id, actor_type, actor_id, action, target_type, target_id, detail, ip)
			 VALUES (?,?,?,?,?,?,?,?)`,
			shared.NewID(), entry.ActorType, entry.ActorID, entry.Action,
			entry.TargetType, entry.TargetID, entry.Detail, entry.IP); err != nil {
			log.Warn("operation log write failed", "err", err)
		}
	}
}

// LoggedOperation — admin log row for the query endpoint.
type LoggedOperation struct {
	ID         string `json:"id"`
	ActorType  string `json:"actor_type"`
	ActorID    string `json:"actor_id"`
	Action     string `json:"action"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Detail     string `json:"detail"`
	IP         string `json:"ip"`
}

// Logs returns the latest operations, optionally filtered by action.
func (p *Provider) Logs(ctx context.Context, action string) ([]*LoggedOperation, error) {
	q := `SELECT id, actor_type, actor_id, action, target_type, target_id, detail, ip
	      FROM ops_operation_log`
	args := []any{}
	if action != "" {
		q += ` WHERE action = ?`
		args = append(args, action)
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, shared.Server("OPS_LOGS", err)
	}
	defer rows.Close()
	var out []*LoggedOperation
	for rows.Next() {
		l := &LoggedOperation{}
		if err := rows.Scan(&l.ID, &l.ActorType, &l.ActorID, &l.Action, &l.TargetType, &l.TargetID, &l.Detail, &l.IP); err != nil {
			return nil, shared.Server("OPS_SCAN", err)
		}
		out = append(out, l)
	}
	return out, nil
}

var _ = json.Marshal

package ops

import (
	"context"
	"encoding/json"
	"fmt"

	"anmo/server/internal/shared"
)

func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

func itoa(n int) string { return fmt.Sprintf("%d", n) }

// insights.go — the three V1 insights (plan §86): low balance, dormant,
// expiring. Reads flow through card/member api.go providers (D7 one-way).
// No AI, no recommendation engine — plain reminder lists.

// InsightItem — one flagged member/card.
type InsightItem struct {
	Kind       string `json:"kind"`
	MemberID   string `json:"member_id"`
	MemberNo   string `json:"member_no"`
	MemberName string `json:"member_name"`
	CardID     string `json:"card_id,omitempty"`
	Detail     string `json:"detail"`
}

// Insights returns the three live lists.
func (p *Provider) Insights(ctx context.Context) (map[string][]InsightItem, error) {
	out := map[string][]InsightItem{
		"LOW_BALANCE": {},
		"DORMANT":     {},
		"EXPIRING":    {},
	}

	cards, err := p.cards.LowBalanceCards(ctx, p.cfg.Business.LowBalanceCount)
	if err != nil {
		return nil, err
	}
	for _, c := range cards {
		name, no := p.memberIdent(ctx, c.MemberID)
		out["LOW_BALANCE"] = append(out["LOW_BALANCE"], InsightItem{
			Kind: "LOW_BALANCE", MemberID: c.MemberID, MemberNo: no, MemberName: name,
			CardID: c.ID, Detail: sprintf("剩余 %d 次", c.RemainingCount),
		})
	}

	members, err := p.members.DormantMembers(ctx, p.cfg.Business.DormantDays)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		detail := "从未到店超过 " + itoa(p.cfg.Business.DormantDays) + " 天"
		if m.LastVisitAt != nil {
			detail = "最后到店 " + m.LastVisitAt.Format("2006-01-02")
		}
		out["DORMANT"] = append(out["DORMANT"], InsightItem{
			Kind: "DORMANT", MemberID: m.ID, MemberNo: m.MemberNo, MemberName: m.Name,
			Detail: detail,
		})
	}

	expiring, err := p.cards.ExpiringCards(ctx, p.cfg.Business.ExpiringDays)
	if err != nil {
		return nil, err
	}
	for _, c := range expiring {
		name, no := p.memberIdent(ctx, c.MemberID)
		until := ""
		if c.ValidUntil != nil {
			until = *c.ValidUntil
		}
		out["EXPIRING"] = append(out["EXPIRING"], InsightItem{
			Kind: "EXPIRING", MemberID: c.MemberID, MemberNo: no, MemberName: name,
			CardID: c.ID, Detail: sprintf("卡将于 %s 过期", until),
		})
	}
	return out, nil
}

// memberIdent resolves member display fields through the member api.
func (p *Provider) memberIdent(ctx context.Context, memberID string) (name, no string) {
	m, err := p.members.Get(ctx, memberID)
	if err != nil {
		return "", ""
	}
	return m.Name, m.MemberNo
}

// RunDaily executes the daily maintenance: expired-card sweep + insight
// snapshot. Production schedules it via cron hitting POST /admin/ops/daily.
func (p *Provider) RunDaily(ctx context.Context) (map[string]any, error) {
	swept, err := p.cards.SweepExpired(ctx)
	if err != nil {
		return nil, err
	}
	insights, err := p.Insights(ctx)
	if err != nil {
		return nil, err
	}
	saved := int64(0)
	today := shared.NowShanghai().Format("2006-01-02")
	for kind, items := range insights {
		for _, item := range items {
			detail, _ := json.Marshal(item)
			if _, err := p.db.ExecContext(ctx,
				`INSERT IGNORE INTO ops_insight_snapshot (id, snapshot_date, kind, member_id, detail)
				 VALUES (?,?,?,?,?)`,
				shared.NewID(), today, kind, item.MemberID, string(detail)); err != nil {
				return nil, shared.Server("OPS_SNAPSHOT", err)
			}
			saved++
		}
	}
	return map[string]any{"expired_swept": swept, "snapshots": saved}, nil
}

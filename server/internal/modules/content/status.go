// status.go — 门店当前状态（goal §17）：动态计算，不落库。
//
// SERVING 服务中：存在 IN_SERVICE 预约，预计空闲 = started_at + 快照时长（打烊截断）
// FREE    空闲中：无 IN_SERVICE，且当前营业时间内还能安排一场最短服务
// BUSY    忙碌中：其余情况（非营业时间 / 打烊前排不下最短服务）
package content

import (
	"context"
	"net/http"
	"time"

	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/modules/service"
	"anmo/server/internal/shared"
)

// Wire injects the cross-module read-only dependencies for store status.
// Two-phase on purpose: appointment needs content's RulesSource at build time,
// so content receives the finished appointment provider afterwards (app.go).
func (p *Provider) Wire(appts *appointment.Provider, services *service.Provider) {
	p.appts = appts
	p.services = services
}

// HandleStoreStatus serves GET /api/store/status.
func (p *Provider) HandleStoreStatus(w http.ResponseWriter, r *http.Request) {
	out, err := p.StoreStatus(r.Context())
	if err != nil {
		shared.Fail(w, err)
		return
	}
	shared.OK(w, out)
}

// cst is the single business timezone (Asia/Shanghai, plan §69).
var cst = time.FixedZone("CST", 8*3600)

type StoreStatus struct {
	Status    string `json:"status"` // FREE | SERVING | BUSY
	FreeAt    string `json:"free_at,omitempty"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

func (p *Provider) StoreStatus(ctx context.Context) (*StoreStatus, error) {
	rules, err := p.BookingRules(ctx)
	if err != nil {
		return nil, err
	}
	out := &StoreStatus{OpenTime: rules.OpenTime, CloseTime: rules.CloseTime}
	if p.appts == nil {
		out.Status = "BUSY"
		return out, nil
	}

	if slot, err := p.appts.ServingNow(ctx); err != nil {
		return nil, err
	} else if slot.Found {
		out.Status = "SERVING"
		freeAt := slot.StartedAt.In(cst).Add(time.Duration(slot.DurationMinutes) * time.Minute)
		if close, ok := endOfDayMinutes(rules.CloseTime); ok {
			closeTime := time.Date(freeAt.Year(), freeAt.Month(), freeAt.Day(), close/60, close%60, 0, 0, cst)
			if freeAt.After(closeTime) {
				freeAt = closeTime
			}
		}
		out.FreeAt = freeAt.Format("15:04")
		return out, nil
	}

	// 无服务中：营业时间内且打烊前还排得下最短服务 → 空闲
	out.Status = "BUSY"
	if p.services != nil {
		minDur, err := p.services.MinActiveDuration(ctx)
		if err != nil {
			return nil, err
		}
		now := shared.NowShanghai()
		open, close := minutes(rules.OpenTime), minutes(rules.CloseTime)
		cur := now.Hour()*60 + now.Minute()
		if minDur > 0 && cur >= open && cur < close && close-cur >= minDur {
			out.Status = "FREE"
		}
	}
	return out, nil
}

func minutes(hhmm string) int {
	t, err := time.ParseInLocation("15:04", hhmm, time.Local)
	if err != nil {
		return 0
	}
	return t.Hour()*60 + t.Minute()
}

// endOfDayMinutes parses close time; ok=false falls back to no truncation.
func endOfDayMinutes(hhmm string) (int, bool) {
	if hhmm == "" {
		return 0, false
	}
	m := minutes(hhmm)
	if m <= 0 {
		return 0, false
	}
	return m, true
}

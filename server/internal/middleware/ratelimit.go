package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"anmo/server/internal/shared"
)

// ratelimit.go — 第九批审查加固（F3/F15）：
//   - AuthRateLimit: 公开鉴权端点按来源 IP 的固定窗口限速，防撞库/暴力注册
//   - BodyLimit: 请求体上限，防恶意大包在 DecodeJSON 里吃内存
//   - ClientIP: 生产在 Cloudflare Tunnel 之后，socket 地址是隧道进程，
//     真实来源取 X-Forwarded-For 首跳，无该头时回落 RemoteAddr

// ClientIP returns the best-known origin address for rate limiting and audit.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		if ip := strings.TrimSpace(xff); ip != "" {
			return ip
		}
	}
	return r.RemoteAddr
}

// authLimiter — fixed-window counter per IP. Not a general-purpose limiter:
// only the handful of public auth endpoints register here, so the map stays
// small; expired IPs are evicted lazily once the map grows past 4096 entries.
type authLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newAuthLimiter(limit int, window time.Duration) *authLimiter {
	return &authLimiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *authLimiter) allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.hits) > 4096 {
		for k, ts := range l.hits {
			if len(ts) == 0 || now.Sub(ts[len(ts)-1]) > l.window {
				delete(l.hits, k)
			}
		}
	}
	var keep []time.Time
	for _, t := range l.hits[ip] {
		if now.Sub(t) <= l.window {
			keep = append(keep, t)
		}
	}
	if len(keep) >= l.limit {
		l.hits[ip] = keep
		return false
	}
	l.hits[ip] = append(keep, now)
	return true
}

// AuthRateLimit throttles the public credential endpoints (POST /api/auth/*,
// POST /admin/auth/login) at 20 req/min per source IP. Failure is a plain 429 —
// never leak which limit tripped.
func AuthRateLimit(next http.Handler) http.Handler {
	lim := newAuthLimiter(20, time.Minute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if r.Method == http.MethodPost &&
			(strings.HasPrefix(p, "/api/auth/") || p == "/admin/auth/login") {
			if !lim.allow(ClientIP(r), time.Now()) {
				shared.NewErr("RATE_LIMITED", "请求过于频繁，请稍后再试", http.StatusTooManyRequests).Write(w)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// BodyLimit caps the request body at n bytes (F15). Oversized reads surface as
// *http.MaxBytesError inside DecodeJSON and map to the handlers' 400 path.
func BodyLimit(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}

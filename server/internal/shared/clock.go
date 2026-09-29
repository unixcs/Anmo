package shared

import (
	"io"
	"log/slog"
	"time"
)

// nowShanghaiLoc is the single business timezone (plan §69: Asia/Shanghai).
var shanghaiLoc = time.FixedZone("+08:00", 8*3600)

// NowShanghai returns the current business wall-clock time.
func NowShanghai() time.Time {
	return time.Now().In(shanghaiLoc)
}

// Shanghai returns t expressed in the business timezone.
func Shanghai(t time.Time) time.Time {
	return t.In(shanghaiLoc)
}

// NewNopLogger returns a discard logger for tests/tools.
func NewNopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

package shared

import (
	"context"
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

// LockHandle releases a named lock.
type LockHandle func()

// NamedLocker is implemented by the connection pool: it takes a MySQL named
// lock on a DEDICATED pooled connection and returns a release func. Unlike an
// in-transaction GET_LOCK, the lock survives until explicitly released —
// callers MUST release only after their transaction has committed (D5).
type NamedLocker interface {
	NamedLock(ctx context.Context, name string, timeoutSecs int) (LockHandle, error)
}

package shared

import (
	"errors"

	"modernc.org/sqlite"
)

// 存储方言辅助（2026-09-28 MySQL→SQLite）：业务代码通过这两个谓词识别
// 可恢复的引擎错误，不直接 import 驱动类型。

// IsDupKey reports a unique-constraint violation (any unique index or
// primary key). SQLite extended codes: 2067 = SQLITE_CONSTRAINT_UNIQUE,
// 1555 = SQLITE_CONSTRAINT_PRIMARYKEY.
func IsDupKey(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	return se.Code() == 2067 || se.Code() == 1555
}

// IsBusy reports SQLITE_BUSY / SQLITE_LOCKED — the transaction was not
// applied (SQLite rolls it back), so it is safe to surface as a retryable
// conflict. Under _txlock=immediate this fires when busy_timeout elapses
// while another write transaction holds the single write lock.
func IsBusy(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	return se.Code() == 5 || se.Code() == 6 // SQLITE_BUSY / SQLITE_LOCKED
}

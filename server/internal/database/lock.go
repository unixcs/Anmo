package database

import (
	"context"
	"database/sql"

	"anmo/server/internal/shared"
)

// NamedLock implements shared.NamedLocker: the MySQL named lock is taken on a
// dedicated pooled connection (not the transaction's), so it stays held until
// the returned release func runs — after the caller's transaction commits.
func (p *Pool) NamedLock(ctx context.Context, name string, timeoutSecs int) (shared.LockHandle, error) {
	conn, err := p.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	var got sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, ?)`, name, timeoutSecs).Scan(&got); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !got.Valid || got.Int64 != 1 {
		_ = conn.Close()
		return nil, shared.Conflict("LOCK_BUSY", "操作繁忙，请重试")
	}
	released := false
	return func() {
		if !released {
			released = true
			_, _ = conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK(?)`, name)
			_ = conn.Close()
		}
	}, nil
}

package shared

import (
	"context"
	"fmt"
)

// NextSeq atomically increments a named counter and returns the new value.
// The row lock (taken by UPDATE) is held until the surrounding transaction
// commits, so concurrent writers block and then read committed values —
// sequential, race-free business numbers. The INSERT IGNORE first ensures the
// row exists for new names (a concurrent insert is waited on by MySQL's
// duplicate-key check, then ignored).
func NextSeq(ctx context.Context, tx Tx, name string) (int64, error) {
	if _, err := tx.ExecContext(ctx,
		`INSERT IGNORE INTO sys_sequence (name, value) VALUES (?, 0)`, name); err != nil {
		return 0, Server("SEQ_INIT", err)
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE sys_sequence SET value = value + 1 WHERE name = ?`, name)
	if err != nil {
		return 0, Server("SEQ_UPDATE", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, Server("SEQ_MISSING", fmt.Errorf("sequence row vanished: %s", name))
	}
	var v int64
	if err := tx.QueryRowContext(ctx,
		`SELECT value FROM sys_sequence WHERE name = ?`, name).Scan(&v); err != nil {
		return 0, Server("SEQ_READ", err)
	}
	return v, nil
}

// SeqDateName builds a per-day sequence name.
func SeqDateName(prefix, yyyymmdd string) string {
	return fmt.Sprintf("%s:%s", prefix, yyyymmdd)
}

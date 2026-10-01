package shared

import (
	"context"
	"crypto/rand"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewID returns a new ULID string (CHAR(26) columns everywhere).
func NewID() string {
	t := time.Now().UTC()
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

// Querier abstracts anything that can run SQL: both the pool (*sql.DB via
// database.Pool) and a transaction (*sql.Tx) satisfy it. Repo functions take
// a Querier so they work inside or outside a transaction.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Tx is a transaction handle; *sql.Tx satisfies it. Module api.go functions
// that must join a caller-owned transaction accept a Tx (D6: the settlement
// module opens the tx and passes it down; callees never open nested
// transactions).
type Tx interface {
	Querier
	Commit() error
	Rollback() error
}

// DB is the connection pool; database.Pool satisfies it.
type DB interface {
	Querier
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	PingContext(ctx context.Context) error
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)
	SetConnMaxLifetime(d time.Duration)
}

// RunInTx executes fn inside a database transaction, committing on nil error
// and rolling back on any error. The rollback error never masks fn's error.
// SQLITE_BUSY on BEGIN/Commit (all writers queue at BEGIN IMMEDIATE; a busy
// timeout expiry surfaces here, F7) is retryable business contention → 409
// LOCK_RETRY, not a 500.
func RunInTx(ctx context.Context, db DB, fn func(tx Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		if IsBusy(err) {
			return Conflict("LOCK_RETRY", "操作繁忙，请重试")
		}
		return Server("TX_BEGIN", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		if IsBusy(err) {
			return Conflict("LOCK_RETRY", "操作繁忙，请重试")
		}
		return Server("TX_COMMIT", err)
	}
	return nil
}

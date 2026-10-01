// Package database owns the SQLite pool and schema migrations.
//
// 存储层 2026-09-28 由 MySQL 迁移至 SQLite（单文件、嵌入纯 Go 驱动，无 CGO）：
//   - 并发模型：WAL + busy_timeout + 所有事务 BEGIN IMMEDIATE（_txlock=immediate）。
//     单写者模型天然串行化写事务，取代 MySQL 的 GET_LOCK 日历锁与 SELECT ... FOR UPDATE
//     （原 D5/D18 锁语义由"整个写事务在 BEGIN 时排队获得写锁"等价承接）。
//   - 时间约定：DATETIME 一律存 Asia/Shanghai 墙上时间字符串 "YYYY-MM-DD HH:MM:SS"；
//     写入（_time_format=datetime）与读取解释（_timezone=Asia/Shanghai）由驱动完成，
//     time.Time 绑定/扫描代码无需感知时区。
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	_ "time/tzdata" // 嵌入时区库：容器/精简镜像内 _timezone=Asia/Shanghai 仍可用

	_ "modernc.org/sqlite"

	"anmo/server/internal/shared"
)

// Pool adapts *sql.DB to shared.DB (BeginTx must return shared.Tx).
type Pool struct {
	*sql.DB
}

func (p *Pool) BeginTx(ctx context.Context, opts *sql.TxOptions) (shared.Tx, error) {
	tx, err := p.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

var (
	_ shared.DB = (*Pool)(nil)
	_ shared.Tx = (*sql.Tx)(nil)
)

// dsn builds the sqlite DSN with the business-wide connection defaults.
// _txlock=immediate makes every transaction take the write lock at BEGIN,
// so a caller never holds locks through a deferred upgrade (no SQLITE_BUSY
// mid-transaction deadlocks); contention beyond busy_timeout surfaces as a
// retryable SQLITE_BUSY.
func dsn(path string) string {
	return "file:" + path +
		"?_txlock=immediate" +
		"&_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(WAL)" +
		// FULL（第九批审查）：单店写频率极低，fsync 开销可忽略；NORMAL 在断电时
		// 可能丢最近已确认交易（收款/核销账本），这里宁可慢也要每笔落盘。
		"&_pragma=synchronous(FULL)" +
		"&_pragma=foreign_keys(1)" +
		"&_timezone=Asia/Shanghai" +
		"&_time_format=datetime"
}

// Open opens (creating if needed) the SQLite database file and returns the
// pool as the shared.DB used by all modules.
func Open(path string) (shared.DB, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// WAL allows readers during a write transaction; keep a modest pool —
	// SQLite serializes writers inside the engine.
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0) // file-backed connection: reuse, no churn
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &Pool{DB: db}, nil
}

// Backup writes a consistent snapshot of the live database to dest via
// `VACUUM INTO` (safe while the server is serving; WAL readers don't block)
// and then verifies the snapshot with integrity_check. Used by the host-side
// cron script (`docker exec anmo-server /app/anmo -backup /backups/...`).
func Backup(db shared.DB, dest string) error {
	if _, err := db.ExecContext(context.Background(),
		`VACUUM INTO ?`, dest); err != nil {
		return fmt.Errorf("vacuum into %s: %w", dest, err)
	}
	check, err := sql.Open("sqlite", "file:"+dest+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return fmt.Errorf("open snapshot: %w", err)
	}
	defer check.Close()
	var verdict string
	if err := check.QueryRowContext(context.Background(),
		`PRAGMA integrity_check`).Scan(&verdict); err != nil {
		return fmt.Errorf("integrity_check: %w", err)
	}
	if verdict != "ok" {
		return fmt.Errorf("integrity_check of %s: %s", dest, verdict)
	}
	return nil
}

// Migrate applies every unapplied migrations/*.sql in filename order and
// records them in schema_migrations. Re-running is a no-op. Each file runs
// inside one transaction (SQLite DDL is transactional, unlike MySQL): a
// half-applied file rolls back, so a re-run can never hit "table exists".
// A failure aborts startup (fail-fast).
func Migrate(ctx context.Context, db shared.DB, dir string) (applied []string, err error) {
	if _, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
		   filename   VARCHAR(255) NOT NULL PRIMARY KEY,
		   applied_at DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours'))
		 )`); err != nil {
		return nil, fmt.Errorf("ensure schema_migrations: %w", err)
	}
	files, err := migrationFiles(dir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		var one string
		if err := db.QueryRowContext(ctx,
			`SELECT filename FROM schema_migrations WHERE filename = ?`, f).Scan(&one); err != nil {
			if err != sql.ErrNoRows {
				return nil, fmt.Errorf("check migration %s: %w", f, err)
			}
			body, err := readMigration(dir, f)
			if err != nil {
				return nil, err
			}
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("begin migration %s: %w", f, err)
			}
			if _, err := tx.ExecContext(ctx, body); err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("apply migration %s: %w", f, err)
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO schema_migrations (filename) VALUES (?)`, f); err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("record migration %s: %w", f, err)
			}
			if err := tx.Commit(); err != nil {
				return nil, fmt.Errorf("commit migration %s: %w", f, err)
			}
			applied = append(applied, f)
		}
	}
	return applied, nil
}

// Package database owns the MySQL pool and schema migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

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

// Open connects to MySQL with business-wide defaults and returns the pool as
// the shared.DB used by all modules.
func Open(dsn string) (shared.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return &Pool{DB: db}, nil
}

// Migrate applies every unapplied migrations/*.sql in filename order and
// records them in schema_migrations. Re-running is a no-op. DDL cannot run
// inside a transaction in MySQL, so each file is executed once and recorded
// immediately; a failure aborts startup (fail-fast).
func Migrate(ctx context.Context, db shared.DB, dir string) (applied []string, err error) {
	if _, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
		   filename   VARCHAR(255) NOT NULL PRIMARY KEY,
		   applied_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
		 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
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
			if _, err := db.ExecContext(ctx, body); err != nil {
				return nil, fmt.Errorf("apply migration %s: %w", f, err)
			}
			if _, err := db.ExecContext(ctx,
				`INSERT INTO schema_migrations (filename) VALUES (?)`, f); err != nil {
				return nil, fmt.Errorf("record migration %s: %w", f, err)
			}
			applied = append(applied, f)
		}
	}
	return applied, nil
}

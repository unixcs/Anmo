// mysql2sqlite — 一次性数据迁移工具：MySQL(生产/开发库) → SQLite。
//
// 独立 go.mod（嵌套模块）：go-sql-driver/mysql 只被本工具引用，
// 主服务模块自 2026-09-28 起完全不含 MySQL 代码与依赖。
//
// 用法（在 yun1 上对生产库执行时，DSN 用 127.0.0.1:33306）：
//
//	go run . -mysql "anmo:PASS@tcp(127.0.0.1:33306)/anmo?parseTime=true&loc=Asia%2FShanghai" \
//	         -sqlite /opt/anmo/data/anmo.db -migrations /opt/anmo/migrations
//
// 流程：建全新 SQLite → 跑 migrations/001-013 建终态 schema → 按外键安全顺序
// 逐表复制（生成列由 SQLite 自动计算，DATE/DATETIME 列统一格式化为
// Asia/Shanghai 墙上时间字符串）→ 行数校验 + foreign_key_check + 完整性检查。
// 迁移期间应停止 MySQL 侧写入（停 server 容器），保证快照一致。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

// copyOrder — 外键依赖安全的复制顺序（父表在前）。
var copyOrder = []string{
	"identity_role", "identity_permission", "identity_user",
	"member", "member_tag", "member_tag_rel",
	"service_category", "service",
	"card_template", "card_service_rule", "member_card", "card_transaction",
	"appointment", "appointment_service", "appointment_status_log", "appointment_closure",
	"payment", "redemption", "redemption_reversal",
	"content_page_config", "content_banner", "content_announcement", "content_system_setting",
	"ops_operation_log", "ops_insight_snapshot",
	"sys_sequence",
}

type colInfo struct {
	Name     string
	DataType string
}

func mysqlColumns(db *sql.DB, table string) ([]colInfo, error) {
	rows, err := db.Query(
		`SELECT column_name, data_type FROM information_schema.columns
		 WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []colInfo
	for rows.Next() {
		var c colInfo
		if err := rows.Scan(&c.Name, &c.DataType); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// sqliteWriteableColumns — PRAGMA table_xinfo 中 hidden=0 的普通列
// （生成列 hidden=2/3，由 SQLite 自动计算，禁止写入）。
func sqliteWriteableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query(`SELECT name FROM pragma_table_xinfo(?) WHERE hidden = 0`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[strings.ToLower(n)] = true
	}
	return out, rows.Err()
}

// convertValue 按列类型归一时间值：MySQL 驱动给出 time.Time 的 DATE/DATETIME
// 列统一转为 Asia/Shanghai 墙上时间字符串（DATE 只留日期），与业务 SQL 的
// 字符串比较语义完全一致。
func convertValue(dataType string, v any, loc *time.Location) any {
	switch t := v.(type) {
	case time.Time:
		t = t.In(loc)
		switch dataType {
		case "date":
			return t.Format("2006-01-02")
		case "datetime", "timestamp":
			return t.Format("2006-01-02 15:04:05")
		}
		return t.Format("2006-01-02 15:04:05")
	case []byte:
		return string(t)
	}
	return v
}

func main() {
	mysqlDSN := flag.String("mysql", "", "source MySQL DSN (parseTime=true&loc=Asia%2FShanghai)")
	sqlitePath := flag.String("sqlite", "", "target SQLite file (must not exist)")
	migDir := flag.String("migrations", "", "migrations dir (SQLite schema source)")
	flag.Parse()
	if *mysqlDSN == "" || *sqlitePath == "" || *migDir == "" {
		flag.Usage()
		os.Exit(2)
	}
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fatal("load timezone: %v", err)
	}
	if _, err := os.Stat(*sqlitePath); err == nil {
		fatal("target %s already exists — refusing to overwrite", *sqlitePath)
	}
	_ = os.MkdirAll(filepath.Dir(*sqlitePath), 0o755)

	src, err := sql.Open("mysql", *mysqlDSN)
	if err != nil {
		fatal("open mysql: %v", err)
	}
	src.SetMaxOpenConns(2)
	if err := src.Ping(); err != nil {
		fatal("ping mysql: %v", err)
	}
	var serverTZ string
	_ = src.QueryRow(`SELECT @@session.time_zone`).Scan(&serverTZ)
	fmt.Printf("[source] mysql connected (session tz=%s)\n", serverTZ)

	dst, err := sql.Open("sqlite", "file:"+*sqlitePath+"?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)&_timezone=Asia/Shanghai&_time_format=datetime")
	if err != nil {
		fatal("open sqlite: %v", err)
	}
	if err := applyMigrations(dst, *migDir); err != nil {
		fatal("migrate sqlite schema: %v", err)
	}
	fmt.Printf("[schema] migrations applied from %s\n", *migDir)

	report := map[string][2]int{} // table -> [mysql, sqlite]
	bad := 0
	for _, table := range copyOrder {
		srcCols, err := mysqlColumns(src, table)
		if err != nil {
			fatal("inspect %s: %v", table, err)
		}
		if len(srcCols) == 0 {
			fmt.Printf("[warn] table %s missing in MySQL, skipped\n", table)
			continue
		}
		dstCols, err := sqliteWriteableColumns(dst, table)
		if err != nil {
			fatal("pragma %s: %v", table, err)
		}
		var names []string
		var types []string
		for _, c := range srcCols {
			if dstCols[strings.ToLower(c.Name)] {
				names = append(names, c.Name)
				types = append(types, c.DataType)
			} else {
				fmt.Printf("[warn] %s.%s skipped (not a plain column in SQLite schema)\n", table, c.Name)
			}
		}
		var mcount, scount int
		_ = src.QueryRow(`SELECT COUNT(*) FROM ` + mquote(table)).Scan(&mcount)
		err = copyTable(src, dst, table, names, types, shanghai)
		if err != nil {
			fatal("copy %s: %v", table, err)
		}
		if err := dst.QueryRow(`SELECT COUNT(*) FROM ` + quote(table)).Scan(&scount); err != nil {
			fatal("count %s: %v", table, err)
		}
		report[table] = [2]int{mcount, scount}
		if mcount != scount {
			bad++
			fmt.Printf("[MISMATCH] %-26s mysql=%d sqlite=%d\n", table, mcount, scount)
		} else {
			fmt.Printf("[ok]       %-26s rows=%d\n", table, scount)
		}
	}

	fmt.Println("\n═══ 校验 ═══")
	var fkIssues int
	if err := dst.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&fkIssues); err != nil {
		fatal("fk check: %v", err)
	}
	fmt.Printf("foreign_key_check violations: %d\n", fkIssues)
	var integ string
	if err := dst.QueryRow(`PRAGMA integrity_check`).Scan(&integ); err != nil || integ != "ok" {
		fatal("integrity_check: %s (%v)", integ, err)
	}
	fmt.Println("integrity_check: ok")
	// 不变量抽查：一预约最多一笔 VALID 收款 / 一预约最多一笔有效核销
	for name, q := range map[string]string{
		"多笔 VALID 收款预约数": `SELECT COUNT(*) FROM (SELECT appointment_id FROM payment WHERE status='VALID' AND appointment_id IS NOT NULL GROUP BY appointment_id HAVING COUNT(*)>1)`,
		"多笔有效核销预约数":     `SELECT COUNT(*) FROM (SELECT appointment_id FROM redemption WHERE status='SUCCESS' AND appointment_id IS NOT NULL GROUP BY appointment_id HAVING COUNT(*)>1)`,
	} {
		var n int
		if err := dst.QueryRow(q).Scan(&n); err != nil {
			fatal("invariant %s: %v", name, err)
		}
		status := "ok"
		if n != 0 {
			status = "VIOLATED"
			bad++
		}
		fmt.Printf("invariant %-24s : %d (%s)\n", name, n, status)
	}

	if bad != 0 || fkIssues != 0 {
		fmt.Printf("\n❌ 迁移校验未通过（%d 项异常）— 保留 SQLite 文件供检查，不要投入服务\n", bad)
		os.Exit(1)
	}
	fmt.Println("\n✅ 迁移完成且校验通过：", *sqlitePath)
}

func copyTable(src, dst *sql.DB, table string, names, types []string, loc *time.Location) error {
	if len(names) == 0 {
		return nil
	}
	sq := make([]string, len(names))
	ph := make([]string, len(names))
	for i := range names {
		sq[i] = mquote(names[i])
		ph[i] = "?"
	}
	rows, err := src.Query(`SELECT ` + strings.Join(sq, ", ") + ` FROM ` + mquote(table))
	if err != nil {
		return err
	}
	defer rows.Close()
	vals := make([]any, len(names))
	ptrs := make([]any, len(names))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	tx, err := dst.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO ` + quote(table) +
		` (` + strings.Join(sq, ", ") + `) VALUES (` + strings.Join(ph, ", ") + `)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	n := 0
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return err
		}
		for i := range vals {
			vals[i] = convertValue(types[i], vals[i], loc)
		}
		if _, err := stmt.Exec(vals...); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return fmt.Errorf("row %d: %w", n+1, err)
		}
		n++
	}
	if err := rows.Err(); err != nil {
		_ = stmt.Close()
		_ = tx.Rollback()
		return err
	}
	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// applyMigrations applies NNN_*.sql from dir in filename order (same contract
// as server/internal/database.Migrate, kept self-contained for this tool).
func applyMigrations(db *sql.DB, dir string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
	   filename VARCHAR(255) NOT NULL PRIMARY KEY,
	   applied_at DATETIME NOT NULL DEFAULT (datetime('now','+8 hours')))`); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, f := range files {
		var one string
		if err := db.QueryRow(`SELECT filename FROM schema_migrations WHERE filename = ?`, f).Scan(&one); err == nil {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply %s: %w", f, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, f); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// mquote quotes a MySQL identifier (backtick); quote() is the SQLite side.
func mquote(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quote(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
	os.Exit(1)
}

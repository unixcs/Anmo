// anmo — 个人到店按摩店服务管理系统 V1 后端。
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"anmo/server/internal/app"
	"anmo/server/internal/config"
	"anmo/server/internal/database"
	"anmo/server/internal/logger"
)

func main() {
	cfgPath := flag.String("config", "", "path to config.yaml")
	migrateOnly := flag.Bool("migrate", false, "apply migrations and exit")
	backupTo := flag.String("backup", "", "write a consistent snapshot (VACUUM INTO) to this path and verify it, then exit")
	daily := flag.Bool("daily", false, "run the daily maintenance sweep (expiry sweep + insights) and exit")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		os.Stderr.WriteString("load config: " + err.Error() + "\n")
		os.Exit(1)
	}
	log := logger.New(cfg.Log.Level)

	// F2（第九批审查）：微信凭据缺省 = dev 兜底 openid（D24）。本地无所谓，
	// 生产若仍为空则顾客身份全走 dev:<code>——启动时必须喊出来。
	if cfg.Wx.AppID == "" || cfg.Wx.Secret == "" {
		log.Warn("wx credentials not configured: wx login falls back to dev openids (dev:<code>); set ANMO_WX_APPID / ANMO_WX_SECRET before production")
	}

	// F4（第九批审查）：库文件缺失/为空 = 大概率是挂卷丢了或路径配错。
	// 静默新建空库会让"数据全没了"伪装成"系统刚上线"。首装显式放行。
	if *backupTo == "" && !*migrateOnly && !*daily && cfg.Database.Path != ":memory:" {
		if info, err := os.Stat(cfg.Database.Path); err != nil || info.Size() == 0 {
			if os.Getenv("ANMO_ALLOW_EMPTY_DB") != "1" {
				log.Error("database file missing or empty — refusing to boot a fresh store",
					"path", cfg.Database.Path,
					"hint", "first install: set ANMO_ALLOW_EMPTY_DB=1 (or run -migrate); wrong volume/path: fix the mount")
				os.Exit(1)
			}
		}
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		log.Error("open database", "err", err)
		os.Exit(1)
	}

	if *backupTo != "" {
		if err := database.Backup(db, *backupTo); err != nil {
			log.Error("backup", "err", err)
			os.Exit(1)
		}
		log.Info("backup ok", "path", *backupTo)
		return
	}

	if *daily {
		out, err := app.RunDaily(db, cfg, log)
		if err != nil {
			log.Error("daily sweep", "err", err)
			os.Exit(1)
		}
		log.Info("daily sweep done", "result", out)
		return
	}

	applied, err := database.Migrate(context.Background(), db, "migrations")
	if err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}
	if len(applied) > 0 {
		log.Info("migrations applied", "count", len(applied))
	}
	if *migrateOnly {
		return
	}

	if err := app.SeedIdentity(db, cfg, log); err != nil {
		log.Error("seed identity", "err", err)
		os.Exit(1)
	}

	handler := app.Build(db, cfg, log)

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Info("listening", "addr", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("shutdown complete")
}

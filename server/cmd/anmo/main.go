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
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		os.Stderr.WriteString("load config: " + err.Error() + "\n")
		os.Exit(1)
	}
	log := logger.New(cfg.Log.Level)

	db, err := database.Open(cfg.MySQL.DSN)
	if err != nil {
		log.Error("open database", "err", err)
		os.Exit(1)
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

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config_invalid", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, err := store.Open(ctx, cfg.MongoDBURI, cfg.MongoDBDatabase)
	if err != nil {
		log.Error("database_connect_failed", "error", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())
	policy := store.RetentionPolicy{AlertDays: cfg.AlertRetentionDays, BriefingDays: cfg.BriefingRetentionDays, AuditDays: cfg.AuditRetentionDays}
	run := func() bool {
		result, runErr := db.RunRetention(ctx, time.Now().UTC(), policy)
		if runErr != nil {
			log.Error("retention_failed", "error", runErr)
			return false
		}
		log.Info("retention_complete", "result", result)
		return true
	}
	if !run() || os.Getenv("MAINTENANCE_RUN_ONCE") == "true" {
		return
	}
	ticker := time.NewTicker(cfg.MaintenanceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

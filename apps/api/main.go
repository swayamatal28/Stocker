package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/httpapi"
	"github.com/stocker-app/stocker/internal/ingest"
	"github.com/stocker-app/stocker/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config_invalid", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := store.Open(ctx, cfg.MongoDBURI, cfg.MongoDBDatabase)
	if err != nil {
		log.Error("database_connect_failed", "error", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())
	if err := db.MigrateLegacyIndexNames(ctx); err != nil {
		log.Error("legacy_index_migration_failed", "error", err)
		os.Exit(1)
	}
	if err := db.MigratePhase4Market(ctx); err != nil {
		log.Error("phase4_market_migration_failed", "error", err)
		os.Exit(1)
	}
	if err := db.EnsureIndexes(ctx); err != nil {
		log.Error("index_initialization_failed", "error", err)
		os.Exit(1)
	}
	if err := db.MigratePhase2News(ctx); err != nil {
		log.Error("phase2_news_migration_failed", "error", err)
		os.Exit(1)
	}
	if err := db.MigratePhase3Intelligence(ctx); err != nil {
		log.Error("phase3_intelligence_migration_failed", "error", err)
		os.Exit(1)
	}
	if err := db.Seed(ctx); err != nil {
		log.Error("seed_failed", "error", err)
		os.Exit(1)
	}
	if cfg.MockProviders {
		policy := ingest.SourcePolicy{
			PollInterval: cfg.IngestPollInterval, Timeout: cfg.IngestTimeout,
			RequestsPerMinute: cfg.IngestRequestsPerMinute, RawRetention: time.Duration(cfg.RawRetention) * 24 * time.Hour,
			Attribution: cfg.IngestAttribution, Licence: cfg.IngestLicence, TermsURL: cfg.IngestTermsURL,
			AutomatedAccessAllowed: true, RobotsChecked: true,
		}
		processor := ingest.NewProcessor(db, nil, nil, log, policy)
		if _, err := processor.RunOnce(ctx, ingest.MockAdapter{}, ingest.Cursor{}); err != nil {
			log.Error("mock_news_seed_failed", "error", err)
			os.Exit(1)
		}
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Error("redis_url_invalid", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		if cfg.RedisRequired {
			log.Error("redis_connect_failed", "error", err)
			os.Exit(1)
		}
		log.Warn("redis_unavailable_optional_in_development", "error", err)
	}
	api := httpapi.New(cfg, db, rdb, log)
	defer api.Close()
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 0, IdleTimeout: 90 * time.Second}
	go func() {
		log.Info("api_started", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("api_failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Error("shutdown_failed", "error", err)
	}
	log.Info("api_stopped")
}

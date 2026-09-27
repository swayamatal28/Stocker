package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/ingest"
	"github.com/stocker-app/stocker/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config_invalid", "error", err)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, err := store.Open(ctx, cfg.MongoDBURI, cfg.MongoDBDatabase)
	if err != nil {
		log.Error("db_connect_failed", "error", err)
		return
	}
	defer db.Close(context.Background())
	if err := db.MigrateLegacyIndexNames(ctx); err != nil {
		log.Error("legacy_index_migration_failed", "error", err)
		return
	}
	if err := db.MigratePhase4Market(ctx); err != nil {
		log.Error("phase4_market_migration_failed", "error", err)
		os.Exit(1)
	}
	if err := db.EnsureIndexes(ctx); err != nil {
		log.Error("index_initialization_failed", "error", err)
		return
	}
	if err := db.MigratePhase2News(ctx); err != nil {
		log.Error("phase2_news_migration_failed", "error", err)
		return
	}
	if err := db.MigratePhase3Intelligence(ctx); err != nil {
		log.Error("phase3_intelligence_migration_failed", "error", err)
		return
	}
	if err := db.Seed(ctx); err != nil {
		log.Error("seed_failed", "error", err)
		return
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Error("redis_url_invalid", "error", err)
		return
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("redis_required_for_ingestion", "error", err)
		return
	}
	if err := rdb.XGroupCreateMkStream(ctx, "stocker:news.created", "analysis-workers", "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Error("redis_stream_group_initialization_failed", "error", err)
		return
	}
	defaultPolicy := ingest.SourcePolicy{
		PollInterval: cfg.IngestPollInterval, Timeout: cfg.IngestTimeout,
		RequestsPerMinute: cfg.IngestRequestsPerMinute, RawRetention: time.Duration(cfg.RawRetention) * 24 * time.Hour,
		Attribution: cfg.IngestAttribution, Licence: cfg.IngestLicence, TermsURL: cfg.IngestTermsURL,
		AutomatedAccessAllowed: cfg.IngestAutomatedAccessAllowed, PolicyExpiresAt: cfg.IngestPolicyExpiresAt,
		RobotsChecked: true,
	}
	if cfg.IngestReplayDeadOnStart {
		replayed, replayErr := db.ReplayDeadNewsEvents(ctx, 100)
		if replayErr != nil {
			log.Error("dead_letter_replay_failed", "error", replayErr)
			return
		}
		log.Info("dead_letter_replay_complete", "count", replayed)
	}
	configured := []ingest.ConfiguredFeed{}
	if cfg.MockProviders {
		if err := defaultPolicy.Validate(time.Now().UTC()); err != nil {
			log.Error("source_policy_invalid", "source", cfg.IngestSourceID, "error", err)
			return
		}
		configured = append(configured, ingest.ConfiguredFeed{Adapter: ingest.MockAdapter{}, Policy: defaultPolicy})
	} else if strings.TrimSpace(cfg.IngestSourcesJSON) != "" {
		configured, err = ingest.ParseFeedSources(cfg.IngestSourcesJSON)
		if err != nil {
			log.Error("source_configuration_invalid", "error", err)
			return
		}
	} else {
		if err := defaultPolicy.Validate(time.Now().UTC()); err != nil {
			log.Error("source_policy_invalid", "source", cfg.IngestSourceID, "error", err)
			return
		}
		adapter, adapterErr := ingest.NewRSSAdapter(ingest.RSSAdapterConfig{
			SourceID: cfg.IngestSourceID, FeedURL: cfg.IngestFeedURL, Attribution: cfg.IngestAttribution,
			Licence: cfg.IngestLicence, Language: cfg.IngestLanguage, Official: cfg.IngestOfficial,
		})
		if adapterErr != nil {
			log.Error("source_adapter_invalid", "source", cfg.IngestSourceID, "error", adapterErr)
			return
		}
		configured = append(configured, ingest.ConfiguredFeed{Adapter: adapter, Policy: defaultPolicy})
	}
	publisher := ingest.NewRedisPublisher(rdb)
	var workers sync.WaitGroup
	for _, source := range configured {
		source := source
		workers.Add(1)
		go func() {
			defer workers.Done()
			runSource(ctx, db, rdb, publisher, log, source)
		}()
	}
	log.Info("ingestion_sources_started", "count", len(configured))
	<-ctx.Done()
	workers.Wait()
	log.Info("worker_stopped")
}

func runSource(ctx context.Context, db *store.Mongo, rdb *redis.Client, publisher ingest.Publisher, log *slog.Logger, source ingest.ConfiguredFeed) {
	adapter, policy := source.Adapter, source.Policy
	processor := ingest.NewProcessor(db, rdb, publisher, log, policy)
	cursor, err := db.SourceCursor(ctx, adapter.ID())
	if err != nil {
		log.Error("source_cursor_load_failed", "source", adapter.ID(), "error", err)
		return
	}
	ticker := time.NewTicker(policy.PollInterval)
	defer ticker.Stop()
	for {
		runCtx, cancel := context.WithTimeout(ctx, policy.Timeout)
		next, err := processor.RunOnce(runCtx, adapter, cursor)
		cancel()
		if next.Value != "" || !next.Since.IsZero() {
			cursor = next
			if saveErr := db.SaveSourceCursor(ctx, adapter.ID(), cursor); saveErr != nil {
				log.Warn("source_cursor_save_failed", "source", adapter.ID(), "error", saveErr)
			}
		}
		if err != nil {
			log.Warn("ingestion_cycle_failed", "source", adapter.ID(), "error", err)
		} else {
			log.Info("ingestion_cycle_complete", "source", adapter.ID())
		}
		select {
		case <-ctx.Done():
			log.Info("ingestion_source_stopped", "source", adapter.ID())
			return
		case <-ticker.C:
		}
	}
}

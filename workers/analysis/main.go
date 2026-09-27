package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/ai"
	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/ingest"
	"github.com/stocker-app/stocker/internal/intelligence"
	"github.com/stocker-app/stocker/internal/store"
)

const streamName = "stocker:news.created"
const consumerGroup = "analysis-workers"

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
	if err := db.MigratePhase3Intelligence(ctx); err != nil {
		log.Error("phase3_migration_failed", "error", err)
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
		log.Error("redis_required_for_analysis", "error", err)
		return
	}
	if err := rdb.XGroupCreateMkStream(ctx, streamName, consumerGroup, "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Error("analysis_group_initialization_failed", "error", err)
		return
	}

	var provider intelligence.Provider = ai.LocalProvider{}
	if cfg.AIProvider == "http-json" {
		provider, err = ai.NewHTTPProvider(cfg.AIEndpoint, cfg.AIAPIKey, cfg.AIModel, cfg.AIRequestTimeout, cfg.AIMaxOutputBytes, cfg.AIRequestCostCents)
		if err != nil {
			log.Error("ai_provider_invalid", "error", err)
			return
		}
	}
	processor := intelligence.NewProcessor(db, provider, ingest.NewRedisPublisher(rdb), intelligence.Config{PromptVersion: cfg.AIPromptVersion, SchemaVersion: cfg.AISchemaVersion, MaxInputChars: cfg.AIMaxInputChars, DailyBudgetCents: cfg.AIDailyBudgetCents, MaxAttempts: cfg.AnalysisMaxAttempts})
	log.Info("analysis_worker_started", "provider", provider.Name(), "model", provider.Model(), "consumer", cfg.AnalysisConsumer)
	recoverPending(ctx, rdb, processor, cfg, log)
	for ctx.Err() == nil {
		streams, readErr := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: consumerGroup, Consumer: cfg.AnalysisConsumer, Streams: []string{streamName, ">"}, Count: 10, Block: 5 * time.Second}).Result()
		if readErr != nil {
			if errors.Is(readErr, redis.Nil) || ctx.Err() != nil {
				continue
			}
			log.Warn("analysis_stream_read_failed", "error", readErr)
			continue
		}
		for _, streamResult := range streams {
			processMessages(ctx, rdb, processor, cfg, log, streamResult.Messages)
		}
	}
}

func recoverPending(ctx context.Context, rdb *redis.Client, processor *intelligence.Processor, cfg config.Config, log *slog.Logger) {
	start := "0-0"
	for ctx.Err() == nil {
		messages, next, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: streamName, Group: consumerGroup, Consumer: cfg.AnalysisConsumer, MinIdle: time.Minute, Start: start, Count: 20}).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				log.Warn("analysis_pending_recovery_failed", "error", err)
			}
			return
		}
		if len(messages) == 0 {
			return
		}
		processMessages(ctx, rdb, processor, cfg, log, messages)
		if next == "0-0" || next == start {
			return
		}
		start = next
	}
}

func processMessages(ctx context.Context, rdb *redis.Client, processor *intelligence.Processor, cfg config.Config, log *slog.Logger, messages []redis.XMessage) {
	for _, message := range messages {
		payload, ok := message.Values["payload"].(string)
		if !ok {
			log.Warn("analysis_event_invalid", "message_id", message.ID)
			_ = rdb.XAck(ctx, streamName, consumerGroup, message.ID).Err()
			continue
		}
		var event domain.NewsEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil || event.ArticleID == "" {
			log.Warn("analysis_event_decode_failed", "message_id", message.ID, "error", err)
			_ = rdb.XAck(ctx, streamName, consumerGroup, message.ID).Err()
			continue
		}
		var processErr error
		for attempt := 1; attempt <= cfg.AnalysisMaxAttempts; attempt++ {
			_, processErr = processor.Process(ctx, event.ArticleID)
			if processErr == nil {
				break
			}
			_ = processor.RecordFailure(ctx, event.ArticleID, processErr)
			if attempt < cfg.AnalysisMaxAttempts {
				timer := time.NewTimer(time.Duration(min(attempt, 5)) * time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}
		if processErr != nil {
			log.Error("analysis_event_dead", "article_id", event.ArticleID, "error", processErr)
		} else {
			log.Info("analysis_complete", "article_id", event.ArticleID)
		}
		if err := rdb.XAck(ctx, streamName, consumerGroup, message.ID).Err(); err != nil {
			log.Warn("analysis_ack_failed", "message_id", message.ID, "error", err)
		}
	}
}

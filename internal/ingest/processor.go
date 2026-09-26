package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/domain"
)

const (
	nearDuplicateWindow    = 72 * time.Hour
	nearDuplicateThreshold = 0.82
	maxPublishAttempts     = 5
)

type Processor struct {
	repo      Repository
	redis     *redis.Client
	publisher Publisher
	log       *slog.Logger
	policy    SourcePolicy
	retry     RetryPolicy

	mu           sync.Mutex
	failures     int
	circuitUntil time.Time
}

func NewProcessor(repo Repository, rdb *redis.Client, pub Publisher, log *slog.Logger, policy SourcePolicy) *Processor {
	return &Processor{
		repo: repo, redis: rdb, publisher: pub, log: log, policy: policy,
		retry: RetryPolicy{MaxAttempts: 4, BaseDelay: 500 * time.Millisecond, MaxDelay: 15 * time.Second},
	}
}

func (p *Processor) RunOnce(ctx context.Context, a Adapter, cursor Cursor) (Cursor, error) {
	if err := p.policy.Validate(time.Now().UTC()); err != nil {
		return cursor, err
	}
	if err := p.checkCircuit(); err != nil {
		return cursor, err
	}
	release, acquired, err := p.acquire(ctx, a.ID())
	if err != nil {
		return cursor, err
	}
	if !acquired {
		return cursor, nil
	}
	defer release()

	started := time.Now()
	batch, err := p.fetchWithRetry(ctx, a, cursor)
	if err != nil {
		p.noteFailure()
		_ = p.repo.RecordFailure(ctx, a.ID(), batch.ParserVersion, err.Error())
		return cursor, err
	}
	p.noteSuccess()
	if err := p.repo.RecordSuccess(ctx, a.ID(), batch.ParserVersion, time.Since(started), p.policy); err != nil {
		return cursor, fmt.Errorf("record source success: %w", err)
	}

	for _, item := range batch.Items {
		if err := ctx.Err(); err != nil {
			return cursor, err
		}
		item.CanonicalURL = CanonicalURL(item.URL)
		if item.RetrievedAt.IsZero() {
			item.RetrievedAt = batch.RetrievedAt
		}
		if item.PublishedAt.IsZero() {
			item.PublishedAt = item.RetrievedAt
		}
		if item.Attribution == "" {
			item.Attribution = p.policy.Attribution
		}
		if item.Licence == "" {
			item.Licence = p.policy.Licence
		}
		hash := ContentHash(item)
		seen, err := p.repo.SeenHash(ctx, hash)
		if err != nil {
			return cursor, err
		}
		if seen {
			continue
		}

		clusterID := hash
		if existing, found, err := p.repo.FindNearDuplicate(ctx, item, nearDuplicateWindow, nearDuplicateThreshold); err != nil {
			return cursor, fmt.Errorf("near-duplicate lookup: %w", err)
		} else if found {
			clusterID = existing
		}

		body, err := json.Marshal(item)
		if err != nil {
			return cursor, fmt.Errorf("encode raw item: %w", err)
		}
		raw := domain.RawDocument{
			SourceID: a.ID(), URL: item.URL, ContentType: "application/json", ParserVersion: batch.ParserVersion,
			RetrievedAt: item.RetrievedAt, DeleteAfter: item.RetrievedAt.Add(p.policy.RawRetention),
			StatusCode: 200, Body: body, Hash: hash,
		}
		if err := p.repo.SaveRaw(ctx, raw); err != nil {
			return cursor, fmt.Errorf("save raw: %w", err)
		}
		if _, _, err := p.repo.SaveNormalized(ctx, a.ID(), item, hash, clusterID, batch.ParserVersion, p.policy); err != nil {
			return cursor, fmt.Errorf("save normalized: %w", err)
		}
	}

	if err := p.PublishPending(ctx); err != nil {
		return batch.Next, err
	}
	return batch.Next, nil
}

func (p *Processor) fetchWithRetry(ctx context.Context, a Adapter, cursor Cursor) (Batch, error) {
	var batch Batch
	var err error
	for attempt := 0; attempt < p.retry.MaxAttempts; attempt++ {
		if err := p.takeRateToken(ctx, a.ID()); err != nil {
			return batch, err
		}
		batch, err = a.Fetch(ctx, cursor)
		if err == nil {
			return batch, nil
		}
		if attempt == p.retry.MaxAttempts-1 {
			break
		}
		delay := p.retry.Delay(attempt)
		p.log.Warn("source_fetch_retry", "source", a.ID(), "attempt", attempt+1, "delay", delay, "error", err)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return batch, ctx.Err()
		case <-timer.C:
		}
	}
	return batch, fmt.Errorf("fetch failed after %d attempts: %w", p.retry.MaxAttempts, err)
}

func (p *Processor) acquire(ctx context.Context, sourceID string) (func(), bool, error) {
	if p.redis == nil {
		return func() {}, true, nil
	}
	lock := "source-lock:" + sourceID
	ok, err := p.redis.SetNX(ctx, lock, "1", p.policy.Timeout+30*time.Second).Result()
	if err != nil {
		return nil, false, err
	}
	return func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.redis.Del(cleanup, lock).Err()
	}, ok, nil
}

func (p *Processor) takeRateToken(ctx context.Context, sourceID string) error {
	if p.redis == nil {
		return nil
	}
	window := time.Now().UTC().Format("200601021504")
	rateKey := "source-rate:" + sourceID + ":" + window
	n, err := p.redis.Incr(ctx, rateKey).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		_ = p.redis.Expire(ctx, rateKey, 70*time.Second).Err()
	}
	if n > int64(p.policy.RequestsPerMinute) {
		return ErrRateLimited
	}
	return nil
}

func (p *Processor) checkCircuit() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Now().Before(p.circuitUntil) {
		return ErrCircuitOpen
	}
	return nil
}

func (p *Processor) noteFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures++
	if p.failures >= 5 {
		p.circuitUntil = time.Now().Add(5 * time.Minute)
	}
}

func (p *Processor) noteSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures = 0
	p.circuitUntil = time.Time{}
}

// PublishPending implements a Mongo-backed outbox. The article stays pending
// until Redis Streams acknowledges XADD; consumers deduplicate by article ID.
func (p *Processor) PublishPending(ctx context.Context) error {
	if p.publisher == nil {
		return nil
	}
	events, err := p.repo.PendingNewsEvents(ctx, 100)
	if err != nil {
		return err
	}
	var publishErr error
	for _, event := range events {
		if err := p.publisher.Publish(ctx, "news.created", event); err != nil {
			_ = p.repo.RecordPublishFailure(ctx, event.ArticleID, err.Error(), maxPublishAttempts)
			publishErr = errors.Join(publishErr, err)
			continue
		}
		if err := p.repo.MarkNewsEventPublished(ctx, event.ArticleID); err != nil {
			publishErr = errors.Join(publishErr, err)
		}
	}
	return publishErr
}

type RedisPublisher struct{ client *redis.Client }

func NewRedisPublisher(c *redis.Client) *RedisPublisher { return &RedisPublisher{c} }

func (p *RedisPublisher) Publish(ctx context.Context, topic string, data any) error {
	if p.client == nil {
		return errors.New("redis publisher is unavailable")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: "stocker:" + topic,
		Values: map[string]any{"payload": string(b), "event_type": topic},
	}).Err()
}

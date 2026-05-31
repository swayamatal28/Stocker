package ingest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

type processorRepo struct {
	rawSaved, normalizedSaved, successes, publishFailures int
	events                                                []domain.NewsEvent
}

func (r *processorRepo) SeenHash(context.Context, string) (bool, error) { return false, nil }
func (r *processorRepo) FindNearDuplicate(context.Context, domain.SourceItem, time.Duration, float64) (string, bool, error) {
	return "", false, nil
}
func (r *processorRepo) SaveRaw(context.Context, domain.RawDocument) error { r.rawSaved++; return nil }
func (r *processorRepo) SaveNormalized(_ context.Context, _ string, item domain.SourceItem, _, _, _ string, _ SourcePolicy) (string, bool, error) {
	r.normalizedSaved++
	r.events = append(r.events, domain.NewsEvent{ArticleID: "article-1", Title: item.Title})
	return "article-1", true, nil
}
func (r *processorRepo) PendingNewsEvents(context.Context, int) ([]domain.NewsEvent, error) {
	return r.events, nil
}
func (r *processorRepo) MarkNewsEventPublished(context.Context, string) error {
	r.events = nil
	return nil
}
func (r *processorRepo) RecordPublishFailure(context.Context, string, string, int) error {
	r.publishFailures++
	return nil
}
func (r *processorRepo) RecordSuccess(context.Context, string, string, time.Duration, SourcePolicy) error {
	r.successes++
	return nil
}
func (r *processorRepo) RecordFailure(context.Context, string, string, string) error { return nil }

type processorPublisher struct{ published int }

func (p *processorPublisher) Publish(context.Context, string, any) error { p.published++; return nil }

type retryAdapter struct{ attempts int }

func (a *retryAdapter) ID() string                    { return "test-feed" }
func (a *retryAdapter) Kind() SourceKind              { return KindRSS }
func (a *retryAdapter) Health(context.Context) Health { return Health{Status: "healthy"} }
func (a *retryAdapter) Fetch(context.Context, Cursor) (Batch, error) {
	a.attempts++
	if a.attempts == 1 {
		return Batch{ParserVersion: "fixture-v1"}, errors.New("temporary failure")
	}
	now := time.Now().UTC()
	return Batch{ParserVersion: "fixture-v1", RetrievedAt: now, Next: Cursor{Since: now}, Items: []domain.SourceItem{{Title: "Saved feed item", Body: "Project-owned fixture", URL: "https://example.invalid/item", PublishedAt: now}}}, nil
}

type oldArticleAdapter struct{ now time.Time }

func (a *oldArticleAdapter) ID() string                    { return "old-article-feed" }
func (a *oldArticleAdapter) Kind() SourceKind              { return KindRSS }
func (a *oldArticleAdapter) Health(context.Context) Health { return Health{Status: "healthy"} }
func (a *oldArticleAdapter) Fetch(context.Context, Cursor) (Batch, error) {
	return Batch{
		ParserVersion: "fixture-v1",
		RetrievedAt:   a.now,
		Next:          Cursor{Since: a.now},
		Items: []domain.SourceItem{{
			Title:       "Old feed item",
			Body:        "This item must never be persisted.",
			URL:         "https://example.invalid/old-item",
			PublishedAt: a.now.Add(-24 * time.Hour),
		}},
	}, nil
}

func TestProcessorRetriesPersistsAndPublishes(t *testing.T) {
	repo := &processorRepo{}
	publisher := &processorPublisher{}
	policy := SourcePolicy{
		PollInterval: time.Minute, Timeout: time.Second, RequestsPerMinute: 2, RawRetention: time.Hour,
		Attribution: "Fixture", Licence: "Project-owned", TermsURL: "https://example.invalid/terms", AutomatedAccessAllowed: true,
	}
	processor := NewProcessor(repo, nil, publisher, slog.New(slog.NewTextHandler(io.Discard, nil)), policy)
	processor.retry = RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}
	adapter := &retryAdapter{}
	if _, err := processor.RunOnce(context.Background(), adapter, Cursor{}); err != nil {
		t.Fatal(err)
	}
	if adapter.attempts != 2 || repo.rawSaved != 1 || repo.normalizedSaved != 1 || repo.successes != 1 || publisher.published != 1 {
		t.Fatalf("unexpected processing counts: attempts=%d raw=%d normalized=%d successes=%d published=%d", adapter.attempts, repo.rawSaved, repo.normalizedSaved, repo.successes, publisher.published)
	}
}

func TestProcessorSkipsArticlesAtLeast24HoursOld(t *testing.T) {
	repo := &processorRepo{}
	policy := SourcePolicy{
		PollInterval: time.Minute, Timeout: time.Second, RequestsPerMinute: 2, RawRetention: time.Hour,
		Attribution: "Fixture", Licence: "Project-owned", TermsURL: "https://example.invalid/terms", AutomatedAccessAllowed: true,
	}
	processor := NewProcessor(repo, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), policy)
	if _, err := processor.RunOnce(context.Background(), &oldArticleAdapter{now: time.Now().UTC()}, Cursor{}); err != nil {
		t.Fatal(err)
	}
	if repo.rawSaved != 0 || repo.normalizedSaved != 0 {
		t.Fatalf("old article was persisted: raw=%d normalized=%d", repo.rawSaved, repo.normalizedSaved)
	}
}

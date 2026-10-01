package ingest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

type loadAdapter struct{ items []domain.SourceItem }

func (adapter loadAdapter) ID() string                    { return "load-fixture" }
func (adapter loadAdapter) Kind() SourceKind              { return KindMock }
func (adapter loadAdapter) Health(context.Context) Health { return Health{Status: "healthy"} }
func (adapter loadAdapter) Fetch(context.Context, Cursor) (Batch, error) {
	return Batch{Items: adapter.items, RetrievedAt: time.Now().UTC(), ParserVersion: "load-v1"}, nil
}

type loadRepository struct{}

func (loadRepository) SeenHash(context.Context, string) (bool, error) { return false, nil }
func (loadRepository) FindNearDuplicate(context.Context, domain.SourceItem, time.Duration, float64) (string, bool, error) {
	return "", false, nil
}
func (loadRepository) SaveRaw(context.Context, domain.RawDocument) error { return nil }
func (loadRepository) SaveNormalized(context.Context, string, domain.SourceItem, string, string, string, SourcePolicy) (string, bool, error) {
	return "load", true, nil
}
func (loadRepository) PendingNewsEvents(context.Context, int) ([]domain.NewsEvent, error) {
	return nil, nil
}
func (loadRepository) MarkNewsEventPublished(context.Context, string) error            { return nil }
func (loadRepository) RecordPublishFailure(context.Context, string, string, int) error { return nil }
func (loadRepository) RecordSuccess(context.Context, string, string, time.Duration, SourcePolicy) error {
	return nil
}
func (loadRepository) RecordFailure(context.Context, string, string, string) error { return nil }

func BenchmarkProcessorBatch1000(b *testing.B) {
	now := time.Now().UTC()
	items := make([]domain.SourceItem, 1000)
	for index := range items {
		items[index] = domain.SourceItem{ExternalID: fmt.Sprintf("load-%d", index), URL: fmt.Sprintf("https://example.invalid/news/%d?utm_source=load", index), Title: fmt.Sprintf("Company event %d", index), Body: "A synthetic benchmark item with enough text to exercise normalization and hashing.", PublishedAt: now, RetrievedAt: now, Synthetic: true}
	}
	policy := SourcePolicy{PollInterval: time.Minute, Timeout: time.Second, RequestsPerMinute: 10000, RawRetention: 24 * time.Hour, Attribution: "Load fixture", Licence: "Project-owned", TermsURL: "https://example.invalid/terms", AutomatedAccessAllowed: true, RobotsChecked: true}
	processor := NewProcessor(loadRepository{}, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), policy)
	adapter := loadAdapter{items: items}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := processor.RunOnce(context.Background(), adapter, Cursor{}); err != nil {
			b.Fatal(err)
		}
	}
}

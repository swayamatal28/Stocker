package intelligence

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stocker-app/stocker/internal/ai"
	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/ingest"
	"github.com/stocker-app/stocker/internal/store"
)

func TestCollectedItemProducesPersistedAnalysisAndSignal(t *testing.T) {
	if os.Getenv("STOCKER_INTEGRATION_TEST") != "1" {
		t.Skip("set STOCKER_INTEGRATION_TEST=1 to run MongoDB integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	db, err := store.Open(ctx, uri, fmt.Sprintf("stocker_phase3_test_%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	defer db.DB.Drop(context.Background())
	if err := db.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	policy := ingest.SourcePolicy{PollInterval: time.Minute, Timeout: 5 * time.Second, RequestsPerMinute: 10, RawRetention: 24 * time.Hour, Attribution: "Synthetic STOCKER fixture", Licence: "Project-owned test fixture", TermsURL: "https://example.invalid/terms", AutomatedAccessAllowed: true, RobotsChecked: true}
	ingestion := ingest.NewProcessor(db, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), policy)
	if _, err := ingestion.RunOnce(ctx, ingest.MockAdapter{}, ingest.Cursor{}); err != nil {
		t.Fatal(err)
	}
	page, err := db.ListNews(ctx, domain.NewsFilter{Page: 1, PageSize: 10})
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("collected item missing: %v", err)
	}

	processor := NewProcessor(db, ai.LocalProvider{}, nil, Config{PromptVersion: "analysis-v1", SchemaVersion: "ai-analysis-v1", MaxInputChars: 24000, DailyBudgetCents: 100, MaxAttempts: 5})
	first, err := processor.Process(ctx, page.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Analysis.ID == "" || len(first.Signals) == 0 || len(first.Analysis.Analysis.Evidence) == 0 {
		t.Fatalf("incomplete intelligence output: %#v", first)
	}
	second, err := processor.Process(ctx, page.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Analysis.ID != first.Analysis.ID || second.Signals[0].ID != first.Signals[0].ID {
		t.Fatal("processing was not idempotent")
	}
	persisted, err := db.AnalysisByArticle(ctx, page.Items[0].ID)
	if err != nil || persisted.Analysis.PromptHash == "" || len(persisted.Signals) != 1 {
		t.Fatalf("persisted output invalid: %v %#v", err, persisted)
	}
}

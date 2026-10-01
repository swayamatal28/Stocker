package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/ai"
	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/ingest"
	"github.com/stocker-app/stocker/internal/intelligence"
	"github.com/stocker-app/stocker/internal/store"
)

func TestPhase3AnalysisAndSignalAPIs(t *testing.T) {
	if os.Getenv("STOCKER_INTEGRATION_TEST") != "1" {
		t.Skip("set STOCKER_INTEGRATION_TEST=1 to run MongoDB integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	db, err := store.Open(ctx, uri, fmt.Sprintf("stocker_phase3_api_test_%d", time.Now().UnixNano()))
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
	if err := db.MigratePhase3Intelligence(ctx); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	policy := ingest.SourcePolicy{PollInterval: time.Minute, Timeout: 5 * time.Second, RequestsPerMinute: 10, RawRetention: 24 * time.Hour, Attribution: "Synthetic STOCKER fixture", Licence: "Project-owned test fixture", TermsURL: "https://example.invalid/terms", AutomatedAccessAllowed: true, RobotsChecked: true}
	ingestion := ingest.NewProcessor(db, nil, nil, log, policy)
	if _, err := ingestion.RunOnce(ctx, ingest.MockAdapter{}, ingest.Cursor{}); err != nil {
		t.Fatal(err)
	}
	page, err := db.ListNews(ctx, domain.NewsFilter{Page: 1, PageSize: 10})
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("collected item missing: %v", err)
	}
	analysis := intelligence.NewProcessor(db, ai.LocalProvider{}, nil, intelligence.Config{PromptVersion: "analysis-v1", SchemaVersion: "ai-analysis-v1", MaxInputChars: 24000, DailyBudgetCents: 100, MaxAttempts: 5})
	if _, err := analysis.Process(ctx, page.Items[0].ID); err != nil {
		t.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond, MaxRetries: 0})
	defer rdb.Close()
	server := New(config.Config{Environment: "test", JWTSecret: "phase-3-api-secret-at-least-32-characters", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour, WebOrigin: "http://localhost:5173"}, db, rdb, log)
	defer server.Close()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	analysisResponse := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/news/"+page.Items[0].ID+"/analysis", "", nil)
	if analysisResponse.status != http.StatusOK {
		t.Fatalf("analysis API status %d: %s", analysisResponse.status, analysisResponse.body)
	}
	signalsResponse := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/stocks/RELIANCE/signals", "", nil)
	if signalsResponse.status != http.StatusOK {
		t.Fatalf("signals API status %d: %s", signalsResponse.status, signalsResponse.body)
	}
}

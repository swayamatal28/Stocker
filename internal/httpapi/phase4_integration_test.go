package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/ingest"
	"github.com/stocker-app/stocker/internal/store"
)

func TestPhase4MarketAPIs(t *testing.T) {
	if os.Getenv("STOCKER_INTEGRATION_TEST") != "1" {
		t.Skip("set STOCKER_INTEGRATION_TEST=1 to run MongoDB integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	db, err := store.Open(ctx, uri, fmt.Sprintf("stocker_phase4_api_test_%d", time.Now().UnixNano()))
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
	policy := ingest.SourcePolicy{PollInterval: 10 * time.Minute, Timeout: 5 * time.Second, RequestsPerMinute: 2, RawRetention: 24 * time.Hour, Attribution: "Phase 4 publisher fixture", Licence: "Project-owned integration fixture", TermsURL: "https://example.invalid/terms", AutomatedAccessAllowed: true, RobotsChecked: true}
	for index := 0; index < 12; index++ {
		item := domain.SourceItem{ExternalID: fmt.Sprintf("ola-%d", index), URL: fmt.Sprintf("https://example.invalid/ola/%d", index), CanonicalURL: fmt.Sprintf("https://example.invalid/ola/%d", index), Title: fmt.Sprintf("Ola Electric update %d", index), Body: "Ola Electric Mobility issued an attributed company update.", Language: "en", ContentType: "text/plain", PublishedAt: time.Now().UTC().Add(-time.Duration(index) * time.Minute), RetrievedAt: time.Now().UTC(), Attribution: policy.Attribution, Licence: policy.Licence}
		hash := ingest.ContentHash(item)
		if _, _, err := db.SaveNormalized(ctx, fmt.Sprintf("publisher-%d", index%6), item, hash, hash[:16], "rss-atom-v1", policy); err != nil {
			t.Fatal(err)
		}
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond, MaxRetries: 0})
	defer rdb.Close()
	server := New(config.Config{Environment: "test", JWTSecret: "phase-4-api-secret-at-least-32-characters", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour, WebOrigin: "http://localhost:5173", MarketProvider: "fixture", MarketRefreshInterval: 5 * time.Minute}, db, rdb, log)
	defer server.Close()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	client := &http.Client{Timeout: 5 * time.Second}

	for _, endpoint := range []string{
		"/api/v1/stocks/search?q=OLA",
		"/api/v1/stocks/OLAELEC",
		"/api/v1/stocks/OLAELEC/quote",
		"/api/v1/stocks/OLAELEC/fundamentals",
		"/api/v1/stocks/OLAELEC/peers",
		"/api/v1/stocks/OLAELEC/risk-flags",
		"/api/v1/market/sectors",
		"/api/v1/market/movers",
		"/api/v1/events?symbol=OLAELEC",
		"/api/v1/stocks/OLAELEC/news",
	} {
		response := doJSON(t, client, http.MethodGet, httpServer.URL+endpoint, "", nil)
		if response.status != http.StatusOK {
			t.Fatalf("%s returned %d: %s", endpoint, response.status, response.body)
		}
	}
	quote := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/stocks/OLAELEC/quote", "", nil)
	if !strings.Contains(string(quote.body), `"dayHigh"`) || !strings.Contains(string(quote.body), `"synthetic":true`) || !strings.Contains(string(quote.body), `"asOf"`) {
		t.Fatalf("quote provenance/freshness fields missing: %s", quote.body)
	}
	fundamentals := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/stocks/OLAELEC/fundamentals", "", nil)
	for _, field := range []string{`"period"`, `"unit"`, `"basis"`, `"source"`} {
		if !strings.Contains(string(fundamentals.body), field) {
			t.Fatalf("fundamental metadata %s missing: %s", field, fundamentals.body)
		}
	}
	stockNews := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/stocks/OLAELEC/news", "", nil)
	if !strings.Contains(string(stockNews.body), `"pageSize":10`) || !strings.Contains(string(stockNews.body), `"total":12`) {
		t.Fatalf("stock news did not return the expected bounded, alias-linked result set: %s", stockNews.body)
	}
}

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
	"github.com/stocker-app/stocker/internal/store"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestPhase6EventTimeEvaluationRejectsLeakedSignal(t *testing.T) {
	if os.Getenv("STOCKER_INTEGRATION_TEST") != "1" {
		t.Skip("set STOCKER_INTEGRATION_TEST=1 to run MongoDB integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	database := fmt.Sprintf("stocker_phase6_test_%d", time.Now().UnixNano())
	db, err := store.Open(ctx, uri, database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	defer db.DB.Drop(context.Background())
	if err := db.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	asOf := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	decision := asOf.Add(-10 * 24 * time.Hour)
	securityID := bson.NewObjectID()
	if _, err := db.DB.Collection("securities").InsertOne(ctx, bson.M{"_id": securityID, "nse_symbol": "EVAL", "company_name": "Evaluation Limited", "sector": "Technology", "active": true, "created_at": decision.Add(-time.Hour), "updated_at": decision.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	quotes := []any{
		bson.M{"security_id": securityID, "source": "phase6-fixture", "last_price": 100.0, "as_of": decision.Add(-time.Minute), "retrieved_at": decision.Add(-30 * time.Second)},
		bson.M{"security_id": securityID, "source": "phase6-fixture", "last_price": 112.0, "as_of": decision.Add(8 * 24 * time.Hour), "retrieved_at": decision.Add(8*24*time.Hour + time.Minute)},
	}
	if _, err := db.DB.Collection("market_quotes").InsertMany(ctx, quotes); err != nil {
		t.Fatal(err)
	}
	signals := []any{
		bson.M{"security_id": securityID, "label": "Positive", "score": 70.0, "confidence": 90, "horizon": "short term", "version": "phase6", "data_fresh_at": decision.Add(-time.Minute), "generated_at": decision},
		bson.M{"security_id": securityID, "label": "Positive", "score": 60.0, "confidence": 75, "horizon": "short term", "version": "phase6-leaked", "data_fresh_at": decision.Add(time.Minute), "generated_at": decision},
	}
	if _, err := db.DB.Collection("signals").InsertMany(ctx, signals); err != nil {
		t.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond, MaxRetries: 0})
	defer rdb.Close()
	cfg := config.Config{Environment: "test", JWTSecret: "phase-6-integration-secret-at-least-32", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour, WebOrigin: "http://localhost:5173", MarketProvider: "fixture", MarketRefreshInterval: 5 * time.Minute, AlertEvaluationInterval: time.Hour}
	api := New(cfg, db, rdb, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer api.Close()
	server := httptest.NewServer(api.Handler())
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	token := registerPhase5User(t, client, server.URL, "phase6@example.com")

	response := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/evaluation/report?asOf="+asOf.Format(time.RFC3339), token, nil)
	if response.status != http.StatusOK {
		t.Fatalf("evaluation status %d: %s", response.status, response.body)
	}
	var result struct {
		Data domain.EvaluationReport `json:"data"`
		Meta struct {
			LookAheadSafe bool `json:"lookAheadSafe"`
		} `json:"meta"`
	}
	decodeJSON(t, response.body, &result)
	if result.Data.CandidateSignals != 2 || result.Data.EvaluatedSignals != 1 || len(result.Data.LeakageViolations) != 1 || result.Meta.LookAheadSafe {
		t.Fatalf("unexpected event-time report: %s", response.body)
	}
	if len(result.Data.Outcomes) != 1 || !result.Data.Outcomes[0].Correct || result.Data.Outcomes[0].ReturnPercent != 12 {
		t.Fatalf("historical outcome is incorrect: %#v", result.Data.Outcomes)
	}
	if count, _ := db.DB.Collection("signal_outcomes").CountDocuments(ctx, bson.M{}); count != 1 {
		t.Fatalf("expected one persisted outcome, got %d", count)
	}
	metrics, err := client.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer metrics.Body.Close()
	buffer := make([]byte, 64*1024)
	n, _ := metrics.Body.Read(buffer)
	if body := string(buffer[:n]); !strings.Contains(body, "stocker_evaluation_leakage_violations 1") || !strings.Contains(body, "stocker_http_requests_total") {
		t.Fatalf("phase 6 metrics missing: %s", body)
	}

	old := asOf.Add(-10 * 24 * time.Hour)
	if _, err := db.DB.Collection("raw_documents").InsertOne(ctx, bson.M{"hash": "phase6-expired", "delete_after": old}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.Collection("briefings").InsertOne(ctx, bson.M{"user_id": bson.NewObjectID(), "kind": "daily", "period_key": "phase6-old", "generated_at": old}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.Collection("evaluation_runs").InsertOne(ctx, bson.M{"version": "old", "as_of": old}); err != nil {
		t.Fatal(err)
	}
	retained, err := db.RunRetention(ctx, asOf, store.RetentionPolicy{Transient: 24 * time.Hour, AlertDays: 1, BriefingDays: 1, AuditDays: 1})
	if err != nil {
		t.Fatal(err)
	}
	if retained.RawDocuments != 1 || retained.Briefings != 1 || retained.EvaluationRuns != 1 {
		t.Fatalf("retention evidence missing: %#v", retained)
	}
}

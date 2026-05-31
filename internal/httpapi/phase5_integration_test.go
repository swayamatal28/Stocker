package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/store"
)

func TestPhase5RuleTriggersOneInspectableUserIsolatedAlert(t *testing.T) {
	if os.Getenv("STOCKER_INTEGRATION_TEST") != "1" {
		t.Skip("set STOCKER_INTEGRATION_TEST=1 to run MongoDB integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	database := fmt.Sprintf("stocker_phase5_test_%d", time.Now().UnixNano())
	db, err := store.Open(ctx, uri, database)
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

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond, MaxRetries: 0})
	defer rdb.Close()
	cfg := config.Config{Environment: "test", JWTSecret: "phase-5-integration-secret-at-least-32", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour, WebOrigin: "http://localhost:5173", MarketProvider: "fixture", MarketRefreshInterval: 5 * time.Minute}
	apiServer := New(cfg, db, rdb, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer apiServer.Close()
	httpServer := httptest.NewServer(apiServer.Handler())
	defer httpServer.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}

	token := registerPhase5User(t, client, httpServer.URL, "phase5-owner@example.com")
	if response := doJSON(t, client, http.MethodPost, httpServer.URL+"/api/v1/portfolio", token, map[string]any{"symbol": "OLAELEC", "quantity": 10, "averageBuyPrice": 40}); response.status != http.StatusCreated {
		t.Fatalf("add portfolio status %d: %s", response.status, response.body)
	}
	threshold := 1.0
	create := doJSON(t, client, http.MethodPost, httpServer.URL+"/api/v1/alert-rules", token, map[string]any{"name": "OLA above one rupee", "symbol": "OLAELEC", "ruleType": "price_above", "threshold": threshold, "minimumConfidence": 80, "minimumSeverity": "low", "cooldownMinutes": 60, "channels": map[string]bool{"inApp": true}, "quietHours": map[string]any{"enabled": false}, "enabled": true})
	if create.status != http.StatusCreated {
		t.Fatalf("create rule status %d: %s", create.status, create.body)
	}
	var created struct {
		Data domain.AlertRule `json:"data"`
	}
	decodeJSON(t, create.body, &created)
	if created.Data.ID == "" {
		t.Fatal("created rule has no ID")
	}

	alerts := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/alerts", token, nil)
	var alertPage struct {
		Data []domain.AlertEvent `json:"data"`
		Meta struct {
			Unread int `json:"unread"`
		} `json:"meta"`
	}
	decodeJSON(t, alerts.body, &alertPage)
	if alerts.status != http.StatusOK || len(alertPage.Data) != 1 || alertPage.Meta.Unread != 1 {
		t.Fatalf("expected one unread alert, status %d: %s", alerts.status, alerts.body)
	}
	event := alertPage.Data[0]
	if event.DeliveredAt == nil || len(event.Evidence) == 0 || event.SourceAsOf.IsZero() || event.ConditionSnapshot["lastPrice"] == nil {
		t.Fatalf("alert is not inspectable and source-backed: %#v", event)
	}

	patch := doJSON(t, client, http.MethodPatch, httpServer.URL+"/api/v1/alert-rules/"+created.Data.ID, token, map[string]any{"name": "OLA above one rupee - checked"})
	if patch.status != http.StatusOK {
		t.Fatalf("patch rule status %d: %s", patch.status, patch.body)
	}
	again := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/alerts", token, nil)
	decodeJSON(t, again.body, &alertPage)
	if len(alertPage.Data) != 1 {
		t.Fatalf("cooldown deduplication failed: %s", again.body)
	}

	read := doJSON(t, client, http.MethodPatch, httpServer.URL+"/api/v1/alerts/"+event.ID+"/read", token, map[string]any{})
	if read.status != http.StatusOK {
		t.Fatalf("mark read status %d: %s", read.status, read.body)
	}
	briefing := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/briefings?kind=daily", token, nil)
	var briefingPage struct {
		Data []domain.Briefing `json:"data"`
	}
	decodeJSON(t, briefing.body, &briefingPage)
	if briefing.status != http.StatusOK || len(briefingPage.Data) != 1 || len(briefingPage.Data[0].Items) == 0 || len(briefingPage.Data[0].Items[0].Evidence) == 0 {
		t.Fatalf("expected evidence-backed briefing: %s", briefing.body)
	}

	otherClient := &http.Client{Timeout: 5 * time.Second}
	otherToken := registerPhase5User(t, otherClient, httpServer.URL, "phase5-other@example.com")
	otherAlerts := doJSON(t, otherClient, http.MethodGet, httpServer.URL+"/api/v1/alerts", otherToken, nil)
	var otherPage struct {
		Data []domain.AlertEvent `json:"data"`
	}
	decodeJSON(t, otherAlerts.body, &otherPage)
	if len(otherPage.Data) != 0 {
		t.Fatal("alert history leaked across users")
	}
	if response := doJSON(t, otherClient, http.MethodPatch, httpServer.URL+"/api/v1/alerts/"+event.ID+"/read", otherToken, map[string]any{}); response.status != http.StatusNotFound {
		t.Fatalf("another user could address the owner's alert: %d %s", response.status, response.body)
	}
	if response := doJSON(t, otherClient, http.MethodPatch, httpServer.URL+"/api/v1/alert-rules/"+created.Data.ID, otherToken, map[string]any{"enabled": false}); response.status != http.StatusNotFound {
		t.Fatalf("another user could address the owner's rule: %d %s", response.status, response.body)
	}

	revoke := doJSON(t, client, http.MethodDelete, httpServer.URL+"/api/v1/alert-rules/"+created.Data.ID, token, nil)
	if revoke.status != http.StatusNoContent {
		t.Fatalf("revoke rule status %d: %s", revoke.status, revoke.body)
	}
	rules := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/alert-rules", token, nil)
	var rulePage struct {
		Data []domain.AlertRule `json:"data"`
	}
	decodeJSON(t, rules.body, &rulePage)
	if len(rulePage.Data) != 0 {
		t.Fatal("revoked rule remains active in rule list")
	}
}

func registerPhase5User(t *testing.T, client *http.Client, baseURL, email string) string {
	t.Helper()
	response := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/auth/register", "", map[string]any{"email": email, "password": "strong-password-123", "displayName": "Phase Five"})
	if response.status != http.StatusCreated {
		t.Fatalf("register status %d: %s", response.status, response.body)
	}
	var result struct {
		AccessToken string `json:"accessToken"`
	}
	decodeJSON(t, response.body, &result)
	return result.AccessToken
}

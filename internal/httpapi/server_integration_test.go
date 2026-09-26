package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/stocker-app/stocker/internal/store"
)

func TestPhase1AuthSearchAndWatchlistFlow(t *testing.T) {
	if os.Getenv("STOCKER_INTEGRATION_TEST") != "1" {
		t.Skip("set STOCKER_INTEGRATION_TEST=1 to run MongoDB integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	database := fmt.Sprintf("stocker_phase1_test_%d", time.Now().UnixNano())
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
	cfg := config.Config{Environment: "test", JWTSecret: "phase-1-integration-secret-at-least-32", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour, WebOrigin: "http://localhost:5173"}
	apiServer := New(cfg, db, rdb, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer apiServer.Close()
	httpServer := httptest.NewServer(apiServer.Handler())
	defer httpServer.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}

	register := doJSON(t, client, http.MethodPost, httpServer.URL+"/api/v1/auth/register", "", map[string]any{"email": "phase1@example.com", "password": "strong-password-123", "displayName": "Phase One"})
	if register.status != http.StatusCreated {
		t.Fatalf("register status %d: %s", register.status, register.body)
	}
	var authResult struct {
		AccessToken string `json:"accessToken"`
	}
	decodeJSON(t, register.body, &authResult)
	if authResult.AccessToken == "" {
		t.Fatal("registration did not return an access token")
	}
	if stream := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/stream", "", nil); stream.status != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated stream to fail, got %d", stream.status)
	}

	me := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/auth/me", authResult.AccessToken, nil)
	if me.status != http.StatusOK {
		t.Fatalf("me status %d: %s", me.status, me.body)
	}
	search := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/stocks/search?q=RELIANCE", "", nil)
	if search.status != http.StatusOK {
		t.Fatalf("search status %d: %s", search.status, search.body)
	}
	add := doJSON(t, client, http.MethodPost, httpServer.URL+"/api/v1/watchlist", authResult.AccessToken, map[string]string{"symbol": "RELIANCE"})
	if add.status != http.StatusCreated {
		t.Fatalf("add status %d: %s", add.status, add.body)
	}
	pause := doJSON(t, client, http.MethodPatch, httpServer.URL+"/api/v1/watchlist/RELIANCE", authResult.AccessToken, map[string]bool{"alertsPaused": true})
	if pause.status != http.StatusOK {
		t.Fatalf("pause status %d: %s", pause.status, pause.body)
	}
	remove := doJSON(t, client, http.MethodDelete, httpServer.URL+"/api/v1/watchlist/RELIANCE", authResult.AccessToken, nil)
	if remove.status != http.StatusNoContent {
		t.Fatalf("remove status %d: %s", remove.status, remove.body)
	}
	list := doJSON(t, client, http.MethodGet, httpServer.URL+"/api/v1/watchlist", authResult.AccessToken, nil)
	var watchlist struct {
		Data []any `json:"data"`
	}
	decodeJSON(t, list.body, &watchlist)
	if list.status != http.StatusOK || len(watchlist.Data) != 0 {
		t.Fatalf("expected empty watchlist, status %d: %s", list.status, list.body)
	}

	refresh := doJSON(t, client, http.MethodPost, httpServer.URL+"/api/v1/auth/refresh", "", nil)
	if refresh.status != http.StatusOK {
		t.Fatalf("refresh status %d: %s", refresh.status, refresh.body)
	}
	logout := doJSON(t, client, http.MethodPost, httpServer.URL+"/api/v1/auth/logout", "", nil)
	if logout.status != http.StatusNoContent {
		t.Fatalf("logout status %d: %s", logout.status, logout.body)
	}
	if again := doJSON(t, client, http.MethodPost, httpServer.URL+"/api/v1/auth/refresh", "", nil); again.status != http.StatusUnauthorized {
		t.Fatalf("expected refresh after logout to fail, got %d", again.status)
	}
}

type integrationResponse struct {
	status int
	body   []byte
}

func doJSON(t *testing.T, client *http.Client, method, url, token string, payload any) integrationResponse {
	t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return integrationResponse{status: response.StatusCode, body: data}
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
}

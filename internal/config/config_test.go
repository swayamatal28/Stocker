package config

import "testing"

func TestLiveIngestionRequiresExplicitSourcePolicy(t *testing.T) {
	t.Setenv("MOCK_PROVIDERS", "false")
	t.Setenv("INGEST_SOURCE_ID", "")
	t.Setenv("INGEST_FEED_URL", "")
	t.Setenv("INGEST_ATTRIBUTION", "")
	t.Setenv("INGEST_LICENCE", "")
	t.Setenv("INGEST_TERMS_URL", "")
	t.Setenv("INGEST_AUTOMATED_ACCESS_ALLOWED", "false")
	if _, err := Load(); err == nil {
		t.Fatal("expected live ingestion without an explicit policy to fail closed")
	}
}

func TestMockIngestionUsesProjectOwnedApproval(t *testing.T) {
	t.Setenv("MOCK_PROVIDERS", "true")
	t.Setenv("INGEST_AUTOMATED_ACCESS_ALLOWED", "false")
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !config.IngestAutomatedAccessAllowed {
		t.Fatal("project-owned mock fixture should be approved")
	}
}

func TestHTTPAIProviderRequiresHTTPSInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "this-is-a-production-length-secret-value")
	t.Setenv("AI_PROVIDER", "http-json")
	t.Setenv("AI_MODEL", "approved-model")
	t.Setenv("AI_ENDPOINT", "http://models.example.com/analyse")
	if _, err := Load(); err == nil {
		t.Fatal("expected non-local production AI endpoint to require HTTPS")
	}
	t.Setenv("AI_ENDPOINT", "https://models.example.com/analyse")
	if _, err := Load(); err != nil {
		t.Fatalf("expected approved HTTPS endpoint: %v", err)
	}
}

func TestPlaintextMarketProviderRequiresExplicitOptIn(t *testing.T) {
	t.Setenv("MARKET_PROVIDER", "indian-stock-api")
	t.Setenv("MARKET_ENDPOINT", "http://65.0.104.9")
	t.Setenv("MARKET_ALLOW_INSECURE_HTTP", "false")
	if _, err := Load(); err == nil {
		t.Fatal("expected plaintext market provider to fail closed")
	}
	t.Setenv("MARKET_ALLOW_INSECURE_HTTP", "true")
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.MarketProvider != "indian-stock-api" {
		t.Fatalf("unexpected market provider: %s", config.MarketProvider)
	}
}

func TestMultiSourceIngestionUsesPerSourceApproval(t *testing.T) {
	t.Setenv("MOCK_PROVIDERS", "false")
	t.Setenv("INGEST_SOURCES_JSON", `[{"id":"publisher"}]`)
	t.Setenv("INGEST_AUTOMATED_ACCESS_ALLOWED", "false")
	if _, err := Load(); err != nil {
		t.Fatalf("multi-source approval is validated per source: %v", err)
	}
}

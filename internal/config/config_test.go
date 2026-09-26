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

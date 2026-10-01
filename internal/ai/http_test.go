package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPProviderUsesBoundedAuthenticatedContract(t *testing.T) {
	document := GroundingDocument{URL: "https://example.invalid/item", Source: "fixture", Title: "Update", Text: "Evidence", PublishedAt: time.Now().UTC()}
	analysis := validAnalysis(document)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing provider authorization")
		}
		var payload struct {
			Model   string          `json:"model"`
			Request GroundedRequest `json:"request"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload.Model != "approved-model" || payload.Request.SystemPolicy == "" {
			t.Errorf("unexpected payload: %#v", payload)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"output": analysis})
	}))
	defer server.Close()
	provider, err := NewHTTPProvider(server.URL, "secret", "approved-model", time.Second, 64<<10, 7)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.Analyze(context.Background(), GroundedRequest{SystemPolicy: "grounded only", Documents: []GroundingDocument{document}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(raw); err != nil {
		t.Fatal(err)
	}
	if provider.EstimatedCostCents(GroundedRequest{}) != 7 {
		t.Fatal("configured cost estimate was not preserved")
	}
}

func TestHTTPProviderRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write(make([]byte, 2048)) }))
	defer server.Close()
	provider, err := NewHTTPProvider(server.URL, "", "model", time.Second, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Analyze(context.Background(), GroundedRequest{}); err == nil {
		t.Fatal("expected oversized response rejection")
	}
}

func TestHTTPProviderTranslatesNonEnglishInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(request.Body).Decode(&payload)
		if payload["task"] != "translate" || payload["sourceLanguage"] != "hi" || payload["targetLanguage"] != "en" {
			t.Errorf("unexpected translation request: %#v", payload)
		}
		_ = json.NewEncoder(writer).Encode(map[string]string{"translation": "The company announced its results."})
	}))
	defer server.Close()
	provider, err := NewHTTPProvider(server.URL, "", "approved-model", time.Second, 4096, 0)
	if err != nil {
		t.Fatal(err)
	}
	translated, err := provider.Translate(context.Background(), "कंपनी ने नतीजे घोषित किए", "hi", "en")
	if err != nil || translated == "" {
		t.Fatalf("translation failed: %v", err)
	}
}

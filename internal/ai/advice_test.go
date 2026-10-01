package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPortfolioAdvisorSendsOnlyPublicStockInputAndValidatesOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatal("missing API key")
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		input := payload["input"].(map[string]any)
		if input["symbol"] != "RELIANCE" || input["quantity"] != nil || input["averageBuyPrice"] != nil || input["userId"] != nil {
			t.Fatalf("unexpected or private input: %#v", input)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"output": map[string]any{
			"summary": "Published figures show both strengths and risks.", "outlook": "mixed",
			"strengths": []string{"Positive cash flow"}, "concerns": []string{"Valuation needs review"}, "whatToWatch": []string{"Next results"},
		}})
	}))
	defer server.Close()
	advisor, err := NewPortfolioAdvisor(server.URL, "secret", "approved-model", time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	advice, err := advisor.Advise(context.Background(), PortfolioAdviceInput{Symbol: "RELIANCE", CompanyName: "Reliance Industries"})
	if err != nil || advice.Outlook != "mixed" {
		t.Fatalf("unexpected advice: %#v, %v", advice, err)
	}
}

func TestPortfolioAdvisorRejectsPlaceholderKey(t *testing.T) {
	if _, err := NewPortfolioAdvisor("https://example.com/advice", "insert AI platform api key", "model", time.Second, 4096); err == nil {
		t.Fatal("expected placeholder key to be unavailable")
	}
}

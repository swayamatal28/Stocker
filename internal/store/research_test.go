package store

import (
	"testing"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

func TestBuildStockResearchUsesTransparentDeterministicChecks(t *testing.T) {
	quote := &domain.MarketQuote{LastPrice: 150, YearLow: 100, YearHigh: 200, Source: "Market source", AsOf: time.Now().UTC()}
	fundamentals := &domain.Fundamentals{Source: "Financial source", AsOf: time.Now().UTC(), Metrics: []domain.FundamentalMetric{
		{Key: "pe_ratio", Value: 18},
		{Key: "roe", Value: 17},
		{Key: "debt_to_equity", Value: .4},
		{Key: "current_ratio", Value: 1.5},
		{Key: "operating_margin", Value: 19},
		{Key: "revenue_growth", Value: 12},
		{Key: "free_cash_flow", Value: 100_000_000},
	}}
	research := BuildStockResearch(quote, fundamentals)
	if research.Coverage != 8 || research.Score < 80 || research.Source != "Financial source" {
		t.Fatalf("unexpected research output: %#v", research)
	}
	for _, check := range research.Checks {
		if check.Formula == "" || check.Explanation == "" {
			t.Fatalf("research check is not explainable: %#v", check)
		}
	}
}

func TestBuildStockResearchDoesNotInventMissingFundamentals(t *testing.T) {
	quote := &domain.MarketQuote{LastPrice: 90, YearLow: 80, YearHigh: 120, Source: "Market source", AsOf: time.Now().UTC()}
	research := BuildStockResearch(quote, nil)
	if research.Coverage != 1 || research.Label != "Not enough financial data" {
		t.Fatalf("missing fundamentals should remain explicit: %#v", research)
	}
}

func TestCalculateDayPnLUsesPreviousClose(t *testing.T) {
	// A price of 110 after a 10% move implies a previous close of 100.
	if got := calculateDayPnL(110, 10, 5); got != 50 {
		t.Fatalf("expected day P&L of 50, got %.2f", got)
	}
}

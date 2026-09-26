package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/stocker-app/stocker/internal/domain"
)

// Provider implementations receive source text as inert data, never as model instructions.
type Provider interface {
	Name() string
	Analyze(context.Context, GroundedRequest) (json.RawMessage, error)
}
type GroundedRequest struct {
	SystemPolicy        string              `json:"systemPolicy"`
	Documents           []GroundingDocument `json:"documents"`
	AllowedNumericFacts map[string]float64  `json:"allowedNumericFacts"`
	SchemaVersion       string              `json:"schemaVersion"`
}
type GroundingDocument struct{ ID, Title, Text, URL, Source string }

var sentiments = map[string]bool{"bullish": true, "bearish": true, "mixed": true, "neutral": true}
var horizons = map[string]bool{"intraday": true, "short term": true, "medium term": true, "long term": true}
var novelties = map[string]bool{"new": true, "partly_known": true, "already_known": true, "unclear": true}
var eventCategories = map[string]bool{"quarterly_results": true, "annual_results": true, "guidance": true, "order_or_contract": true, "merger_or_acquisition": true, "fundraising": true, "dividend_bonus_split": true, "buyback": true, "promoter_or_institutional_transaction": true, "management_change": true, "regulatory_action": true, "litigation": true, "credit_rating": true, "product_launch": true, "capacity_expansion": true, "plant_shutdown": true, "accident_or_disruption": true, "fraud_or_governance": true, "macroeconomic": true, "rbi_policy": true, "government_policy": true, "tax_or_tariff": true, "commodity_price": true, "currency_movement": true, "sector_development": true, "other": true}

func Validate(raw []byte) (domain.AIAnalysis, error) {
	var a domain.AIAnalysis
	d := json.NewDecoder(stringsReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return a, fmt.Errorf("invalid AI schema: %w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return a, errors.New("AI output must contain exactly one JSON object")
	}
	if a.Summary == "" || !sentiments[a.Sentiment] || !horizons[a.TimeHorizon] || !novelties[a.Novelty] || !eventCategories[a.EventCategory] {
		return a, errors.New("required AI fields are missing or invalid")
	}
	if a.SentimentScore < -100 || a.SentimentScore > 100 || a.Materiality < 0 || a.Materiality > 100 || a.Confidence < 0 || a.Confidence > 100 {
		return a, errors.New("AI scores outside allowed ranges")
	}
	if len(a.Evidence) == 0 {
		return a, errors.New("at least one evidence reference is required")
	}
	return a, nil
}

type stringReader []byte

func (s *stringReader) Read(p []byte) (int, error) {
	if len(*s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, *s)
	*s = (*s)[n:]
	return n, nil
}
func stringsReader(b []byte) *stringReader { s := stringReader(b); return &s }

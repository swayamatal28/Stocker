package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

// PortfolioAdviceInput contains only public company information. Portfolio
// quantities, cost prices, user identifiers and application secrets must never
// be included in requests to an external model.
type PortfolioAdviceInput struct {
	Symbol       string                    `json:"symbol"`
	CompanyName  string                    `json:"companyName"`
	Quote        *domain.MarketQuote       `json:"quote,omitempty"`
	Fundamentals *domain.Fundamentals      `json:"fundamentals,omitempty"`
	Research     *domain.StockResearch     `json:"ruleBasedResearch,omitempty"`
	RecentNews   []PortfolioAdviceNewsItem `json:"recentNews"`
}

type PortfolioAdviceNewsItem struct {
	Title       string    `json:"title"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"publishedAt"`
}

type PortfolioAdvice struct {
	Summary     string   `json:"summary"`
	Outlook     string   `json:"outlook"`
	Strengths   []string `json:"strengths"`
	Concerns    []string `json:"concerns"`
	WhatToWatch []string `json:"whatToWatch"`
}

type PortfolioAdvisor struct {
	endpoint, apiKey, model string
	client                  *http.Client
	maxOutputBytes          int
}

func NewPortfolioAdvisor(endpoint, apiKey, model string, timeout time.Duration, maxOutputBytes int) (*PortfolioAdvisor, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("AI endpoint must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(apiKey) == "" || strings.HasPrefix(strings.ToLower(strings.TrimSpace(apiKey)), "insert ") {
		return nil, errors.New("AI API key is not configured")
	}
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("AI model is required")
	}
	if maxOutputBytes < 1024 {
		maxOutputBytes = 64 << 10
	}
	return &PortfolioAdvisor{endpoint: endpoint, apiKey: apiKey, model: model, client: &http.Client{Timeout: timeout}, maxOutputBytes: maxOutputBytes}, nil
}

func (advisor *PortfolioAdvisor) Name() string { return "configured AI provider" }

func (advisor *PortfolioAdvisor) Advise(ctx context.Context, input PortfolioAdviceInput) (PortfolioAdvice, error) {
	payload, err := json.Marshal(map[string]any{
		"model": advisor.model,
		"task":  "portfolio_stock_advice",
		"instructions": []string{
			"Use only the supplied public company facts and recent headlines.",
			"Do not infer the user's quantity, cost price, identity, risk tolerance, or financial situation.",
			"Return JSON with summary, outlook, strengths, concerns and whatToWatch.",
			"outlook must be bullish, bearish, mixed, or neutral; avoid guarantees and direct trade instructions.",
		},
		"input": input,
	})
	if err != nil {
		return PortfolioAdvice{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, advisor.endpoint, bytes.NewReader(payload))
	if err != nil {
		return PortfolioAdvice{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+advisor.apiKey)
	response, err := advisor.client.Do(request)
	if err != nil {
		return PortfolioAdvice{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(advisor.maxOutputBytes)+1))
	if err != nil {
		return PortfolioAdvice{}, err
	}
	if len(body) > advisor.maxOutputBytes {
		return PortfolioAdvice{}, errors.New("AI provider response exceeded configured limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return PortfolioAdvice{}, fmt.Errorf("AI provider returned status %d", response.StatusCode)
	}
	var envelope struct {
		Output json.RawMessage `json:"output"`
	}
	raw := json.RawMessage(body)
	if json.Unmarshal(body, &envelope) == nil && len(envelope.Output) > 0 {
		raw = envelope.Output
	}
	var advice PortfolioAdvice
	if err := json.Unmarshal(raw, &advice); err != nil {
		return PortfolioAdvice{}, fmt.Errorf("decode AI portfolio advice: %w", err)
	}
	if err := validatePortfolioAdvice(advice); err != nil {
		return PortfolioAdvice{}, err
	}
	return advice, nil
}

func validatePortfolioAdvice(advice PortfolioAdvice) error {
	allowed := map[string]bool{"bullish": true, "bearish": true, "mixed": true, "neutral": true}
	advice.Outlook = strings.ToLower(strings.TrimSpace(advice.Outlook))
	if !allowed[advice.Outlook] {
		return errors.New("AI portfolio advice returned an invalid outlook")
	}
	if strings.TrimSpace(advice.Summary) == "" || len(advice.Summary) > 1200 {
		return errors.New("AI portfolio advice summary is missing or too long")
	}
	for _, list := range [][]string{advice.Strengths, advice.Concerns, advice.WhatToWatch} {
		if len(list) > 6 {
			return errors.New("AI portfolio advice list is too long")
		}
		for _, item := range list {
			if strings.TrimSpace(item) == "" || len(item) > 400 {
				return errors.New("AI portfolio advice contains an invalid list item")
			}
		}
	}
	return nil
}

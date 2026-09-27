package market

import (
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

const maxResponseBytes = 1 << 20

type PublicHTTPProvider struct {
	base   *url.URL
	client *http.Client
}

func NewPublicHTTPProvider(endpoint string, timeout time.Duration) (*PublicHTTPProvider, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return nil, errors.New("invalid public market endpoint")
	}
	parsed.RawQuery, parsed.Fragment = "", ""
	return &PublicHTTPProvider{base: parsed, client: &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("market provider redirects are disabled")
		},
	}}, nil
}

func (p *PublicHTTPProvider) Name() string { return "indian-stock-api-experimental" }

func (p *PublicHTTPProvider) endpoint(path string, query url.Values) string {
	u := *p.base
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	u.RawQuery = query.Encode()
	return u.String()
}

func (p *PublicHTTPProvider) get(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	// This adapter intentionally has no credential, cookie, authorization, or
	// caller-header pathway. Only public symbols/search terms leave STOCKER.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "STOCKER/1.0 market-research-client")
	response, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("market provider returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode market provider response: %w", err)
	}
	return nil
}

func (p *PublicHTTPProvider) Search(ctx context.Context, query string) ([]Instrument, error) {
	query = strings.TrimSpace(query)
	if query == "" || len(query) > 100 {
		return []Instrument{}, nil
	}
	var payload struct {
		Status  string `json:"status"`
		Results []struct {
			Symbol      string `json:"symbol"`
			CompanyName string `json:"company_name"`
			Sector      string `json:"sector"`
			Industry    string `json:"industry"`
		} `json:"results"`
		Query        string `json:"query"`
		TotalResults int    `json:"total_results"`
		Note         string `json:"note"`
		Timestamp    string `json:"timestamp"`
	}
	endpoint := p.endpoint("search", url.Values{"q": []string{query}})
	if err := p.get(ctx, endpoint, &payload); err != nil {
		if errors.Is(err, ErrNotFound) {
			return []Instrument{}, nil
		}
		return nil, err
	}
	results := make([]Instrument, 0, len(payload.Results))
	for _, result := range payload.Results {
		symbol, ok := CleanSymbol(result.Symbol)
		if !ok || strings.TrimSpace(result.CompanyName) == "" {
			continue
		}
		results = append(results, Instrument{NSESymbol: symbol, CompanyName: result.CompanyName, Sector: defaultText(result.Sector, "Unclassified"), Industry: defaultText(result.Industry, "Unclassified"), Exchange: "NSE", Source: p.Name(), SourceURL: endpoint})
		if len(results) == 20 {
			break
		}
	}
	return results, nil
}

func (p *PublicHTTPProvider) Snapshot(ctx context.Context, symbol string) (Snapshot, error) {
	clean, ok := CleanSymbol(symbol)
	if !ok {
		return Snapshot{}, ErrNotFound
	}
	var payload struct {
		Status   string `json:"status"`
		Symbol   string `json:"symbol"`
		Exchange string `json:"exchange"`
		Data     struct {
			CompanyName      string  `json:"company_name"`
			LastPrice        float64 `json:"last_price"`
			Change           float64 `json:"change"`
			PercentChange    float64 `json:"percent_change"`
			PreviousClose    float64 `json:"previous_close"`
			Open             float64 `json:"open"`
			DayHigh          float64 `json:"day_high"`
			DayLow           float64 `json:"day_low"`
			YearHigh         float64 `json:"year_high"`
			YearLow          float64 `json:"year_low"`
			Volume           int64   `json:"volume"`
			MarketCap        float64 `json:"market_cap"`
			PERatio          float64 `json:"pe_ratio"`
			DividendYield    float64 `json:"dividend_yield"`
			BookValue        float64 `json:"book_value"`
			EarningsPerShare float64 `json:"earnings_per_share"`
			Sector           string  `json:"sector"`
			Industry         string  `json:"industry"`
			Currency         string  `json:"currency"`
			LastUpdate       string  `json:"last_update"`
			Timestamp        string  `json:"timestamp"`
		} `json:"data"`
		Ticker         string         `json:"ticker"`
		ResponseFormat string         `json:"response_format"`
		Alternate      map[string]any `json:"alternate_exchange"`
	}
	endpoint := p.endpoint("stock", url.Values{"symbol": []string{clean}, "res": []string{"num"}})
	if err := p.get(ctx, endpoint, &payload); err != nil {
		return Snapshot{}, err
	}
	if payload.Status != "success" || payload.Data.LastPrice <= 0 {
		return Snapshot{}, ErrNotFound
	}
	asOf := parseProviderTime(payload.Data.LastUpdate, payload.Data.Timestamp)
	instrument := Instrument{NSESymbol: clean, CompanyName: payload.Data.CompanyName, Sector: defaultText(payload.Data.Sector, "Unclassified"), Industry: defaultText(payload.Data.Industry, "Unclassified"), Exchange: defaultText(payload.Exchange, "NSE"), Source: p.Name(), SourceURL: endpoint}
	quote := domain.MarketQuote{LastPrice: payload.Data.LastPrice, Change: payload.Data.Change, ChangePercent: payload.Data.PercentChange, PreviousClose: payload.Data.PreviousClose, Open: payload.Data.Open, DayHigh: payload.Data.DayHigh, DayLow: payload.Data.DayLow, YearHigh: payload.Data.YearHigh, YearLow: payload.Data.YearLow, Volume: payload.Data.Volume}
	metrics := []domain.FundamentalMetric{
		{Key: "market_cap", Label: "Market capitalisation", Value: payload.Data.MarketCap, Unit: payload.Data.Currency, Period: "current", Basis: "provider snapshot"},
		{Key: "pe_ratio", Label: "P/E ratio", Value: payload.Data.PERatio, Unit: "x", Period: "TTM", Basis: "trailing"},
		{Key: "dividend_yield", Label: "Dividend yield", Value: payload.Data.DividendYield, Unit: "%", Period: "TTM", Basis: "trailing"},
		{Key: "book_value", Label: "Book value per share", Value: payload.Data.BookValue, Unit: payload.Data.Currency, Period: "latest available", Basis: "per share"},
		{Key: "eps", Label: "Earnings per share", Value: payload.Data.EarningsPerShare, Unit: payload.Data.Currency, Period: "TTM", Basis: "trailing"},
	}
	return snapshotFrom(instrument, quote, metrics, p.Name(), endpoint, asOf, false), nil
}

func parseProviderTime(values ...string) time.Time {
	for _, value := range values {
		value = strings.TrimSpace(value)
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if parsed, err := time.Parse(layout, value); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Now().UTC()
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" || strings.EqualFold(value, "N/A") {
		return fallback
	}
	return strings.TrimSpace(value)
}

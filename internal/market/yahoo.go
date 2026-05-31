package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

// YahooProvider reads public, credential-free Yahoo Finance endpoints. It never
// receives or forwards application secrets, cookies, users, or portfolios.
type YahooProvider struct {
	base   *url.URL
	client *http.Client
}

func NewYahooProvider(endpoint string, timeout time.Duration) (*YahooProvider, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return nil, errors.New("Yahoo market endpoint must be an absolute HTTPS URL")
	}
	parsed.RawQuery, parsed.Fragment = "", ""
	return &YahooProvider{base: parsed, client: &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("market provider redirects are disabled")
		},
	}}, nil
}

func (p *YahooProvider) Name() string { return "Yahoo Finance (delayed)" }

func (p *YahooProvider) endpoint(path string, query url.Values) string {
	u := *p.base
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	u.RawQuery = query.Encode()
	return u.String()
}

func (p *YahooProvider) get(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
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
		return fmt.Errorf("Yahoo Finance returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(target); err != nil {
		return fmt.Errorf("decode Yahoo Finance response: %w", err)
	}
	return nil
}

func (p *YahooProvider) Search(ctx context.Context, query string) ([]Instrument, error) {
	query = strings.TrimSpace(query)
	if query == "" || len(query) > 100 {
		return []Instrument{}, nil
	}
	var payload struct {
		Quotes []struct {
			Symbol       string `json:"symbol"`
			ShortName    string `json:"shortname"`
			LongName     string `json:"longname"`
			Exchange     string `json:"exchange"`
			QuoteType    string `json:"quoteType"`
			Sector       string `json:"sector"`
			Industry     string `json:"industry"`
			SectorDisp   string `json:"sectorDisp"`
			IndustryDisp string `json:"industryDisp"`
		} `json:"quotes"`
	}
	endpoint := p.endpoint("v1/finance/search", url.Values{
		"q": {query}, "quotesCount": {"20"}, "newsCount": {"0"},
	})
	if err := p.get(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	results := make([]Instrument, 0, len(payload.Quotes))
	seen := map[string]struct{}{}
	for _, result := range payload.Quotes {
		if !strings.EqualFold(result.QuoteType, "EQUITY") || (!strings.HasSuffix(strings.ToUpper(result.Symbol), ".NS") && !strings.EqualFold(result.Exchange, "NSI")) {
			continue
		}
		symbol, ok := CleanSymbol(result.Symbol)
		companyName := defaultText(result.LongName, result.ShortName)
		if !ok || companyName == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		sector := defaultText(result.Sector, result.SectorDisp)
		industry := defaultText(result.Industry, result.IndustryDisp)
		results = append(results, Instrument{NSESymbol: symbol, CompanyName: companyName, Sector: sector, Industry: industry, Exchange: "NSE", Source: p.Name(), SourceURL: endpoint})
		if len(results) == 20 {
			break
		}
	}
	return results, nil
}

func (p *YahooProvider) Snapshot(ctx context.Context, symbol string) (Snapshot, error) {
	clean, ok := CleanSymbol(symbol)
	if !ok {
		return Snapshot{}, ErrNotFound
	}
	ticker := clean + ".NS"
	var payload struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Currency             string  `json:"currency"`
					Symbol               string  `json:"symbol"`
					ExchangeName         string  `json:"exchangeName"`
					FullExchangeName     string  `json:"fullExchangeName"`
					InstrumentType       string  `json:"instrumentType"`
					RegularMarketTime    int64   `json:"regularMarketTime"`
					RegularMarketPrice   float64 `json:"regularMarketPrice"`
					ChartPreviousClose   float64 `json:"chartPreviousClose"`
					PreviousClose        float64 `json:"previousClose"`
					RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
					RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
					FiftyTwoWeekHigh     float64 `json:"fiftyTwoWeekHigh"`
					FiftyTwoWeekLow      float64 `json:"fiftyTwoWeekLow"`
					RegularMarketVolume  int64   `json:"regularMarketVolume"`
					LongName             string  `json:"longName"`
					ShortName            string  `json:"shortName"`
				} `json:"meta"`
				Indicators struct {
					Quote []struct {
						Open   []*float64 `json:"open"`
						High   []*float64 `json:"high"`
						Low    []*float64 `json:"low"`
						Volume []*int64   `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error any `json:"error"`
		} `json:"chart"`
	}
	endpoint := p.endpoint("v8/finance/chart/"+ticker, url.Values{"range": {"1d"}, "interval": {"1d"}})
	if err := p.get(ctx, endpoint, &payload); err != nil {
		return Snapshot{}, err
	}
	if payload.Chart.Error != nil || len(payload.Chart.Result) == 0 {
		return Snapshot{}, ErrNotFound
	}
	result := payload.Chart.Result[0]
	meta := result.Meta
	if meta.RegularMarketPrice <= 0 || !strings.EqualFold(meta.InstrumentType, "EQUITY") {
		return Snapshot{}, ErrNotFound
	}
	previousClose := meta.ChartPreviousClose
	if previousClose <= 0 {
		previousClose = meta.PreviousClose
	}
	change, changePercent := 0.0, 0.0
	if previousClose > 0 {
		change = meta.RegularMarketPrice - previousClose
		changePercent = change / previousClose * 100
	}
	open, high, low, volume := 0.0, meta.RegularMarketDayHigh, meta.RegularMarketDayLow, meta.RegularMarketVolume
	if len(result.Indicators.Quote) > 0 {
		quote := result.Indicators.Quote[0]
		open = lastFloat(quote.Open)
		if high <= 0 {
			high = lastFloat(quote.High)
		}
		if low <= 0 {
			low = lastFloat(quote.Low)
		}
		if volume <= 0 {
			volume = lastInt(quote.Volume)
		}
	}
	asOf := time.Now().UTC()
	if meta.RegularMarketTime > 0 {
		asOf = time.Unix(meta.RegularMarketTime, 0).UTC()
	}
	companyName := defaultText(meta.LongName, meta.ShortName)
	if companyName == "" {
		companyName = clean
	}
	instrument := Instrument{NSESymbol: clean, CompanyName: companyName, Exchange: "NSE", Source: p.Name(), SourceURL: endpoint}
	// Yahoo's chart response supplies prices but not sector metadata. Its search
	// response contains the classification for NSE equities, so enrich the
	// snapshot when available. Failure is deliberately non-fatal: live prices
	// should still update, and the store preserves any existing classification.
	if matches, searchErr := p.Search(ctx, clean); searchErr == nil {
		for _, candidate := range matches {
			if candidate.NSESymbol != clean {
				continue
			}
			instrument.Sector = candidate.Sector
			instrument.Industry = candidate.Industry
			if candidate.CompanyName != "" {
				instrument.CompanyName = candidate.CompanyName
			}
			break
		}
	}
	quote := domain.MarketQuote{
		LastPrice: roundMarket(meta.RegularMarketPrice), Change: roundMarket(change), ChangePercent: roundMarket(changePercent),
		PreviousClose: roundMarket(previousClose), Open: roundMarket(open), DayHigh: roundMarket(high), DayLow: roundMarket(low),
		YearHigh: roundMarket(meta.FiftyTwoWeekHigh), YearLow: roundMarket(meta.FiftyTwoWeekLow), Volume: volume,
	}
	metrics, fundamentalAsOf, fundamentalURL := p.fundamentals(ctx, ticker)
	snapshot := snapshotFrom(instrument, quote, metrics, p.Name(), endpoint, asOf, false)
	if len(metrics) > 0 {
		snapshot.Fundamentals.SourceURL = fundamentalURL
		if !fundamentalAsOf.IsZero() {
			snapshot.Fundamentals.AsOf = fundamentalAsOf
		}
	}
	return snapshot, nil
}

type yahooFundamentalPoint struct {
	AsOfDate      string `json:"asOfDate"`
	PeriodType    string `json:"periodType"`
	CurrencyCode  string `json:"currencyCode"`
	ReportedValue struct {
		Raw float64 `json:"raw"`
	} `json:"reportedValue"`
}

func (p *YahooProvider) fundamentals(ctx context.Context, ticker string) ([]domain.FundamentalMetric, time.Time, string) {
	types := []string{
		"trailingMarketCap", "trailingPeRatio", "trailingDilutedEPS", "trailingTotalRevenue", "trailingNetIncome",
		"annualTotalDebt", "annualStockholdersEquity", "annualCurrentAssets", "annualCurrentLiabilities",
		"annualOperatingCashFlow", "annualFreeCashFlow", "annualTotalRevenue", "annualNetIncome", "annualOperatingIncome",
	}
	now := time.Now().UTC()
	endpoint := p.endpoint("ws/fundamentals-timeseries/v1/finance/timeseries/"+ticker, url.Values{
		"symbol": {ticker}, "type": {strings.Join(types, ",")},
		"period1": {fmt.Sprintf("%d", now.AddDate(-5, 0, 0).Unix())}, "period2": {fmt.Sprintf("%d", now.Unix())},
	})
	var payload struct {
		Timeseries struct {
			Result []map[string]json.RawMessage `json:"result"`
			Error  any                          `json:"error"`
		} `json:"timeseries"`
	}
	if err := p.get(ctx, endpoint, &payload); err != nil || payload.Timeseries.Error != nil {
		return nil, time.Time{}, endpoint
	}
	series := map[string][]yahooFundamentalPoint{}
	latestAsOf := time.Time{}
	for _, result := range payload.Timeseries.Result {
		var meta struct {
			Type []string `json:"type"`
		}
		if raw := result["meta"]; len(raw) > 0 {
			_ = json.Unmarshal(raw, &meta)
		}
		if len(meta.Type) == 0 {
			continue
		}
		key := meta.Type[0]
		if raw := result[key]; len(raw) > 0 {
			var points []yahooFundamentalPoint
			if json.Unmarshal(raw, &points) == nil && len(points) > 0 {
				series[key] = points
				if parsed, err := time.Parse("2006-01-02", points[len(points)-1].AsOfDate); err == nil && parsed.After(latestAsOf) {
					latestAsOf = parsed.UTC()
				}
			}
		}
	}
	latest := func(key string) (yahooFundamentalPoint, bool) {
		points := series[key]
		if len(points) == 0 {
			return yahooFundamentalPoint{}, false
		}
		return points[len(points)-1], true
	}
	metrics := []domain.FundamentalMetric{}
	add := func(key, label, sourceKey, unit, period, basis string) {
		if point, ok := latest(sourceKey); ok {
			metrics = append(metrics, domain.FundamentalMetric{Key: key, Label: label, Value: point.ReportedValue.Raw, Unit: defaultText(point.CurrencyCode, unit), Period: defaultText(point.PeriodType, period), Basis: basis})
		}
	}
	add("market_cap", "Market capitalisation", "trailingMarketCap", "INR", "current", "latest available")
	add("pe_ratio", "P/E ratio", "trailingPeRatio", "x", "TTM", "share price divided by trailing earnings per share")
	add("eps", "Earnings per share", "trailingDilutedEPS", "INR", "TTM", "diluted")
	add("revenue", "Revenue", "trailingTotalRevenue", "INR", "TTM", "latest available")
	add("net_income", "Net income", "trailingNetIncome", "INR", "TTM", "latest available")
	add("total_debt", "Total debt", "annualTotalDebt", "INR", "FY", "latest annual")
	add("shareholders_equity", "Shareholder equity", "annualStockholdersEquity", "INR", "FY", "latest annual")
	add("current_assets", "Current assets", "annualCurrentAssets", "INR", "FY", "latest annual")
	add("current_liabilities", "Current liabilities", "annualCurrentLiabilities", "INR", "FY", "latest annual")
	add("operating_cash_flow", "Operating cash flow", "annualOperatingCashFlow", "INR", "FY", "latest annual")
	add("free_cash_flow", "Free cash flow", "annualFreeCashFlow", "INR", "FY", "latest annual")
	derive := func(key, label, unit, formula string, numerator, denominator float64) {
		if denominator != 0 {
			metrics = append(metrics, domain.FundamentalMetric{Key: key, Label: label, Value: roundMarket(numerator / denominator), Unit: unit, Period: "latest annual", Basis: formula})
		}
	}
	debt, debtOK := latest("annualTotalDebt")
	equity, equityOK := latest("annualStockholdersEquity")
	if debtOK && equityOK {
		derive("debt_to_equity", "Debt to equity", "x", "total debt divided by shareholder equity", debt.ReportedValue.Raw, equity.ReportedValue.Raw)
	}
	assets, assetsOK := latest("annualCurrentAssets")
	liabilities, liabilitiesOK := latest("annualCurrentLiabilities")
	if assetsOK && liabilitiesOK {
		derive("current_ratio", "Current ratio", "x", "current assets divided by current liabilities", assets.ReportedValue.Raw, liabilities.ReportedValue.Raw)
	}
	netIncome, incomeOK := latest("annualNetIncome")
	if incomeOK && equityOK {
		derive("roe", "Return on equity", "%", "net income divided by shareholder equity", netIncome.ReportedValue.Raw*100, equity.ReportedValue.Raw)
	}
	operatingIncome, operatingOK := latest("annualOperatingIncome")
	revenue, revenueOK := latest("annualTotalRevenue")
	if operatingOK && revenueOK {
		derive("operating_margin", "Operating margin", "%", "operating income divided by revenue", operatingIncome.ReportedValue.Raw*100, revenue.ReportedValue.Raw)
	}
	if points := series["annualTotalRevenue"]; len(points) >= 2 && points[len(points)-2].ReportedValue.Raw != 0 {
		growth := (points[len(points)-1].ReportedValue.Raw/points[len(points)-2].ReportedValue.Raw - 1) * 100
		metrics = append(metrics, domain.FundamentalMetric{Key: "revenue_growth", Label: "Annual revenue growth", Value: roundMarket(growth), Unit: "%", Period: "latest annual", Basis: "latest full year compared with previous full year"})
	}
	return metrics, latestAsOf, endpoint
}

func lastFloat(values []*float64) float64 {
	for index := len(values) - 1; index >= 0; index-- {
		if values[index] != nil {
			return *values[index]
		}
	}
	return 0
}

func lastInt(values []*int64) int64 {
	for index := len(values) - 1; index >= 0; index-- {
		if values[index] != nil {
			return *values[index]
		}
	}
	return 0
}

func roundMarket(value float64) float64 { return math.Round(value*100) / 100 }

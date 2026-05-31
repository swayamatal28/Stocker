package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestYahooProviderParsesNSESearchAndQuoteWithoutCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("X-API-Key") != "" {
			t.Fatal("Yahoo provider must not send credentials or cookies")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/finance/search":
			_, _ = w.Write([]byte(`{"quotes":[{"exchange":"NSI","shortname":"OLA ELECTRIC MOBILITY LTD","longname":"Ola Electric Mobility Limited","quoteType":"EQUITY","symbol":"OLAELEC.NS","sector":"Consumer Cyclical","industry":"Auto Manufacturers"},{"exchange":"BSE","longname":"Ola Electric Mobility Limited","quoteType":"EQUITY","symbol":"OLAELEC.BO","sector":"Consumer Cyclical","industry":"Auto Manufacturers"}]}`))
		case "/v8/finance/chart/OLAELEC.NS":
			_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"INR","symbol":"OLAELEC.NS","exchangeName":"NSI","fullExchangeName":"NSE","instrumentType":"EQUITY","regularMarketTime":1790330399,"regularMarketPrice":38.48,"chartPreviousClose":42.52,"regularMarketDayHigh":42.5,"regularMarketDayLow":38.01,"fiftyTwoWeekHigh":58.65,"fiftyTwoWeekLow":22.25,"regularMarketVolume":194452956,"longName":"Ola Electric Mobility Limited"},"indicators":{"quote":[{"open":[41.5],"high":[42.5],"low":[38.01],"volume":[194452956]}]}}],"error":null}}`))
		case "/ws/fundamentals-timeseries/v1/finance/timeseries/OLAELEC.NS":
			_, _ = w.Write([]byte(`{"timeseries":{"result":[{"meta":{"type":["trailingPeRatio"]},"trailingPeRatio":[{"asOfDate":"2026-06-30","periodType":"TTM","reportedValue":{"raw":25}}]},{"meta":{"type":["annualTotalDebt"]},"annualTotalDebt":[{"asOfDate":"2026-03-31","periodType":"FY","currencyCode":"INR","reportedValue":{"raw":20}}]},{"meta":{"type":["annualStockholdersEquity"]},"annualStockholdersEquity":[{"asOfDate":"2026-03-31","periodType":"FY","currencyCode":"INR","reportedValue":{"raw":100}}]},{"meta":{"type":["annualTotalRevenue"]},"annualTotalRevenue":[{"asOfDate":"2025-03-31","reportedValue":{"raw":100}},{"asOfDate":"2026-03-31","reportedValue":{"raw":120}}]}],"error":null}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	provider := &YahooProvider{base: base, client: server.Client()}
	results, err := provider.Search(context.Background(), "OLA")
	if err != nil || len(results) != 1 || results[0].NSESymbol != "OLAELEC" {
		t.Fatalf("unexpected Yahoo search result: %#v, %v", results, err)
	}
	snapshot, err := provider.Snapshot(context.Background(), "OLAELEC")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Quote.LastPrice != 38.48 || snapshot.Quote.DayHigh != 42.5 || snapshot.Quote.DayLow != 38.01 || snapshot.Quote.Synthetic {
		t.Fatalf("unexpected live quote: %#v", snapshot.Quote)
	}
	if snapshot.Quote.AsOf.Equal(time.Time{}) || snapshot.Instrument.CompanyName != "Ola Electric Mobility Limited" {
		t.Fatalf("missing live quote metadata: %#v", snapshot)
	}
	if snapshot.Instrument.Sector != "Consumer Cyclical" || snapshot.Instrument.Industry != "Auto Manufacturers" {
		t.Fatalf("missing live classification metadata: %#v", snapshot.Instrument)
	}
	if len(snapshot.Fundamentals.Metrics) < 4 {
		t.Fatalf("expected live fundamental and calculated metrics: %#v", snapshot.Fundamentals)
	}
}

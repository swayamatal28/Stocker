package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPublicHTTPProviderSendsOnlyPublicRequestData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("X-API-Key") != "" {
			t.Fatal("public market adapter must never send credentials or cookies")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search":
			if r.URL.Query().Get("q") != "OLA" {
				t.Fatalf("unexpected public search term: %q", r.URL.Query().Get("q"))
			}
			_, _ = w.Write([]byte(`{"status":"success","query":"OLA","total_results":1,"results":[{"symbol":"OLAELEC","company_name":"Ola Electric Mobility Limited","sector":"Consumer Discretionary","industry":"Electric Mobility","source":"yahoo","api_url":"/stock?symbol=OLAELEC"}],"timestamp":"2026-09-27 10:00:00"}`))
		case "/stock":
			if r.URL.Query().Get("symbol") != "OLAELEC" || r.URL.Query().Get("res") != "num" {
				t.Fatalf("unexpected stock query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"status":"success","symbol":"OLAELEC","exchange":"NSE","ticker":"OLAELEC.NS","response_format":"numeric_only","data":{"company_name":"Ola Electric Mobility Limited","last_price":42,"change":1.1,"percent_change":2.69,"previous_close":40.9,"open":41.05,"day_high":42.7,"day_low":40.8,"year_high":102.5,"year_low":30.75,"volume":17234000,"market_cap":185000000000,"pe_ratio":0,"dividend_yield":0,"book_value":25.28,"earnings_per_share":-5.4,"sector":"Consumer Discretionary","industry":"Electric Mobility","currency":"INR","last_update":"2026-09-27","timestamp":"2026-09-27 10:00:00"},"alternate_exchange":{"exchange":"BSE"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	provider, err := NewPublicHTTPProvider(server.URL, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	results, err := provider.Search(context.Background(), "OLA")
	if err != nil || len(results) != 1 || results[0].NSESymbol != "OLAELEC" {
		t.Fatalf("unexpected search result: %#v, %v", results, err)
	}
	snapshot, err := provider.Snapshot(context.Background(), "OLAELEC")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Quote.DayHigh != 42.7 || snapshot.Quote.DayLow != 40.8 || snapshot.Quote.Source != provider.Name() {
		t.Fatalf("unexpected quote: %#v", snapshot.Quote)
	}
	if len(snapshot.Fundamentals.Metrics) != 5 || snapshot.Fundamentals.Metrics[1].Period != "TTM" {
		t.Fatalf("fundamental metadata missing: %#v", snapshot.Fundamentals)
	}
}

func TestPublicHTTPProviderRejectsRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("redirect target must not be contacted") }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer server.Close()
	provider, err := NewPublicHTTPProvider(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Search(context.Background(), "OLA"); err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("expected redirect rejection, got %v", err)
	}
}

func TestFixtureIncludesOlaElectricIdentity(t *testing.T) {
	results, err := (FixtureProvider{}).Search(context.Background(), "OLA")
	if err != nil || len(results) != 1 {
		t.Fatalf("unexpected fixture result: %#v, %v", results, err)
	}
	if results[0].ISIN != "INE0LXG01040" || results[0].BSECode != "544225" {
		t.Fatalf("OLA identity is incomplete: %#v", results[0])
	}
}

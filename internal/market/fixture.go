package market

import (
	"context"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

type FixtureProvider struct{}

func (FixtureProvider) Name() string { return "stocker-market-fixture" }

var fixtureInstruments = []Instrument{
	{NSESymbol: "RELIANCE", BSECode: "500325", ISIN: "INE002A01018", CompanyName: "Reliance Industries Limited", Sector: "Energy", Industry: "Oil, Gas & Consumable Fuels", Exchange: "NSE"},
	{NSESymbol: "HDFCBANK", BSECode: "500180", ISIN: "INE040A01034", CompanyName: "HDFC Bank Limited", Sector: "Financial Services", Industry: "Banks", Exchange: "NSE"},
	{NSESymbol: "INFY", BSECode: "500209", ISIN: "INE009A01021", CompanyName: "Infosys Limited", Sector: "Information Technology", Industry: "IT Services & Consulting", Exchange: "NSE"},
	{NSESymbol: "TCS", BSECode: "532540", ISIN: "INE467B01029", CompanyName: "Tata Consultancy Services Limited", Sector: "Information Technology", Industry: "IT Services & Consulting", Exchange: "NSE"},
	{NSESymbol: "ITC", BSECode: "500875", ISIN: "INE154A01025", CompanyName: "ITC Limited", Sector: "Consumer Staples", Industry: "Diversified FMCG", Exchange: "NSE"},
	{NSESymbol: "LT", BSECode: "500510", ISIN: "INE018A01030", CompanyName: "Larsen & Toubro Limited", Sector: "Industrials", Industry: "Engineering", Exchange: "NSE"},
	{NSESymbol: "OLAELEC", BSECode: "544225", ISIN: "INE0LXG01040", CompanyName: "Ola Electric Mobility Limited", Sector: "Consumer Discretionary", Industry: "Automobiles - Electric Mobility", Exchange: "NSE"},
}

func (FixtureProvider) Search(_ context.Context, query string) ([]Instrument, error) {
	query = strings.ToUpper(strings.TrimSpace(query))
	results := []Instrument{}
	for _, item := range fixtureInstruments {
		if strings.Contains(strings.ToUpper(item.CompanyName), query) || strings.HasPrefix(item.NSESymbol, query) || strings.HasPrefix(item.BSECode, query) || strings.HasPrefix(item.ISIN, query) {
			item.Source = "STOCKER synthetic market fixture"
			item.SourceURL = "https://example.invalid/stocker-market-fixture"
			results = append(results, item)
		}
	}
	return results, nil
}

func (FixtureProvider) Snapshot(_ context.Context, symbol string) (Snapshot, error) {
	clean, ok := CleanSymbol(symbol)
	if !ok {
		return Snapshot{}, ErrNotFound
	}
	prices := map[string]domain.MarketQuote{
		"RELIANCE": {LastPrice: 1392.40, Change: 9.90, ChangePercent: .72, PreviousClose: 1382.50, Open: 1381, DayHigh: 1401.20, DayLow: 1378.35, YearHigh: 1608.80, YearLow: 1114.85, Volume: 8210050},
		"HDFCBANK": {LastPrice: 983.15, Change: 5.55, ChangePercent: .57, PreviousClose: 977.60, Open: 978.40, DayHigh: 988.30, DayLow: 974.50, YearHigh: 1021.70, YearLow: 812.15, Volume: 10342110},
		"INFY":     {LastPrice: 1518.20, Change: -13.90, ChangePercent: -.91, PreviousClose: 1532.10, Open: 1530, DayHigh: 1538.40, DayLow: 1511.25, YearHigh: 2006.80, YearLow: 1307.10, Volume: 5903210},
		"TCS":      {LastPrice: 3124.60, Change: 17.35, ChangePercent: .56, PreviousClose: 3107.25, Open: 3110, DayHigh: 3140.20, DayLow: 3098.40, YearHigh: 4592.25, YearLow: 2866.55, Volume: 2219030},
		"ITC":      {LastPrice: 421.80, Change: -1.25, ChangePercent: -.30, PreviousClose: 423.05, Open: 423.20, DayHigh: 425.10, DayLow: 420.15, YearHigh: 528.55, YearLow: 390.15, Volume: 11231000},
		"LT":       {LastPrice: 3684.90, Change: 31.20, ChangePercent: .85, PreviousClose: 3653.70, Open: 3658, DayHigh: 3701.40, DayLow: 3642.80, YearHigh: 3963.50, YearLow: 2965.30, Volume: 1849220},
		"OLAELEC":  {LastPrice: 42, Change: 1.10, ChangePercent: 2.69, PreviousClose: 40.90, Open: 41.05, DayHigh: 42.70, DayLow: 40.80, YearHigh: 102.50, YearLow: 30.75, Volume: 17234000},
	}
	for _, item := range fixtureInstruments {
		if item.NSESymbol != clean && item.BSECode != clean && item.ISIN != clean {
			continue
		}
		item.Source = "STOCKER synthetic market fixture"
		item.SourceURL = "https://example.invalid/stocker-market-fixture"
		quote := prices[item.NSESymbol]
		metrics := []domain.FundamentalMetric{
			{Key: "market_cap", Label: "Market capitalisation", Value: quote.LastPrice * 1000000, Unit: "INR", Period: "current", Basis: "synthetic demonstration"},
			{Key: "pe_ratio", Label: "P/E ratio", Value: 0, Unit: "x", Period: "TTM", Basis: "not available in fixture"},
		}
		return snapshotFrom(item, quote, metrics, item.Source, item.SourceURL, time.Now().UTC().Add(-15*time.Minute), true), nil
	}
	return Snapshot{}, ErrNotFound
}

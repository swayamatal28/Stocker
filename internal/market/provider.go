package market

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

var ErrNotFound = errors.New("market instrument not found")

var symbolPattern = regexp.MustCompile(`^[A-Z0-9&.-]{1,24}$`)

type Instrument struct {
	NSESymbol   string
	BSECode     string
	ISIN        string
	CompanyName string
	Sector      string
	Industry    string
	Exchange    string
	Source      string
	SourceURL   string
}

type Snapshot struct {
	Instrument   Instrument
	Quote        domain.MarketQuote
	Fundamentals domain.Fundamentals
}

type Provider interface {
	Name() string
	Search(context.Context, string) ([]Instrument, error)
	Snapshot(context.Context, string) (Snapshot, error)
}

func CleanSymbol(value string) (string, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.TrimSuffix(strings.TrimSuffix(value, ".NS"), ".BO")
	return value, symbolPattern.MatchString(value)
}

func snapshotFrom(instrument Instrument, quote domain.MarketQuote, metrics []domain.FundamentalMetric, source, sourceURL string, asOf time.Time, synthetic bool) Snapshot {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	now := time.Now().UTC()
	quote.Symbol = instrument.NSESymbol
	quote.Exchange = instrument.Exchange
	quote.Currency = "INR"
	quote.Source = source
	quote.SourceURL = sourceURL
	quote.AsOf = asOf.UTC()
	quote.RetrievedAt = now
	quote.IsDelayed = true
	quote.Synthetic = synthetic
	return Snapshot{Instrument: instrument, Quote: quote, Fundamentals: domain.Fundamentals{
		Symbol: instrument.NSESymbol, Metrics: metrics, Source: source, SourceURL: sourceURL,
		AsOf: asOf.UTC(), RetrievedAt: now, Synthetic: synthetic,
	}}
}

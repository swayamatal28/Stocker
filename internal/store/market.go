package store

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stocker-app/stocker/internal/domain"
)

type fundamentalsDocument struct {
	SecurityID  bson.ObjectID              `bson:"security_id"`
	Symbol      string                     `bson:"symbol"`
	Metrics     []domain.FundamentalMetric `bson:"metrics"`
	Source      string                     `bson:"source"`
	SourceURL   string                     `bson:"source_url,omitempty"`
	AsOf        time.Time                  `bson:"as_of"`
	RetrievedAt time.Time                  `bson:"retrieved_at"`
	Synthetic   bool                       `bson:"synthetic"`
}

func (m *Mongo) UpsertMarketSecurity(ctx context.Context, nseSymbol, bseCode, isin, companyName, sector, industry, source, sourceURL string) error {
	nseSymbol = strings.ToUpper(strings.TrimSpace(nseSymbol))
	bseCode = strings.ToUpper(strings.TrimSpace(bseCode))
	isin = strings.ToUpper(strings.TrimSpace(isin))
	if nseSymbol == "" && bseCode == "" {
		return ErrNotFound
	}
	filter := bson.M{"nse_symbol": nseSymbol}
	if nseSymbol == "" {
		filter = bson.M{"bse_code": bseCode}
	}
	set := bson.M{"active": true, "updated_at": time.Now().UTC()}
	for key, value := range map[string]string{"nse_symbol": nseSymbol, "bse_code": bseCode, "isin": isin, "company_name": strings.TrimSpace(companyName), "sector": strings.TrimSpace(sector), "industry": strings.TrimSpace(industry), "source": strings.TrimSpace(source), "source_url": strings.TrimSpace(sourceURL)} {
		if value != "" {
			set[key] = value
		}
	}
	aliases := []string{}
	companyAlias := strings.ToUpper(strings.TrimSpace(companyName))
	for _, suffix := range []string{" LIMITED", " LTD.", " LTD"} {
		companyAlias = strings.TrimSuffix(companyAlias, suffix)
	}
	if len(companyAlias) >= 5 {
		aliases = append(aliases, companyAlias)
	}
	update := bson.M{"$set": set, "$setOnInsert": bson.M{"created_at": time.Now().UTC()}}
	if len(aliases) > 0 {
		update["$addToSet"] = bson.M{"aliases": bson.M{"$each": aliases}}
	}
	_, err := m.DB.Collection("securities").UpdateOne(ctx, filter, update,
		options.UpdateOne().SetUpsert(true))
	return err
}

func (m *Mongo) SaveMarketSnapshot(ctx context.Context, symbol string, quote domain.MarketQuote, fundamentals domain.Fundamentals) error {
	security, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return err
	}
	if quote.AsOf.IsZero() || quote.Source == "" || quote.LastPrice <= 0 {
		return errors.New("market quote requires a positive price, source, and as-of timestamp")
	}
	if quote.RetrievedAt.IsZero() {
		quote.RetrievedAt = time.Now().UTC()
	}
	quoteDoc := quoteDocument{
		SecurityID: security.ID, LastPrice: quote.LastPrice, Change: quote.Change, ChangePercent: quote.ChangePercent,
		PreviousClose: quote.PreviousClose, Open: quote.Open, High: quote.DayHigh, Low: quote.DayLow, YearHigh: quote.YearHigh,
		YearLow: quote.YearLow, Volume: quote.Volume, Currency: quote.Currency, Source: quote.Source, SourceURL: quote.SourceURL,
		IsDelayed: quote.IsDelayed, Synthetic: quote.Synthetic, AsOf: quote.AsOf.UTC(), RetrievedAt: quote.RetrievedAt.UTC(),
	}
	_, err = m.DB.Collection("market_quotes").UpdateOne(ctx,
		bson.M{"security_id": security.ID, "source": quote.Source, "as_of": quote.AsOf.UTC()},
		bson.M{"$set": quoteDoc}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return err
	}
	if len(fundamentals.Metrics) == 0 {
		return nil
	}
	if fundamentals.AsOf.IsZero() {
		fundamentals.AsOf = quote.AsOf
	}
	if fundamentals.RetrievedAt.IsZero() {
		fundamentals.RetrievedAt = quote.RetrievedAt
	}
	document := fundamentalsDocument{SecurityID: security.ID, Symbol: security.NSESymbol, Metrics: fundamentals.Metrics, Source: fundamentals.Source, SourceURL: fundamentals.SourceURL, AsOf: fundamentals.AsOf.UTC(), RetrievedAt: fundamentals.RetrievedAt.UTC(), Synthetic: fundamentals.Synthetic}
	_, err = m.DB.Collection("fundamentals").UpdateOne(ctx,
		bson.M{"security_id": security.ID, "source": document.Source, "as_of": document.AsOf},
		bson.M{"$set": document}, options.UpdateOne().SetUpsert(true))
	return err
}

func quoteFromDocument(symbol string, document quoteDocument) domain.MarketQuote {
	return domain.MarketQuote{Symbol: symbol, Exchange: "NSE", Currency: document.Currency, LastPrice: document.LastPrice, Change: document.Change, ChangePercent: document.ChangePercent, PreviousClose: document.PreviousClose, Open: document.Open, DayHigh: document.High, DayLow: document.Low, YearHigh: document.YearHigh, YearLow: document.YearLow, Volume: document.Volume, Source: document.Source, SourceURL: document.SourceURL, AsOf: document.AsOf, RetrievedAt: document.RetrievedAt, IsDelayed: document.IsDelayed, Synthetic: document.Synthetic}
}

func (m *Mongo) MarketQuoteBySymbol(ctx context.Context, symbol string) (domain.MarketQuote, error) {
	security, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return domain.MarketQuote{}, err
	}
	var document quoteDocument
	err = m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": security.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.MarketQuote{}, ErrNotFound
	}
	return quoteFromDocument(security.NSESymbol, document), err
}

func (m *Mongo) MarketQuoteFresh(ctx context.Context, symbol string, maximumAge time.Duration) (bool, error) {
	quote, err := m.MarketQuoteBySymbol(ctx, symbol)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	stamp := quote.RetrievedAt
	if stamp.IsZero() {
		stamp = quote.AsOf
	}
	return time.Since(stamp) <= maximumAge, nil
}

func (m *Mongo) FundamentalsBySymbol(ctx context.Context, symbol string) (domain.Fundamentals, error) {
	security, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return domain.Fundamentals{}, err
	}
	var document fundamentalsDocument
	err = m.DB.Collection("fundamentals").FindOne(ctx, bson.M{"security_id": security.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Fundamentals{}, ErrNotFound
	}
	return domain.Fundamentals{Symbol: security.NSESymbol, Metrics: document.Metrics, Source: document.Source, SourceURL: document.SourceURL, AsOf: document.AsOf, RetrievedAt: document.RetrievedAt, Synthetic: document.Synthetic}, err
}

func (m *Mongo) SectorSnapshots(ctx context.Context) ([]domain.SectorSnapshot, error) {
	cur, err := m.DB.Collection("securities").Find(ctx, bson.M{"active": true})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	type accumulator struct {
		item        domain.SectorSnapshot
		changeTotal float64
		signalTotal float64
		signalCount int
	}
	bySector := map[string]*accumulator{}
	for cur.Next(ctx) {
		var security securityDocument
		if err := cur.Decode(&security); err != nil {
			return nil, err
		}
		sector := security.Sector
		if sector == "" {
			sector = "Unclassified"
		}
		entry := bySector[sector]
		if entry == nil {
			entry = &accumulator{item: domain.SectorSnapshot{Sector: sector}}
			bySector[sector] = entry
		}
		var quote quoteDocument
		if err := m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": security.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&quote); err == nil {
			entry.item.CompanyCount++
			entry.changeTotal += quote.ChangePercent
			if quote.ChangePercent > 0 {
				entry.item.Advances++
			} else if quote.ChangePercent < 0 {
				entry.item.Declines++
			}
			if quote.AsOf.After(entry.item.LatestMarketAsOf) {
				entry.item.LatestMarketAsOf = quote.AsOf
			}
		}
		var signal signalDocument
		if err := m.DB.Collection("signals").FindOne(ctx, bson.M{"security_id": security.ID}, options.FindOne().SetSort(bson.D{{Key: "generated_at", Value: -1}})).Decode(&signal); err == nil {
			entry.signalTotal += signal.Score
			entry.signalCount++
			if signal.GeneratedAt.After(entry.item.LatestEvidenceAt) {
				entry.item.LatestEvidenceAt = signal.GeneratedAt
			}
		}
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.SectorSnapshot, 0, len(bySector))
	for _, entry := range bySector {
		if entry.item.CompanyCount > 0 {
			entry.item.AverageChange = round2(entry.changeTotal / float64(entry.item.CompanyCount))
		}
		if entry.signalCount > 0 {
			entry.item.AverageSentiment = round2(entry.signalTotal / float64(entry.signalCount))
		}
		result = append(result, entry.item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].AverageChange > result[j].AverageChange })
	return result, nil
}

func (m *Mongo) MarketMovers(ctx context.Context, limit int) ([]domain.MarketMover, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	cur, err := m.DB.Collection("securities").Find(ctx, bson.M{"active": true})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	result := []domain.MarketMover{}
	for cur.Next(ctx) {
		var security securityDocument
		if err := cur.Decode(&security); err != nil {
			return nil, err
		}
		var quote quoteDocument
		if err := m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": security.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&quote); err != nil {
			continue
		}
		result = append(result, domain.MarketMover{Symbol: security.NSESymbol, CompanyName: security.CompanyName, Sector: security.Sector, LastPrice: quote.LastPrice, ChangePercent: quote.ChangePercent, Volume: quote.Volume, AsOf: quote.AsOf, Source: quote.Source})
	}
	sort.Slice(result, func(i, j int) bool { return math.Abs(result[i].ChangePercent) > math.Abs(result[j].ChangePercent) })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, cur.Err()
}

func (m *Mongo) MarketEvents(ctx context.Context, symbol string, from time.Time, limit int) ([]domain.MarketEvent, error) {
	if limit < 1 || limit > 100 {
		limit = 30
	}
	filter := bson.M{"created_at": bson.M{"$gte": from.UTC()}}
	cur, err := m.DB.Collection("ai_analyses").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit*3)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	wanted := strings.ToUpper(strings.TrimSpace(symbol))
	result := []domain.MarketEvent{}
	for cur.Next(ctx) && len(result) < limit {
		var analysis analysisDocument
		if err := cur.Decode(&analysis); err != nil {
			return nil, err
		}
		var article newsDocument
		if err := m.DB.Collection("normalized_articles").FindOne(ctx, bson.M{"_id": analysis.ArticleID}).Decode(&article); err != nil {
			continue
		}
		matchedSymbol := ""
		for _, linked := range article.Symbols {
			if wanted == "" || strings.EqualFold(linked, wanted) {
				matchedSymbol = linked
				break
			}
		}
		if wanted != "" && matchedSymbol == "" {
			continue
		}
		result = append(result, domain.MarketEvent{ID: analysis.ID.Hex(), Symbol: matchedSymbol, Title: article.Title, Category: analysis.Analysis.EventCategory, EventAt: article.PublishedAt, Source: article.SourceName, SourceURL: article.CanonicalURL, EvidenceID: article.ID.Hex(), Official: article.Official, Synthetic: article.Synthetic})
	}
	return result, cur.Err()
}

func (m *Mongo) PeersBySymbol(ctx context.Context, symbol string, limit int) ([]domain.PeerSnapshot, error) {
	security, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > 20 {
		limit = 5
	}
	cur, err := m.DB.Collection("securities").Find(ctx, bson.M{"active": true, "sector": security.Sector, "_id": bson.M{"$ne": security.ID}}, options.Find().SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	result := []domain.PeerSnapshot{}
	for cur.Next(ctx) {
		var peer securityDocument
		if err := cur.Decode(&peer); err != nil {
			return nil, err
		}
		item := domain.PeerSnapshot{Symbol: peer.NSESymbol, CompanyName: peer.CompanyName}
		var quote quoteDocument
		if err := m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": peer.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&quote); err == nil {
			item.LastPrice, item.ChangePercent = quote.LastPrice, quote.ChangePercent
		}
		var fundamentals fundamentalsDocument
		if err := m.DB.Collection("fundamentals").FindOne(ctx, bson.M{"security_id": peer.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&fundamentals); err == nil {
			item.PERatio = metricValue(fundamentals.Metrics, "pe_ratio")
			item.MarketCap = metricValue(fundamentals.Metrics, "market_cap")
		}
		result = append(result, item)
	}
	return result, cur.Err()
}

func (m *Mongo) RiskFlagsBySymbol(ctx context.Context, symbol string) ([]domain.RiskFlag, error) {
	quote, quoteErr := m.MarketQuoteBySymbol(ctx, symbol)
	if quoteErr != nil && !errors.Is(quoteErr, ErrNotFound) {
		return nil, quoteErr
	}
	flags := []domain.RiskFlag{}
	if errors.Is(quoteErr, ErrNotFound) {
		flags = append(flags, domain.RiskFlag{Code: "missing_market_data", Severity: "high", Title: "Market data unavailable", Explanation: "No provider snapshot is available; price-based conclusions must not be drawn."})
		return flags, nil
	}
	if time.Since(quote.AsOf) > 30*time.Minute {
		flags = append(flags, domain.RiskFlag{Code: "stale_market_data", Severity: "medium", Title: "Delayed market snapshot", Explanation: "The latest price is older than 30 minutes. Always inspect the displayed source timestamp."})
	}
	if quote.YearLow > 0 && quote.LastPrice <= quote.YearLow*1.1 {
		flags = append(flags, domain.RiskFlag{Code: "near_year_low", Severity: "medium", Title: "Near 52-week low", Explanation: "The latest price is within 10% of the provider-reported 52-week low."})
	}
	if quote.ChangePercent <= -3 {
		flags = append(flags, domain.RiskFlag{Code: "sharp_daily_decline", Severity: "medium", Title: "Sharp daily decline", Explanation: "The provider reports a daily decline of at least 3%. Check attributed news for a cause."})
	}
	if strings.Contains(strings.ToLower(quote.Source), "experimental") {
		flags = append(flags, domain.RiskFlag{Code: "unverified_provider", Severity: "info", Title: "Experimental data source", Explanation: "This free upstream source is not represented as licensed or guaranteed real-time."})
	}
	if fundamentals, err := m.FundamentalsBySymbol(ctx, symbol); err == nil && metricValue(fundamentals.Metrics, "eps") < 0 {
		flags = append(flags, domain.RiskFlag{Code: "negative_eps", Severity: "medium", Title: "Negative trailing earnings", Explanation: "The latest provider snapshot reports negative trailing earnings per share."})
	}
	return flags, nil
}

func (m *Mongo) StockIntelligenceBySymbol(ctx context.Context, symbol string) (domain.StockIntelligence, error) {
	security, err := m.SecurityBySymbol(ctx, symbol)
	if err != nil {
		return domain.StockIntelligence{}, err
	}
	result := domain.StockIntelligence{Security: security, Peers: []domain.PeerSnapshot{}, RiskFlags: []domain.RiskFlag{}}
	if quote, quoteErr := m.MarketQuoteBySymbol(ctx, symbol); quoteErr == nil {
		result.Quote = &quote
	}
	if fundamentals, fundamentalErr := m.FundamentalsBySymbol(ctx, symbol); fundamentalErr == nil {
		result.Fundamentals = &fundamentals
	}
	result.Peers, err = m.PeersBySymbol(ctx, symbol, 5)
	if err != nil {
		return result, err
	}
	result.RiskFlags, err = m.RiskFlagsBySymbol(ctx, symbol)
	return result, err
}

func metricValue(metrics []domain.FundamentalMetric, key string) float64 {
	for _, metric := range metrics {
		if metric.Key == key {
			return metric.Value
		}
	}
	return 0
}

func round2(value float64) float64 { return math.Round(value*100) / 100 }

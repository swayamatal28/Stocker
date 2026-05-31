package store

import (
	"context"
	"errors"
	"fmt"
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
	for key, value := range map[string]string{"nse_symbol": nseSymbol, "bse_code": bseCode, "isin": isin, "company_name": strings.TrimSpace(companyName), "source": strings.TrimSpace(source), "source_url": strings.TrimSpace(sourceURL)} {
		if value != "" {
			set[key] = value
		}
	}
	// Quote/search providers often omit classification and historically sent the
	// placeholder "Unclassified". Never let that erase a useful sector already
	// obtained from a richer directory response. A later meaningful value is
	// still allowed to repair an incomplete record.
	if usefulClassification(sector) {
		set["sector"] = strings.TrimSpace(sector)
	}
	if usefulClassification(industry) {
		set["industry"] = strings.TrimSpace(industry)
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

func usefulClassification(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "unclassified", "unknown", "n/a", "na", "not available", "other":
		return false
	default:
		return true
	}
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
		// Sector performance is a price-based view. Do not render empty groups
		// created only by an old signal or an incomplete directory record.
		if entry.item.CompanyCount == 0 {
			continue
		}
		entry.item.AverageChange = round2(entry.changeTotal / float64(entry.item.CompanyCount))
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
		flags = append(flags, domain.RiskFlag{Code: "missing_market_data", Severity: "high", Title: "Price information unavailable", Explanation: "A recent price is not available, so no price-based outlook can be shown."})
		return flags, nil
	}
	if time.Since(quote.AsOf) > 30*time.Minute {
		flags = append(flags, domain.RiskFlag{Code: "stale_market_data", Severity: "medium", Title: "Price may be delayed", Explanation: "The latest price is more than 30 minutes old. Check the displayed time before relying on it."})
	}
	if quote.YearLow > 0 && quote.LastPrice <= quote.YearLow*1.1 {
		flags = append(flags, domain.RiskFlag{Code: "near_year_low", Severity: "medium", Title: "Near 52-week low", Explanation: "The latest price is within 10% of its reported 52-week low."})
	}
	if quote.ChangePercent <= -3 {
		flags = append(flags, domain.RiskFlag{Code: "sharp_daily_decline", Severity: "medium", Title: "Sharp daily decline", Explanation: "The price has fallen by at least 3% today. Check recent company news for a possible reason."})
	}
	if strings.Contains(strings.ToLower(quote.Source), "experimental") {
		flags = append(flags, domain.RiskFlag{Code: "unverified_provider", Severity: "info", Title: "Price source limitation", Explanation: "This free price source may be delayed and does not guarantee real-time information."})
	}
	if fundamentals, err := m.FundamentalsBySymbol(ctx, symbol); err == nil && metricValue(fundamentals.Metrics, "eps") < 0 {
		flags = append(flags, domain.RiskFlag{Code: "negative_eps", Severity: "medium", Title: "Negative recent earnings", Explanation: "The latest available figures show negative earnings per share over the past 12 months."})
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
		result.Research = BuildStockResearch(result.Quote, &fundamentals)
	} else if result.Quote != nil {
		result.Research = BuildStockResearch(result.Quote, nil)
	}
	result.Peers, err = m.PeersBySymbol(ctx, symbol, 5)
	if err != nil {
		return result, err
	}
	result.RiskFlags, err = m.RiskFlagsBySymbol(ctx, symbol)
	return result, err
}

// BuildStockResearch applies published accounting ratios and conservative,
// explicit thresholds. It is deterministic: no model, prediction, or learned
// weight is involved. Thresholds are context aids, not buy/sell rules.
func BuildStockResearch(quote *domain.MarketQuote, fundamentals *domain.Fundamentals) *domain.StockResearch {
	checks := []domain.ResearchCheck{}
	points := 0
	add := func(check domain.ResearchCheck, score int) {
		checks = append(checks, check)
		points += score
	}
	metric := func(key string) (domain.FundamentalMetric, bool) {
		if fundamentals == nil {
			return domain.FundamentalMetric{}, false
		}
		for _, item := range fundamentals.Metrics {
			if item.Key == key {
				return item, true
			}
		}
		return domain.FundamentalMetric{}, false
	}
	status := func(value float64, positive, caution bool) (string, int) {
		if positive {
			return "positive", 1
		}
		if caution {
			return "caution", -1
		}
		return "neutral", 0
	}
	if item, ok := metric("pe_ratio"); ok && item.Value != 0 {
		s, score := status(item.Value, item.Value > 0 && item.Value <= 20, item.Value < 0 || item.Value > 40)
		add(domain.ResearchCheck{Key: item.Key, Title: "Price compared with earnings", Status: s, Value: item.Value, Display: formatResearch(item.Value, "x"), Explanation: "A lower positive P/E can mean the price is modest relative to recent earnings, but useful ranges differ by industry.", Formula: "P/E = share price ÷ earnings per share"}, score)
	}
	if item, ok := metric("roe"); ok {
		s, score := status(item.Value, item.Value >= 15, item.Value < 8)
		add(domain.ResearchCheck{Key: item.Key, Title: "Return on shareholder money", Status: s, Value: item.Value, Display: formatResearch(item.Value, "%"), Explanation: "ROE shows how much profit the company generated relative to shareholder equity.", Formula: "ROE = net income ÷ shareholder equity × 100"}, score)
	}
	if item, ok := metric("debt_to_equity"); ok {
		s, score := status(item.Value, item.Value <= .5, item.Value > 1.5)
		add(domain.ResearchCheck{Key: item.Key, Title: "Debt compared with equity", Status: s, Value: item.Value, Display: formatResearch(item.Value, "x"), Explanation: "Lower debt relative to equity generally leaves more room to handle difficult periods; norms vary for banks and financial firms.", Formula: "Debt to equity = total debt ÷ shareholder equity"}, score)
	}
	if item, ok := metric("current_ratio"); ok {
		s, score := status(item.Value, item.Value >= 1.2 && item.Value <= 2.5, item.Value < 1)
		add(domain.ResearchCheck{Key: item.Key, Title: "Short-term financial cushion", Status: s, Value: item.Value, Display: formatResearch(item.Value, "x"), Explanation: "The current ratio compares assets expected within a year with obligations due within a year.", Formula: "Current ratio = current assets ÷ current liabilities"}, score)
	}
	if item, ok := metric("operating_margin"); ok {
		s, score := status(item.Value, item.Value >= 15, item.Value < 8)
		add(domain.ResearchCheck{Key: item.Key, Title: "Operating profit margin", Status: s, Value: item.Value, Display: formatResearch(item.Value, "%"), Explanation: "Operating margin shows how much operating profit remains from each rupee of revenue before interest and tax.", Formula: "Operating margin = operating income ÷ revenue × 100"}, score)
	}
	if item, ok := metric("revenue_growth"); ok {
		s, score := status(item.Value, item.Value >= 10, item.Value < 0)
		add(domain.ResearchCheck{Key: item.Key, Title: "Annual revenue trend", Status: s, Value: item.Value, Display: formatResearch(item.Value, "%"), Explanation: "This compares the latest full-year revenue with the previous full year.", Formula: "Revenue growth = (latest revenue ÷ previous revenue − 1) × 100"}, score)
	}
	if item, ok := metric("free_cash_flow"); ok {
		s, score := status(item.Value, item.Value > 0, item.Value < 0)
		add(domain.ResearchCheck{Key: item.Key, Title: "Free cash flow", Status: s, Value: item.Value, Display: formatCrores(item.Value), Explanation: "Positive free cash flow means operations and capital spending left cash available during the reported year.", Formula: "Free cash flow = operating cash flow − capital spending"}, score)
	}
	if quote != nil && quote.YearHigh > quote.YearLow && quote.LastPrice > 0 {
		position := (quote.LastPrice - quote.YearLow) / (quote.YearHigh - quote.YearLow) * 100
		s, score := status(position, position >= 25 && position <= 75, position >= 90)
		add(domain.ResearchCheck{Key: "year_range_position", Title: "Position in 52-week range", Status: s, Value: round2(position), Display: formatResearch(position, "%"), Explanation: "This gives price context only. Being near a high or low does not by itself show whether a stock is cheap or expensive.", Formula: "Position = (price − 52-week low) ÷ (52-week high − 52-week low) × 100"}, score)
	}
	coverage := len(checks)
	score := 0
	if coverage > 0 {
		score = int(math.Round(50 + float64(points)*50/float64(coverage)))
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
	}
	label := "Not enough financial data"
	if coverage >= 3 {
		switch {
		case score >= 70:
			label = "Fundamentals look strong on available checks"
		case score >= 55:
			label = "Fundamentals look balanced on available checks"
		case score >= 40:
			label = "Available fundamentals are mixed"
		default:
			label = "Available fundamentals need caution"
		}
	}
	research := &domain.StockResearch{Score: score, Label: label, Coverage: coverage, Checks: checks, Disclaimer: "Rule-based research only. Thresholds vary by industry and this is not a buy or sell recommendation."}
	if fundamentals != nil {
		research.Source = fundamentals.Source
		research.AsOf = fundamentals.AsOf
	} else if quote != nil {
		research.Source = quote.Source
		research.AsOf = quote.AsOf
	}
	return research
}

func formatResearch(value float64, suffix string) string {
	return fmt.Sprintf("%.2f%s", value, suffix)
}

func formatCrores(value float64) string {
	return fmt.Sprintf("₹%.2f Cr", value/10_000_000)
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

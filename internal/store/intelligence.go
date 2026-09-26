package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stocker-app/stocker/internal/domain"
)

type analysisDocument struct {
	ID                  bson.ObjectID     `bson:"_id,omitempty"`
	ArticleID           bson.ObjectID     `bson:"article_id"`
	Provider            string            `bson:"provider"`
	Model               string            `bson:"model"`
	PromptVersion       string            `bson:"prompt_version"`
	PromptHash          string            `bson:"prompt_hash"`
	SchemaVersion       string            `bson:"schema_version"`
	DetectedLanguage    string            `bson:"detected_language"`
	AnalysisLanguage    string            `bson:"analysis_language"`
	TranslationProvider string            `bson:"translation_provider"`
	TranslationApplied  bool              `bson:"translation_applied"`
	Analysis            domain.AIAnalysis `bson:"analysis"`
	InputUnits          int               `bson:"input_units"`
	OutputUnits         int               `bson:"output_units"`
	CostCents           int               `bson:"cost_cents"`
	CreatedAt           time.Time         `bson:"created_at"`
}

func (m *Mongo) AnalysisContext(ctx context.Context, articleID string) (domain.AnalysisArticle, error) {
	id, err := bson.ObjectIDFromHex(articleID)
	if err != nil {
		return domain.AnalysisArticle{}, ErrNotFound
	}
	if err := m.resolveArticleEntities(ctx, id); err != nil {
		return domain.AnalysisArticle{}, err
	}
	article, err := m.NewsByID(ctx, articleID)
	if err != nil {
		return domain.AnalysisArticle{}, err
	}
	candidateSymbols := uniqueUpper(article.Symbols)
	symbols := []string{}
	facts := map[string]float64{}
	for _, symbol := range candidateSymbols {
		var security securityDocument
		if err := m.DB.Collection("securities").FindOne(ctx, bson.M{"nse_symbol": symbol}).Decode(&security); err != nil {
			continue
		}
		symbols = append(symbols, security.NSESymbol)
		var quote quoteDocument
		if err := m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": security.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&quote); err == nil {
			facts[symbol+".last_price"] = quote.LastPrice
			facts[symbol+".change_percent"] = quote.ChangePercent
			if quote.Volume > 0 {
				facts[symbol+".volume"] = float64(quote.Volume)
			}
		}
	}
	return domain.AnalysisArticle{Article: article, NumericFacts: facts, LinkedSymbols: symbols}, nil
}

func (m *Mongo) resolveArticleEntities(ctx context.Context, articleID bson.ObjectID) error {
	var article newsDocument
	if err := m.DB.Collection("normalized_articles").FindOne(ctx, bson.M{"_id": articleID}).Decode(&article); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}
	haystack := strings.ToUpper(article.Title + " " + article.BodyText)
	explicit := map[string]struct{}{}
	for _, symbol := range article.Symbols {
		explicit[strings.ToUpper(symbol)] = struct{}{}
	}
	cur, err := m.DB.Collection("securities").Find(ctx, bson.M{"active": true})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	symbols := append([]string(nil), article.Symbols...)
	sectors := append([]string(nil), article.Sectors...)
	for cur.Next(ctx) {
		var security securityDocument
		if err := cur.Decode(&security); err != nil {
			return err
		}
		_, selected := explicit[security.NSESymbol]
		method := "declared_symbol"
		if !selected && security.NSESymbol != "" {
			selected = tokenPresent(haystack, security.NSESymbol)
			method = "deterministic_symbol"
		}
		if !selected {
			name := strings.TrimSpace(strings.TrimSuffix(strings.ToUpper(security.CompanyName), " LIMITED"))
			selected = len(name) >= 5 && strings.Contains(haystack, name)
			method = "deterministic_company_name"
		}
		if !selected {
			continue
		}
		_, err := m.DB.Collection("article_security_links").UpdateOne(ctx,
			bson.M{"article_id": articleID, "security_id": security.ID},
			bson.M{"$setOnInsert": bson.M{"article_id": articleID, "security_id": security.ID, "link_method": method, "confidence": 100, "created_at": time.Now().UTC()}},
			options.UpdateOne().SetUpsert(true))
		if err != nil {
			return err
		}
		symbols = append(symbols, security.NSESymbol)
		sectors = append(sectors, security.Sector)
	}
	if err := cur.Err(); err != nil {
		return err
	}
	_, err = m.DB.Collection("normalized_articles").UpdateOne(ctx, bson.M{"_id": articleID}, bson.M{"$set": bson.M{"symbols": uniqueUpper(symbols), "sectors": uniqueStrings(sectors)}})
	return err
}

func tokenPresent(text, token string) bool {
	matched, _ := regexp.MatchString(`(^|[^A-Z0-9])`+regexp.QuoteMeta(token)+`([^A-Z0-9]|$)`, text)
	return matched
}

func (m *Mongo) ReserveAIBudget(ctx context.Context, provider string, costCents, dailyLimit int) (bool, error) {
	if costCents == 0 {
		return true, nil
	}
	date := time.Now().UTC().Format("2006-01-02")
	_, err := m.DB.Collection("ai_daily_budgets").UpdateOne(ctx, bson.M{"provider": provider, "date": date}, bson.M{"$setOnInsert": bson.M{"provider": provider, "date": date, "spent_cents": 0, "created_at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return false, err
	}
	result, err := m.DB.Collection("ai_daily_budgets").UpdateOne(ctx,
		bson.M{"provider": provider, "date": date, "spent_cents": bson.M{"$lte": dailyLimit - costCents}},
		bson.M{"$inc": bson.M{"spent_cents": costCents}, "$set": bson.M{"updated_at": time.Now().UTC()}})
	if err != nil {
		return false, err
	}
	return result.ModifiedCount == 1, nil
}

func (m *Mongo) SaveIntelligence(ctx context.Context, output domain.IntelligenceOutput) (domain.IntelligenceOutput, bool, error) {
	articleID, err := bson.ObjectIDFromHex(output.Analysis.ArticleID)
	if err != nil {
		return output, false, ErrNotFound
	}
	now := time.Now().UTC()
	_, err = m.DB.Collection("prompt_versions").UpdateOne(ctx,
		bson.M{"version": output.Analysis.PromptVersion, "prompt_hash": output.Analysis.PromptHash},
		bson.M{"$setOnInsert": bson.M{"version": output.Analysis.PromptVersion, "prompt_hash": output.Analysis.PromptHash, "schema_version": output.Analysis.SchemaVersion, "system_policy": output.Analysis.PromptText, "created_at": now}},
		options.UpdateOne().SetUpsert(true))
	if err != nil {
		return output, false, err
	}
	document := analysisDocument{
		ArticleID: articleID, Provider: output.Analysis.Provider, Model: output.Analysis.Model,
		PromptVersion: output.Analysis.PromptVersion, PromptHash: output.Analysis.PromptHash, SchemaVersion: output.Analysis.SchemaVersion,
		DetectedLanguage: output.Analysis.DetectedLanguage, AnalysisLanguage: output.Analysis.AnalysisLanguage,
		TranslationProvider: output.Analysis.TranslationProvider, TranslationApplied: output.Analysis.TranslationApplied,
		Analysis: output.Analysis.Analysis, InputUnits: output.Analysis.Usage.InputUnits,
		OutputUnits: output.Analysis.Usage.OutputUnits, CostCents: output.Analysis.Usage.CostCents, CreatedAt: now,
	}
	inserted, err := m.DB.Collection("ai_analyses").InsertOne(ctx, document)
	created := err == nil
	if mongo.IsDuplicateKeyError(err) {
		if findErr := m.DB.Collection("ai_analyses").FindOne(ctx, bson.M{"article_id": articleID, "prompt_version": document.PromptVersion, "provider": document.Provider, "model": document.Model}).Decode(&document); findErr != nil {
			return output, false, findErr
		}
	} else if err != nil {
		return output, false, err
	} else {
		document.ID = inserted.InsertedID.(bson.ObjectID)
	}
	output.Analysis.ID = document.ID.Hex()
	output.Analysis.CreatedAt = document.CreatedAt
	for _, evidence := range output.Analysis.Analysis.Evidence {
		hash := sha256.Sum256([]byte(evidence.URL + "\x00" + evidence.Excerpt))
		_, evidenceErr := m.DB.Collection("evidence_references").InsertOne(ctx, bson.M{
			"analysis_id": document.ID, "article_id": articleID, "evidence_hash": fmt.Sprintf("%x", hash[:]),
			"label": evidence.Label, "url": evidence.URL, "source": evidence.Source, "excerpt": evidence.Excerpt,
			"published_at": evidence.PublishedAt, "created_at": now,
		})
		if evidenceErr != nil && !mongo.IsDuplicateKeyError(evidenceErr) {
			return output, created, evidenceErr
		}
	}
	for index := range output.Signals {
		signal := &output.Signals[index]
		var security securityDocument
		if err := m.DB.Collection("securities").FindOne(ctx, bson.M{"nse_symbol": strings.ToUpper(signal.Symbol)}).Decode(&security); err != nil {
			return output, created, err
		}
		signalDoc := signalDocument{
			SecurityID: security.ID, AnalysisID: document.ID, ArticleID: articleID, Symbol: security.NSESymbol,
			Label: signal.Label, Score: float64(signal.Strength), Confidence: signal.Confidence, Horizon: signal.Horizon,
			Version: signal.Version, Reasons: signal.Reasons, Risks: signal.Risks, Invalidators: signal.Invalidators,
			InputSnapshot: bson.M{"analysis_id": document.ID, "article_id": articleID, "prompt_hash": document.PromptHash},
			DataFreshAt:   signal.FreshAt, GeneratedAt: now, Sources: signal.Sources,
		}
		result, signalErr := m.DB.Collection("signals").InsertOne(ctx, signalDoc)
		if mongo.IsDuplicateKeyError(signalErr) {
			var existing signalDocument
			if findErr := m.DB.Collection("signals").FindOne(ctx, bson.M{"analysis_id": document.ID, "security_id": security.ID, "version": signal.Version}).Decode(&existing); findErr != nil {
				return output, created, findErr
			}
			signal.ID = existing.ID.Hex()
			signal.GeneratedAt = existing.GeneratedAt
		} else if signalErr != nil {
			return output, created, signalErr
		} else {
			signal.ID = result.InsertedID.(bson.ObjectID).Hex()
			signal.GeneratedAt = now
		}
		signal.AnalysisID = document.ID.Hex()
	}
	_, err = m.DB.Collection("normalized_articles").UpdateOne(ctx, bson.M{"_id": articleID}, bson.M{"$set": bson.M{"analysis_state": "complete", "analysis_completed_at": now, "analysis_error": ""}})
	if err == nil {
		_, _ = m.DB.Collection("analysis_dead_letters").UpdateOne(ctx, bson.M{"event_key": "analysis:" + output.Analysis.ArticleID, "status": "dead"}, bson.M{"$set": bson.M{"status": "resolved", "resolved_at": now}})
	}
	return output, created, err
}

func (m *Mongo) MarkAnalysisFailure(ctx context.Context, articleID, message string, maxAttempts int) error {
	id, err := bson.ObjectIDFromHex(articleID)
	if err != nil {
		return ErrNotFound
	}
	var article newsDocument
	err = m.DB.Collection("normalized_articles").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"analysis_attempts": 1}, "$set": bson.M{"analysis_error": message, "analysis_last_attempt_at": time.Now().UTC()}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&article)
	if err != nil {
		return err
	}
	if article.AnalysisAttempts < maxAttempts {
		return nil
	}
	_, err = m.DB.Collection("normalized_articles").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"analysis_state": "dead"}})
	if err != nil {
		return err
	}
	_, err = m.DB.Collection("analysis_dead_letters").UpdateOne(ctx, bson.M{"event_key": "analysis:" + articleID}, bson.M{"$set": bson.M{"event_key": "analysis:" + articleID, "article_id": id, "error": message, "attempts": article.AnalysisAttempts, "status": "dead", "failed_at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
	return err
}

func (m *Mongo) AnalysisByArticle(ctx context.Context, articleID string) (domain.IntelligenceOutput, error) {
	id, err := bson.ObjectIDFromHex(articleID)
	if err != nil {
		return domain.IntelligenceOutput{}, ErrNotFound
	}
	var document analysisDocument
	err = m.DB.Collection("ai_analyses").FindOne(ctx, bson.M{"article_id": id}, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.IntelligenceOutput{}, ErrNotFound
	}
	if err != nil {
		return domain.IntelligenceOutput{}, err
	}
	output := domain.IntelligenceOutput{Analysis: toAnalysisRecord(document)}
	cur, err := m.DB.Collection("signals").Find(ctx, bson.M{"analysis_id": document.ID}, options.Find().SetSort(bson.D{{Key: "generated_at", Value: -1}}))
	if err != nil {
		return output, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var signalDoc signalDocument
		if err := cur.Decode(&signalDoc); err != nil {
			return output, err
		}
		output.Signals = append(output.Signals, toSignal(signalDoc))
	}
	return output, cur.Err()
}

func (m *Mongo) SignalsBySymbol(ctx context.Context, symbol string, limit int) ([]domain.Signal, error) {
	var security securityDocument
	if err := m.DB.Collection("securities").FindOne(ctx, bson.M{"nse_symbol": strings.ToUpper(strings.TrimSpace(symbol))}).Decode(&security); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	cur, err := m.DB.Collection("signals").Find(ctx, bson.M{"security_id": security.ID}, options.Find().SetSort(bson.D{{Key: "generated_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []domain.Signal{}
	for cur.Next(ctx) {
		var document signalDocument
		if err := cur.Decode(&document); err != nil {
			return nil, err
		}
		out = append(out, toSignal(document))
	}
	return out, cur.Err()
}

func toAnalysisRecord(document analysisDocument) domain.AnalysisRecord {
	return domain.AnalysisRecord{ID: document.ID.Hex(), ArticleID: document.ArticleID.Hex(), Provider: document.Provider, Model: document.Model, PromptVersion: document.PromptVersion, PromptHash: document.PromptHash, SchemaVersion: document.SchemaVersion, DetectedLanguage: document.DetectedLanguage, AnalysisLanguage: document.AnalysisLanguage, TranslationProvider: document.TranslationProvider, TranslationApplied: document.TranslationApplied, Analysis: document.Analysis, Usage: domain.AIUsage{InputUnits: document.InputUnits, OutputUnits: document.OutputUnits, CostCents: document.CostCents}, CreatedAt: document.CreatedAt}
}

func toSignal(document signalDocument) domain.Signal {
	signal := domain.Signal{ID: document.ID.Hex(), Symbol: document.Symbol, Label: document.Label, Horizon: document.Horizon, Version: document.Version, Strength: int(document.Score), Confidence: document.Confidence, Reasons: document.Reasons, Risks: document.Risks, Invalidators: document.Invalidators, FreshAt: document.DataFreshAt, GeneratedAt: document.GeneratedAt, Sources: document.Sources}
	if !document.AnalysisID.IsZero() {
		signal.AnalysisID = document.AnalysisID.Hex()
	}
	return signal
}

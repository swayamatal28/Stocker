package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stocker-app/stocker/internal/domain"
)

func (m *Mongo) EvaluationCases(ctx context.Context, asOf time.Time, limit int) ([]domain.BacktestCase, error) {
	if limit < 1 || limit > 10000 {
		limit = 5000
	}
	cursor, err := m.DB.Collection("signals").Find(ctx, bson.M{"generated_at": bson.M{"$lte": asOf}}, options.Find().SetSort(bson.D{{Key: "generated_at", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	cases := []domain.BacktestCase{}
	for cursor.Next(ctx) {
		var signal signalDocument
		if err := cursor.Decode(&signal); err != nil {
			return nil, err
		}
		var security securityDocument
		if err := m.DB.Collection("securities").FindOne(ctx, bson.M{"_id": signal.SecurityID}).Decode(&security); err != nil {
			continue
		}
		candidate := domain.BacktestCase{SignalID: signal.ID.Hex(), Symbol: security.NSESymbol, Sector: security.Sector, Horizon: signal.Horizon, Strength: int(signal.Score), Confidence: signal.Confidence, DecisionAt: signal.GeneratedAt, TargetAt: signal.GeneratedAt.Add(evaluationHorizon(signal.Horizon)), FeatureTimes: map[string]time.Time{"signal_data": signal.DataFreshAt}}
		if !signal.AnalysisID.IsZero() {
			var analysis analysisDocument
			if err := m.DB.Collection("ai_analyses").FindOne(ctx, bson.M{"_id": signal.AnalysisID}).Decode(&analysis); err == nil {
				candidate.Category = analysis.Analysis.EventCategory
				candidate.FeatureTimes["analysis"] = analysis.CreatedAt
				var article newsDocument
				if err := m.DB.Collection("normalized_articles").FindOne(ctx, bson.M{"_id": analysis.ArticleID}).Decode(&article); err == nil {
					candidate.FeatureTimes["article"] = article.RetrievedAt
				}
			}
		}
		var entry quoteDocument
		entryErr := m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": signal.SecurityID, "as_of": bson.M{"$lte": signal.GeneratedAt}, "retrieved_at": bson.M{"$lte": signal.GeneratedAt}}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&entry)
		if entryErr == nil {
			candidate.EntryPrice = entry.LastPrice
			candidate.FeatureTimes["entry_quote"] = entry.RetrievedAt
		} else if !errors.Is(entryErr, mongo.ErrNoDocuments) {
			return nil, entryErr
		}
		var exit quoteDocument
		exitErr := m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": signal.SecurityID, "as_of": bson.M{"$gte": candidate.TargetAt}, "retrieved_at": bson.M{"$lte": asOf}}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: 1}})).Decode(&exit)
		if exitErr == nil {
			candidate.ExitPrice = exit.LastPrice
			candidate.ExitAsOf = exit.AsOf
			candidate.ExitAvailableAt = exit.RetrievedAt
		} else if !errors.Is(exitErr, mongo.ErrNoDocuments) {
			return nil, exitErr
		}
		cases = append(cases, candidate)
	}
	return cases, cursor.Err()
}

func (m *Mongo) SaveEvaluationReport(ctx context.Context, report domain.EvaluationReport) error {
	for _, outcome := range report.Outcomes {
		signalID, err := bson.ObjectIDFromHex(outcome.SignalID)
		if err != nil {
			continue
		}
		document := bson.M{"signal_id": signalID, "symbol": outcome.Symbol, "sector": outcome.Sector, "category": outcome.Category, "horizon": outcome.Horizon, "confidence_band": outcome.ConfidenceBand, "strength": outcome.Strength, "confidence": outcome.Confidence, "decision_at": outcome.DecisionAt, "target_at": outcome.TargetAt, "exit_as_of": outcome.ExitAsOf, "entry_price": outcome.EntryPrice, "exit_price": outcome.ExitPrice, "return_percent": outcome.ReturnPercent, "prediction": outcome.Prediction, "actual": outcome.Actual, "correct": outcome.Correct, "evaluated_at": outcome.EvaluatedAt, "version": outcome.Version}
		_, err = m.DB.Collection("signal_outcomes").UpdateOne(ctx, bson.M{"signal_id": signalID, "version": outcome.Version, "exit_as_of": outcome.ExitAsOf}, bson.M{"$setOnInsert": document}, options.UpdateOne().SetUpsert(true))
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return err
		}
	}
	for _, violation := range report.LeakageViolations {
		signalID, err := bson.ObjectIDFromHex(violation.SignalID)
		if err != nil {
			continue
		}
		document := bson.M{"signal_id": signalID, "field": violation.Field, "available_at": violation.AvailableAt, "decision_at": violation.DecisionAt, "reason": violation.Reason, "version": report.Version, "detected_at": report.AsOf}
		_, err = m.DB.Collection("evaluation_leakage").UpdateOne(ctx, bson.M{"signal_id": signalID, "field": violation.Field, "version": report.Version, "available_at": violation.AvailableAt}, bson.M{"$setOnInsert": document}, options.UpdateOne().SetUpsert(true))
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return err
		}
	}
	_, err := m.DB.Collection("evaluation_runs").InsertOne(ctx, bson.M{"version": report.Version, "as_of": report.AsOf, "candidate_signals": report.CandidateSignals, "evaluated_signals": report.EvaluatedSignals, "pending_signals": report.PendingSignals, "leakage_violation_count": len(report.LeakageViolations), "slices": report.Slices, "created_at": time.Now().UTC()})
	return err
}

func evaluationHorizon(horizon string) time.Duration {
	switch horizon {
	case "intraday":
		return 8 * time.Hour
	case "short term":
		return 7 * 24 * time.Hour
	case "medium term":
		return 30 * 24 * time.Hour
	case "long term":
		return 90 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

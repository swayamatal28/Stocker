package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type RetentionPolicy struct {
	Transient    time.Duration
	AlertDays    int
	BriefingDays int
	AuditDays    int
}

type RetentionResult struct {
	RefreshSessions        int64 `json:"refreshSessions"`
	RawDocuments           int64 `json:"rawDocuments"`
	NewsArticles           int64 `json:"newsArticles"`
	Analyses               int64 `json:"analyses"`
	Signals                int64 `json:"signals"`
	MarketQuotes           int64 `json:"marketQuotes"`
	Fundamentals           int64 `json:"fundamentals"`
	SourceHealthMetrics    int64 `json:"sourceHealthMetrics"`
	DeadLetters            int64 `json:"deadLetters"`
	NotificationDeliveries int64 `json:"notificationDeliveries"`
	AlertEvents            int64 `json:"alertEvents"`
	AlertRules             int64 `json:"alertRules"`
	Briefings              int64 `json:"briefings"`
	EvaluationRuns         int64 `json:"evaluationRuns"`
}

// RunRetention keeps identity, user, portfolio, current source-policy and
// security records. Time-sensitive news and derived records are removed once
// they reach the configured transient window. Portfolio market data is exempt
// because it is required to display the user's holdings.
func (m *Mongo) RunRetention(ctx context.Context, now time.Time, policy RetentionPolicy) (RetentionResult, error) {
	var result RetentionResult
	if policy.Transient < time.Hour || policy.AlertDays < 1 || policy.BriefingDays < 1 || policy.AuditDays < 1 {
		return result, errors.New("retention periods must be positive")
	}
	now = now.UTC()
	cutoff := now.Add(-policy.Transient)

	deleteInto := func(collection string, filter bson.M, destination *int64) error {
		deleted, err := m.DB.Collection(collection).DeleteMany(ctx, filter)
		if err == nil && destination != nil {
			*destination += deleted.DeletedCount
		}
		return err
	}

	articleIDs, err := m.objectIDs(ctx, "normalized_articles", bson.M{"published_at": bson.M{"$lte": cutoff}})
	if err != nil {
		return result, err
	}
	analysisFilter := bson.M{"created_at": bson.M{"$lte": cutoff}}
	if len(articleIDs) > 0 {
		analysisFilter = bson.M{"$or": bson.A{analysisFilter, bson.M{"article_id": bson.M{"$in": articleIDs}}}}
	}
	analysisIDs, err := m.objectIDs(ctx, "ai_analyses", analysisFilter)
	if err != nil {
		return result, err
	}
	signalClauses := bson.A{bson.M{"generated_at": bson.M{"$lte": cutoff}}}
	if len(articleIDs) > 0 {
		signalClauses = append(signalClauses, bson.M{"article_id": bson.M{"$in": articleIDs}})
	}
	if len(analysisIDs) > 0 {
		signalClauses = append(signalClauses, bson.M{"analysis_id": bson.M{"$in": analysisIDs}})
	}
	signalIDs, err := m.objectIDs(ctx, "signals", bson.M{"$or": signalClauses})
	if err != nil {
		return result, err
	}
	alertIDs, err := m.objectIDs(ctx, "alert_events", bson.M{"triggered_at": bson.M{"$lte": cutoff}})
	if err != nil {
		return result, err
	}

	if len(alertIDs) > 0 {
		if err := deleteInto("notification_deliveries", bson.M{"alert_event_id": bson.M{"$in": alertIDs}}, &result.NotificationDeliveries); err != nil {
			return result, err
		}
		if err := deleteInto("alert_events", bson.M{"_id": bson.M{"$in": alertIDs}}, &result.AlertEvents); err != nil {
			return result, err
		}
	}
	if len(signalIDs) > 0 {
		for _, collection := range []string{"signal_outcomes", "evaluation_leakage"} {
			if err := deleteInto(collection, bson.M{"signal_id": bson.M{"$in": signalIDs}}, nil); err != nil {
				return result, err
			}
		}
		if err := deleteInto("signals", bson.M{"_id": bson.M{"$in": signalIDs}}, &result.Signals); err != nil {
			return result, err
		}
	}
	if len(analysisIDs) > 0 {
		if err := deleteInto("evidence_references", bson.M{"analysis_id": bson.M{"$in": analysisIDs}}, nil); err != nil {
			return result, err
		}
		if err := deleteInto("ai_analyses", bson.M{"_id": bson.M{"$in": analysisIDs}}, &result.Analyses); err != nil {
			return result, err
		}
	}
	if len(articleIDs) > 0 {
		if err := deleteInto("article_security_links", bson.M{"article_id": bson.M{"$in": articleIDs}}, nil); err != nil {
			return result, err
		}
		if err := deleteInto("normalized_articles", bson.M{"_id": bson.M{"$in": articleIDs}}, &result.NewsArticles); err != nil {
			return result, err
		}
	}

	if err := deleteInto("raw_documents", bson.M{"$or": bson.A{bson.M{"retrieved_at": bson.M{"$lte": cutoff}}, bson.M{"delete_after": bson.M{"$lte": now}}}}, &result.RawDocuments); err != nil {
		return result, err
	}
	if err := deleteInto("source_health_metrics", bson.M{"checked_at": bson.M{"$lte": cutoff}}, &result.SourceHealthMetrics); err != nil {
		return result, err
	}
	for _, dead := range []struct {
		collection string
		field      string
	}{{"ingestion_dead_letters", "failed_at"}, {"analysis_dead_letters", "failed_at"}} {
		if err := deleteInto(dead.collection, bson.M{dead.field: bson.M{"$lte": cutoff}}, &result.DeadLetters); err != nil {
			return result, err
		}
	}

	portfolioSecurityIDs, err := m.portfolioSecurityIDs(ctx)
	if err != nil {
		return result, err
	}
	marketFilter := bson.M{"retrieved_at": bson.M{"$lte": cutoff}}
	if len(portfolioSecurityIDs) > 0 {
		marketFilter["security_id"] = bson.M{"$nin": portfolioSecurityIDs}
	}
	if err := deleteInto("market_quotes", marketFilter, &result.MarketQuotes); err != nil {
		return result, err
	}
	if err := deleteInto("fundamentals", marketFilter, &result.Fundamentals); err != nil {
		return result, err
	}
	if err := deleteInto("briefings", bson.M{"generated_at": bson.M{"$lte": cutoff}}, &result.Briefings); err != nil {
		return result, err
	}
	if err := deleteInto("evaluation_runs", bson.M{"as_of": bson.M{"$lte": cutoff}}, &result.EvaluationRuns); err != nil {
		return result, err
	}
	if err := deleteInto("alert_rules", bson.M{"revoked_at": bson.M{"$lte": cutoff}}, &result.AlertRules); err != nil {
		return result, err
	}
	if err := deleteInto("refresh_sessions", bson.M{"$or": bson.A{bson.M{"expires_at": bson.M{"$lte": now}}, bson.M{"revoked_at": bson.M{"$lte": cutoff}}}}, &result.RefreshSessions); err != nil {
		return result, err
	}
	return result, nil
}

func (m *Mongo) portfolioSecurityIDs(ctx context.Context) (bson.A, error) {
	cursor, err := m.DB.Collection("watchlists").Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"items.security_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	seen := map[bson.ObjectID]bool{}
	result := bson.A{}
	for cursor.Next(ctx) {
		var document struct {
			Items []struct {
				SecurityID bson.ObjectID `bson:"security_id"`
			} `bson:"items"`
		}
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		for _, item := range document.Items {
			if !item.SecurityID.IsZero() && !seen[item.SecurityID] {
				seen[item.SecurityID] = true
				result = append(result, item.SecurityID)
			}
		}
	}
	return result, cursor.Err()
}

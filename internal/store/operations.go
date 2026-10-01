package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type OperationalSnapshot struct {
	IngestionDeadLetters int64
	AnalysisDeadLetters  int64
	LeakageViolations    int64
	DeferredAlerts       int64
	UnhealthySources     int64
	AIDailySpendCents    int64
}

func (snapshot OperationalSnapshot) Prometheus() string {
	return fmt.Sprintf("# HELP stocker_ingestion_dead_letters Unresolved ingestion dead letters.\n# TYPE stocker_ingestion_dead_letters gauge\nstocker_ingestion_dead_letters %d\n# HELP stocker_analysis_dead_letters Unresolved analysis dead letters.\n# TYPE stocker_analysis_dead_letters gauge\nstocker_analysis_dead_letters %d\n# HELP stocker_evaluation_leakage_violations Signals rejected for look-ahead leakage.\n# TYPE stocker_evaluation_leakage_violations gauge\nstocker_evaluation_leakage_violations %d\n# HELP stocker_alerts_deferred Alerts waiting for quiet hours to end.\n# TYPE stocker_alerts_deferred gauge\nstocker_alerts_deferred %d\n# HELP stocker_sources_unhealthy Sources whose latest health state is not healthy.\n# TYPE stocker_sources_unhealthy gauge\nstocker_sources_unhealthy %d\n# HELP stocker_ai_daily_spend_cents Current recorded AI daily spend in cents.\n# TYPE stocker_ai_daily_spend_cents gauge\nstocker_ai_daily_spend_cents %d\n", snapshot.IngestionDeadLetters, snapshot.AnalysisDeadLetters, snapshot.LeakageViolations, snapshot.DeferredAlerts, snapshot.UnhealthySources, snapshot.AIDailySpendCents)
}

func (m *Mongo) OperationalMetrics(ctx context.Context) (OperationalSnapshot, error) {
	var snapshot OperationalSnapshot
	queries := []struct {
		collection string
		filter     bson.M
		dest       *int64
	}{
		{"ingestion_dead_letters", bson.M{"resolved_at": bson.M{"$exists": false}}, &snapshot.IngestionDeadLetters},
		{"analysis_dead_letters", bson.M{"resolved_at": bson.M{"$exists": false}}, &snapshot.AnalysisDeadLetters},
		{"evaluation_leakage", bson.M{}, &snapshot.LeakageViolations},
		{"alert_events", bson.M{"delivery_status": "deferred"}, &snapshot.DeferredAlerts},
	}
	for _, query := range queries {
		count, err := m.DB.Collection(query.collection).CountDocuments(ctx, query.filter)
		if err != nil {
			return snapshot, err
		}
		*query.dest = count
	}
	healthCursor, err := m.DB.Collection("source_health_metrics").Aggregate(ctx, bson.A{
		bson.M{"$sort": bson.M{"checked_at": -1}},
		bson.M{"$group": bson.M{"_id": "$source_key", "status": bson.M{"$first": "$status"}}},
		bson.M{"$match": bson.M{"status": bson.M{"$ne": "healthy"}}},
		bson.M{"$count": "total"},
	})
	if err != nil {
		return snapshot, err
	}
	if healthCursor.Next(ctx) {
		var total struct {
			Total int64 `bson:"total"`
		}
		if err := healthCursor.Decode(&total); err != nil {
			return snapshot, err
		}
		snapshot.UnhealthySources = total.Total
	}
	healthCursor.Close(ctx)
	cursor, err := m.DB.Collection("ai_daily_budgets").Aggregate(ctx, bson.A{bson.M{"$match": bson.M{"date": time.Now().UTC().Format("2006-01-02")}}, bson.M{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$spent_cents"}}}})
	if err != nil {
		return snapshot, err
	}
	defer cursor.Close(ctx)
	if cursor.Next(ctx) {
		var total struct {
			Total int64 `bson:"total"`
		}
		if err := cursor.Decode(&total); err != nil {
			return snapshot, err
		}
		snapshot.AIDailySpendCents = total.Total
	}
	if err := cursor.Err(); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// PurgeSyntheticRuntimeData removes only records that are explicitly marked as
// synthetic, mock, fixture, or demo data. User accounts and genuine collected
// or provider-backed records are preserved.
func (m *Mongo) PurgeSyntheticRuntimeData(ctx context.Context) error {
	mockText := bson.Regex{Pattern: "(?i)(mock|fixture|synthetic|example\\.invalid)", Options: ""}
	mockSource := bson.M{"$or": bson.A{
		bson.M{"source_key": bson.Regex{Pattern: "^mock-", Options: "i"}},
		bson.M{"source_type": "mock"},
		bson.M{"synthetic": true},
		bson.M{"canonical_url": mockText},
	}}

	articleIDs, err := m.objectIDs(ctx, "normalized_articles", mockSource)
	if err != nil {
		return fmt.Errorf("find synthetic articles: %w", err)
	}
	analysisIDs := []bson.ObjectID{}
	if len(articleIDs) > 0 {
		analysisIDs, err = m.objectIDs(ctx, "ai_analyses", bson.M{"article_id": bson.M{"$in": articleIDs}})
		if err != nil {
			return fmt.Errorf("find synthetic analyses: %w", err)
		}
	}

	signalFilter := bson.M{"$or": bson.A{
		bson.M{"input_snapshot.demo": true},
		bson.M{"sources.url": mockText},
		bson.M{"sources.source": mockText},
	}}
	if len(articleIDs) > 0 {
		signalFilter["$or"] = append(signalFilter["$or"].(bson.A), bson.M{"article_id": bson.M{"$in": articleIDs}})
	}
	if len(analysisIDs) > 0 {
		signalFilter["$or"] = append(signalFilter["$or"].(bson.A), bson.M{"analysis_id": bson.M{"$in": analysisIDs}})
	}
	signalIDs, err := m.objectIDs(ctx, "signals", signalFilter)
	if err != nil {
		return fmt.Errorf("find synthetic signals: %w", err)
	}

	alertFilter := bson.M{"$or": bson.A{
		bson.M{"condition_snapshot.source": mockText},
		bson.M{"evidence.url": mockText},
		bson.M{"evidence.source": mockText},
	}}
	alertIDs, err := m.objectIDs(ctx, "alert_events", alertFilter)
	if err != nil {
		return fmt.Errorf("find synthetic alerts: %w", err)
	}

	deletions := []struct {
		collection string
		filter     any
	}{
		{"market_quotes", bson.M{"$or": bson.A{bson.M{"synthetic": true}, bson.M{"source": mockText}, bson.M{"source_url": mockText}}}},
		{"fundamentals", bson.M{"$or": bson.A{bson.M{"synthetic": true}, bson.M{"source": mockText}, bson.M{"source_url": mockText}}}},
		{"market_indices", bson.M{"source": mockText}},
		{"briefings", bson.M{"synthetic": true}},
		{"raw_documents", bson.M{"$or": bson.A{bson.M{"source_key": bson.Regex{Pattern: "^mock-", Options: "i"}}, bson.M{"source_url": mockText}}}},
		{"news_sources", bson.M{"$or": bson.A{bson.M{"key": bson.Regex{Pattern: "^mock-", Options: "i"}}, bson.M{"synthetic": true}, bson.M{"base_url": mockText}}}},
		{"source_policies", bson.M{"source_key": bson.Regex{Pattern: "^mock-", Options: "i"}}},
		{"source_states", bson.M{"source_key": bson.Regex{Pattern: "^mock-", Options: "i"}}},
		{"source_health_metrics", bson.M{"source_key": bson.Regex{Pattern: "^mock-", Options: "i"}}},
		{"ingestion_dead_letters", bson.M{"$or": bson.A{bson.M{"source_key": bson.Regex{Pattern: "^mock-", Options: "i"}}, bson.M{"payload": mockText}}}},
	}
	if len(signalIDs) > 0 {
		deletions = append(deletions,
			struct {
				collection string
				filter     any
			}{"signal_outcomes", bson.M{"signal_id": bson.M{"$in": signalIDs}}},
			struct {
				collection string
				filter     any
			}{"evaluation_leakage", bson.M{"signal_id": bson.M{"$in": signalIDs}}},
		)
	}
	if len(alertIDs) > 0 {
		deletions = append(deletions, struct {
			collection string
			filter     any
		}{"notification_deliveries", bson.M{"alert_event_id": bson.M{"$in": alertIDs}}})
	}
	if len(analysisIDs) > 0 {
		deletions = append(deletions, struct {
			collection string
			filter     any
		}{"evidence_references", bson.M{"analysis_id": bson.M{"$in": analysisIDs}}})
	}
	if len(articleIDs) > 0 {
		deletions = append(deletions, struct {
			collection string
			filter     any
		}{"article_security_links", bson.M{"article_id": bson.M{"$in": articleIDs}}})
	}
	for _, deletion := range deletions {
		if _, err := m.DB.Collection(deletion.collection).DeleteMany(ctx, deletion.filter); err != nil {
			return fmt.Errorf("purge %s: %w", deletion.collection, err)
		}
	}
	if len(alertIDs) > 0 {
		if _, err := m.DB.Collection("alert_events").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": alertIDs}}); err != nil {
			return fmt.Errorf("purge alert_events: %w", err)
		}
	}
	if len(signalIDs) > 0 {
		if _, err := m.DB.Collection("signals").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": signalIDs}}); err != nil {
			return fmt.Errorf("purge signals: %w", err)
		}
	}
	if len(analysisIDs) > 0 {
		if _, err := m.DB.Collection("ai_analyses").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": analysisIDs}}); err != nil {
			return fmt.Errorf("purge ai_analyses: %w", err)
		}
	}
	if len(articleIDs) > 0 {
		if _, err := m.DB.Collection("normalized_articles").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": articleIDs}}); err != nil {
			return fmt.Errorf("purge normalized_articles: %w", err)
		}
	}
	if _, err := m.DB.Collection("securities").UpdateMany(ctx, bson.M{"$or": bson.A{bson.M{"source": mockText}, bson.M{"source_url": mockText}}}, bson.M{"$unset": bson.M{"source": "", "source_url": ""}}); err != nil {
		return fmt.Errorf("remove fixture security attribution: %w", err)
	}
	if len(signalIDs) > 0 || len(analysisIDs) > 0 || len(articleIDs) > 0 {
		if _, err := m.DB.Collection("evaluation_runs").DeleteMany(ctx, bson.M{}); err != nil {
			return fmt.Errorf("clear derived evaluation runs: %w", err)
		}
	}
	return nil
}

func (m *Mongo) objectIDs(ctx context.Context, collection string, filter any) ([]bson.ObjectID, error) {
	cursor, err := m.DB.Collection(collection).Find(ctx, filter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	result := []bson.ObjectID{}
	for cursor.Next(ctx) {
		var document struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		result = append(result, document.ID)
	}
	return result, cursor.Err()
}

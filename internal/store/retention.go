package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type RetentionPolicy struct {
	AlertDays    int
	BriefingDays int
	AuditDays    int
}

type RetentionResult struct {
	RefreshSessions        int64 `json:"refreshSessions"`
	RawDocuments           int64 `json:"rawDocuments"`
	NotificationDeliveries int64 `json:"notificationDeliveries"`
	AlertEvents            int64 `json:"alertEvents"`
	AlertRules             int64 `json:"alertRules"`
	Briefings              int64 `json:"briefings"`
	EvaluationRuns         int64 `json:"evaluationRuns"`
}

func (m *Mongo) RunRetention(ctx context.Context, now time.Time, policy RetentionPolicy) (RetentionResult, error) {
	var result RetentionResult
	if policy.AlertDays < 1 || policy.BriefingDays < 1 || policy.AuditDays < 1 {
		return result, errors.New("retention periods must be positive")
	}
	now = now.UTC()
	alertCutoff := now.AddDate(0, 0, -policy.AlertDays)
	briefingCutoff := now.AddDate(0, 0, -policy.BriefingDays)
	auditCutoff := now.AddDate(0, 0, -policy.AuditDays)

	deleteInto := func(collection string, filter bson.M, destination *int64) error {
		deleted, err := m.DB.Collection(collection).DeleteMany(ctx, filter)
		if err == nil {
			*destination = deleted.DeletedCount
		}
		return err
	}
	if err := deleteInto("refresh_sessions", bson.M{"$or": bson.A{bson.M{"expires_at": bson.M{"$lte": now}}, bson.M{"revoked_at": bson.M{"$lte": alertCutoff}}}}, &result.RefreshSessions); err != nil {
		return result, err
	}
	if err := deleteInto("raw_documents", bson.M{"delete_after": bson.M{"$lte": now}}, &result.RawDocuments); err != nil {
		return result, err
	}
	alertFilter := bson.M{"triggered_at": bson.M{"$lte": alertCutoff}, "$or": bson.A{bson.M{"read_at": bson.M{"$exists": true}}, bson.M{"delivery_status": "delivered"}}}
	cursor, err := m.DB.Collection("alert_events").Find(ctx, alertFilter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return result, err
	}
	defer cursor.Close(ctx)
	ids := bson.A{}
	for cursor.Next(ctx) {
		var item struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&item); err != nil {
			return result, err
		}
		ids = append(ids, item.ID)
	}
	if err := cursor.Err(); err != nil {
		return result, err
	}
	if len(ids) > 0 {
		if err := deleteInto("notification_deliveries", bson.M{"alert_event_id": bson.M{"$in": ids}}, &result.NotificationDeliveries); err != nil {
			return result, err
		}
	}
	if err := deleteInto("alert_events", alertFilter, &result.AlertEvents); err != nil {
		return result, err
	}
	if err := deleteInto("alert_rules", bson.M{"revoked_at": bson.M{"$lte": alertCutoff}}, &result.AlertRules); err != nil {
		return result, err
	}
	if err := deleteInto("briefings", bson.M{"generated_at": bson.M{"$lte": briefingCutoff}}, &result.Briefings); err != nil {
		return result, err
	}
	if err := deleteInto("evaluation_runs", bson.M{"as_of": bson.M{"$lte": auditCutoff}}, &result.EvaluationRuns); err != nil {
		return result, err
	}
	return result, nil
}

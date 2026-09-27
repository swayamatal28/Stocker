package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stocker-app/stocker/internal/domain"
)

const alertRuleLimit = 50

type alertRuleDocument struct {
	ID                bson.ObjectID        `bson:"_id,omitempty"`
	UserID            bson.ObjectID        `bson:"user_id"`
	Name              string               `bson:"name"`
	Symbol            string               `bson:"symbol"`
	RuleType          string               `bson:"rule_type"`
	Threshold         *float64             `bson:"threshold,omitempty"`
	EventCategories   []string             `bson:"event_categories,omitempty"`
	MinimumConfidence int                  `bson:"minimum_confidence"`
	MinimumSeverity   string               `bson:"minimum_severity"`
	CooldownMinutes   int                  `bson:"cooldown_minutes"`
	Channels          domain.AlertChannels `bson:"channels"`
	QuietHours        domain.QuietHours    `bson:"quiet_hours"`
	Enabled           bool                 `bson:"enabled"`
	CreatedAt         time.Time            `bson:"created_at"`
	UpdatedAt         time.Time            `bson:"updated_at"`
	RevokedAt         *time.Time           `bson:"revoked_at,omitempty"`
}

type alertEventDocument struct {
	ID                    bson.ObjectID     `bson:"_id,omitempty"`
	UserID                bson.ObjectID     `bson:"user_id"`
	RuleID                bson.ObjectID     `bson:"rule_id"`
	RuleName              string            `bson:"rule_name"`
	RuleType              string            `bson:"rule_type"`
	Symbol                string            `bson:"symbol"`
	Severity              string            `bson:"severity"`
	Title                 string            `bson:"title"`
	Explanation           string            `bson:"explanation"`
	Confidence            int               `bson:"confidence"`
	Evidence              []domain.Evidence `bson:"evidence"`
	ConditionSnapshot     map[string]any    `bson:"condition_snapshot"`
	SourceAsOf            time.Time         `bson:"source_as_of"`
	TriggeredAt           time.Time         `bson:"triggered_at"`
	DeliverAfter          time.Time         `bson:"deliver_after"`
	DeliveredAt           *time.Time        `bson:"delivered_at,omitempty"`
	ReadAt                *time.Time        `bson:"read_at,omitempty"`
	DeliveryStatus        string            `bson:"delivery_status"`
	ExternalChannelStatus map[string]string `bson:"external_channel_status,omitempty"`
	DeduplicationKey      string            `bson:"deduplication_key"`
}

type briefingDocument struct {
	ID          bson.ObjectID         `bson:"_id,omitempty"`
	UserID      bson.ObjectID         `bson:"user_id"`
	Kind        string                `bson:"kind"`
	Title       string                `bson:"title"`
	Summary     string                `bson:"summary"`
	Items       []domain.BriefingItem `bson:"items"`
	GeneratedAt time.Time             `bson:"generated_at"`
	PeriodKey   string                `bson:"period_key"`
	Synthetic   bool                  `bson:"synthetic"`
}

func (m *Mongo) CreateAlertRule(ctx context.Context, userID string, rule domain.AlertRule) (domain.AlertRule, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return domain.AlertRule{}, ErrNotFound
	}
	watchlisted, _, err := m.WatchlistAlertState(ctx, userID, rule.Symbol)
	if err != nil {
		return domain.AlertRule{}, err
	}
	if !watchlisted {
		return domain.AlertRule{}, ErrNotWatchlisted
	}
	count, err := m.DB.Collection("alert_rules").CountDocuments(ctx, bson.M{"user_id": uid, "revoked_at": bson.M{"$exists": false}})
	if err != nil {
		return domain.AlertRule{}, err
	}
	if count >= alertRuleLimit {
		return domain.AlertRule{}, ErrAlertRuleLimit
	}
	now := time.Now().UTC()
	document := alertRuleDocument{UserID: uid, Name: rule.Name, Symbol: rule.Symbol, RuleType: rule.RuleType, Threshold: rule.Threshold, EventCategories: rule.EventCategories, MinimumConfidence: rule.MinimumConfidence, MinimumSeverity: rule.MinimumSeverity, CooldownMinutes: rule.CooldownMinutes, Channels: rule.Channels, QuietHours: rule.QuietHours, Enabled: rule.Enabled, CreatedAt: now, UpdatedAt: now}
	result, err := m.DB.Collection("alert_rules").InsertOne(ctx, document)
	if err != nil {
		return domain.AlertRule{}, err
	}
	document.ID = result.InsertedID.(bson.ObjectID)
	return toAlertRule(document), nil
}

func (m *Mongo) AlertRules(ctx context.Context, userID string) ([]domain.AlertRule, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrNotFound
	}
	cursor, err := m.DB.Collection("alert_rules").Find(ctx, bson.M{"user_id": uid, "revoked_at": bson.M{"$exists": false}}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rules := []domain.AlertRule{}
	for cursor.Next(ctx) {
		var document alertRuleDocument
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		rules = append(rules, toAlertRule(document))
	}
	return rules, cursor.Err()
}

func (m *Mongo) AlertRuleByID(ctx context.Context, userID, id string) (domain.AlertRule, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return domain.AlertRule{}, ErrNotFound
	}
	ruleID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.AlertRule{}, ErrNotFound
	}
	var document alertRuleDocument
	err = m.DB.Collection("alert_rules").FindOne(ctx, bson.M{"_id": ruleID, "user_id": uid, "revoked_at": bson.M{"$exists": false}}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.AlertRule{}, ErrNotFound
	}
	return toAlertRule(document), err
}

func (m *Mongo) UpdateAlertRule(ctx context.Context, userID string, rule domain.AlertRule) (domain.AlertRule, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return domain.AlertRule{}, ErrNotFound
	}
	ruleID, err := bson.ObjectIDFromHex(rule.ID)
	if err != nil {
		return domain.AlertRule{}, ErrNotFound
	}
	watchlisted, _, err := m.WatchlistAlertState(ctx, userID, rule.Symbol)
	if err != nil {
		return domain.AlertRule{}, err
	}
	if !watchlisted {
		return domain.AlertRule{}, ErrNotWatchlisted
	}
	now := time.Now().UTC()
	set := bson.M{"name": rule.Name, "symbol": rule.Symbol, "rule_type": rule.RuleType, "threshold": rule.Threshold, "event_categories": rule.EventCategories, "minimum_confidence": rule.MinimumConfidence, "minimum_severity": rule.MinimumSeverity, "cooldown_minutes": rule.CooldownMinutes, "channels": rule.Channels, "quiet_hours": rule.QuietHours, "enabled": rule.Enabled, "updated_at": now}
	var document alertRuleDocument
	err = m.DB.Collection("alert_rules").FindOneAndUpdate(ctx, bson.M{"_id": ruleID, "user_id": uid, "revoked_at": bson.M{"$exists": false}}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.AlertRule{}, ErrNotFound
	}
	return toAlertRule(document), err
}

func (m *Mongo) RevokeAlertRule(ctx context.Context, userID, id string) error {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return ErrNotFound
	}
	ruleID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrNotFound
	}
	now := time.Now().UTC()
	result, err := m.DB.Collection("alert_rules").UpdateOne(ctx, bson.M{"_id": ruleID, "user_id": uid, "revoked_at": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"enabled": false, "revoked_at": now, "updated_at": now}})
	if err == nil && result.ModifiedCount == 0 {
		return ErrNotFound
	}
	return err
}

func (m *Mongo) ActiveAlertRules(ctx context.Context) ([]domain.AlertRule, error) {
	cursor, err := m.DB.Collection("alert_rules").Find(ctx, bson.M{"enabled": true, "revoked_at": bson.M{"$exists": false}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rules := []domain.AlertRule{}
	for cursor.Next(ctx) {
		var document alertRuleDocument
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		rules = append(rules, toAlertRule(document))
	}
	return rules, cursor.Err()
}

func (m *Mongo) WatchlistAlertState(ctx context.Context, userID, symbol string) (watchlisted, paused bool, err error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, false, ErrNotFound
	}
	security, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return false, false, err
	}
	var watchlist watchlistDocument
	err = m.DB.Collection("watchlists").FindOne(ctx, bson.M{"user_id": uid, "items.security_id": security.ID}).Decode(&watchlist)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	for _, item := range watchlist.Items {
		if item.SecurityID == security.ID {
			return true, item.AlertsPaused, nil
		}
	}
	return false, false, nil
}

func (m *Mongo) AlertCandidate(ctx context.Context, rule domain.AlertRule) (domain.AlertCandidate, error) {
	security, err := m.securityDocumentBySymbol(ctx, rule.Symbol)
	if err != nil {
		return domain.AlertCandidate{}, err
	}
	if rule.RuleType != "event" {
		quote, err := m.MarketQuoteBySymbol(ctx, rule.Symbol)
		if err != nil {
			return domain.AlertCandidate{}, err
		}
		evidence := []domain.Evidence{{Label: "Timestamped market snapshot", URL: quote.SourceURL, Source: quote.Source, Excerpt: fmt.Sprintf("%s last price %.2f %s; daily change %.2f%%; market as of %s.", quote.Symbol, quote.LastPrice, quote.Currency, quote.ChangePercent, quote.AsOf.Format(time.RFC3339)), PublishedAt: quote.AsOf}}
		severity := "low"
		if math.Abs(quote.ChangePercent) >= 5 {
			severity = "high"
		} else if math.Abs(quote.ChangePercent) >= 2 {
			severity = "medium"
		}
		return domain.AlertCandidate{Symbol: security.NSESymbol, Kind: "threshold", Title: "Market threshold reached", Explanation: "The latest persisted provider snapshot satisfies the configured threshold.", Severity: severity, Confidence: 100, Price: quote.LastPrice, ChangePercent: quote.ChangePercent, SourceAsOf: quote.AsOf, Evidence: evidence, ConditionSnapshot: map[string]any{"lastPrice": quote.LastPrice, "changePercent": quote.ChangePercent, "source": quote.Source, "sourceAsOf": quote.AsOf}}, nil
	}
	var signal signalDocument
	err = m.DB.Collection("signals").FindOne(ctx, bson.M{"security_id": security.ID}, options.FindOne().SetSort(bson.D{{Key: "generated_at", Value: -1}})).Decode(&signal)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.AlertCandidate{}, ErrNotFound
	}
	if err != nil {
		return domain.AlertCandidate{}, err
	}
	var analysis analysisDocument
	if err := m.DB.Collection("ai_analyses").FindOne(ctx, bson.M{"_id": signal.AnalysisID}).Decode(&analysis); err != nil {
		return domain.AlertCandidate{}, err
	}
	var article newsDocument
	if err := m.DB.Collection("normalized_articles").FindOne(ctx, bson.M{"_id": signal.ArticleID}).Decode(&article); err != nil {
		return domain.AlertCandidate{}, err
	}
	severity := severityFromMateriality(analysis.Analysis.Materiality)
	return domain.AlertCandidate{Symbol: security.NSESymbol, Kind: "event", Category: analysis.Analysis.EventCategory, ClusterID: article.ClusterID, Title: article.Title, Explanation: analysis.Analysis.Summary, Severity: severity, Confidence: analysis.Analysis.Confidence, SourceAsOf: signal.DataFreshAt, Evidence: analysis.Analysis.Evidence, ConditionSnapshot: map[string]any{"eventCategory": analysis.Analysis.EventCategory, "materiality": analysis.Analysis.Materiality, "signalLabel": signal.Label, "signalStrength": int(signal.Score), "analysisId": analysis.ID.Hex(), "articleId": article.ID.Hex()}}, nil
}

func (m *Mongo) SaveAlertEvent(ctx context.Context, event domain.AlertEvent) (domain.AlertEvent, bool, error) {
	uid, err := bson.ObjectIDFromHex(event.UserID)
	if err != nil {
		return domain.AlertEvent{}, false, ErrNotFound
	}
	ruleID, err := bson.ObjectIDFromHex(event.RuleID)
	if err != nil {
		return domain.AlertEvent{}, false, ErrNotFound
	}
	document := alertEventDocument{UserID: uid, RuleID: ruleID, RuleName: event.RuleName, RuleType: event.RuleType, Symbol: event.Symbol, Severity: event.Severity, Title: event.Title, Explanation: event.Explanation, Confidence: event.Confidence, Evidence: event.Evidence, ConditionSnapshot: event.ConditionSnapshot, SourceAsOf: event.SourceAsOf, TriggeredAt: event.TriggeredAt, DeliverAfter: event.DeliverAfter, DeliveredAt: event.DeliveredAt, ReadAt: event.ReadAt, DeliveryStatus: event.DeliveryStatus, ExternalChannelStatus: event.ExternalChannelStatus, DeduplicationKey: event.DeduplicationKey}
	result, err := m.DB.Collection("alert_events").InsertOne(ctx, document)
	if mongo.IsDuplicateKeyError(err) {
		var existing alertEventDocument
		if findErr := m.DB.Collection("alert_events").FindOne(ctx, bson.M{"deduplication_key": event.DeduplicationKey}).Decode(&existing); findErr != nil {
			return domain.AlertEvent{}, false, findErr
		}
		return toAlertEvent(existing), false, nil
	}
	if err != nil {
		return domain.AlertEvent{}, false, err
	}
	document.ID = result.InsertedID.(bson.ObjectID)
	return toAlertEvent(document), true, nil
}

func (m *Mongo) AlertRuleInCooldown(ctx context.Context, userID, ruleID string, since time.Time) (bool, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, ErrNotFound
	}
	rid, err := bson.ObjectIDFromHex(ruleID)
	if err != nil {
		return false, ErrNotFound
	}
	count, err := m.DB.Collection("alert_events").CountDocuments(ctx, bson.M{"user_id": uid, "rule_id": rid, "triggered_at": bson.M{"$gt": since}}, options.Count().SetLimit(1))
	return count > 0, err
}

func (m *Mongo) DeliverAlert(ctx context.Context, eventID string, deliveredAt time.Time) (domain.AlertEvent, bool, error) {
	id, err := bson.ObjectIDFromHex(eventID)
	if err != nil {
		return domain.AlertEvent{}, false, ErrNotFound
	}
	result, err := m.DB.Collection("notification_deliveries").InsertOne(ctx, bson.M{"alert_event_id": id, "channel": "in_app", "status": "delivered", "delivered_at": deliveredAt})
	if mongo.IsDuplicateKeyError(err) {
		var existing alertEventDocument
		findErr := m.DB.Collection("alert_events").FindOne(ctx, bson.M{"_id": id}).Decode(&existing)
		return toAlertEvent(existing), false, findErr
	}
	if err != nil || result.InsertedID == nil {
		return domain.AlertEvent{}, false, err
	}
	var document alertEventDocument
	err = m.DB.Collection("alert_events").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"delivery_status": "delivered", "delivered_at": deliveredAt}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&document)
	return toAlertEvent(document), err == nil, err
}

func (m *Mongo) CancelAlertDelivery(ctx context.Context, eventID, reason string) error {
	id, err := bson.ObjectIDFromHex(eventID)
	if err != nil {
		return ErrNotFound
	}
	result, err := m.DB.Collection("alert_events").UpdateOne(ctx, bson.M{"_id": id, "delivered_at": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"delivery_status": "cancelled", "cancellation_reason": reason}})
	if err == nil && result.MatchedCount == 0 {
		return ErrNotFound
	}
	return err
}

func (m *Mongo) PendingAlerts(ctx context.Context, now time.Time, limit int) ([]domain.AlertEvent, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	cursor, err := m.DB.Collection("alert_events").Find(ctx, bson.M{"delivery_status": "deferred_quiet_hours", "deliver_after": bson.M{"$lte": now}}, options.Find().SetSort(bson.D{{Key: "deliver_after", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	events := []domain.AlertEvent{}
	for cursor.Next(ctx) {
		var document alertEventDocument
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		events = append(events, toAlertEvent(document))
	}
	return events, cursor.Err()
}

func (m *Mongo) Alerts(ctx context.Context, userID string, limit int) ([]domain.AlertEvent, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrNotFound
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	cursor, err := m.DB.Collection("alert_events").Find(ctx, bson.M{"user_id": uid}, options.Find().SetSort(bson.D{{Key: "triggered_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	events := []domain.AlertEvent{}
	for cursor.Next(ctx) {
		var document alertEventDocument
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		events = append(events, toAlertEvent(document))
	}
	return events, cursor.Err()
}

func (m *Mongo) MarkAlertRead(ctx context.Context, userID, id string) (domain.AlertEvent, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return domain.AlertEvent{}, ErrNotFound
	}
	idValue, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.AlertEvent{}, ErrNotFound
	}
	now := time.Now().UTC()
	var document alertEventDocument
	err = m.DB.Collection("alert_events").FindOneAndUpdate(ctx, bson.M{"_id": idValue, "user_id": uid}, bson.M{"$set": bson.M{"read_at": now}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.AlertEvent{}, ErrNotFound
	}
	return toAlertEvent(document), err
}

func (m *Mongo) GenerateBriefing(ctx context.Context, userID, kind string, now time.Time) (domain.Briefing, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return domain.Briefing{}, ErrNotFound
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	titles := map[string]string{"morning": "Morning watchlist briefing", "closing": "Closing watchlist briefing", "daily": "Daily watchlist briefing"}
	title, ok := titles[kind]
	if !ok {
		return domain.Briefing{}, errors.New("briefing kind must be morning, closing, or daily")
	}
	watchlist, err := m.Watchlist(ctx, userID)
	if err != nil {
		return domain.Briefing{}, err
	}
	items := []domain.BriefingItem{}
	synthetic := false
	for _, watched := range watchlist {
		symbol := watched.NSESymbol
		signals, signalErr := m.SignalsBySymbol(ctx, symbol, 1)
		if signalErr == nil && len(signals) > 0 && len(signals[0].Sources) > 0 {
			signal := signals[0]
			items = append(items, domain.BriefingItem{Symbol: symbol, Headline: signal.Label, Explanation: strings.Join(signal.Reasons, " "), Severity: severityFromStrength(signal.Strength), Confidence: signal.Confidence, AsOf: signal.FreshAt, Evidence: signal.Sources})
			continue
		}
		quote, quoteErr := m.MarketQuoteBySymbol(ctx, symbol)
		if quoteErr != nil {
			continue
		}
		synthetic = synthetic || quote.Synthetic
		items = append(items, domain.BriefingItem{Symbol: symbol, Headline: fmt.Sprintf("%s %.2f%%", symbol, quote.ChangePercent), Explanation: fmt.Sprintf("Latest persisted price %.2f %s, market as of %s.", quote.LastPrice, quote.Currency, quote.AsOf.Format(time.RFC3339)), Severity: severityFromMove(quote.ChangePercent), Confidence: 100, AsOf: quote.AsOf, Evidence: []domain.Evidence{{Label: "Timestamped market snapshot", URL: quote.SourceURL, Source: quote.Source, Excerpt: fmt.Sprintf("%s last price %.2f %s and daily change %.2f%%.", symbol, quote.LastPrice, quote.Currency, quote.ChangePercent), PublishedAt: quote.AsOf}}})
	}
	summary := fmt.Sprintf("%d sourced update(s) across %d monitored companies.", len(items), len(watchlist))
	periodKey := now.UTC().Format("2006-01-02")
	document := briefingDocument{UserID: uid, Kind: kind, Title: title, Summary: summary, Items: items, GeneratedAt: now.UTC(), PeriodKey: periodKey, Synthetic: synthetic}
	update := bson.M{"$set": document, "$setOnInsert": bson.M{"created_at": now.UTC()}}
	_, err = m.DB.Collection("briefings").UpdateOne(ctx, bson.M{"user_id": uid, "kind": kind, "period_key": periodKey}, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return domain.Briefing{}, err
	}
	if err := m.DB.Collection("briefings").FindOne(ctx, bson.M{"user_id": uid, "kind": kind, "period_key": periodKey}).Decode(&document); err != nil {
		return domain.Briefing{}, err
	}
	return toBriefing(document), nil
}

func toAlertRule(document alertRuleDocument) domain.AlertRule {
	return domain.AlertRule{ID: document.ID.Hex(), UserID: document.UserID.Hex(), Name: document.Name, Symbol: document.Symbol, RuleType: document.RuleType, Threshold: document.Threshold, EventCategories: document.EventCategories, MinimumConfidence: document.MinimumConfidence, MinimumSeverity: document.MinimumSeverity, CooldownMinutes: document.CooldownMinutes, Channels: document.Channels, QuietHours: document.QuietHours, Enabled: document.Enabled, CreatedAt: document.CreatedAt, UpdatedAt: document.UpdatedAt}
}

func toAlertEvent(document alertEventDocument) domain.AlertEvent {
	return domain.AlertEvent{ID: document.ID.Hex(), UserID: document.UserID.Hex(), DeduplicationKey: document.DeduplicationKey, RuleID: document.RuleID.Hex(), RuleName: document.RuleName, RuleType: document.RuleType, Symbol: document.Symbol, Severity: document.Severity, Title: document.Title, Explanation: document.Explanation, Confidence: document.Confidence, Evidence: document.Evidence, ConditionSnapshot: document.ConditionSnapshot, SourceAsOf: document.SourceAsOf, TriggeredAt: document.TriggeredAt, DeliverAfter: document.DeliverAfter, DeliveredAt: document.DeliveredAt, ReadAt: document.ReadAt, DeliveryStatus: document.DeliveryStatus, ExternalChannelStatus: document.ExternalChannelStatus}
}

func toBriefing(document briefingDocument) domain.Briefing {
	return domain.Briefing{ID: document.ID.Hex(), Kind: document.Kind, Title: document.Title, Summary: document.Summary, Items: document.Items, GeneratedAt: document.GeneratedAt, PeriodKey: document.PeriodKey, Synthetic: document.Synthetic}
}

func severityFromMateriality(value int) string {
	if value >= 75 {
		return "high"
	}
	if value >= 40 {
		return "medium"
	}
	return "low"
}

func severityFromStrength(value int) string {
	if value < 0 {
		value = -value
	}
	if value >= 70 {
		return "high"
	}
	if value >= 40 {
		return "medium"
	}
	return "low"
}

func severityFromMove(value float64) string {
	if math.Abs(value) >= 5 {
		return "high"
	}
	if math.Abs(value) >= 2 {
		return "medium"
	}
	return "low"
}

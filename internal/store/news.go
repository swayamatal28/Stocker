package store

import (
	"context"
	"errors"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/ingest"
)

type newsDocument struct {
	ID               bson.ObjectID `bson:"_id,omitempty"`
	RawDocumentID    bson.ObjectID `bson:"raw_document_id,omitempty"`
	SourceKey        string        `bson:"source_key"`
	SourceName       string        `bson:"source_name"`
	SourceType       string        `bson:"source_type"`
	ExternalID       string        `bson:"external_id"`
	CanonicalURL     string        `bson:"canonical_url"`
	Title            string        `bson:"title"`
	BodyText         string        `bson:"body_text"`
	Author           string        `bson:"author"`
	Language         string        `bson:"language"`
	Attribution      string        `bson:"attribution"`
	Licence          string        `bson:"licence"`
	ContentHash      string        `bson:"content_hash"`
	ClusterID        string        `bson:"cluster_id"`
	ParserVersion    string        `bson:"parser_version"`
	PublishedAt      time.Time     `bson:"published_at"`
	RetrievedAt      time.Time     `bson:"retrieved_at"`
	Symbols          []string      `bson:"symbols"`
	Sectors          []string      `bson:"sectors"`
	Official         bool          `bson:"official"`
	Synthetic        bool          `bson:"synthetic"`
	DuplicateCount   int           `bson:"duplicate_count,omitempty"`
	OutboxState      string        `bson:"outbox_state"`
	DeliveryAttempts int           `bson:"delivery_attempts"`
	AnalysisState    string        `bson:"analysis_state"`
	AnalysisAttempts int           `bson:"analysis_attempts"`
}

func toNewsArticle(d newsDocument) domain.NewsArticle {
	return domain.NewsArticle{
		ID: d.ID.Hex(), ClusterID: d.ClusterID, Title: d.Title, Body: d.BodyText,
		SourceID: d.SourceKey, SourceName: d.SourceName, SourceType: d.SourceType,
		URL: d.CanonicalURL, Author: d.Author, Language: d.Language, Attribution: d.Attribution,
		Licence: d.Licence, ParserVersion: d.ParserVersion, PublishedAt: d.PublishedAt,
		RetrievedAt: d.RetrievedAt, Symbols: nonNil(d.Symbols), Sectors: nonNil(d.Sectors),
		Official: d.Official, Synthetic: d.Synthetic, DuplicateCount: max(d.DuplicateCount, 1),
	}
}

func (m *Mongo) ensureSource(ctx context.Context, key, parserVersion string, item domain.SourceItem, policy ingest.SourcePolicy) error {
	sourceType := "rss_atom"
	if item.Synthetic || strings.HasPrefix(key, "mock-") {
		sourceType = "mock"
	}
	baseURL := ""
	if parsed, err := url.Parse(item.CanonicalURL); err == nil {
		baseURL = parsed.Scheme + "://" + parsed.Host
	}
	now := time.Now().UTC()
	_, err := m.DB.Collection("news_sources").UpdateOne(ctx, bson.M{"key": key}, bson.M{
		"$set": bson.M{
			"name": defaultValue(item.Attribution, key), "source_type": sourceType, "base_url": baseURL,
			"official": item.Official, "synthetic": item.Synthetic, "updated_at": now,
		},
		"$setOnInsert": bson.M{"key": key, "created_at": now},
	}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return err
	}
	policyDocument := bson.M{
		"source_key": key, "licence": policy.Licence, "attribution": policy.Attribution,
		"terms_url": policy.TermsURL, "automated_access_allowed": policy.AutomatedAccessAllowed,
		"poll_interval_seconds": int(policy.PollInterval.Seconds()), "requests_per_minute": policy.RequestsPerMinute,
		"timeout_seconds": int(policy.Timeout.Seconds()), "raw_retention_days": int(policy.RawRetention.Hours() / 24),
		"parser_version": parserVersion, "policy_expires_at": policy.PolicyExpiresAt, "updated_at": now,
	}
	_, err = m.DB.Collection("source_policies").UpdateOne(ctx, bson.M{"source_key": key}, bson.M{"$set": policyDocument}, options.UpdateOne().SetUpsert(true))
	return err
}

func (m *Mongo) SaveRaw(ctx context.Context, d domain.RawDocument) error {
	_, err := m.DB.Collection("raw_documents").UpdateOne(ctx,
		bson.M{"source_key": d.SourceID, "content_hash": d.Hash},
		bson.M{"$setOnInsert": bson.M{
			"source_key": d.SourceID, "source_url": d.URL, "canonical_url": d.URL,
			"http_status": d.StatusCode, "content_type": d.ContentType, "content_hash": d.Hash,
			"parser_version": d.ParserVersion, "body": d.Body, "retrieved_at": d.RetrievedAt, "delete_after": d.DeleteAfter,
		}}, options.UpdateOne().SetUpsert(true))
	return err
}

func (m *Mongo) FindNearDuplicate(ctx context.Context, item domain.SourceItem, window time.Duration, threshold float64) (string, bool, error) {
	start := item.PublishedAt.Add(-window)
	end := item.PublishedAt.Add(window)
	filter := bson.M{"published_at": bson.M{"$gte": start, "$lte": end}, "language": item.Language}
	cur, err := m.DB.Collection("normalized_articles").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "published_at", Value: -1}}).SetLimit(200).
			SetProjection(bson.M{"title": 1, "body_text": 1, "cluster_id": 1, "content_hash": 1}))
	if err != nil {
		return "", false, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var candidate newsDocument
		if err := cur.Decode(&candidate); err != nil {
			return "", false, err
		}
		score := math.Max(jaccard(item.Title, candidate.Title), jaccard(item.Title+" "+item.Body, candidate.Title+" "+candidate.BodyText))
		if score >= threshold {
			clusterID := candidate.ClusterID
			if clusterID == "" {
				clusterID = candidate.ContentHash
			}
			return clusterID, true, nil
		}
	}
	return "", false, cur.Err()
}

func (m *Mongo) SaveNormalized(ctx context.Context, sourceKey string, item domain.SourceItem, hash, clusterID, parserVersion string, policy ingest.SourcePolicy) (string, bool, error) {
	if err := m.ensureSource(ctx, sourceKey, parserVersion, item, policy); err != nil {
		return "", false, err
	}
	var raw struct {
		ID bson.ObjectID `bson:"_id"`
	}
	_ = m.DB.Collection("raw_documents").FindOne(ctx, bson.M{"source_key": sourceKey, "content_hash": hash}).Decode(&raw)
	sourceType := "rss_atom"
	if item.Synthetic {
		sourceType = "mock"
	}
	document := newsDocument{
		RawDocumentID: raw.ID, SourceKey: sourceKey, SourceName: defaultValue(item.Attribution, sourceKey), SourceType: sourceType,
		ExternalID: item.ExternalID, CanonicalURL: item.CanonicalURL, Title: item.Title, BodyText: item.Body,
		Author: item.Author, Language: defaultValue(item.Language, "en"), Attribution: item.Attribution, Licence: item.Licence,
		ContentHash: hash, ClusterID: clusterID, ParserVersion: parserVersion, PublishedAt: item.PublishedAt,
		RetrievedAt: item.RetrievedAt, Symbols: uniqueUpper(item.Symbols), Sectors: uniqueStrings(item.Sectors),
		Official: item.Official, Synthetic: item.Synthetic, OutboxState: "pending", DeliveryAttempts: 0, AnalysisState: "pending", AnalysisAttempts: 0,
	}
	res, err := m.DB.Collection("normalized_articles").UpdateOne(ctx, bson.M{"content_hash": hash}, bson.M{
		"$setOnInsert": document,
	}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return "", false, err
	}
	var saved newsDocument
	if err := m.DB.Collection("normalized_articles").FindOne(ctx, bson.M{"content_hash": hash}).Decode(&saved); err != nil {
		return "", false, err
	}
	if err := m.linkArticleSecurities(ctx, saved.ID, item); err != nil {
		return "", false, err
	}
	if res.UpsertedCount > 0 {
		count, _ := m.DB.Collection("normalized_articles").CountDocuments(ctx, bson.M{"cluster_id": clusterID})
		_, _ = m.DB.Collection("normalized_articles").UpdateMany(ctx, bson.M{"cluster_id": clusterID}, bson.M{"$set": bson.M{"duplicate_count": count}})
	}
	return saved.ID.Hex(), res.UpsertedCount > 0, nil
}

func (m *Mongo) linkArticleSecurities(ctx context.Context, articleID bson.ObjectID, item domain.SourceItem) error {
	cur, err := m.DB.Collection("securities").Find(ctx, bson.M{"active": true})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	explicit := map[string]struct{}{}
	for _, symbol := range item.Symbols {
		explicit[strings.ToUpper(strings.TrimSpace(symbol))] = struct{}{}
	}
	haystack := strings.ToUpper(item.Title + " " + item.Body)
	symbols := uniqueUpper(item.Symbols)
	sectors := uniqueStrings(item.Sectors)
	for cur.Next(ctx) {
		var security securityDocument
		if err := cur.Decode(&security); err != nil {
			return err
		}
		_, selected := explicit[security.NSESymbol]
		method := "declared_symbol"
		if !selected {
			selected, method = securityMention(haystack, security)
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

func securityMention(haystack string, security securityDocument) (bool, string) {
	if security.NSESymbol != "" {
		matched, _ := regexp.MatchString(`(^|[^A-Z0-9])`+regexp.QuoteMeta(security.NSESymbol)+`([^A-Z0-9]|$)`, haystack)
		if matched {
			return true, "deterministic_symbol"
		}
	}
	aliases := append([]string(nil), security.Aliases...)
	name := strings.TrimSpace(strings.TrimSuffix(strings.ToUpper(security.CompanyName), " LIMITED"))
	if name != "" {
		aliases = append(aliases, name)
	}
	for _, alias := range aliases {
		alias = strings.ToUpper(strings.TrimSpace(alias))
		if len(alias) >= 5 && strings.Contains(haystack, alias) {
			return true, "deterministic_company_alias"
		}
	}
	return false, ""
}

func (m *Mongo) ListNews(ctx context.Context, filter domain.NewsFilter) (domain.NewsPage, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 50 {
		filter.PageSize = 20
	}
	match := bson.M{}
	if filter.Query != "" {
		pattern := regexp.QuoteMeta(strings.TrimSpace(filter.Query))
		match["$or"] = bson.A{bson.M{"title": bson.M{"$regex": pattern, "$options": "i"}}, bson.M{"body_text": bson.M{"$regex": pattern, "$options": "i"}}}
	}
	if filter.Stock != "" {
		match["symbols"] = strings.ToUpper(strings.TrimSpace(filter.Stock))
	}
	if filter.Sector != "" {
		match["sectors"] = bson.M{"$regex": "^" + regexp.QuoteMeta(strings.TrimSpace(filter.Sector)) + "$", "$options": "i"}
	}
	if filter.Source != "" {
		match["source_key"] = strings.TrimSpace(filter.Source)
	}
	if filter.Language != "" {
		match["language"] = strings.ToLower(strings.TrimSpace(filter.Language))
	}
	if filter.OfficialOnly {
		match["official"] = true
	}
	if filter.From != nil || filter.To != nil {
		rangeFilter := bson.M{}
		if filter.From != nil {
			rangeFilter["$gte"] = *filter.From
		}
		if filter.To != nil {
			rangeFilter["$lte"] = *filter.To
		}
		match["published_at"] = rangeFilter
	}
	skip := int64((filter.Page - 1) * filter.PageSize)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$sort", Value: bson.D{{Key: "published_at", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$cluster_id", "$content_hash"}}}}, {Key: "document", Value: bson.D{{Key: "$first", Value: "$$ROOT"}}}, {Key: "duplicate_count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
		{{Key: "$set", Value: bson.D{{Key: "document.duplicate_count", Value: "$duplicate_count"}}}},
		{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$document"}}}},
		{{Key: "$sort", Value: bson.D{{Key: "published_at", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$facet", Value: bson.D{
			{Key: "data", Value: bson.A{bson.D{{Key: "$skip", Value: skip}}, bson.D{{Key: "$limit", Value: int64(filter.PageSize)}}}},
			{Key: "meta", Value: bson.A{bson.D{{Key: "$count", Value: "total"}}}},
		}}},
	}
	cur, err := m.DB.Collection("normalized_articles").Aggregate(ctx, pipeline)
	if err != nil {
		return domain.NewsPage{}, err
	}
	defer cur.Close(ctx)
	var result []struct {
		Data []newsDocument `bson:"data"`
		Meta []struct {
			Total int64 `bson:"total"`
		} `bson:"meta"`
	}
	if err := cur.All(ctx, &result); err != nil {
		return domain.NewsPage{}, err
	}
	page := domain.NewsPage{Items: []domain.NewsArticle{}, Page: filter.Page, PageSize: filter.PageSize}
	if len(result) == 0 {
		return page, nil
	}
	for _, document := range result[0].Data {
		page.Items = append(page.Items, toNewsArticle(document))
	}
	if len(result[0].Meta) > 0 {
		page.Total = result[0].Meta[0].Total
	}
	page.TotalPages = int(math.Ceil(float64(page.Total) / float64(page.PageSize)))
	return page, nil
}

func (m *Mongo) NewsByID(ctx context.Context, id string) (domain.NewsArticle, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.NewsArticle{}, ErrNotFound
	}
	var document newsDocument
	err = m.DB.Collection("normalized_articles").FindOne(ctx, bson.M{"_id": objectID}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.NewsArticle{}, ErrNotFound
	}
	if err != nil {
		return domain.NewsArticle{}, err
	}
	clusterID := document.ClusterID
	if clusterID == "" {
		clusterID = document.ContentHash
		document.ClusterID = clusterID
	}
	count, err := m.DB.Collection("normalized_articles").CountDocuments(ctx, bson.M{"$or": bson.A{bson.M{"cluster_id": clusterID}, bson.M{"content_hash": clusterID}}})
	if err != nil {
		return domain.NewsArticle{}, err
	}
	document.DuplicateCount = int(count)
	return toNewsArticle(document), nil
}

func (m *Mongo) PendingNewsEvents(ctx context.Context, limit int) ([]domain.NewsEvent, error) {
	cur, err := m.DB.Collection("normalized_articles").Find(ctx,
		bson.M{"outbox_state": "pending", "delivery_attempts": bson.M{"$lt": 5}},
		options.Find().SetSort(bson.D{{Key: "retrieved_at", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	events := []domain.NewsEvent{}
	for cur.Next(ctx) {
		var document newsDocument
		if err := cur.Decode(&document); err != nil {
			return nil, err
		}
		events = append(events, domain.NewsEvent{ArticleID: document.ID.Hex(), Title: document.Title, SourceID: document.SourceKey, PublishedAt: document.PublishedAt, Symbols: nonNil(document.Symbols)})
	}
	return events, cur.Err()
}

func (m *Mongo) MarkNewsEventPublished(ctx context.Context, articleID string) error {
	id, err := bson.ObjectIDFromHex(articleID)
	if err != nil {
		return ErrNotFound
	}
	_, err = m.DB.Collection("normalized_articles").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"outbox_state": "published", "published_event_at": time.Now().UTC(), "last_publish_error": ""}})
	return err
}

func (m *Mongo) RecordPublishFailure(ctx context.Context, articleID, message string, maxAttempts int) error {
	id, err := bson.ObjectIDFromHex(articleID)
	if err != nil {
		return ErrNotFound
	}
	var document newsDocument
	err = m.DB.Collection("normalized_articles").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{
		"$inc": bson.M{"delivery_attempts": 1},
		"$set": bson.M{"last_publish_error": message, "last_publish_attempt_at": time.Now().UTC()},
	}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&document)
	if err != nil {
		return err
	}
	if document.DeliveryAttempts < maxAttempts {
		return nil
	}
	_, err = m.DB.Collection("normalized_articles").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"outbox_state": "dead"}})
	if err != nil {
		return err
	}
	_, err = m.DB.Collection("ingestion_dead_letters").UpdateOne(ctx, bson.M{"event_key": "news.created:" + articleID}, bson.M{"$set": bson.M{
		"event_key": "news.created:" + articleID, "article_id": id, "topic": "news.created", "error": message,
		"attempts": document.DeliveryAttempts, "failed_at": time.Now().UTC(), "status": "dead",
	}}, options.UpdateOne().SetUpsert(true))
	return err
}

func (m *Mongo) ReplayDeadNewsEvents(ctx context.Context, limit int) (int64, error) {
	cur, err := m.DB.Collection("ingestion_dead_letters").Find(ctx, bson.M{"status": "dead"}, options.Find().SetLimit(int64(limit)))
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)
	var replayed int64
	for cur.Next(ctx) {
		var dead struct {
			ID        bson.ObjectID `bson:"_id"`
			ArticleID bson.ObjectID `bson:"article_id"`
		}
		if err := cur.Decode(&dead); err != nil {
			return replayed, err
		}
		if _, err := m.DB.Collection("normalized_articles").UpdateOne(ctx, bson.M{"_id": dead.ArticleID}, bson.M{"$set": bson.M{"outbox_state": "pending", "delivery_attempts": 0}}); err != nil {
			return replayed, err
		}
		_, _ = m.DB.Collection("ingestion_dead_letters").UpdateOne(ctx, bson.M{"_id": dead.ID}, bson.M{"$set": bson.M{"status": "replayed", "replayed_at": time.Now().UTC()}})
		replayed++
	}
	return replayed, cur.Err()
}

func (m *Mongo) RecordSuccess(ctx context.Context, key, parserVersion string, latency time.Duration, policy ingest.SourcePolicy) error {
	if err := m.ensureSource(ctx, key, parserVersion, domain.SourceItem{Attribution: policy.Attribution, Licence: policy.Licence, Synthetic: strings.HasPrefix(key, "mock-")}, policy); err != nil {
		return err
	}
	_, err := m.DB.Collection("source_health_metrics").InsertOne(ctx, bson.M{
		"source_key": key, "status": "healthy", "parser_version": parserVersion, "latency_ms": latency.Milliseconds(),
		"consecutive_failures": 0, "circuit_state": "closed", "checked_at": time.Now().UTC(),
	})
	return err
}

func (m *Mongo) RecordFailure(ctx context.Context, key, parserVersion, message string) error {
	consecutive := 1
	var latest struct {
		Consecutive int `bson:"consecutive_failures"`
	}
	if err := m.DB.Collection("source_health_metrics").FindOne(ctx, bson.M{"source_key": key}, options.FindOne().SetSort(bson.D{{Key: "checked_at", Value: -1}})).Decode(&latest); err == nil {
		consecutive = latest.Consecutive + 1
	}
	state := "closed"
	if consecutive >= 5 {
		state = "open"
	}
	_, err := m.DB.Collection("source_health_metrics").InsertOne(ctx, bson.M{
		"source_key": key, "status": "degraded", "parser_version": parserVersion, "parser_failures": 1,
		"consecutive_failures": consecutive, "circuit_state": state, "error": message, "checked_at": time.Now().UTC(),
	})
	return err
}

func (m *Mongo) SourceHealth(ctx context.Context) ([]domain.SourceHealth, error) {
	cur, err := m.DB.Collection("news_sources").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "key", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []domain.SourceHealth{}
	for cur.Next(ctx) {
		var source struct {
			Key        string `bson:"key"`
			Name       string `bson:"name"`
			SourceType string `bson:"source_type"`
			Synthetic  bool   `bson:"synthetic"`
		}
		if err := cur.Decode(&source); err != nil {
			return nil, err
		}
		var policy struct {
			Licence             string `bson:"licence"`
			ParserVersion       string `bson:"parser_version"`
			PollIntervalSeconds int    `bson:"poll_interval_seconds"`
		}
		_ = m.DB.Collection("source_policies").FindOne(ctx, bson.M{"source_key": source.Key}).Decode(&policy)
		health := domain.SourceHealth{ID: source.Key, Name: source.Name, SourceType: source.SourceType, Status: "unknown", ParserVersion: policy.ParserVersion, Licence: policy.Licence, PollIntervalSeconds: policy.PollIntervalSeconds, Synthetic: source.Synthetic}
		var metric struct {
			Status              string    `bson:"status"`
			CheckedAt           time.Time `bson:"checked_at"`
			ConsecutiveFailures int       `bson:"consecutive_failures"`
			CircuitState        string    `bson:"circuit_state"`
		}
		if err := m.DB.Collection("source_health_metrics").FindOne(ctx, bson.M{"source_key": source.Key}, options.FindOne().SetSort(bson.D{{Key: "checked_at", Value: -1}})).Decode(&metric); err == nil {
			health.Status, health.ConsecutiveFailures, health.CircuitState = metric.Status, metric.ConsecutiveFailures, metric.CircuitState
			if metric.Status == "healthy" {
				health.LastSuccess = &metric.CheckedAt
			} else {
				health.LastFailure = &metric.CheckedAt
			}
		}
		out = append(out, health)
	}
	return out, cur.Err()
}

func (m *Mongo) SourceCursor(ctx context.Context, sourceKey string) (ingest.Cursor, error) {
	var state struct {
		Value string    `bson:"cursor_value"`
		Since time.Time `bson:"cursor_since"`
	}
	err := m.DB.Collection("source_states").FindOne(ctx, bson.M{"source_key": sourceKey}).Decode(&state)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ingest.Cursor{}, nil
	}
	return ingest.Cursor{Value: state.Value, Since: state.Since}, err
}

func (m *Mongo) SaveSourceCursor(ctx context.Context, sourceKey string, cursor ingest.Cursor) error {
	_, err := m.DB.Collection("source_states").UpdateOne(ctx, bson.M{"source_key": sourceKey}, bson.M{
		"$set":         bson.M{"cursor_value": cursor.Value, "cursor_since": cursor.Since, "updated_at": time.Now().UTC()},
		"$setOnInsert": bson.M{"source_key": sourceKey, "created_at": time.Now().UTC()},
	}, options.UpdateOne().SetUpsert(true))
	return err
}

// MigratePhase2News safely backfills documents written by the Phase 1 mock
// processor so each legacy item remains its own cluster and enters the outbox.
func (m *Mongo) MigratePhase2News(ctx context.Context) error {
	_, err := m.DB.Collection("normalized_articles").UpdateMany(ctx, bson.M{}, mongo.Pipeline{
		{{Key: "$set", Value: bson.D{
			{Key: "cluster_id", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$cluster_id", "$content_hash"}}}},
			{Key: "source_name", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$source_name", "$attribution"}}}},
			{Key: "source_type", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$source_type", "mock"}}}},
			{Key: "symbols", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$symbols", bson.A{}}}}},
			{Key: "sectors", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$sectors", bson.A{}}}}},
			{Key: "outbox_state", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$outbox_state", "pending"}}}},
			{Key: "delivery_attempts", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$delivery_attempts", 0}}}},
		}}},
	})
	return err
}

func (m *Mongo) MigratePhase3Intelligence(ctx context.Context) error {
	_, err := m.DB.Collection("normalized_articles").UpdateMany(ctx, bson.M{}, mongo.Pipeline{
		{{Key: "$set", Value: bson.D{
			{Key: "analysis_state", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$analysis_state", "pending"}}}},
			{Key: "analysis_attempts", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$analysis_attempts", 0}}}},
		}}},
	})
	if err != nil {
		return err
	}
	_, err = m.DB.Collection("signal_versions").UpdateOne(ctx, bson.M{"version": "signal-v2"}, bson.M{"$setOnInsert": bson.M{
		"version": "signal-v2", "description": "Phase 3 grounded evidence scoring",
		"weights":    bson.M{"news_sentiment": .16, "materiality": .13, "source_reliability": .08, "ai_confidence": .09, "recency": .08, "confirmations": .07, "price_movement": .11, "volume_anomaly": .08, "sector_movement": .05, "index_movement": .04, "valuation": .05, "financial_health": .08, "contradiction": .08},
		"created_at": time.Now().UTC(),
	}}, options.UpdateOne().SetUpsert(true))
	return err
}

func jaccard(left, right string) float64 {
	a, b := wordSet(left), wordSet(right)
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for word := range a {
		if _, ok := b[word]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	return float64(intersection) / float64(union)
}

func wordSet(value string) map[string]struct{} {
	value = strings.ToLower(value)
	value = regexp.MustCompile(`[^\p{L}\p{N}]+`).ReplaceAllString(value, " ")
	words := strings.Fields(value)
	out := make(map[string]struct{}, len(words))
	for _, word := range words {
		if len([]rune(word)) > 2 {
			out[word] = struct{}{}
		}
	}
	return out
}

func uniqueUpper(values []string) []string {
	out := uniqueStrings(values)
	for i := range out {
		out[i] = strings.ToUpper(out[i])
	}
	sort.Strings(out)
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[strings.ToLower(value)] = value
		}
	}
	out := make([]string, 0, len(seen))
	for _, value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/stocker-app/stocker/internal/domain"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")
var ErrWatchlistLimit = errors.New("watchlist limit is 10")

type Mongo struct {
	Client *mongo.Client
	DB     *mongo.Database
}

type userDocument struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Email        string        `bson:"email"`
	DisplayName  string        `bson:"display_name"`
	PasswordHash string        `bson:"password_hash"`
	Role         string        `bson:"role"`
	CreatedAt    time.Time     `bson:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"`
	DeletedAt    *time.Time    `bson:"deleted_at,omitempty"`
}

type securityDocument struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	NSESymbol   string        `bson:"nse_symbol,omitempty"`
	BSECode     string        `bson:"bse_code,omitempty"`
	ISIN        string        `bson:"isin"`
	CompanyName string        `bson:"company_name"`
	Sector      string        `bson:"sector"`
	Industry    string        `bson:"industry"`
	Active      bool          `bson:"active"`
	CreatedAt   time.Time     `bson:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"`
}

type quoteDocument struct {
	SecurityID    bson.ObjectID `bson:"security_id"`
	LastPrice     float64       `bson:"last_price"`
	Open          float64       `bson:"open,omitempty"`
	High          float64       `bson:"high,omitempty"`
	Low           float64       `bson:"low,omitempty"`
	PreviousClose float64       `bson:"previous_close,omitempty"`
	Volume        int64         `bson:"volume,omitempty"`
	ChangePercent float64       `bson:"change_percent"`
	Source        string        `bson:"source"`
	IsDelayed     bool          `bson:"is_delayed"`
	AsOf          time.Time     `bson:"as_of"`
}

type signalDocument struct {
	ID            bson.ObjectID     `bson:"_id,omitempty"`
	SecurityID    bson.ObjectID     `bson:"security_id"`
	AnalysisID    bson.ObjectID     `bson:"analysis_id,omitempty"`
	ArticleID     bson.ObjectID     `bson:"article_id,omitempty"`
	Symbol        string            `bson:"symbol,omitempty"`
	Label         string            `bson:"label"`
	Score         float64           `bson:"score"`
	Confidence    int               `bson:"confidence"`
	Horizon       string            `bson:"horizon"`
	Version       string            `bson:"version"`
	Reasons       []string          `bson:"reasons"`
	Risks         []string          `bson:"risks"`
	Invalidators  []string          `bson:"invalidators"`
	InputSnapshot bson.M            `bson:"input_snapshot"`
	DataFreshAt   time.Time         `bson:"data_fresh_at"`
	GeneratedAt   time.Time         `bson:"generated_at"`
	Sources       []domain.Evidence `bson:"sources,omitempty"`
}

type watchlistEntry struct {
	SecurityID   bson.ObjectID `bson:"security_id"`
	AlertsPaused bool          `bson:"alerts_paused"`
	AddedAt      time.Time     `bson:"added_at"`
}

type watchlistDocument struct {
	ID        bson.ObjectID    `bson:"_id,omitempty"`
	UserID    bson.ObjectID    `bson:"user_id"`
	Items     []watchlistEntry `bson:"items"`
	UpdatedAt time.Time        `bson:"updated_at"`
}

func Open(ctx context.Context, uri, database string) (*Mongo, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetAppName("stocker-api").SetServerSelectionTimeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	m := &Mongo{Client: client, DB: client.Database(database)}
	if err := m.Ping(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return m, nil
}

func (m *Mongo) Close(ctx context.Context) error { return m.Client.Disconnect(ctx) }
func (m *Mongo) Ping(ctx context.Context) error  { return m.Client.Ping(ctx, readpref.Primary()) }

// MigrateLegacyIndexNames repairs a Phase 1 startup bug where a shared mutable
// index-options pointer named unrelated unique indexes "content_hash_1". The
// key definitions were correct, but MongoDB refuses equivalent indexes with a
// different requested name. Dropping only those misnamed indexes lets the
// normal idempotent index initializer recreate them with stable names.
func (m *Mongo) MigrateLegacyIndexNames(ctx context.Context) error {
	collections := []string{"users", "refresh_sessions", "securities", "watchlists", "raw_documents", "news_sources", "source_policies", "article_security_links", "alert_events"}
	for _, collection := range collections {
		cursor, err := m.DB.Collection(collection).Indexes().List(ctx)
		if err != nil {
			return fmt.Errorf("list indexes for %s: %w", collection, err)
		}
		var indexes []struct {
			Name string `bson:"name"`
		}
		if err := cursor.All(ctx, &indexes); err != nil {
			return fmt.Errorf("decode indexes for %s: %w", collection, err)
		}
		for _, index := range indexes {
			if index.Name != "content_hash_1" {
				continue
			}
			if err := m.DB.Collection(collection).Indexes().DropOne(ctx, index.Name); err != nil {
				return fmt.Errorf("drop legacy index %s on %s: %w", index.Name, collection, err)
			}
		}
	}
	return nil
}

func (m *Mongo) EnsureIndexes(ctx context.Context) error {
	indexes := map[string][]mongo.IndexModel{
		"users": {{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"refresh_sessions": {
			{Keys: bson.D{{Key: "token_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
		"securities": {
			{Keys: bson.D{{Key: "isin", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "nse_symbol", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
			{Keys: bson.D{{Key: "bse_code", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
			{Keys: bson.D{{Key: "company_name", Value: "text"}, {Key: "nse_symbol", Value: "text"}, {Key: "bse_code", Value: "text"}, {Key: "isin", Value: "text"}}},
		},
		"watchlists":    {{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"market_quotes": {{Keys: bson.D{{Key: "security_id", Value: 1}, {Key: "as_of", Value: -1}}}},
		"normalized_articles": {
			{Keys: bson.D{{Key: "content_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "title", Value: "text"}, {Key: "body_text", Value: "text"}}},
			{Keys: bson.D{{Key: "published_at", Value: -1}, {Key: "source_key", Value: 1}}},
			{Keys: bson.D{{Key: "cluster_id", Value: 1}, {Key: "published_at", Value: -1}}},
			{Keys: bson.D{{Key: "symbols", Value: 1}, {Key: "published_at", Value: -1}}},
			{Keys: bson.D{{Key: "outbox_state", Value: 1}, {Key: "delivery_attempts", Value: 1}}},
		},
		"raw_documents":          {{Keys: bson.D{{Key: "source_key", Value: 1}, {Key: "content_hash", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "delete_after", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}},
		"news_sources":           {{Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"source_policies":        {{Keys: bson.D{{Key: "source_key", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"source_states":          {{Keys: bson.D{{Key: "source_key", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"source_health_metrics":  {{Keys: bson.D{{Key: "source_key", Value: 1}, {Key: "checked_at", Value: -1}}}},
		"article_security_links": {{Keys: bson.D{{Key: "article_id", Value: 1}, {Key: "security_id", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"ai_analyses": {
			{Keys: bson.D{{Key: "article_id", Value: 1}, {Key: "prompt_version", Value: 1}, {Key: "provider", Value: 1}, {Key: "model", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "created_at", Value: -1}}},
		},
		"evidence_references":    {{Keys: bson.D{{Key: "analysis_id", Value: 1}, {Key: "evidence_hash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"ai_daily_budgets":       {{Keys: bson.D{{Key: "provider", Value: 1}, {Key: "date", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"prompt_versions":        {{Keys: bson.D{{Key: "version", Value: 1}, {Key: "prompt_hash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"signal_versions":        {{Keys: bson.D{{Key: "version", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"analysis_dead_letters":  {{Keys: bson.D{{Key: "event_key", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"ingestion_dead_letters": {{Keys: bson.D{{Key: "event_key", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"signals": {
			{Keys: bson.D{{Key: "security_id", Value: 1}, {Key: "generated_at", Value: -1}}},
			{Keys: bson.D{{Key: "analysis_id", Value: 1}, {Key: "security_id", Value: 1}, {Key: "version", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		},
		"alert_events": {{Keys: bson.D{{Key: "deduplication_key", Value: 1}}, Options: options.Index().SetUnique(true)}},
	}
	for collection, models := range indexes {
		if _, err := m.DB.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("indexes for %s: %w", collection, err)
		}
	}
	return nil
}

func (m *Mongo) Seed(ctx context.Context) error {
	now := time.Now().UTC()
	seeds := []securityDocument{
		{NSESymbol: "RELIANCE", BSECode: "500325", ISIN: "INE002A01018", CompanyName: "Reliance Industries Limited", Sector: "Energy", Industry: "Oil, Gas & Consumable Fuels", Active: true},
		{NSESymbol: "HDFCBANK", BSECode: "500180", ISIN: "INE040A01034", CompanyName: "HDFC Bank Limited", Sector: "Financial Services", Industry: "Banks", Active: true},
		{NSESymbol: "INFY", BSECode: "500209", ISIN: "INE009A01021", CompanyName: "Infosys Limited", Sector: "Information Technology", Industry: "IT Services & Consulting", Active: true},
		{NSESymbol: "TCS", BSECode: "532540", ISIN: "INE467B01029", CompanyName: "Tata Consultancy Services Limited", Sector: "Information Technology", Industry: "IT Services & Consulting", Active: true},
		{NSESymbol: "ITC", BSECode: "500875", ISIN: "INE154A01025", CompanyName: "ITC Limited", Sector: "Consumer Staples", Industry: "Diversified FMCG", Active: true},
		{NSESymbol: "LT", BSECode: "500510", ISIN: "INE018A01030", CompanyName: "Larsen & Toubro Limited", Sector: "Industrials", Industry: "Engineering", Active: true},
	}
	for _, seed := range seeds {
		seed.CreatedAt, seed.UpdatedAt = now, now
		_, err := m.DB.Collection("securities").UpdateOne(ctx, bson.M{"isin": seed.ISIN}, bson.M{"$setOnInsert": seed}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	quotes := map[string]quoteDocument{
		"RELIANCE": {LastPrice: 1392.40, Open: 1381, High: 1401.20, Low: 1378.35, PreviousClose: 1382.50, Volume: 8210050, ChangePercent: .72},
		"HDFCBANK": {LastPrice: 983.15, PreviousClose: 977.60, ChangePercent: .57},
		"INFY":     {LastPrice: 1518.20, PreviousClose: 1532.10, ChangePercent: -.91},
	}
	for symbol, q := range quotes {
		var s securityDocument
		if err := m.DB.Collection("securities").FindOne(ctx, bson.M{"nse_symbol": symbol}).Decode(&s); err != nil {
			return err
		}
		q.SecurityID = s.ID
		q.Source = "Mock market provider"
		q.IsDelayed = true
		q.AsOf = now.Add(-15 * time.Minute)
		_, err := m.DB.Collection("market_quotes").UpdateOne(ctx, bson.M{"security_id": s.ID, "source": q.Source}, bson.M{"$set": q}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	for symbol, value := range map[string]float64{"NIFTY50": 24894.25, "SENSEX": 81207.40} {
		_, err := m.DB.Collection("market_indices").UpdateOne(ctx, bson.M{"symbol": symbol, "source": "Mock market provider"}, bson.M{"$set": bson.M{"symbol": symbol, "value": value, "change_percent": .6, "source": "Mock market provider", "as_of": now.Add(-15 * time.Minute)}}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	var reliance securityDocument
	if err := m.DB.Collection("securities").FindOne(ctx, bson.M{"nse_symbol": "RELIANCE"}).Decode(&reliance); err != nil {
		return err
	}
	_, err := m.DB.Collection("signals").UpdateOne(ctx, bson.M{"security_id": reliance.ID, "version": "signal-v1", "fixture": true}, bson.M{"$setOnInsert": signalDocument{SecurityID: reliance.ID, Label: "Positive setup", Score: 42.6, Confidence: 73, Horizon: "medium term", Version: "signal-v1", Reasons: []string{"Synthetic capacity-investment event is potentially material", "Price context is mildly supportive"}, Risks: []string{"Mock event is not a real announcement", "Execution timing and returns remain uncertain"}, Invalidators: []string{"No primary-source confirmation", "Material project delay or cancellation"}, InputSnapshot: bson.M{"demo": true, "evidence_count": 1}, DataFreshAt: now.Add(-15 * time.Minute), GeneratedAt: now}}, options.UpdateOne().SetUpsert(true))
	return err
}

func toUser(d userDocument) domain.User {
	return domain.User{ID: d.ID.Hex(), Email: d.Email, DisplayName: d.DisplayName, PasswordHash: d.PasswordHash, Role: d.Role, CreatedAt: d.CreatedAt}
}
func (m *Mongo) CreateUser(ctx context.Context, email, name, passwordHash string) (domain.User, error) {
	now := time.Now().UTC()
	d := userDocument{Email: strings.ToLower(strings.TrimSpace(email)), DisplayName: strings.TrimSpace(name), PasswordHash: passwordHash, Role: "user", CreatedAt: now, UpdatedAt: now}
	res, err := m.DB.Collection("users").InsertOne(ctx, d)
	if mongo.IsDuplicateKeyError(err) {
		return domain.User{}, ErrConflict
	}
	if err != nil {
		return domain.User{}, err
	}
	d.ID = res.InsertedID.(bson.ObjectID)
	return toUser(d), nil
}
func (m *Mongo) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	var d userDocument
	err := m.DB.Collection("users").FindOne(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(email)), "deleted_at": bson.M{"$exists": false}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.User{}, ErrNotFound
	}
	return toUser(d), err
}
func (m *Mongo) UserByID(ctx context.Context, id string) (domain.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.User{}, ErrNotFound
	}
	var d userDocument
	err = m.DB.Collection("users").FindOne(ctx, bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.User{}, ErrNotFound
	}
	return toUser(d), err
}
func (m *Mongo) UserRole(ctx context.Context, id string) (string, error) {
	u, err := m.UserByID(ctx, id)
	return u.Role, err
}
func (m *Mongo) CreateRefreshSession(ctx context.Context, uid string, hash []byte, expires time.Time) error {
	id, err := bson.ObjectIDFromHex(uid)
	if err != nil {
		return ErrNotFound
	}
	_, err = m.DB.Collection("refresh_sessions").InsertOne(ctx, bson.M{"user_id": id, "token_hash": hash, "created_at": time.Now().UTC(), "expires_at": expires})
	return err
}
func (m *Mongo) RotateRefreshSession(ctx context.Context, oldHash, newHash []byte, expires time.Time) (string, error) {
	var d struct {
		UserID bson.ObjectID `bson:"user_id"`
	}
	err := m.DB.Collection("refresh_sessions").FindOneAndUpdate(ctx, bson.M{"token_hash": oldHash, "revoked_at": bson.M{"$exists": false}, "expires_at": bson.M{"$gt": time.Now().UTC()}}, bson.M{"$set": bson.M{"revoked_at": time.Now().UTC(), "replaced_by_hash": newHash}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err = m.DB.Collection("refresh_sessions").InsertOne(ctx, bson.M{"user_id": d.UserID, "token_hash": newHash, "created_at": time.Now().UTC(), "expires_at": expires}); err != nil {
		return "", err
	}
	return d.UserID.Hex(), nil
}
func (m *Mongo) RevokeRefreshSession(ctx context.Context, hash []byte) error {
	_, err := m.DB.Collection("refresh_sessions").UpdateOne(ctx, bson.M{"token_hash": hash, "revoked_at": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"revoked_at": time.Now().UTC()}})
	return err
}

func (m *Mongo) hydrateSecurity(ctx context.Context, d securityDocument) (domain.Security, error) {
	s := domain.Security{ID: d.ID.Hex(), NSESymbol: d.NSESymbol, BSECode: d.BSECode, ISIN: d.ISIN, CompanyName: d.CompanyName, Sector: d.Sector, Industry: d.Industry, AsOf: d.UpdatedAt, Source: "Seed security master", Signal: "Insufficient evidence"}
	var q quoteDocument
	if err := m.DB.Collection("market_quotes").FindOne(ctx, bson.M{"security_id": d.ID}, options.FindOne().SetSort(bson.D{{Key: "as_of", Value: -1}})).Decode(&q); err == nil {
		s.Price = q.LastPrice
		s.ChangePercent = q.ChangePercent
		s.AsOf = q.AsOf
		s.Source = q.Source
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return s, err
	}
	var sig signalDocument
	if err := m.DB.Collection("signals").FindOne(ctx, bson.M{"security_id": d.ID}, options.FindOne().SetSort(bson.D{{Key: "generated_at", Value: -1}})).Decode(&sig); err == nil {
		s.Signal = sig.Label
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return s, err
	}
	count, err := m.DB.Collection("article_security_links").CountDocuments(ctx, bson.M{"security_id": d.ID})
	s.NewsCount = int(count)
	return s, err
}
func (m *Mongo) SearchSecurities(ctx context.Context, q string, limit int) ([]domain.Security, error) {
	pattern := regexp.QuoteMeta(strings.TrimSpace(q))
	filter := bson.M{"active": true, "$or": bson.A{bson.M{"company_name": bson.M{"$regex": pattern, "$options": "i"}}, bson.M{"nse_symbol": bson.M{"$regex": "^" + pattern, "$options": "i"}}, bson.M{"bse_code": bson.M{"$regex": "^" + pattern}}, bson.M{"isin": bson.M{"$regex": "^" + pattern, "$options": "i"}}}}
	cur, err := m.DB.Collection("securities").Find(ctx, filter, options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "company_name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []domain.Security{}
	for cur.Next(ctx) {
		var d securityDocument
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		s, err := m.hydrateSecurity(ctx, d)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, cur.Err()
}
func (m *Mongo) securityDocumentBySymbol(ctx context.Context, symbol string) (securityDocument, error) {
	var d securityDocument
	value := strings.ToUpper(strings.TrimSpace(symbol))
	err := m.DB.Collection("securities").FindOne(ctx, bson.M{"$or": bson.A{bson.M{"nse_symbol": value}, bson.M{"bse_code": value}, bson.M{"isin": value}}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		err = ErrNotFound
	}
	return d, err
}
func (m *Mongo) SecurityBySymbol(ctx context.Context, symbol string) (domain.Security, error) {
	d, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return domain.Security{}, err
	}
	return m.hydrateSecurity(ctx, d)
}

func (m *Mongo) Watchlist(ctx context.Context, uid string) ([]domain.WatchlistItem, error) {
	userID, err := bson.ObjectIDFromHex(uid)
	if err != nil {
		return nil, ErrNotFound
	}
	var d watchlistDocument
	err = m.DB.Collection("watchlists").FindOne(ctx, bson.M{"user_id": userID}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return []domain.WatchlistItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]domain.WatchlistItem, 0, len(d.Items))
	for _, entry := range d.Items {
		var sec securityDocument
		if err := m.DB.Collection("securities").FindOne(ctx, bson.M{"_id": entry.SecurityID}).Decode(&sec); err != nil {
			continue
		}
		s, err := m.hydrateSecurity(ctx, sec)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.WatchlistItem{Security: s, AlertsPaused: entry.AlertsPaused, AddedAt: entry.AddedAt})
	}
	return out, nil
}
func (m *Mongo) AddWatchlist(ctx context.Context, uid, symbol string) (domain.WatchlistItem, error) {
	userID, err := bson.ObjectIDFromHex(uid)
	if err != nil {
		return domain.WatchlistItem{}, ErrNotFound
	}
	sec, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return domain.WatchlistItem{}, err
	}
	entry := watchlistEntry{SecurityID: sec.ID, AddedAt: time.Now().UTC()}
	filter := bson.M{"user_id": userID, "items.security_id": bson.M{"$ne": sec.ID}, "$expr": bson.M{"$lt": bson.A{bson.M{"$size": bson.M{"$ifNull": bson.A{"$items", bson.A{}}}}, 10}}}
	result, err := m.DB.Collection("watchlists").UpdateOne(ctx, filter, bson.M{"$push": bson.M{"items": entry}, "$set": bson.M{"updated_at": time.Now().UTC()}})
	if err != nil {
		return domain.WatchlistItem{}, err
	}
	if result.MatchedCount == 0 {
		_, insertErr := m.DB.Collection("watchlists").InsertOne(ctx, watchlistDocument{UserID: userID, Items: []watchlistEntry{entry}, UpdatedAt: time.Now().UTC()})
		if mongo.IsDuplicateKeyError(insertErr) {
			var current watchlistDocument
			if err := m.DB.Collection("watchlists").FindOne(ctx, bson.M{"user_id": userID}).Decode(&current); err != nil {
				return domain.WatchlistItem{}, err
			}
			for _, v := range current.Items {
				if v.SecurityID == sec.ID {
					return domain.WatchlistItem{}, ErrConflict
				}
			}
			if len(current.Items) >= 10 {
				return domain.WatchlistItem{}, ErrWatchlistLimit
			}
			result, err = m.DB.Collection("watchlists").UpdateOne(ctx, filter, bson.M{"$push": bson.M{"items": entry}, "$set": bson.M{"updated_at": time.Now().UTC()}})
			if err != nil {
				return domain.WatchlistItem{}, err
			}
			if result.ModifiedCount == 0 {
				return domain.WatchlistItem{}, ErrConflict
			}
		} else if insertErr != nil {
			return domain.WatchlistItem{}, insertErr
		}
	}
	s, err := m.hydrateSecurity(ctx, sec)
	return domain.WatchlistItem{Security: s, AddedAt: entry.AddedAt}, err
}
func (m *Mongo) DeleteWatchlist(ctx context.Context, uid, symbol string) error {
	userID, err := bson.ObjectIDFromHex(uid)
	if err != nil {
		return ErrNotFound
	}
	sec, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return err
	}
	res, err := m.DB.Collection("watchlists").UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$pull": bson.M{"items": bson.M{"security_id": sec.ID}}, "$set": bson.M{"updated_at": time.Now().UTC()}})
	if err == nil && res.ModifiedCount == 0 {
		return ErrNotFound
	}
	return err
}
func (m *Mongo) PauseWatchlist(ctx context.Context, uid, symbol string, paused bool) error {
	userID, err := bson.ObjectIDFromHex(uid)
	if err != nil {
		return ErrNotFound
	}
	sec, err := m.securityDocumentBySymbol(ctx, symbol)
	if err != nil {
		return err
	}
	res, err := m.DB.Collection("watchlists").UpdateOne(ctx, bson.M{"user_id": userID, "items.security_id": sec.ID}, bson.M{"$set": bson.M{"items.$.alerts_paused": paused, "updated_at": time.Now().UTC()}})
	if err == nil && res.ModifiedCount == 0 {
		return ErrNotFound
	}
	return err
}

func (m *Mongo) Overview(ctx context.Context) (map[string]any, error) {
	cur, err := m.DB.Collection("market_indices").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "symbol", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	indices := []map[string]any{}
	for cur.Next(ctx) {
		var d struct {
			Symbol        string    `bson:"symbol"`
			Value         float64   `bson:"value"`
			ChangePercent float64   `bson:"change_percent"`
			Source        string    `bson:"source"`
			AsOf          time.Time `bson:"as_of"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		indices = append(indices, map[string]any{"symbol": d.Symbol, "value": d.Value, "changePercent": d.ChangePercent, "source": d.Source, "asOf": d.AsOf})
	}
	return map[string]any{"indices": indices, "mood": map[string]any{"label": "Cautiously positive", "score": 62, "explanation": "Breadth is positive, while mixed global cues keep conviction moderate.", "asOf": time.Now().UTC(), "source": "Mock market provider — delayed demonstration data"}, "breadth": map[string]int{"advances": 1378, "declines": 982, "unchanged": 126}}, cur.Err()
}

func (m *Mongo) SeenHash(ctx context.Context, hash string) (bool, error) {
	// Only a normalized article proves processing completed. A raw document may
	// remain after a crash between the raw and normalized writes; treating that
	// partial state as seen would permanently strand the item.
	n, err := m.DB.Collection("normalized_articles").CountDocuments(ctx, bson.M{"content_hash": hash}, options.Count().SetLimit(1))
	return n > 0, err
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stocker-app/stocker/internal/auth"
	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/market"
	"github.com/stocker-app/stocker/internal/store"
)

type Server struct {
	cfg    config.Config
	db     *store.Mongo
	redis  *redis.Client
	auth   *auth.Service
	hub    *Hub
	market *market.Service
	log    *slog.Logger
	router *gin.Engine
	cancel context.CancelFunc
}
type ctxKey string

const userIDKey ctxKey = "user_id"

func New(cfg config.Config, db *store.Mongo, rdb *redis.Client, log *slog.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	marketService, err := market.NewService(cfg, db, log)
	if err != nil {
		log.Error("market_service_initialization_failed", "error", err)
	}
	s := &Server{cfg: cfg, db: db, redis: rdb, auth: auth.New(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, db), hub: NewHub(), market: marketService, log: log, cancel: cancel}
	s.router = s.routes()
	go s.bridgeNewsEvents(ctx)
	return s
}
func (s *Server) Handler() http.Handler { return s.router }
func (s *Server) Close()                { s.cancel() }

func (s *Server) routes() *gin.Engine {
	if s.cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), s.requestLog(), s.securityHeaders(), cors.New(cors.Config{AllowOrigins: []string{s.cfg.WebOrigin}, AllowMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Authorization", "Content-Type", "X-CSRF-Token"}, ExposeHeaders: []string{"X-Request-ID"}, AllowCredentials: true, MaxAge: 12 * time.Hour}), s.rateLimit())
	r.GET("/health/live", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/health/ready", s.ready)
	r.GET("/metrics", s.metrics)
	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", s.register)
	v1.POST("/auth/login", s.login)
	v1.POST("/auth/refresh", s.requireTrustedOrigin(), s.refresh)
	v1.POST("/auth/logout", s.requireTrustedOrigin(), s.logout)
	v1.GET("/market/overview", s.overview)
	v1.GET("/market/sectors", s.marketSectors)
	v1.GET("/market/movers", s.marketMovers)
	v1.GET("/events", s.marketEvents)
	v1.GET("/stocks/search", s.search)
	v1.GET("/stocks/:symbol", s.stock)
	v1.GET("/stocks/:symbol/quote", s.stockQuote)
	v1.GET("/stocks/:symbol/fundamentals", s.stockFundamentals)
	v1.GET("/stocks/:symbol/peers", s.stockPeers)
	v1.GET("/stocks/:symbol/risk-flags", s.stockRiskFlags)
	v1.GET("/stocks/:symbol/news", s.stockNews)
	v1.GET("/stocks/:symbol/signals", s.stockSignals)
	v1.GET("/news", s.news)
	v1.GET("/news/:id", s.newsDetail)
	v1.GET("/news/:id/analysis", s.newsAnalysis)
	v1.GET("/system/source-health", s.sourceHealth)
	protected := v1.Group("")
	protected.Use(s.requireAuth())
	protected.GET("/auth/me", s.me)
	protected.GET("/stream", s.stream)
	protected.GET("/watchlist", s.watchlist)
	protected.POST("/watchlist", s.addWatchlist)
	protected.DELETE("/watchlist/:symbol", s.deleteWatchlist)
	protected.PATCH("/watchlist/:symbol", s.pauseWatchlist)
	protected.GET("/alerts", s.alerts)
	protected.POST("/alert-rules", s.notImplemented("Alert-rule management is scheduled for Phase 5."))
	protected.PATCH("/alert-rules/:id", s.notImplemented("Alert-rule management is scheduled for Phase 5."))
	protected.GET("/briefings", s.briefings)
	return r
}
func (s *Server) ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		c.JSON(503, gin.H{"status": "unavailable", "database": "down"})
		return
	}
	if err := s.redis.Ping(ctx).Err(); err != nil {
		if s.cfg.RedisRequired {
			c.JSON(503, gin.H{"status": "unavailable", "redis": "down"})
			return
		}
		c.JSON(200, gin.H{"status": "ready", "redis": "unavailable (optional in development)"})
		return
	}
	c.JSON(200, gin.H{"status": "ready"})
}
func (s *Server) metrics(c *gin.Context) {
	subscribers, dropped := s.hub.Stats()
	c.Header("Content-Type", "text/plain; version=0.0.4")
	c.String(200, "# HELP stocker_up Whether the API is running.\n# TYPE stocker_up gauge\nstocker_up 1\n# HELP stocker_sse_subscribers Current authenticated SSE subscribers.\n# TYPE stocker_sse_subscribers gauge\nstocker_sse_subscribers %d\n# HELP stocker_sse_dropped_events Events dropped for slow subscribers.\n# TYPE stocker_sse_dropped_events counter\nstocker_sse_dropped_events %d\n", subscribers, dropped)
}
func (s *Server) requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		s.log.Info("http_request", "method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds(), "client_ip", c.ClientIP())
	}
}
func (s *Server) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}
func (s *Server) rateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health/live" {
			c.Next()
			return
		}
		ctx := c.Request.Context()
		key := "rl:" + c.ClientIP() + ":" + time.Now().UTC().Format("200601021504")
		n, err := s.redis.Incr(ctx, key).Result()
		if err == nil && n == 1 {
			s.redis.Expire(ctx, key, 70*time.Second)
		}
		if err == nil && n > 120 {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(429, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
			return
		}
		claims, err := s.auth.Parse(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid access token"})
			return
		}
		c.Set(string(userIDKey), claims.Subject)
		c.Next()
	}
}
func (s *Server) requireTrustedOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && origin != s.cfg.WebOrigin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "untrusted request origin"})
			return
		}
		c.Next()
	}
}

type authInput struct {
	Email       string `json:"email" binding:"required,email,max=254"`
	Password    string `json:"password" binding:"required,min=12,max=128"`
	DisplayName string `json:"displayName" binding:"omitempty,min=2,max=80"`
}

func (s *Server) register(c *gin.Context) {
	var in authInput
	if err := c.ShouldBindJSON(&in); err != nil {
		bad(c, err)
		return
	}
	h, err := auth.HashPassword(in.Password)
	if err != nil {
		bad(c, err)
		return
	}
	u, err := s.db.CreateUser(c, in.Email, in.DisplayName, h)
	if errors.Is(err, store.ErrConflict) {
		c.JSON(409, gin.H{"error": "an account already exists"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	s.issue(c, u, http.StatusCreated)
}
func (s *Server) login(c *gin.Context) {
	var in authInput
	if err := c.ShouldBindJSON(&in); err != nil {
		bad(c, err)
		return
	}
	u, err := s.db.UserByEmail(c, in.Email)
	if err != nil || auth.CheckPassword(u.PasswordHash, in.Password) != nil {
		c.JSON(401, gin.H{"error": "invalid email or password"})
		return
	}
	s.issue(c, u, 200)
}
func (s *Server) issue(c *gin.Context, u domain.User, status int) {
	access, refresh, expires, err := s.auth.NewPair(c, u.ID, u.Role)
	if err != nil {
		fail(c, s.log, err)
		return
	}
	s.setRefresh(c, refresh, expires)
	c.JSON(status, gin.H{"accessToken": access, "expiresIn": int(s.cfg.AccessTokenTTL.Seconds()), "user": gin.H{"id": u.ID, "email": u.Email, "displayName": u.DisplayName, "role": u.Role}})
}
func (s *Server) refresh(c *gin.Context) {
	token, err := c.Cookie("stocker_refresh")
	if err != nil {
		c.JSON(401, gin.H{"error": "refresh session missing"})
		return
	}
	access, fresh, expires, err := s.auth.Rotate(c, token)
	if err != nil {
		c.JSON(401, gin.H{"error": "refresh session invalid"})
		return
	}
	s.setRefresh(c, fresh, expires)
	c.JSON(200, gin.H{"accessToken": access, "expiresIn": int(s.cfg.AccessTokenTTL.Seconds())})
}
func (s *Server) logout(c *gin.Context) {
	if token, err := c.Cookie("stocker_refresh"); err == nil {
		_ = s.auth.Revoke(c, token)
	}
	c.SetCookie("stocker_refresh", "", -1, "/api/v1/auth", "", s.cfg.CookieSecure, true)
	c.Status(204)
}
func (s *Server) me(c *gin.Context) {
	u, err := s.db.UserByID(c, c.GetString(string(userIDKey)))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user no longer exists"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": u.ID, "email": u.Email, "displayName": u.DisplayName, "role": u.Role}})
}
func (s *Server) setRefresh(c *gin.Context, token string, expires time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{Name: "stocker_refresh", Value: token, Path: "/api/v1/auth", Expires: expires, MaxAge: int(time.Until(expires).Seconds()), HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode})
}

func (s *Server) search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 1 {
		c.JSON(200, gin.H{"data": []any{}, "meta": gin.H{"query": q}})
		return
	}
	var items []domain.Security
	var err error
	if s.market != nil {
		items, err = s.market.Search(c, q)
	} else {
		items, err = s.db.SearchSecurities(c, q, 20)
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "meta": gin.H{"query": q, "count": len(items)}})
}
func (s *Server) stock(c *gin.Context) {
	if s.market != nil {
		s.market.RefreshBestEffort(c, c.Param("symbol"))
	}
	item, err := s.db.StockIntelligenceBySymbol(c, c.Param("symbol"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(404, gin.H{"error": "security not found"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(200, gin.H{"data": item, "meta": gin.H{"provider": s.marketProviderName(), "disclaimer": "Informational research only — not financial advice."}})
}
func (s *Server) overview(c *gin.Context) {
	v, err := s.db.Overview(c)
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(200, gin.H{"data": v, "meta": gin.H{"delivery": "Provider timestamps and delay labels are authoritative; never assume tick-level real time.", "provider": s.marketProviderName()}})
}

func (s *Server) marketSectors(c *gin.Context) {
	items, err := s.db.SectorSnapshots(c)
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"count": len(items), "computedFrom": "latest persisted quote and signal per security"}})
}

func (s *Server) marketMovers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	items, err := s.db.MarketMovers(c, limit)
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"count": len(items), "provider": s.marketProviderName()}})
}

func (s *Server) marketEvents(c *gin.Context) {
	from := time.Now().UTC().AddDate(0, -3, 0)
	if value := c.Query("from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			bad(c, errors.New("from must be an RFC3339 timestamp"))
			return
		}
		from = parsed
	}
	items, err := s.db.MarketEvents(c, c.Query("symbol"), from, 50)
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"count": len(items), "derivedFrom": "persisted, analyzed source evidence"}})
}

func (s *Server) stockQuote(c *gin.Context) {
	if s.market != nil {
		s.market.RefreshBestEffort(c, c.Param("symbol"))
	}
	quote, err := s.db.MarketQuoteBySymbol(c, c.Param("symbol"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "market quote not available"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quote, "meta": gin.H{"provider": s.marketProviderName(), "freshness": "inspect asOf and isDelayed"}})
}

func (s *Server) stockFundamentals(c *gin.Context) {
	if s.market != nil {
		s.market.RefreshBestEffort(c, c.Param("symbol"))
	}
	items, err := s.db.FundamentalsBySymbol(c, c.Param("symbol"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "fundamentals not available"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"periodCompatibility": "each metric carries its own period, unit, and basis"}})
}

func (s *Server) stockPeers(c *gin.Context) {
	items, err := s.db.PeersBySymbol(c, c.Param("symbol"), 5)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "security not found"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"basis": "same-sector securities with available snapshots"}})
}

func (s *Server) stockRiskFlags(c *gin.Context) {
	items, err := s.db.RiskFlagsBySymbol(c, c.Param("symbol"))
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"deterministic": true, "disclaimer": "Risk flags are descriptive checks, not investment advice."}})
}

func (s *Server) marketProviderName() string {
	if s.market == nil {
		return "unavailable"
	}
	return s.market.ProviderName()
}

func (s *Server) watchlist(c *gin.Context) {
	items, err := s.db.Watchlist(c, c.GetString(string(userIDKey)))
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "meta": gin.H{"count": len(items), "limit": 10}})
}
func (s *Server) addWatchlist(c *gin.Context) {
	var in struct {
		Symbol string `json:"symbol" binding:"required,max=24"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		bad(c, err)
		return
	}
	if s.market != nil {
		s.market.RefreshBestEffort(c, in.Symbol)
	}
	item, err := s.db.AddWatchlist(c, c.GetString(string(userIDKey)), in.Symbol)
	if errors.Is(err, store.ErrWatchlistLimit) {
		c.JSON(422, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, store.ErrConflict) {
		c.JSON(409, gin.H{"error": "stock already on watchlist"})
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(404, gin.H{"error": "security not found"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(201, gin.H{"data": item})
}
func (s *Server) deleteWatchlist(c *gin.Context) {
	err := s.db.DeleteWatchlist(c, c.GetString(string(userIDKey)), c.Param("symbol"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(404, gin.H{"error": "watchlist item not found"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.Status(204)
}
func (s *Server) pauseWatchlist(c *gin.Context) {
	var in struct {
		AlertsPaused bool `json:"alertsPaused"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		bad(c, err)
		return
	}
	if err := s.db.PauseWatchlist(c, c.GetString(string(userIDKey)), c.Param("symbol"), in.AlertsPaused); err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"symbol": c.Param("symbol"), "alertsPaused": in.AlertsPaused}})
}

func (s *Server) stream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	ch, done := s.hub.Subscribe()
	defer done()
	c.SSEvent("connected", gin.H{"at": time.Now().UTC(), "freshness": "near-real-time subject to provider delay"})
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case b, ok := <-ch:
			if !ok {
				return
			}
			c.Writer.Write([]byte("event: update\ndata: " + string(b) + "\n\n"))
			c.Writer.Flush()
		case t := <-ticker.C:
			c.SSEvent("heartbeat", gin.H{"at": t.UTC()})
			c.Writer.Flush()
		}
	}
}

func (s *Server) bridgeNewsEvents(ctx context.Context) {
	lastNewsID, lastAnalysisID := "$", "$"
	for ctx.Err() == nil {
		streams, err := s.redis.XRead(ctx, &redis.XReadArgs{Streams: []string{"stocker:news.created", "stocker:analysis.completed", lastNewsID, lastAnalysisID}, Count: 100, Block: 5 * time.Second}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			s.log.Debug("news_event_bridge_unavailable", "error", err)
			timer := time.NewTimer(2 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			continue
		}
		for _, stream := range streams {
			for _, message := range stream.Messages {
				eventName := "news.created"
				if stream.Stream == "stocker:analysis.completed" {
					lastAnalysisID, eventName = message.ID, "analysis.completed"
				} else {
					lastNewsID = message.ID
				}
				payload, ok := message.Values["payload"].(string)
				if !ok {
					continue
				}
				var event any
				if json.Unmarshal([]byte(payload), &event) == nil {
					s.hub.Publish(eventName, event)
				}
			}
		}
	}
}
func (s *Server) news(c *gin.Context) {
	s.listNews(c, "")
}
func (s *Server) stockNews(c *gin.Context) {
	s.listNews(c, c.Param("symbol"))
}
func (s *Server) listNews(c *gin.Context, stock string) {
	filter, err := parseNewsFilter(c, stock)
	if err != nil {
		bad(c, err)
		return
	}
	page, err := s.db.ListNews(c, filter)
	if err != nil {
		fail(c, s.log, err)
		return
	}
	allSynthetic := len(page.Items) > 0
	for _, article := range page.Items {
		allSynthetic = allSynthetic && article.Synthetic
	}
	c.JSON(200, gin.H{"data": page.Items, "meta": gin.H{
		"page": page.Page, "pageSize": page.PageSize, "total": page.Total, "totalPages": page.TotalPages,
		"clustered": true, "syntheticOnly": allSynthetic,
	}})
}
func (s *Server) newsDetail(c *gin.Context) {
	article, err := s.db.NewsByID(c, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(404, gin.H{"error": "news item not found"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(200, gin.H{"data": article, "meta": gin.H{
		"evidenceStatus": "collected source item; analysis is available separately when complete",
		"disclaimer":     "Informational research only — not financial advice.",
	}})
}
func (s *Server) newsAnalysis(c *gin.Context) {
	output, err := s.db.AnalysisByArticle(c, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "analysis not available"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": output, "meta": gin.H{"appendOnly": true, "disclaimer": "Probabilistic research output, not financial advice."}})
}
func (s *Server) stockSignals(c *gin.Context) {
	signals, err := s.db.SignalsBySymbol(c, c.Param("symbol"), 20)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "security not found"})
		return
	}
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": signals, "meta": gin.H{"appendOnly": true, "count": len(signals), "disclaimer": "Probabilistic research output, not financial advice."}})
}
func (s *Server) sourceHealth(c *gin.Context) {
	health, err := s.db.SourceHealth(c)
	if err != nil {
		fail(c, s.log, err)
		return
	}
	c.JSON(200, gin.H{"data": health, "meta": gin.H{"persisted": true}})
}

func parseNewsFilter(c *gin.Context, stock string) (domain.NewsFilter, error) {
	filter := domain.NewsFilter{
		Query: strings.TrimSpace(c.Query("q")), Stock: strings.TrimSpace(stock),
		Sector: strings.TrimSpace(c.Query("sector")), Source: strings.TrimSpace(c.Query("source")),
		Language: strings.TrimSpace(c.Query("language")), Page: 1, PageSize: 20,
	}
	if stock != "" {
		filter.PageSize = 10
	}
	if filter.Stock == "" {
		filter.Stock = strings.TrimSpace(c.Query("stock"))
	}
	if len(filter.Query) > 200 || len(filter.Stock) > 32 || len(filter.Sector) > 100 || len(filter.Source) > 100 || len(filter.Language) > 16 {
		return filter, errors.New("one or more news filters are too long")
	}
	var err error
	if value := c.Query("officialOnly"); value != "" {
		filter.OfficialOnly, err = strconv.ParseBool(value)
		if err != nil {
			return filter, errors.New("officialOnly must be true or false")
		}
	}
	if value := c.Query("page"); value != "" {
		filter.Page, err = strconv.Atoi(value)
		if err != nil || filter.Page < 1 {
			return filter, errors.New("page must be a positive integer")
		}
	}
	if value := c.Query("pageSize"); value != "" {
		filter.PageSize, err = strconv.Atoi(value)
		if err != nil || filter.PageSize < 1 || filter.PageSize > 50 {
			return filter, errors.New("pageSize must be between 1 and 50")
		}
	}
	for key, target := range map[string]**time.Time{"from": &filter.From, "to": &filter.To} {
		if value := c.Query(key); value != "" {
			parsed, parseErr := time.Parse(time.RFC3339, value)
			if parseErr != nil {
				return filter, fmt.Errorf("%s must be an RFC3339 timestamp", key)
			}
			*target = &parsed
		}
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return filter, errors.New("from must not be after to")
	}
	return filter, nil
}
func (s *Server) alerts(c *gin.Context) {
	c.JSON(200, gin.H{"data": []any{}, "meta": gin.H{"deduplication": "cluster+rule+cooldown"}})
}
func (s *Server) briefings(c *gin.Context) {
	c.JSON(200, gin.H{"data": []any{}, "meta": gin.H{"message": "Briefings are generated only from collected evidence."}})
}
func (s *Server) notImplemented(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented", "detail": message})
	}
}
func bad(c *gin.Context, err error) {
	c.JSON(400, gin.H{"error": "invalid request", "detail": err.Error()})
}
func fail(c *gin.Context, log *slog.Logger, err error) {
	log.Error("request_failed", "error", err)
	c.JSON(500, gin.H{"error": "internal server error"})
}

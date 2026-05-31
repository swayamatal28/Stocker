package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment                  string
	HTTPAddr                     string
	MongoDBURI                   string
	MongoDBDatabase              string
	RedisURL                     string
	RedisRequired                bool
	JWTSecret                    string
	AccessTokenTTL               time.Duration
	RefreshTokenTTL              time.Duration
	WebOrigin                    string
	CookieSecure                 bool
	MockProviders                bool
	RawRetention                 int
	IngestSourceID               string
	IngestSourcesJSON            string
	IngestSourcesFile            string
	IngestFeedURL                string
	IngestAttribution            string
	IngestLicence                string
	IngestTermsURL               string
	IngestLanguage               string
	IngestOfficial               bool
	IngestAutomatedAccessAllowed bool
	IngestRequestsPerMinute      int
	IngestPollInterval           time.Duration
	IngestTimeout                time.Duration
	IngestPolicyExpiresAt        *time.Time
	IngestReplayDeadOnStart      bool
	AIProvider                   string
	AIEndpoint                   string
	AIAPIKey                     string
	AIModel                      string
	AIPromptVersion              string
	AISchemaVersion              string
	AIRequestTimeout             time.Duration
	AIMaxInputChars              int
	AIMaxOutputBytes             int
	AIRequestCostCents           int
	AIDailyBudgetCents           int
	AnalysisConsumer             string
	AnalysisMaxAttempts          int
	MarketProvider               string
	MarketEndpoint               string
	MarketAllowInsecureHTTP      bool
	MarketRequestTimeout         time.Duration
	MarketRefreshInterval        time.Duration
	AlertEvaluationInterval      time.Duration
	MaintenanceInterval          time.Duration
	TransientRetention           time.Duration
	AlertRetentionDays           int
	BriefingRetentionDays        int
	AuditRetentionDays           int
	OTELExporterEndpoint         string
}

func Load() (Config, error) {
	c := Config{
		Environment:                  env("APP_ENV", "development"),
		HTTPAddr:                     env("HTTP_ADDR", ":8080"),
		MongoDBURI:                   env("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDBDatabase:              env("MONGODB_DATABASE", "stocker"),
		RedisURL:                     env("REDIS_URL", "redis://localhost:6379/0"),
		RedisRequired:                envBool("REDIS_REQUIRED", false),
		JWTSecret:                    env("JWT_SECRET", "development-only-secret-change-me-now"),
		WebOrigin:                    env("WEB_ORIGIN", "http://localhost:5173"),
		CookieSecure:                 envBool("COOKIE_SECURE", false),
		MockProviders:                envBool("MOCK_PROVIDERS", false),
		RawRetention:                 envInt("RAW_RETENTION_DAYS", 7),
		IngestSourceID:               env("INGEST_SOURCE_ID", ""),
		IngestSourcesJSON:            env("INGEST_SOURCES_JSON", ""),
		IngestSourcesFile:            env("INGEST_SOURCES_FILE", ""),
		IngestFeedURL:                env("INGEST_FEED_URL", ""),
		IngestAttribution:            env("INGEST_ATTRIBUTION", ""),
		IngestLicence:                env("INGEST_LICENCE", ""),
		IngestTermsURL:               env("INGEST_TERMS_URL", ""),
		IngestLanguage:               env("INGEST_LANGUAGE", "en"),
		IngestOfficial:               envBool("INGEST_OFFICIAL", false),
		IngestAutomatedAccessAllowed: envBool("INGEST_AUTOMATED_ACCESS_ALLOWED", false),
		IngestRequestsPerMinute:      envInt("INGEST_REQUESTS_PER_MINUTE", 10),
		IngestReplayDeadOnStart:      envBool("INGEST_REPLAY_DEAD_ON_START", false),
		AIProvider:                   env("AI_PROVIDER", "local-deterministic"),
		AIEndpoint:                   env("AI_ENDPOINT", ""),
		AIAPIKey:                     env("AI_API_KEY", ""),
		AIModel:                      env("AI_MODEL", "stocker-grounded-rules-v1"),
		AIPromptVersion:              env("AI_PROMPT_VERSION", "analysis-v1"),
		AISchemaVersion:              env("AI_SCHEMA_VERSION", "ai-analysis-v1"),
		AIMaxInputChars:              envInt("AI_MAX_INPUT_CHARS", 24000),
		AIMaxOutputBytes:             envInt("AI_MAX_OUTPUT_BYTES", 65536),
		AIRequestCostCents:           envInt("AI_REQUEST_COST_CENTS", 0),
		AIDailyBudgetCents:           envInt("AI_DAILY_BUDGET_CENTS", 100),
		AnalysisConsumer:             env("ANALYSIS_CONSUMER", "analysis-local-1"),
		AnalysisMaxAttempts:          envInt("ANALYSIS_MAX_ATTEMPTS", 5),
		MarketProvider:               env("MARKET_PROVIDER", "yahoo-finance"),
		MarketEndpoint:               env("MARKET_ENDPOINT", "https://query1.finance.yahoo.com"),
		MarketAllowInsecureHTTP:      envBool("MARKET_ALLOW_INSECURE_HTTP", false),
		AlertRetentionDays:           envInt("ALERT_RETENTION_DAYS", 365),
		BriefingRetentionDays:        envInt("BRIEFING_RETENTION_DAYS", 90),
		AuditRetentionDays:           envInt("AUDIT_RETENTION_DAYS", 730),
		OTELExporterEndpoint:         env("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}
	if strings.TrimSpace(c.IngestSourcesJSON) == "" && strings.TrimSpace(c.IngestSourcesFile) != "" {
		contents, readErr := os.ReadFile(c.IngestSourcesFile)
		if readErr != nil {
			return c, fmt.Errorf("INGEST_SOURCES_FILE: %w", readErr)
		}
		if len(contents) > 1<<20 {
			return c, fmt.Errorf("INGEST_SOURCES_FILE exceeds 1 MiB")
		}
		c.IngestSourcesJSON = string(contents)
	}
	if c.MockProviders {
		c.IngestAutomatedAccessAllowed = true
	} else {
		if strings.TrimSpace(c.IngestSourcesJSON) == "" {
			for _, key := range []string{"INGEST_SOURCE_ID", "INGEST_FEED_URL", "INGEST_ATTRIBUTION", "INGEST_LICENCE", "INGEST_TERMS_URL"} {
				if os.Getenv(key) == "" {
					return c, fmt.Errorf("%s is required when MOCK_PROVIDERS=false", key)
				}
			}
		}
		if strings.TrimSpace(c.IngestSourcesJSON) == "" && !c.IngestAutomatedAccessAllowed {
			return c, fmt.Errorf("INGEST_AUTOMATED_ACCESS_ALLOWED must be explicitly true when MOCK_PROVIDERS=false")
		}
	}
	var err error
	if c.AccessTokenTTL, err = time.ParseDuration(env("ACCESS_TOKEN_TTL", "15m")); err != nil {
		return c, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	if c.RefreshTokenTTL, err = time.ParseDuration(env("REFRESH_TOKEN_TTL", "720h")); err != nil {
		return c, fmt.Errorf("REFRESH_TOKEN_TTL: %w", err)
	}
	if c.IngestPollInterval, err = time.ParseDuration(env("INGEST_POLL_INTERVAL", "1m")); err != nil {
		return c, fmt.Errorf("INGEST_POLL_INTERVAL: %w", err)
	}
	if c.IngestTimeout, err = time.ParseDuration(env("INGEST_TIMEOUT", "15s")); err != nil {
		return c, fmt.Errorf("INGEST_TIMEOUT: %w", err)
	}
	if c.AIRequestTimeout, err = time.ParseDuration(env("AI_REQUEST_TIMEOUT", "30s")); err != nil {
		return c, fmt.Errorf("AI_REQUEST_TIMEOUT: %w", err)
	}
	if c.MarketRequestTimeout, err = time.ParseDuration(env("MARKET_REQUEST_TIMEOUT", "8s")); err != nil {
		return c, fmt.Errorf("MARKET_REQUEST_TIMEOUT: %w", err)
	}
	if c.MarketRefreshInterval, err = time.ParseDuration(env("MARKET_REFRESH_INTERVAL", "5m")); err != nil {
		return c, fmt.Errorf("MARKET_REFRESH_INTERVAL: %w", err)
	}
	if c.AlertEvaluationInterval, err = time.ParseDuration(env("ALERT_EVALUATION_INTERVAL", "1m")); err != nil {
		return c, fmt.Errorf("ALERT_EVALUATION_INTERVAL: %w", err)
	}
	if c.MaintenanceInterval, err = time.ParseDuration(env("MAINTENANCE_INTERVAL", "24h")); err != nil {
		return c, fmt.Errorf("MAINTENANCE_INTERVAL: %w", err)
	}
	if c.TransientRetention, err = time.ParseDuration(env("TRANSIENT_RETENTION", "24h")); err != nil {
		return c, fmt.Errorf("TRANSIENT_RETENTION: %w", err)
	}
	if c.AlertEvaluationInterval < 10*time.Second {
		return c, fmt.Errorf("ALERT_EVALUATION_INTERVAL must be at least 10s")
	}
	if c.MaintenanceInterval < time.Hour || c.TransientRetention < time.Hour || c.AlertRetentionDays < 1 || c.BriefingRetentionDays < 1 || c.AuditRetentionDays < 1 {
		return c, fmt.Errorf("maintenance interval and retention periods are invalid")
	}
	if value := os.Getenv("INGEST_POLICY_EXPIRES_AT"); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return c, fmt.Errorf("INGEST_POLICY_EXPIRES_AT: %w", parseErr)
		}
		c.IngestPolicyExpiresAt = &parsed
	}
	if c.Environment == "production" && len(c.JWTSecret) < 32 {
		return c, fmt.Errorf("JWT_SECRET must contain at least 32 characters in production")
	}
	if c.Environment == "production" {
		mongoSecure := strings.HasPrefix(strings.ToLower(c.MongoDBURI), "mongodb+srv://") || strings.Contains(strings.ToLower(c.MongoDBURI), "tls=true")
		if !mongoSecure {
			return c, fmt.Errorf("MONGODB_URI must enable TLS in production")
		}
		if !strings.HasPrefix(strings.ToLower(c.RedisURL), "rediss://") {
			return c, fmt.Errorf("REDIS_URL must use TLS (rediss://) in production")
		}
		if !c.RedisRequired {
			return c, fmt.Errorf("REDIS_REQUIRED must be true in production")
		}
		if !c.CookieSecure {
			return c, fmt.Errorf("COOKIE_SECURE must be true in production")
		}
		if strings.TrimSpace(c.OTELExporterEndpoint) == "" {
			return c, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required in production")
		}
	}
	if c.OTELExporterEndpoint != "" {
		endpoint, parseErr := url.Parse(c.OTELExporterEndpoint)
		if parseErr != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
			return c, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT must be an absolute HTTP(S) URL")
		}
		if c.Environment == "production" && endpoint.Scheme != "https" {
			return c, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT must use HTTPS in production")
		}
	}
	if c.AIProvider != "local-deterministic" && c.AIProvider != "http-json" {
		return c, fmt.Errorf("AI_PROVIDER must be local-deterministic or http-json")
	}
	if c.AIProvider == "http-json" && (c.AIEndpoint == "" || c.AIModel == "") {
		return c, fmt.Errorf("AI_ENDPOINT and AI_MODEL are required when AI_PROVIDER=http-json")
	}
	if c.AIProvider == "http-json" {
		endpoint, parseErr := url.Parse(c.AIEndpoint)
		if parseErr != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
			return c, fmt.Errorf("AI_ENDPOINT must be an absolute HTTP(S) URL")
		}
		if c.Environment == "production" && endpoint.Scheme != "https" && endpoint.Hostname() != "localhost" && endpoint.Hostname() != "127.0.0.1" {
			return c, fmt.Errorf("AI_ENDPOINT must use HTTPS outside the local machine in production")
		}
	}
	if c.AIMaxInputChars < 1000 || c.AIMaxOutputBytes < 1024 || c.AIDailyBudgetCents < 0 || c.AIRequestCostCents < 0 || c.AnalysisMaxAttempts < 1 {
		return c, fmt.Errorf("AI limits and budgets are invalid")
	}
	if c.MarketProvider != "fixture" && c.MarketProvider != "indian-stock-api" && c.MarketProvider != "yahoo-finance" {
		return c, fmt.Errorf("MARKET_PROVIDER must be fixture, indian-stock-api, or yahoo-finance")
	}
	if c.MarketRequestTimeout <= 0 || c.MarketRefreshInterval < time.Minute {
		return c, fmt.Errorf("market timeout and refresh interval are invalid")
	}
	if c.MarketProvider == "indian-stock-api" {
		endpoint, parseErr := url.Parse(c.MarketEndpoint)
		if parseErr != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
			return c, fmt.Errorf("MARKET_ENDPOINT must be an absolute HTTP(S) URL")
		}
		if endpoint.Scheme == "http" && !c.MarketAllowInsecureHTTP {
			return c, fmt.Errorf("MARKET_ALLOW_INSECURE_HTTP must be explicitly true for a plaintext market endpoint")
		}
	}
	if c.MarketProvider == "yahoo-finance" {
		endpoint, parseErr := url.Parse(c.MarketEndpoint)
		if parseErr != nil || endpoint.Host == "" || endpoint.Scheme != "https" {
			return c, fmt.Errorf("MARKET_ENDPOINT must be an absolute HTTPS URL for yahoo-finance")
		}
	}
	return c, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func envBool(k string, fallback bool) bool {
	v, err := strconv.ParseBool(env(k, strconv.FormatBool(fallback)))
	if err != nil {
		return fallback
	}
	return v
}
func envInt(k string, fallback int) int {
	v, err := strconv.Atoi(env(k, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return v
}

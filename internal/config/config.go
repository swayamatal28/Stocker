package config

import (
	"fmt"
	"os"
	"strconv"
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
		MockProviders:                envBool("MOCK_PROVIDERS", true),
		RawRetention:                 envInt("RAW_RETENTION_DAYS", 7),
		IngestSourceID:               env("INGEST_SOURCE_ID", "mock-exchange"),
		IngestFeedURL:                env("INGEST_FEED_URL", ""),
		IngestAttribution:            env("INGEST_ATTRIBUTION", "Synthetic STOCKER fixture"),
		IngestLicence:                env("INGEST_LICENCE", "Project-owned test fixture"),
		IngestTermsURL:               env("INGEST_TERMS_URL", "https://example.invalid/project-owned-fixture"),
		IngestLanguage:               env("INGEST_LANGUAGE", "en"),
		IngestOfficial:               envBool("INGEST_OFFICIAL", false),
		IngestAutomatedAccessAllowed: envBool("INGEST_AUTOMATED_ACCESS_ALLOWED", false),
		IngestRequestsPerMinute:      envInt("INGEST_REQUESTS_PER_MINUTE", 10),
		IngestReplayDeadOnStart:      envBool("INGEST_REPLAY_DEAD_ON_START", false),
	}
	if c.MockProviders {
		c.IngestAutomatedAccessAllowed = true
	} else {
		for _, key := range []string{"INGEST_SOURCE_ID", "INGEST_FEED_URL", "INGEST_ATTRIBUTION", "INGEST_LICENCE", "INGEST_TERMS_URL"} {
			if os.Getenv(key) == "" {
				return c, fmt.Errorf("%s is required when MOCK_PROVIDERS=false", key)
			}
		}
		if !c.IngestAutomatedAccessAllowed {
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

package ingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type FeedSourceDefinition struct {
	ID                     string `json:"id"`
	Enabled                bool   `json:"enabled"`
	FeedURL                string `json:"feedUrl"`
	Attribution            string `json:"attribution"`
	Licence                string `json:"licence"`
	TermsURL               string `json:"termsUrl"`
	Language               string `json:"language"`
	Official               bool   `json:"official"`
	AutomatedAccessAllowed bool   `json:"automatedAccessAllowed"`
	RobotsChecked          bool   `json:"robotsChecked"`
	RequestsPerMinute      int    `json:"requestsPerMinute"`
	PollInterval           string `json:"pollInterval"`
	Timeout                string `json:"timeout"`
	RawRetentionDays       int    `json:"rawRetentionDays"`
	PolicyExpiresAt        string `json:"policyExpiresAt"`
}

type ConfiguredFeed struct {
	Adapter Adapter
	Policy  SourcePolicy
}

func ParseFeedSources(raw string) ([]ConfiguredFeed, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var definitions []FeedSourceDefinition
	if err := decoder.Decode(&definitions); err != nil {
		return nil, fmt.Errorf("parse INGEST_SOURCES_JSON: %w", err)
	}
	seen := map[string]struct{}{}
	feeds := []ConfiguredFeed{}
	for _, definition := range definitions {
		if !definition.Enabled {
			continue
		}
		if _, exists := seen[definition.ID]; exists {
			return nil, fmt.Errorf("duplicate source ID %q", definition.ID)
		}
		seen[definition.ID] = struct{}{}
		if !definition.RobotsChecked {
			return nil, fmt.Errorf("source %s robots policy must be checked before enabling", definition.ID)
		}
		pollInterval, err := time.ParseDuration(definition.PollInterval)
		if err != nil {
			return nil, fmt.Errorf("source %s pollInterval: %w", definition.ID, err)
		}
		timeout, err := time.ParseDuration(definition.Timeout)
		if err != nil {
			return nil, fmt.Errorf("source %s timeout: %w", definition.ID, err)
		}
		var expiry *time.Time
		if definition.PolicyExpiresAt != "" {
			parsed, parseErr := time.Parse(time.RFC3339, definition.PolicyExpiresAt)
			if parseErr != nil {
				return nil, fmt.Errorf("source %s policyExpiresAt: %w", definition.ID, parseErr)
			}
			expiry = &parsed
		}
		policy := SourcePolicy{
			PollInterval: pollInterval, Timeout: timeout, RequestsPerMinute: definition.RequestsPerMinute,
			RawRetention: time.Duration(definition.RawRetentionDays) * 24 * time.Hour,
			Attribution:  definition.Attribution, Licence: definition.Licence, TermsURL: definition.TermsURL,
			AutomatedAccessAllowed: definition.AutomatedAccessAllowed, RobotsChecked: definition.RobotsChecked, PolicyExpiresAt: expiry,
		}
		if err := policy.Validate(time.Now().UTC()); err != nil {
			return nil, fmt.Errorf("source %s policy: %w", definition.ID, err)
		}
		adapter, err := NewRSSAdapter(RSSAdapterConfig{SourceID: definition.ID, FeedURL: definition.FeedURL, Attribution: definition.Attribution, Licence: definition.Licence, Language: definition.Language, Official: definition.Official})
		if err != nil {
			return nil, fmt.Errorf("source %s adapter: %w", definition.ID, err)
		}
		feeds = append(feeds, ConfiguredFeed{Adapter: adapter, Policy: policy})
	}
	if len(feeds) == 0 {
		return nil, errors.New("INGEST_SOURCES_JSON has no enabled, approved sources")
	}
	return feeds, nil
}

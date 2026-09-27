package alerts

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/market"
	"github.com/stocker-app/stocker/internal/store"
)

var clockPattern = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d$`)

type Notifier func(userID string, event domain.AlertEvent)
type Refresher func(context.Context, string)

type Service struct {
	db       *store.Mongo
	log      *slog.Logger
	notify   Notifier
	refresh  Refresher
	interval time.Duration
}

func NewService(db *store.Mongo, log *slog.Logger, notify Notifier, refresh Refresher, interval time.Duration) *Service {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Service{db: db, log: log, notify: notify, refresh: refresh, interval: interval}
}

func NormalizeAndValidate(rule domain.AlertRule) (domain.AlertRule, error) {
	rule.Name = strings.TrimSpace(rule.Name)
	if len(rule.Name) < 3 || len(rule.Name) > 80 {
		return rule, errors.New("name must contain 3 to 80 characters")
	}
	symbol, ok := market.CleanSymbol(rule.Symbol)
	if !ok {
		return rule, errors.New("symbol is invalid")
	}
	rule.Symbol = symbol
	rule.RuleType = strings.ToLower(strings.TrimSpace(rule.RuleType))
	validTypes := map[string]bool{"event": true, "price_above": true, "price_below": true, "change_above": true, "change_below": true}
	if !validTypes[rule.RuleType] {
		return rule, errors.New("ruleType must be event, price_above, price_below, change_above, or change_below")
	}
	if rule.RuleType == "event" {
		rule.Threshold = nil
		if len(rule.EventCategories) > 10 {
			return rule, errors.New("eventCategories cannot contain more than 10 values")
		}
		seen := map[string]bool{}
		categories := []string{}
		for _, category := range rule.EventCategories {
			category = strings.ToLower(strings.TrimSpace(category))
			if category == "" || len(category) > 50 || seen[category] {
				continue
			}
			seen[category] = true
			categories = append(categories, category)
		}
		sort.Strings(categories)
		rule.EventCategories = categories
	} else {
		if rule.Threshold == nil || *rule.Threshold <= 0 || math.IsNaN(*rule.Threshold) || math.IsInf(*rule.Threshold, 0) {
			return rule, errors.New("a positive finite threshold is required for threshold rules")
		}
		if strings.HasPrefix(rule.RuleType, "change_") && *rule.Threshold > 100 {
			return rule, errors.New("percentage threshold cannot exceed 100")
		}
		rule.EventCategories = nil
	}
	if rule.MinimumConfidence < 0 || rule.MinimumConfidence > 100 {
		return rule, errors.New("minimumConfidence must be between 0 and 100")
	}
	rule.MinimumSeverity = strings.ToLower(strings.TrimSpace(rule.MinimumSeverity))
	if rule.MinimumSeverity == "" {
		rule.MinimumSeverity = "low"
	}
	if severityRank(rule.MinimumSeverity) == 0 {
		return rule, errors.New("minimumSeverity must be low, medium, or high")
	}
	if rule.CooldownMinutes == 0 {
		rule.CooldownMinutes = 60
	}
	if rule.CooldownMinutes < 1 || rule.CooldownMinutes > 10080 {
		return rule, errors.New("cooldownMinutes must be between 1 and 10080")
	}
	if !rule.Channels.InApp {
		return rule, errors.New("in-app delivery must remain enabled until another consented provider is configured")
	}
	if rule.QuietHours.Enabled {
		if !clockPattern.MatchString(rule.QuietHours.Start) || !clockPattern.MatchString(rule.QuietHours.End) || rule.QuietHours.Start == rule.QuietHours.End {
			return rule, errors.New("quiet hours require different HH:MM start and end values")
		}
		if rule.QuietHours.Timezone == "" {
			rule.QuietHours.Timezone = "Asia/Kolkata"
		}
		if _, err := time.LoadLocation(rule.QuietHours.Timezone); err != nil {
			return rule, errors.New("quiet-hours timezone is invalid")
		}
	} else {
		rule.QuietHours = domain.QuietHours{Enabled: false, Timezone: "Asia/Kolkata"}
	}
	return rule, nil
}

func (service *Service) EvaluateUser(ctx context.Context, userID string, now time.Time) ([]domain.AlertEvent, error) {
	rules, err := service.db.AlertRules(ctx, userID)
	if err != nil {
		return nil, err
	}
	return service.evaluateRules(ctx, rules, now)
}

func (service *Service) EvaluateAll(ctx context.Context, now time.Time) ([]domain.AlertEvent, error) {
	rules, err := service.db.ActiveAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	return service.evaluateRules(ctx, rules, now)
}

func (service *Service) evaluateRules(ctx context.Context, rules []domain.AlertRule, now time.Time) ([]domain.AlertEvent, error) {
	created := []domain.AlertEvent{}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if service.refresh != nil {
			service.refresh(ctx, rule.Symbol)
		}
		event, made, err := service.EvaluateRule(ctx, rule, now)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			return created, err
		}
		if made {
			created = append(created, event)
		}
	}
	return created, nil
}

func (service *Service) EvaluateRule(ctx context.Context, rule domain.AlertRule, now time.Time) (domain.AlertEvent, bool, error) {
	watchlisted, paused, err := service.db.WatchlistAlertState(ctx, rule.UserID, rule.Symbol)
	if err != nil || !watchlisted || paused || !rule.Enabled {
		return domain.AlertEvent{}, false, err
	}
	candidate, err := service.db.AlertCandidate(ctx, rule)
	if err != nil {
		return domain.AlertEvent{}, false, err
	}
	if !matches(rule, candidate) || candidate.Confidence < rule.MinimumConfidence || severityRank(candidate.Severity) < severityRank(rule.MinimumSeverity) {
		return domain.AlertEvent{}, false, nil
	}
	if len(candidate.Evidence) == 0 {
		return domain.AlertEvent{}, false, nil
	}
	dedupSource := candidate.ClusterID
	if dedupSource == "" {
		cooldown := time.Duration(rule.CooldownMinutes) * time.Minute
		inCooldown, cooldownErr := service.db.AlertRuleInCooldown(ctx, rule.UserID, rule.ID, now.UTC().Add(-cooldown))
		if cooldownErr != nil {
			return domain.AlertEvent{}, false, cooldownErr
		}
		if inCooldown {
			return domain.AlertEvent{}, false, nil
		}
		dedupSource = fmt.Sprintf("cooldown:%d", now.UTC().Truncate(cooldown).Unix())
	}
	digest := sha256.Sum256([]byte(rule.UserID + "\x00" + rule.ID + "\x00" + dedupSource))
	deliverAfter, quiet := quietHoursEnd(now, rule.QuietHours)
	status := "ready"
	if quiet {
		status = "deferred_quiet_hours"
	} else {
		deliverAfter = now.UTC()
	}
	external := map[string]string{}
	if rule.Channels.Browser {
		external["browser"] = "not_configured"
	}
	if rule.Channels.Email {
		external["email"] = "not_configured"
	}
	if rule.Channels.Telegram {
		external["telegram"] = "not_configured"
	}
	title := candidate.Title
	if candidate.Kind == "threshold" {
		title = fmt.Sprintf("%s: %s", rule.Symbol, thresholdLabel(rule))
	}
	event := domain.AlertEvent{UserID: rule.UserID, RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.RuleType, Symbol: rule.Symbol, Severity: candidate.Severity, Title: title, Explanation: candidate.Explanation, Confidence: candidate.Confidence, Evidence: candidate.Evidence, ConditionSnapshot: candidate.ConditionSnapshot, SourceAsOf: candidate.SourceAsOf, TriggeredAt: now.UTC(), DeliverAfter: deliverAfter.UTC(), DeliveryStatus: status, ExternalChannelStatus: external, DeduplicationKey: fmt.Sprintf("%x", digest[:])}
	saved, created, err := service.db.SaveAlertEvent(ctx, event)
	if err != nil || !created {
		return saved, false, err
	}
	if !quiet {
		saved, _, err = service.db.DeliverAlert(ctx, saved.ID, now.UTC())
		if err != nil {
			return saved, true, err
		}
		service.publish(saved)
	}
	return saved, true, nil
}

func (service *Service) DeliverDue(ctx context.Context, now time.Time) error {
	events, err := service.db.PendingAlerts(ctx, now.UTC(), 100)
	if err != nil {
		return err
	}
	for _, event := range events {
		rule, ruleErr := service.db.AlertRuleByID(ctx, event.UserID, event.RuleID)
		watchlisted, paused, stateErr := service.db.WatchlistAlertState(ctx, event.UserID, event.Symbol)
		if ruleErr != nil || stateErr != nil || !rule.Enabled || !watchlisted || paused {
			_ = service.db.CancelAlertDelivery(ctx, event.ID, "rule revoked, disabled, or watchlist alerts paused")
			continue
		}
		delivered, fresh, deliverErr := service.db.DeliverAlert(ctx, event.ID, now.UTC())
		if deliverErr != nil {
			return deliverErr
		}
		if fresh {
			service.publish(delivered)
		}
	}
	return nil
}

func (service *Service) Run(ctx context.Context) {
	service.runOnce(ctx)
	ticker := time.NewTicker(service.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			service.runOnce(runCtx, now)
			cancel()
		}
	}
}

func (service *Service) runOnce(ctx context.Context, values ...time.Time) {
	now := time.Now().UTC()
	if len(values) > 0 {
		now = values[0]
	}
	if _, err := service.EvaluateAll(ctx, now); err != nil && ctx.Err() == nil {
		service.log.Warn("alert_evaluation_failed", "error", err)
	}
	if err := service.DeliverDue(ctx, now); err != nil && ctx.Err() == nil {
		service.log.Warn("alert_delivery_failed", "error", err)
	}
}

func (service *Service) publish(event domain.AlertEvent) {
	if service.notify != nil {
		service.notify(event.UserID, event)
	}
}

func matches(rule domain.AlertRule, candidate domain.AlertCandidate) bool {
	if rule.RuleType == "event" {
		if len(rule.EventCategories) == 0 {
			return true
		}
		for _, category := range rule.EventCategories {
			if strings.EqualFold(category, candidate.Category) {
				return true
			}
		}
		return false
	}
	if rule.Threshold == nil {
		return false
	}
	switch rule.RuleType {
	case "price_above":
		return candidate.Price >= *rule.Threshold
	case "price_below":
		return candidate.Price <= *rule.Threshold
	case "change_above":
		return candidate.ChangePercent >= *rule.Threshold
	case "change_below":
		return candidate.ChangePercent <= -*rule.Threshold
	default:
		return false
	}
}

func thresholdLabel(rule domain.AlertRule) string {
	if rule.Threshold == nil {
		return "threshold reached"
	}
	switch rule.RuleType {
	case "price_above":
		return fmt.Sprintf("price at or above %.2f", *rule.Threshold)
	case "price_below":
		return fmt.Sprintf("price at or below %.2f", *rule.Threshold)
	case "change_above":
		return fmt.Sprintf("daily gain at or above %.2f%%", *rule.Threshold)
	case "change_below":
		return fmt.Sprintf("daily decline at or beyond %.2f%%", *rule.Threshold)
	default:
		return "threshold reached"
	}
}

func severityRank(value string) int {
	switch strings.ToLower(value) {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	default:
		return 0
	}
}

func quietHoursEnd(now time.Time, quiet domain.QuietHours) (time.Time, bool) {
	if !quiet.Enabled {
		return now.UTC(), false
	}
	location, err := time.LoadLocation(quiet.Timezone)
	if err != nil {
		return now.UTC(), false
	}
	local := now.In(location)
	start, startErr := time.Parse("15:04", quiet.Start)
	end, endErr := time.Parse("15:04", quiet.End)
	if startErr != nil || endErr != nil {
		return now.UTC(), false
	}
	minutes := local.Hour()*60 + local.Minute()
	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()
	inside := false
	endDay := local
	if startMinutes < endMinutes {
		inside = minutes >= startMinutes && minutes < endMinutes
	} else {
		inside = minutes >= startMinutes || minutes < endMinutes
		if minutes >= startMinutes {
			endDay = endDay.AddDate(0, 0, 1)
		}
	}
	if !inside {
		return now.UTC(), false
	}
	endAt := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), end.Hour(), end.Minute(), 0, 0, location)
	return endAt.UTC(), true
}

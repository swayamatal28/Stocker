package alerts

import (
	"testing"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

func TestNormalizeAndValidateDefaultsSafeDelivery(t *testing.T) {
	threshold := 2.5
	rule, err := NormalizeAndValidate(domain.AlertRule{Name: "Large daily gain", Symbol: " reliance.ns ", RuleType: "change_above", Threshold: &threshold, Channels: domain.AlertChannels{InApp: true}, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if rule.Symbol != "RELIANCE" || rule.MinimumSeverity != "low" || rule.CooldownMinutes != 60 || rule.QuietHours.Timezone != "Asia/Kolkata" {
		t.Fatalf("unexpected normalized rule: %#v", rule)
	}
}

func TestNormalizeAndValidateRejectsUnconsentedOnlyDelivery(t *testing.T) {
	_, err := NormalizeAndValidate(domain.AlertRule{Name: "Material event", Symbol: "OLAELEC", RuleType: "event", Channels: domain.AlertChannels{Email: true}, Enabled: true})
	if err == nil {
		t.Fatal("expected an alert without in-app delivery to fail closed")
	}
}

func TestQuietHoursAcrossMidnight(t *testing.T) {
	quiet := domain.QuietHours{Enabled: true, Start: "22:00", End: "07:00", Timezone: "Asia/Kolkata"}
	now := time.Date(2026, 9, 27, 18, 0, 0, 0, time.UTC) // 23:30 IST
	deliverAt, inside := quietHoursEnd(now, quiet)
	if !inside {
		t.Fatal("expected time to fall inside quiet hours")
	}
	expected := time.Date(2026, 9, 28, 1, 30, 0, 0, time.UTC)
	if !deliverAt.Equal(expected) {
		t.Fatalf("expected delivery at %s, got %s", expected, deliverAt)
	}
}

func TestThresholdMatching(t *testing.T) {
	threshold := 3.0
	rule := domain.AlertRule{RuleType: "change_below", Threshold: &threshold}
	if !matches(rule, domain.AlertCandidate{ChangePercent: -3.2}) || matches(rule, domain.AlertCandidate{ChangePercent: -2.9}) {
		t.Fatal("change_below must compare against the negative magnitude")
	}
}

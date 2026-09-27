package evaluation

import (
	"testing"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

func TestEvaluateRejectsLookAheadAndReportsSlices(t *testing.T) {
	decision := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	asOf := decision.Add(40 * 24 * time.Hour)
	cases := []domain.BacktestCase{
		{SignalID: "good", Symbol: "RELIANCE", Sector: "Energy", Category: "capacity_expansion", Horizon: "short term", Strength: 60, Confidence: 88, DecisionAt: decision, TargetAt: decision.Add(7 * 24 * time.Hour), EntryPrice: 100, ExitPrice: 110, ExitAsOf: decision.Add(8 * 24 * time.Hour), ExitAvailableAt: decision.Add(8 * 24 * time.Hour), FeatureTimes: map[string]time.Time{"article": decision.Add(-time.Hour), "entry_quote": decision.Add(-time.Minute)}},
		{SignalID: "leaked", Symbol: "INFY", Sector: "Information Technology", Category: "earnings", Horizon: "short term", Strength: -60, Confidence: 72, DecisionAt: decision, TargetAt: decision.Add(7 * 24 * time.Hour), EntryPrice: 100, ExitPrice: 90, ExitAsOf: decision.Add(8 * 24 * time.Hour), ExitAvailableAt: decision.Add(8 * 24 * time.Hour), FeatureTimes: map[string]time.Time{"revised_fundamental": decision.Add(time.Minute)}},
	}
	report := Evaluate(cases, asOf)
	if report.EvaluatedSignals != 1 || len(report.LeakageViolations) != 1 || report.Outcomes[0].ReturnPercent != 10 || !report.Outcomes[0].Correct {
		t.Fatalf("unexpected report: %#v", report)
	}
	foundBand := false
	for _, slice := range report.Slices {
		if slice.Dimension == "confidenceBand" && slice.Value == "85-100" && slice.Accuracy == 100 && slice.Precision == 100 {
			foundBand = true
		}
	}
	if !foundBand {
		t.Fatalf("confidence-band report missing: %#v", report.Slices)
	}
}

func TestEvaluateKeepsOutcomePendingUntilTargetAndAvailability(t *testing.T) {
	decision := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	report := Evaluate([]domain.BacktestCase{{SignalID: "pending", DecisionAt: decision, TargetAt: decision.Add(24 * time.Hour), EntryPrice: 100, ExitPrice: 105, ExitAsOf: decision.Add(time.Hour), ExitAvailableAt: decision.Add(time.Hour)}}, decision.Add(2*time.Hour))
	if report.PendingSignals != 1 || report.EvaluatedSignals != 0 {
		t.Fatalf("future outcome was evaluated early: %#v", report)
	}
}

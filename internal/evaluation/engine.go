package evaluation

import (
	"math"
	"sort"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

const Version = "event-time-v1"

func HorizonDuration(horizon string) time.Duration {
	switch horizon {
	case "intraday":
		return 8 * time.Hour
	case "short term":
		return 7 * 24 * time.Hour
	case "medium term":
		return 30 * 24 * time.Hour
	case "long term":
		return 90 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

func Evaluate(cases []domain.BacktestCase, asOf time.Time) domain.EvaluationReport {
	report := domain.EvaluationReport{Version: Version, AsOf: asOf.UTC(), CandidateSignals: len(cases), Outcomes: []domain.SignalOutcome{}, LeakageViolations: []domain.LeakageViolation{}, Slices: []domain.EvaluationSlice{}}
	for _, candidate := range cases {
		violations := detectLeakage(candidate)
		if len(violations) > 0 {
			report.LeakageViolations = append(report.LeakageViolations, violations...)
			continue
		}
		if candidate.EntryPrice <= 0 || candidate.ExitPrice <= 0 || candidate.ExitAsOf.Before(candidate.TargetAt) || candidate.ExitAvailableAt.After(asOf) {
			report.PendingSignals++
			continue
		}
		change := ((candidate.ExitPrice - candidate.EntryPrice) / candidate.EntryPrice) * 100
		prediction := direction(float64(candidate.Strength), 25)
		actual := direction(change, 1)
		report.Outcomes = append(report.Outcomes, domain.SignalOutcome{SignalID: candidate.SignalID, Symbol: candidate.Symbol, Sector: candidate.Sector, Category: candidate.Category, Horizon: candidate.Horizon, ConfidenceBand: ConfidenceBand(candidate.Confidence), Strength: candidate.Strength, Confidence: candidate.Confidence, DecisionAt: candidate.DecisionAt.UTC(), TargetAt: candidate.TargetAt.UTC(), ExitAsOf: candidate.ExitAsOf.UTC(), EntryPrice: round(candidate.EntryPrice), ExitPrice: round(candidate.ExitPrice), ReturnPercent: round(change), Prediction: prediction, Actual: actual, Correct: prediction == actual, EvaluatedAt: asOf.UTC(), Version: Version})
	}
	report.EvaluatedSignals = len(report.Outcomes)
	report.PendingSignals += report.CandidateSignals - report.EvaluatedSignals - uniqueViolationSignals(report.LeakageViolations) - report.PendingSignals
	report.Slices = buildSlices(report.Outcomes)
	return report
}

func detectLeakage(candidate domain.BacktestCase) []domain.LeakageViolation {
	violations := []domain.LeakageViolation{}
	for field, availableAt := range candidate.FeatureTimes {
		if availableAt.IsZero() || !availableAt.After(candidate.DecisionAt) {
			continue
		}
		violations = append(violations, domain.LeakageViolation{SignalID: candidate.SignalID, Field: field, AvailableAt: availableAt.UTC(), DecisionAt: candidate.DecisionAt.UTC(), Reason: "feature became available after the simulated decision time"})
	}
	sort.Slice(violations, func(i, j int) bool { return violations[i].Field < violations[j].Field })
	return violations
}

func ConfidenceBand(confidence int) string {
	switch {
	case confidence >= 85:
		return "85-100"
	case confidence >= 70:
		return "70-84"
	case confidence >= 50:
		return "50-69"
	default:
		return "0-49"
	}
}

func buildSlices(outcomes []domain.SignalOutcome) []domain.EvaluationSlice {
	type counter struct{ evaluated, correct, predictedPositive, truePositive int }
	counts := map[string]*counter{}
	for _, outcome := range outcomes {
		values := map[string]string{"confidenceBand": outcome.ConfidenceBand, "category": outcome.Category, "sector": outcome.Sector, "horizon": outcome.Horizon}
		for dimension, value := range values {
			if value == "" {
				value = "unknown"
			}
			key := dimension + "\x00" + value
			if counts[key] == nil {
				counts[key] = &counter{}
			}
			current := counts[key]
			current.evaluated++
			if outcome.Correct {
				current.correct++
			}
			if outcome.Prediction == "positive" {
				current.predictedPositive++
				if outcome.Actual == "positive" {
					current.truePositive++
				}
			}
		}
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	slices := make([]domain.EvaluationSlice, 0, len(keys))
	for _, key := range keys {
		separator := 0
		for index := range key {
			if key[index] == 0 {
				separator = index
				break
			}
		}
		current := counts[key]
		accuracy := percent(current.correct, current.evaluated)
		precision := percent(current.truePositive, current.predictedPositive)
		slices = append(slices, domain.EvaluationSlice{Dimension: key[:separator], Value: key[separator+1:], Evaluated: current.evaluated, Correct: current.correct, PredictedPositive: current.predictedPositive, TruePositive: current.truePositive, Accuracy: accuracy, Precision: precision})
	}
	return slices
}

func direction(value, neutralBand float64) string {
	if value >= neutralBand {
		return "positive"
	}
	if value <= -neutralBand {
		return "negative"
	}
	return "neutral"
}

func percent(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return round(float64(numerator) / float64(denominator) * 100)
}

func round(value float64) float64 { return math.Round(value*100) / 100 }

func uniqueViolationSignals(violations []domain.LeakageViolation) int {
	seen := map[string]bool{}
	for _, violation := range violations {
		seen[violation.SignalID] = true
	}
	return len(seen)
}

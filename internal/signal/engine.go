package signal

import (
	"math"
	"time"
)

type Weights struct{ NewsSentiment, Materiality, SourceReliability, AIConfidence, Recency, Confirmations, PriceMovement, VolumeAnomaly, SectorMovement, IndexMovement, Valuation, FinancialHealth, Contradiction float64 }
type Inputs struct {
	NewsSentiment                                                                                                                                                                float64
	Materiality, SourceReliability, AIConfidence, Recency, Confirmations, PriceMovement, VolumeAnomaly, SectorMovement, IndexMovement, Valuation, FinancialHealth, Contradiction float64
	EvidenceCount                                                                                                                                                                int
	FreshAt                                                                                                                                                                      time.Time
}
type Result struct {
	Label        string   `json:"label"`
	Score        float64  `json:"score"`
	Confidence   int      `json:"confidence"`
	Version      string   `json:"version"`
	Reasons      []string `json:"reasons"`
	Risks        []string `json:"risks"`
	Invalidators []string `json:"invalidators"`
}

var DefaultWeights = Weights{.16, .13, .08, .09, .08, .07, .11, .08, .05, .04, .05, .08, .08}

func Score(in Inputs, w Weights, version string) Result {
	if in.EvidenceCount == 0 || in.AIConfidence < 25 || in.SourceReliability < 20 {
		return Result{Label: "Insufficient evidence", Version: version, Confidence: int(in.AIConfidence), Risks: []string{"Evidence or source reliability is below the minimum threshold."}}
	}
	s := in.NewsSentiment*w.NewsSentiment + signed(in.Materiality, in.NewsSentiment)*w.Materiality + signed(in.SourceReliability, in.NewsSentiment)*w.SourceReliability + signed(in.AIConfidence, in.NewsSentiment)*w.AIConfidence + signed(in.Recency, in.NewsSentiment)*w.Recency + signed(in.Confirmations, in.NewsSentiment)*w.Confirmations + in.PriceMovement*w.PriceMovement + in.VolumeAnomaly*w.VolumeAnomaly + in.SectorMovement*w.SectorMovement + in.IndexMovement*w.IndexMovement + in.Valuation*w.Valuation + in.FinancialHealth*w.FinancialHealth - in.Contradiction*w.Contradiction
	s = math.Max(-100, math.Min(100, s))
	label := "Neutral/watch"
	if s >= 60 {
		label = "Strong positive setup"
	} else if s >= 25 {
		label = "Positive setup"
	} else if s <= -60 {
		label = "Strong negative setup"
	} else if s <= -25 {
		label = "Negative setup"
	}
	conf := int(math.Min(100, (in.AIConfidence+in.SourceReliability+math.Min(100, float64(in.EvidenceCount)*20))/3))
	return Result{Label: label, Score: math.Round(s*10) / 10, Confidence: conf, Version: version, Reasons: []string{"News, market context, and fundamentals were evaluated together."}, Risks: []string{"New filings or market moves can invalidate this setup."}, Invalidators: []string{"Contradictory official announcement", "Material price reversal on confirming volume"}}
}
func signed(v, direction float64) float64 {
	if direction < 0 {
		return -v
	}
	return v
}

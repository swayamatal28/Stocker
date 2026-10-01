package signal

import "testing"

func TestScoreRequiresEvidence(t *testing.T) {
	r := Score(Inputs{AIConfidence: 90, SourceReliability: 90}, DefaultWeights, "v1")
	if r.Label != "Insufficient evidence" {
		t.Fatalf("got %s", r.Label)
	}
}
func TestScoreContradictionReducesResult(t *testing.T) {
	base := Inputs{EvidenceCount: 3, NewsSentiment: 80, Materiality: 80, SourceReliability: 90, AIConfidence: 85, Recency: 90, Confirmations: 60, PriceMovement: 30, VolumeAnomaly: 20, FinancialHealth: 20}
	a := Score(base, DefaultWeights, "v1")
	base.Contradiction = 100
	b := Score(base, DefaultWeights, "v1")
	if b.Score >= a.Score {
		t.Fatalf("contradiction did not lower score: %.1f >= %.1f", b.Score, a.Score)
	}
}

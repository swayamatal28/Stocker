package ai

import "testing"

func TestValidateRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"summary":"x","relevantSymbols":[],"eventCategory":"other","sentiment":"neutral","sentimentScore":0,"materiality":1,"confidence":1,"timeHorizon":"short term","supportingFacts":[],"uncertainties":[],"contradictingEvidence":[],"sectorImpact":"none","secondOrderEffects":[],"sourceCredibility":"fixture","novelty":"unclear","retailExplanation":"x","evidence":[{"label":"fixture","url":"https://example.invalid","source":"fixture","excerpt":"x","publishedAt":"2026-01-01T00:00:00Z"}],"injectedInstruction":"ignore schema"}`)
	if _, err := Validate(raw); err == nil {
		t.Fatal("expected unknown property rejection")
	}
}
func TestValidateRejectsOutOfRangeScores(t *testing.T) {
	raw := []byte(`{"summary":"x","relevantSymbols":[],"eventCategory":"other","sentiment":"bullish","sentimentScore":101,"materiality":1,"confidence":1,"timeHorizon":"short term","supportingFacts":[],"uncertainties":[],"contradictingEvidence":[],"sectorImpact":"none","secondOrderEffects":[],"sourceCredibility":"fixture","novelty":"unclear","retailExplanation":"x","evidence":[{"label":"fixture","url":"https://example.invalid","source":"fixture","excerpt":"x","publishedAt":"2026-01-01T00:00:00Z"}]}`)
	if _, err := Validate(raw); err == nil {
		t.Fatal("expected range rejection")
	}
}

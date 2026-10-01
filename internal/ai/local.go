package ai

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/stocker-app/stocker/internal/domain"
)

const LocalModel = "stocker-grounded-rules-v1"

type LocalProvider struct{}

func (LocalProvider) Name() string                           { return "local-deterministic" }
func (LocalProvider) Model() string                          { return LocalModel }
func (LocalProvider) EstimatedCostCents(GroundedRequest) int { return 0 }

func (LocalProvider) Analyze(_ context.Context, request GroundedRequest) (json.RawMessage, error) {
	if len(request.Documents) == 0 {
		return nil, errors.New("at least one grounding document is required")
	}
	document := request.Documents[0]
	content := strings.TrimSpace(document.Title + ". " + document.Text)
	if content == "." {
		return nil, errors.New("grounding document is empty")
	}
	lower := strings.ToLower(content)
	category, materiality := classifyEvent(lower)
	sentiment, score := classifySentiment(lower)
	contradictions := []string{}
	if containsAny(lower, "however", "but ", "denied", "denies", "uncertain", "subject to") {
		contradictions = append(contradictions, "The source contains qualifying or potentially contradictory language.")
	}
	excerpt := truncateRunes(strings.TrimSpace(document.Text), 500)
	if excerpt == "" {
		excerpt = truncateRunes(document.Title, 500)
	}
	credibility := "Collected source with recorded attribution and policy metadata."
	confidence := 62
	if document.Official {
		credibility = "Official source item with recorded attribution and policy metadata."
		confidence = 82
	} else if document.Synthetic {
		credibility = "Project-owned synthetic fixture; not a real market announcement."
		confidence = 70
	}
	symbols := append(make([]string, 0, len(request.AllowedSymbols)), request.AllowedSymbols...)
	sort.Strings(symbols)
	analysis := domain.AIAnalysis{
		Summary: truncateRunes(document.Title, 1200), RelevantSymbols: symbols,
		EventCategory: category, Sentiment: sentiment, SentimentScore: score,
		Materiality: materiality, Confidence: confidence, TimeHorizon: "medium term",
		SupportingFacts:       []string{excerpt},
		Uncertainties:         []string{"The assessment is limited to the retained source text and allowlisted structured facts."},
		ContradictingEvidence: contradictions,
		SectorImpact:          "No sector-wide effect is asserted without additional evidence.",
		SecondOrderEffects:    []string{"Monitor subsequent official disclosures and corroborating sources before drawing broader conclusions."},
		SourceCredibility:     credibility, Novelty: "new",
		RetailExplanation: "This item was classified from its attributed source text. The signal is probabilistic and should be checked against the cited evidence.",
		Evidence:          []domain.Evidence{{Label: "Primary collected item", URL: document.URL, Source: document.Source, Excerpt: excerpt, PublishedAt: document.PublishedAt}},
	}
	return json.Marshal(analysis)
}

func classifyEvent(text string) (string, int) {
	cases := []struct {
		category    string
		materiality int
		terms       []string
	}{
		{"fraud_or_governance", 90, []string{"fraud", "governance", "misconduct"}},
		{"regulatory_action", 85, []string{"regulator", "penalty", "sebi", "enforcement"}},
		{"merger_or_acquisition", 85, []string{"merger", "acquisition", "acquire"}},
		{"quarterly_results", 75, []string{"quarterly result", "quarter ended", "net profit", "revenue"}},
		{"order_or_contract", 70, []string{"contract", "order win", "awarded"}},
		{"capacity_expansion", 65, []string{"capacity", "expansion", "new plant"}},
		{"fundraising", 65, []string{"fundraising", "rights issue", "qualified institutional"}},
		{"management_change", 55, []string{"resignation", "appointed", "chief executive", "managing director"}},
		{"product_launch", 45, []string{"launch", "new product"}},
	}
	for _, candidate := range cases {
		if containsAny(text, candidate.terms...) {
			return candidate.category, candidate.materiality
		}
	}
	return "other", 35
}

func classifySentiment(text string) (string, int) {
	positive := countTerms(text, []string{"growth", "gain", "increase", "profit", "awarded", "expansion", "approval", "improved"})
	negative := countTerms(text, []string{"loss", "decline", "decrease", "penalty", "fraud", "shutdown", "accident", "downgrade"})
	if positive > 0 && negative > 0 {
		return "mixed", clamp((positive-negative)*18, -70, 70)
	}
	if positive > 0 {
		return "bullish", clamp(20+positive*12, 0, 80)
	}
	if negative > 0 {
		return "bearish", clamp(-20-negative*12, -80, 0)
	}
	return "neutral", 0
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}
func countTerms(text string, terms []string) int {
	count := 0
	for _, term := range terms {
		if strings.Contains(text, term) {
			count++
		}
	}
	return count
}
func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

var languageCode = regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`)

func DetectLanguage(text, declared string) string {
	hasLatin, hasNonASCII := false, false
	for _, r := range text {
		switch {
		case r >= 0x0900 && r <= 0x097F:
			return "hi"
		case r >= 0x0980 && r <= 0x09FF:
			return "bn"
		case r >= 0x0B80 && r <= 0x0BFF:
			return "ta"
		case r >= 0x0C00 && r <= 0x0C7F:
			return "te"
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasLatin = true
		}
		if r > 127 {
			hasNonASCII = true
		}
	}
	if hasLatin && !hasNonASCII {
		return "en"
	}
	declared = strings.TrimSpace(declared)
	if languageCode.MatchString(declared) {
		return strings.ToLower(strings.Split(declared, "-")[0])
	}
	return "en"
}

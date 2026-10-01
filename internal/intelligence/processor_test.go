package intelligence

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stocker-app/stocker/internal/ai"
	"github.com/stocker-app/stocker/internal/domain"
)

type testRepository struct {
	input       domain.AnalysisArticle
	budget      bool
	saved       domain.IntelligenceOutput
	reserveCost int
}

func (repo *testRepository) AnalysisContext(context.Context, string) (domain.AnalysisArticle, error) {
	return repo.input, nil
}
func (repo *testRepository) ReserveAIBudget(_ context.Context, _ string, cost, _ int) (bool, error) {
	repo.reserveCost = cost
	return repo.budget, nil
}
func (repo *testRepository) SaveIntelligence(_ context.Context, output domain.IntelligenceOutput) (domain.IntelligenceOutput, bool, error) {
	repo.saved = output
	output.Analysis.ID = "analysis"
	return output, true, nil
}
func (repo *testRepository) MarkAnalysisFailure(context.Context, string, string, int) error {
	return nil
}

type pricedProvider struct {
	cost   int
	called bool
	raw    json.RawMessage
}

func (provider *pricedProvider) Name() string                              { return "priced" }
func (provider *pricedProvider) Model() string                             { return "model-v1" }
func (provider *pricedProvider) EstimatedCostCents(ai.GroundedRequest) int { return provider.cost }
func (provider *pricedProvider) Analyze(context.Context, ai.GroundedRequest) (json.RawMessage, error) {
	provider.called = true
	return provider.raw, nil
}

func TestProcessorCreatesGroundedSignal(t *testing.T) {
	repo := &testRepository{input: analysisInput(), budget: true}
	processor := NewProcessor(repo, ai.LocalProvider{}, nil, Config{PromptVersion: "prompt-v1", SchemaVersion: "ai-analysis-v1", MaxInputChars: 10000, DailyBudgetCents: 100, MaxAttempts: 3})
	output, err := processor.Process(context.Background(), repo.input.Article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if output.Analysis.PromptHash == "" || output.Analysis.Provider != "local-deterministic" {
		t.Fatalf("missing provider metadata: %#v", output.Analysis)
	}
	if len(output.Signals) != 1 || output.Signals[0].Symbol != "RELIANCE" || len(output.Signals[0].Sources) == 0 {
		t.Fatalf("unexpected signals: %#v", output.Signals)
	}
}

func TestLocalProcessorKeepsUntranslatedSourceTextConservatively(t *testing.T) {
	input := analysisInput()
	input.Article.Title = "बाज़ार में मिला-जुला कारोबार"
	input.Article.Body = "प्रकाशित स्रोत का संक्षिप्त विवरण।"
	input.Article.Language = "hi"
	input.Article.Symbols = []string{}
	input.LinkedSymbols = []string{}
	repo := &testRepository{input: input, budget: true}
	processor := NewProcessor(repo, ai.LocalProvider{}, nil, Config{PromptVersion: "prompt-v1", SchemaVersion: "ai-analysis-v1", MaxInputChars: 10000, DailyBudgetCents: 100, MaxAttempts: 3})
	output, err := processor.Process(context.Background(), input.Article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if output.Analysis.DetectedLanguage != "hi" || output.Analysis.AnalysisLanguage != "hi" || output.Analysis.TranslationApplied {
		t.Fatalf("unexpected language metadata: %#v", output.Analysis)
	}
}

func TestProcessorStopsBeforeProviderWhenBudgetIsExhausted(t *testing.T) {
	repo := &testRepository{input: analysisInput(), budget: false}
	provider := &pricedProvider{cost: 11}
	processor := NewProcessor(repo, provider, nil, Config{PromptVersion: "v1", SchemaVersion: "v1", MaxInputChars: 1000, DailyBudgetCents: 10})
	_, err := processor.Process(context.Background(), repo.input.Article.ID)
	if err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("expected budget error, got %v", err)
	}
	if provider.called {
		t.Fatal("provider called after budget rejection")
	}
	if repo.reserveCost != 11 {
		t.Fatalf("expected 11 cent reservation, got %d", repo.reserveCost)
	}
}

func TestProcessorRejectsHallucinatedProviderOutput(t *testing.T) {
	input := analysisInput()
	document := ai.GroundingDocument{URL: input.Article.URL, Source: input.Article.SourceName, Title: input.Article.Title, Text: input.Article.Body, PublishedAt: input.Article.PublishedAt}
	analysis := testAnalysis(document)
	analysis.Summary = "The company guaranteed a 999 percent return."
	raw, _ := json.Marshal(analysis)
	repo := &testRepository{input: input, budget: true}
	provider := &pricedProvider{raw: raw}
	processor := NewProcessor(repo, provider, nil, Config{PromptVersion: "v1", SchemaVersion: "v1", MaxInputChars: 1000, DailyBudgetCents: 10})
	if _, err := processor.Process(context.Background(), input.Article.ID); err == nil {
		t.Fatal("expected grounding rejection")
	}
}

func analysisInput() domain.AnalysisArticle {
	now := time.Now().UTC()
	return domain.AnalysisArticle{Article: domain.NewsArticle{ID: "507f1f77bcf86cd799439011", Title: "RELIANCE announces capacity expansion", Body: "The company announced a capacity expansion.", URL: "https://example.invalid/item", SourceName: "fixture", Language: "en", PublishedAt: now.Add(-time.Hour), RetrievedAt: now, Symbols: []string{"RELIANCE"}, Synthetic: true, DuplicateCount: 1}, LinkedSymbols: []string{"RELIANCE"}, NumericFacts: map[string]float64{"RELIANCE.last_price": 1400}}
}

func testAnalysis(document ai.GroundingDocument) domain.AIAnalysis {
	return domain.AIAnalysis{Summary: document.Title, RelevantSymbols: []string{"RELIANCE"}, EventCategory: "capacity_expansion", Sentiment: "neutral", SentimentScore: 0, Materiality: 60, Confidence: 70, TimeHorizon: "medium term", SupportingFacts: []string{document.Text}, Uncertainties: []string{}, ContradictingEvidence: []string{}, SectorImpact: "No sector effect asserted.", SecondOrderEffects: []string{}, SourceCredibility: "fixture", Novelty: "new", RetailExplanation: "Inspect the cited source.", Evidence: []domain.Evidence{{Label: "source", URL: document.URL, Source: document.Source, Excerpt: document.Text, PublishedAt: document.PublishedAt}}}
}

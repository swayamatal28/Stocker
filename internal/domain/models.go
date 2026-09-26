package domain

import "time"

type Security struct {
	ID            string    `json:"id"`
	NSESymbol     string    `json:"nseSymbol,omitempty"`
	BSECode       string    `json:"bseCode,omitempty"`
	ISIN          string    `json:"isin"`
	CompanyName   string    `json:"companyName"`
	Sector        string    `json:"sector"`
	Industry      string    `json:"industry"`
	Price         float64   `json:"price"`
	ChangePercent float64   `json:"changePercent"`
	Signal        string    `json:"signal"`
	NewsCount     int       `json:"newsCount"`
	AsOf          time.Time `json:"asOf"`
	Source        string    `json:"source"`
}

type WatchlistItem struct {
	Security
	AlertsPaused bool      `json:"alertsPaused"`
	AddedAt      time.Time `json:"addedAt"`
}

type User struct {
	ID, Email, DisplayName, PasswordHash, Role string
	CreatedAt                                  time.Time
}

type SourceItem struct {
	ExternalID, URL, CanonicalURL, Title, Body, Author, Language, ContentType string
	PublishedAt, RetrievedAt                                                  time.Time
	Attribution, Licence                                                      string
	Symbols, Sectors                                                          []string
	Official, Synthetic                                                       bool
	Headers                                                                   map[string]string
}
type RawDocument struct {
	SourceID, URL, ContentType, ParserVersion string
	RetrievedAt, DeleteAfter                  time.Time
	StatusCode                                int
	Body                                      []byte
	Hash                                      string
}

type Evidence struct {
	Label       string    `json:"label" bson:"label"`
	URL         string    `json:"url" bson:"url"`
	Source      string    `json:"source" bson:"source"`
	Excerpt     string    `json:"excerpt" bson:"excerpt"`
	PublishedAt time.Time `json:"publishedAt" bson:"published_at"`
}

type AIAnalysis struct {
	Summary               string     `json:"summary" bson:"summary"`
	RelevantSymbols       []string   `json:"relevantSymbols" bson:"relevant_symbols"`
	EventCategory         string     `json:"eventCategory" bson:"event_category"`
	Sentiment             string     `json:"sentiment" bson:"sentiment"`
	SentimentScore        int        `json:"sentimentScore" bson:"sentiment_score"`
	Materiality           int        `json:"materiality" bson:"materiality"`
	Confidence            int        `json:"confidence" bson:"confidence"`
	TimeHorizon           string     `json:"timeHorizon" bson:"time_horizon"`
	SupportingFacts       []string   `json:"supportingFacts" bson:"supporting_facts"`
	Uncertainties         []string   `json:"uncertainties" bson:"uncertainties"`
	ContradictingEvidence []string   `json:"contradictingEvidence" bson:"contradicting_evidence"`
	SectorImpact          string     `json:"sectorImpact" bson:"sector_impact"`
	SecondOrderEffects    []string   `json:"secondOrderEffects" bson:"second_order_effects"`
	SourceCredibility     string     `json:"sourceCredibility" bson:"source_credibility"`
	Novelty               string     `json:"novelty" bson:"novelty"`
	RetailExplanation     string     `json:"retailExplanation" bson:"retail_explanation"`
	Evidence              []Evidence `json:"evidence" bson:"evidence"`
}

type Signal struct {
	ID           string     `json:"id" bson:"-"`
	AnalysisID   string     `json:"analysisId" bson:"-"`
	Symbol       string     `json:"symbol" bson:"symbol"`
	Label        string     `json:"label" bson:"label"`
	Horizon      string     `json:"horizon" bson:"horizon"`
	Version      string     `json:"version" bson:"version"`
	Strength     int        `json:"strength" bson:"strength"`
	Confidence   int        `json:"confidence" bson:"confidence"`
	Reasons      []string   `json:"reasons" bson:"reasons"`
	Risks        []string   `json:"risks" bson:"risks"`
	Invalidators []string   `json:"invalidators" bson:"invalidators"`
	FreshAt      time.Time  `json:"freshAt" bson:"data_fresh_at"`
	GeneratedAt  time.Time  `json:"generatedAt" bson:"generated_at"`
	Sources      []Evidence `json:"sources" bson:"sources"`
}

type AnalysisArticle struct {
	Article       NewsArticle        `json:"article"`
	NumericFacts  map[string]float64 `json:"numericFacts"`
	LinkedSymbols []string           `json:"linkedSymbols"`
}

type AIUsage struct {
	InputUnits  int `json:"inputUnits"`
	OutputUnits int `json:"outputUnits"`
	CostCents   int `json:"costCents"`
}

type AnalysisRecord struct {
	ID                  string     `json:"id"`
	ArticleID           string     `json:"articleId"`
	Provider            string     `json:"provider"`
	Model               string     `json:"model"`
	PromptVersion       string     `json:"promptVersion"`
	PromptHash          string     `json:"promptHash"`
	PromptText          string     `json:"-"`
	SchemaVersion       string     `json:"schemaVersion"`
	DetectedLanguage    string     `json:"detectedLanguage"`
	AnalysisLanguage    string     `json:"analysisLanguage"`
	TranslationProvider string     `json:"translationProvider"`
	TranslationApplied  bool       `json:"translationApplied"`
	Analysis            AIAnalysis `json:"analysis"`
	Usage               AIUsage    `json:"usage"`
	CreatedAt           time.Time  `json:"createdAt"`
}

type IntelligenceOutput struct {
	Analysis AnalysisRecord `json:"analysis"`
	Signals  []Signal       `json:"signals"`
}

// NewsArticle is the public, policy-aware representation of a collected item.
// Body contains only the text the source policy permits STOCKER to retain and
// display; callers must always keep the attribution and canonical URL visible.
type NewsArticle struct {
	ID             string    `json:"id"`
	ClusterID      string    `json:"clusterId"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	SourceID       string    `json:"sourceId"`
	SourceName     string    `json:"sourceName"`
	SourceType     string    `json:"sourceType"`
	URL            string    `json:"url"`
	Author         string    `json:"author,omitempty"`
	Language       string    `json:"language"`
	Attribution    string    `json:"attribution"`
	Licence        string    `json:"licence"`
	ParserVersion  string    `json:"parserVersion"`
	PublishedAt    time.Time `json:"publishedAt"`
	RetrievedAt    time.Time `json:"retrievedAt"`
	Symbols        []string  `json:"symbols"`
	Sectors        []string  `json:"sectors"`
	Official       bool      `json:"official"`
	Synthetic      bool      `json:"synthetic"`
	DuplicateCount int       `json:"duplicateCount"`
}

type NewsFilter struct {
	Query, Stock, Sector, Source, Language string
	OfficialOnly                           bool
	From, To                               *time.Time
	Page, PageSize                         int
}

type NewsPage struct {
	Items      []NewsArticle
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

type SourceHealth struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	SourceType          string     `json:"sourceType"`
	Status              string     `json:"status"`
	ParserVersion       string     `json:"parserVersion"`
	Licence             string     `json:"licence"`
	LastSuccess         *time.Time `json:"lastSuccess,omitempty"`
	LastFailure         *time.Time `json:"lastFailure,omitempty"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	CircuitState        string     `json:"circuitState"`
	PollIntervalSeconds int        `json:"pollIntervalSeconds"`
	Synthetic           bool       `json:"synthetic"`
}

type NewsEvent struct {
	ArticleID   string    `json:"articleId"`
	Title       string    `json:"title"`
	SourceID    string    `json:"sourceId"`
	PublishedAt time.Time `json:"publishedAt"`
	Symbols     []string  `json:"symbols"`
}

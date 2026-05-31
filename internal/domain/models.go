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

type MarketQuote struct {
	Symbol        string    `json:"symbol"`
	Exchange      string    `json:"exchange"`
	Currency      string    `json:"currency"`
	LastPrice     float64   `json:"lastPrice"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"changePercent"`
	PreviousClose float64   `json:"previousClose"`
	Open          float64   `json:"open"`
	DayHigh       float64   `json:"dayHigh"`
	DayLow        float64   `json:"dayLow"`
	YearHigh      float64   `json:"yearHigh"`
	YearLow       float64   `json:"yearLow"`
	Volume        int64     `json:"volume"`
	Source        string    `json:"source"`
	SourceURL     string    `json:"sourceUrl,omitempty"`
	AsOf          time.Time `json:"asOf"`
	RetrievedAt   time.Time `json:"retrievedAt"`
	IsDelayed     bool      `json:"isDelayed"`
	Synthetic     bool      `json:"synthetic"`
}

type FundamentalMetric struct {
	Key    string  `json:"key" bson:"key"`
	Label  string  `json:"label" bson:"label"`
	Value  float64 `json:"value" bson:"value"`
	Unit   string  `json:"unit" bson:"unit"`
	Period string  `json:"period" bson:"period"`
	Basis  string  `json:"basis" bson:"basis"`
}

type Fundamentals struct {
	Symbol      string              `json:"symbol"`
	Metrics     []FundamentalMetric `json:"metrics"`
	Source      string              `json:"source"`
	SourceURL   string              `json:"sourceUrl,omitempty"`
	AsOf        time.Time           `json:"asOf"`
	RetrievedAt time.Time           `json:"retrievedAt"`
	Synthetic   bool                `json:"synthetic"`
}

type SectorSnapshot struct {
	Sector           string    `json:"sector"`
	CompanyCount     int       `json:"companyCount"`
	Advances         int       `json:"advances"`
	Declines         int       `json:"declines"`
	AverageChange    float64   `json:"averageChangePercent"`
	AverageSentiment float64   `json:"averageSentiment"`
	LatestEvidenceAt time.Time `json:"latestEvidenceAt,omitempty"`
	LatestMarketAsOf time.Time `json:"latestMarketAsOf,omitempty"`
}

type MarketMover struct {
	Symbol        string    `json:"symbol"`
	CompanyName   string    `json:"companyName"`
	Sector        string    `json:"sector"`
	LastPrice     float64   `json:"lastPrice"`
	ChangePercent float64   `json:"changePercent"`
	Volume        int64     `json:"volume"`
	AsOf          time.Time `json:"asOf"`
	Source        string    `json:"source"`
}

type MarketEvent struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol,omitempty"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	EventAt    time.Time `json:"eventAt"`
	Source     string    `json:"source"`
	SourceURL  string    `json:"sourceUrl"`
	EvidenceID string    `json:"evidenceId,omitempty"`
	Official   bool      `json:"official"`
	Synthetic  bool      `json:"synthetic"`
}

type PeerSnapshot struct {
	Symbol        string  `json:"symbol"`
	CompanyName   string  `json:"companyName"`
	LastPrice     float64 `json:"lastPrice"`
	ChangePercent float64 `json:"changePercent"`
	PERatio       float64 `json:"peRatio"`
	MarketCap     float64 `json:"marketCap"`
}

type RiskFlag struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
}

type StockIntelligence struct {
	Security     Security       `json:"security"`
	Quote        *MarketQuote   `json:"quote,omitempty"`
	Fundamentals *Fundamentals  `json:"fundamentals,omitempty"`
	Peers        []PeerSnapshot `json:"peers"`
	RiskFlags    []RiskFlag     `json:"riskFlags"`
	Research     *StockResearch `json:"research,omitempty"`
}

type ResearchCheck struct {
	Key         string  `json:"key"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	Value       float64 `json:"value"`
	Display     string  `json:"display"`
	Explanation string  `json:"explanation"`
	Formula     string  `json:"formula"`
}

type StockResearch struct {
	Score      int             `json:"score"`
	Label      string          `json:"label"`
	Coverage   int             `json:"coverage"`
	Checks     []ResearchCheck `json:"checks"`
	Source     string          `json:"source"`
	AsOf       time.Time       `json:"asOf"`
	Disclaimer string          `json:"disclaimer"`
}

type PortfolioItem struct {
	Security
	Quantity        float64   `json:"quantity"`
	AverageBuyPrice float64   `json:"averageBuyPrice"`
	InvestedValue   float64   `json:"investedValue"`
	CurrentValue    float64   `json:"currentValue"`
	PnL             float64   `json:"pnl"`
	PnLPercent      float64   `json:"pnlPercent"`
	DayPnL          float64   `json:"dayPnl"`
	DetailsComplete bool      `json:"detailsComplete"`
	AlertsPaused    bool      `json:"alertsPaused"`
	AddedAt         time.Time `json:"addedAt"`
}

type PortfolioSummary struct {
	InvestedValue float64 `json:"investedValue"`
	CurrentValue  float64 `json:"currentValue"`
	PnL           float64 `json:"pnl"`
	PnLPercent    float64 `json:"pnlPercent"`
	DayPnL        float64 `json:"dayPnl"`
	HoldingCount  int     `json:"holdingCount"`
}

type AlertChannels struct {
	InApp    bool `json:"inApp" bson:"in_app"`
	Browser  bool `json:"browser" bson:"browser"`
	Email    bool `json:"email" bson:"email"`
	Telegram bool `json:"telegram" bson:"telegram"`
}

type QuietHours struct {
	Enabled  bool   `json:"enabled" bson:"enabled"`
	Start    string `json:"start" bson:"start"`
	End      string `json:"end" bson:"end"`
	Timezone string `json:"timezone" bson:"timezone"`
}

type AlertRule struct {
	ID                string        `json:"id"`
	UserID            string        `json:"-"`
	Name              string        `json:"name"`
	Symbol            string        `json:"symbol"`
	RuleType          string        `json:"ruleType"`
	Threshold         *float64      `json:"threshold,omitempty"`
	EventCategories   []string      `json:"eventCategories,omitempty"`
	MinimumConfidence int           `json:"minimumConfidence"`
	MinimumSeverity   string        `json:"minimumSeverity"`
	CooldownMinutes   int           `json:"cooldownMinutes"`
	Channels          AlertChannels `json:"channels"`
	QuietHours        QuietHours    `json:"quietHours"`
	Enabled           bool          `json:"enabled"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`
}

type AlertEvent struct {
	ID                    string            `json:"id"`
	UserID                string            `json:"-"`
	DeduplicationKey      string            `json:"-"`
	RuleID                string            `json:"ruleId"`
	RuleName              string            `json:"ruleName"`
	RuleType              string            `json:"ruleType"`
	Symbol                string            `json:"symbol"`
	Severity              string            `json:"severity"`
	Title                 string            `json:"title"`
	Explanation           string            `json:"explanation"`
	Confidence            int               `json:"confidence"`
	Evidence              []Evidence        `json:"evidence"`
	ConditionSnapshot     map[string]any    `json:"conditionSnapshot"`
	SourceAsOf            time.Time         `json:"sourceAsOf"`
	TriggeredAt           time.Time         `json:"triggeredAt"`
	DeliverAfter          time.Time         `json:"deliverAfter"`
	DeliveredAt           *time.Time        `json:"deliveredAt,omitempty"`
	ReadAt                *time.Time        `json:"readAt,omitempty"`
	DeliveryStatus        string            `json:"deliveryStatus"`
	ExternalChannelStatus map[string]string `json:"externalChannelStatus,omitempty"`
}

type AlertCandidate struct {
	Symbol            string
	Kind              string
	Category          string
	ClusterID         string
	Title             string
	Explanation       string
	Severity          string
	Confidence        int
	Price             float64
	ChangePercent     float64
	SourceAsOf        time.Time
	Evidence          []Evidence
	ConditionSnapshot map[string]any
}

type BriefingItem struct {
	Symbol      string     `json:"symbol" bson:"symbol"`
	Headline    string     `json:"headline" bson:"headline"`
	Explanation string     `json:"explanation" bson:"explanation"`
	Severity    string     `json:"severity" bson:"severity"`
	Confidence  int        `json:"confidence" bson:"confidence"`
	AsOf        time.Time  `json:"asOf" bson:"as_of"`
	Evidence    []Evidence `json:"evidence" bson:"evidence"`
}

type Briefing struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Title       string         `json:"title"`
	Summary     string         `json:"summary"`
	Items       []BriefingItem `json:"items"`
	GeneratedAt time.Time      `json:"generatedAt"`
	PeriodKey   string         `json:"periodKey"`
	Synthetic   bool           `json:"synthetic"`
}

type BacktestCase struct {
	SignalID        string
	Symbol          string
	Sector          string
	Category        string
	Horizon         string
	Strength        int
	Confidence      int
	DecisionAt      time.Time
	TargetAt        time.Time
	EntryPrice      float64
	ExitPrice       float64
	ExitAsOf        time.Time
	ExitAvailableAt time.Time
	FeatureTimes    map[string]time.Time
}

type SignalOutcome struct {
	SignalID       string    `json:"signalId" bson:"signal_id"`
	Symbol         string    `json:"symbol" bson:"symbol"`
	Sector         string    `json:"sector" bson:"sector"`
	Category       string    `json:"category" bson:"category"`
	Horizon        string    `json:"horizon" bson:"horizon"`
	ConfidenceBand string    `json:"confidenceBand" bson:"confidence_band"`
	Strength       int       `json:"strength" bson:"strength"`
	Confidence     int       `json:"confidence" bson:"confidence"`
	DecisionAt     time.Time `json:"decisionAt" bson:"decision_at"`
	TargetAt       time.Time `json:"targetAt" bson:"target_at"`
	ExitAsOf       time.Time `json:"exitAsOf" bson:"exit_as_of"`
	EntryPrice     float64   `json:"entryPrice" bson:"entry_price"`
	ExitPrice      float64   `json:"exitPrice" bson:"exit_price"`
	ReturnPercent  float64   `json:"returnPercent" bson:"return_percent"`
	Prediction     string    `json:"prediction" bson:"prediction"`
	Actual         string    `json:"actual" bson:"actual"`
	Correct        bool      `json:"correct" bson:"correct"`
	EvaluatedAt    time.Time `json:"evaluatedAt" bson:"evaluated_at"`
	Version        string    `json:"version" bson:"version"`
}

type LeakageViolation struct {
	SignalID    string    `json:"signalId" bson:"signal_id"`
	Field       string    `json:"field" bson:"field"`
	AvailableAt time.Time `json:"availableAt" bson:"available_at"`
	DecisionAt  time.Time `json:"decisionAt" bson:"decision_at"`
	Reason      string    `json:"reason" bson:"reason"`
}

type EvaluationSlice struct {
	Dimension         string  `json:"dimension"`
	Value             string  `json:"value"`
	Evaluated         int     `json:"evaluated"`
	Correct           int     `json:"correct"`
	PredictedPositive int     `json:"predictedPositive"`
	TruePositive      int     `json:"truePositive"`
	Accuracy          float64 `json:"accuracy"`
	Precision         float64 `json:"precision"`
}

type EvaluationReport struct {
	Version           string             `json:"version"`
	AsOf              time.Time          `json:"asOf"`
	CandidateSignals  int                `json:"candidateSignals"`
	EvaluatedSignals  int                `json:"evaluatedSignals"`
	PendingSignals    int                `json:"pendingSignals"`
	LeakageViolations []LeakageViolation `json:"leakageViolations"`
	Outcomes          []SignalOutcome    `json:"outcomes"`
	Slices            []EvaluationSlice  `json:"slices"`
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

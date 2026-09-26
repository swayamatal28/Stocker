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
	Label, URL, Source, Excerpt string
	PublishedAt                 time.Time
}

type AIAnalysis struct {
	Summary               string     `json:"summary"`
	RelevantSymbols       []string   `json:"relevantSymbols"`
	EventCategory         string     `json:"eventCategory"`
	Sentiment             string     `json:"sentiment"`
	SentimentScore        int        `json:"sentimentScore"`
	Materiality           int        `json:"materiality"`
	Confidence            int        `json:"confidence"`
	TimeHorizon           string     `json:"timeHorizon"`
	SupportingFacts       []string   `json:"supportingFacts"`
	Uncertainties         []string   `json:"uncertainties"`
	ContradictingEvidence []string   `json:"contradictingEvidence"`
	SectorImpact          string     `json:"sectorImpact"`
	SecondOrderEffects    []string   `json:"secondOrderEffects"`
	SourceCredibility     string     `json:"sourceCredibility"`
	Novelty               string     `json:"novelty"`
	RetailExplanation     string     `json:"retailExplanation"`
	Evidence              []Evidence `json:"evidence"`
}

type Signal struct {
	ID, Symbol, Label, Horizon, Version string
	Strength, Confidence                int
	Reasons, Risks, Invalidators        []string
	FreshAt                             time.Time
	Sources                             []Evidence
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

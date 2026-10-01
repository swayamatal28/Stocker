package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

// Adapter isolates a source's transport and parser. Implementations must honour
// SourcePolicy and must never bypass access controls, robots rules, or paywalls.
type Adapter interface {
	ID() string
	Kind() SourceKind
	Fetch(context.Context, Cursor) (Batch, error)
	Health(context.Context) Health
}

type SourceKind string

const (
	KindExchange SourceKind = "exchange_announcement"
	KindAPI      SourceKind = "json_api"
	KindRSS      SourceKind = "rss_atom"
	KindHTML     SourceKind = "permitted_html"
	KindMock     SourceKind = "mock"
)

type Cursor struct {
	Value string
	Since time.Time
}
type Batch struct {
	Items         []domain.SourceItem
	Next          Cursor
	ParserVersion string
	RetrievedAt   time.Time
}
type Health struct {
	Status              string
	LastSuccess         *time.Time
	LastError           string
	Latency             time.Duration
	ConsecutiveFailures int
}
type SourcePolicy struct {
	PollInterval, Timeout  time.Duration
	RequestsPerMinute      int
	RawRetention           time.Duration
	Attribution, Licence   string
	TermsURL               string
	AutomatedAccessAllowed bool
	PolicyExpiresAt        *time.Time
	RobotsChecked          bool
}

func (p SourcePolicy) Validate(now time.Time) error {
	if !p.AutomatedAccessAllowed {
		return ErrSourceNotApproved
	}
	if strings.TrimSpace(p.TermsURL) == "" || strings.TrimSpace(p.Attribution) == "" || strings.TrimSpace(p.Licence) == "" {
		return fmt.Errorf("%w: terms URL, attribution, and licence are required", ErrSourceNotApproved)
	}
	if p.PolicyExpiresAt != nil && !p.PolicyExpiresAt.After(now) {
		return fmt.Errorf("%w: policy expired at %s", ErrSourceNotApproved, p.PolicyExpiresAt.UTC().Format(time.RFC3339))
	}
	if p.PollInterval <= 0 || p.Timeout <= 0 || p.RequestsPerMinute <= 0 || p.RawRetention <= 0 {
		return errors.New("source policy timing, rate, and retention values must be positive")
	}
	return nil
}

type Publisher interface {
	Publish(context.Context, string, any) error
}
type Repository interface {
	SeenHash(context.Context, string) (bool, error)
	FindNearDuplicate(context.Context, domain.SourceItem, time.Duration, float64) (string, bool, error)
	SaveRaw(context.Context, domain.RawDocument) error
	SaveNormalized(context.Context, string, domain.SourceItem, string, string, string, SourcePolicy) (string, bool, error)
	PendingNewsEvents(context.Context, int) ([]domain.NewsEvent, error)
	MarkNewsEventPublished(context.Context, string) error
	RecordPublishFailure(context.Context, string, string, int) error
	RecordSuccess(context.Context, string, string, time.Duration, SourcePolicy) error
	RecordFailure(context.Context, string, string, string) error
}

func ContentHash(item domain.SourceItem) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(item.Title+" "+item.Body), " "))
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}
func CanonicalURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	q := u.Query()
	for key := range q {
		l := strings.ToLower(key)
		if strings.HasPrefix(l, "utm_") || l == "ref" || l == "source" {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	u.Host = strings.ToLower(u.Host)
	return u.String()
}

type RetryPolicy struct {
	MaxAttempts         int
	BaseDelay, MaxDelay time.Duration
}

func (r RetryPolicy) Delay(attempt int) time.Duration {
	d := r.BaseDelay * time.Duration(1<<min(attempt, 10))
	if d > r.MaxDelay {
		d = r.MaxDelay
	}
	jitter := time.Duration(rand.Int64N(max(int64(d/3), 1)))
	return d + jitter
}

var ErrCircuitOpen = errors.New("source circuit breaker open")
var ErrRateLimited = errors.New("source rate limit reached")
var ErrSourceNotApproved = errors.New("source policy does not approve automated access")

// MockAdapter is deliberately transparent and contains no copied third-party content.
// It proves the adapter contract when no licensed source credentials are configured.
type MockAdapter struct{}

func (MockAdapter) ID() string       { return "mock-exchange" }
func (MockAdapter) Kind() SourceKind { return KindMock }
func (MockAdapter) Health(context.Context) Health {
	now := time.Now().UTC()
	return Health{Status: "healthy", LastSuccess: &now, Latency: 2 * time.Millisecond}
}
func (MockAdapter) Fetch(_ context.Context, c Cursor) (Batch, error) {
	now := time.Now().UTC()
	return Batch{ParserVersion: "mock-v2", RetrievedAt: now, Next: Cursor{Value: now.Format(time.RFC3339), Since: now}, Items: []domain.SourceItem{{ExternalID: "demo-reliance-capex", URL: "https://example.invalid/mock/reliance-capex", CanonicalURL: "https://example.invalid/mock/reliance-capex", Title: "Demonstration: Reliance announces capacity investment", Body: "Synthetic fixture for local development. The company described a multi-year capacity investment; no real financial values are asserted.", Author: "STOCKER fixture", Language: "en", ContentType: "text/plain", PublishedAt: now.Add(-20 * time.Minute), RetrievedAt: now, Attribution: "Synthetic STOCKER demo content", Licence: "Project-owned test fixture", Symbols: []string{"RELIANCE"}, Sectors: []string{"Energy"}, Synthetic: true}}}, nil
}

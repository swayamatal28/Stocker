package ingest

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
)

const maxFeedBytes = 2 << 20

var htmlTag = regexp.MustCompile(`<[^>]+>`)

type RSSAdapterConfig struct {
	SourceID, FeedURL, Attribution, Licence, Language string
	Official                                          bool
	Client                                            *http.Client
}

type RSSAdapter struct {
	cfg RSSAdapterConfig
}

func NewRSSAdapter(cfg RSSAdapterConfig) (*RSSAdapter, error) {
	u, err := url.Parse(cfg.FeedURL)
	if err != nil || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
		return nil, errors.New("feed URL must use HTTPS (HTTP is allowed only for local tests)")
	}
	if strings.TrimSpace(cfg.SourceID) == "" || strings.TrimSpace(cfg.Attribution) == "" || strings.TrimSpace(cfg.Licence) == "" {
		return nil, errors.New("source ID, attribution, and licence are required")
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 15 * time.Second}
	}
	return &RSSAdapter{cfg: cfg}, nil
}

func (a *RSSAdapter) ID() string       { return a.cfg.SourceID }
func (a *RSSAdapter) Kind() SourceKind { return KindRSS }
func (a *RSSAdapter) Health(context.Context) Health {
	return Health{Status: "configured"}
}

func (a *RSSAdapter) Fetch(ctx context.Context, cursor Cursor) (Batch, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.FeedURL, nil)
	if err != nil {
		return Batch{}, err
	}
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")
	req.Header.Set("User-Agent", "STOCKER/0.2 evidence-research feed reader")
	if cursor.Value != "" {
		req.Header.Set("If-Modified-Since", cursor.Value)
	}
	res, err := a.cfg.Client.Do(req)
	if err != nil {
		return Batch{}, err
	}
	defer res.Body.Close()
	now := time.Now().UTC()
	if res.StatusCode == http.StatusNotModified {
		return Batch{ParserVersion: "rss-atom-v1", RetrievedAt: now, Next: cursor}, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Batch{}, fmt.Errorf("feed returned HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxFeedBytes+1))
	if err != nil {
		return Batch{}, err
	}
	if len(body) > maxFeedBytes {
		return Batch{}, errors.New("feed exceeds 2 MiB safety limit")
	}
	items, err := parseFeed(body, now, a.cfg)
	if err != nil {
		return Batch{}, err
	}
	filtered := items[:0]
	for _, item := range items {
		if cursor.Since.IsZero() || item.PublishedAt.After(cursor.Since) {
			filtered = append(filtered, item)
		}
	}
	lastModified := res.Header.Get("Last-Modified")
	if lastModified == "" {
		lastModified = now.Format(http.TimeFormat)
	}
	return Batch{
		Items: filtered, ParserVersion: "rss-atom-v1", RetrievedAt: now,
		Next: Cursor{Value: lastModified, Since: newestTime(filtered, cursor.Since)},
	}, nil
}

type feedEnvelope struct {
	XMLName xml.Name
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
	Entries []atomEntry `xml:"entry"`
}

type rssItem struct {
	GUID        string   `xml:"guid"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Author      string   `xml:"author"`
	PubDate     string   `xml:"pubDate"`
	Categories  []string `xml:"category"`
}

type atomEntry struct {
	ID        string `xml:"id"`
	Title     string `xml:"title"`
	Summary   string `xml:"summary"`
	Content   string `xml:"content"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Author    struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Links []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
	Categories []struct {
		Term string `xml:"term,attr"`
	} `xml:"category"`
}

func parseFeed(data []byte, retrievedAt time.Time, cfg RSSAdapterConfig) ([]domain.SourceItem, error) {
	var feed feedEnvelope
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parse RSS/Atom: %w", err)
	}
	items := make([]domain.SourceItem, 0, len(feed.Channel.Items)+len(feed.Entries))
	for _, v := range feed.Channel.Items {
		published := parseFeedTime(v.PubDate, retrievedAt)
		item := domain.SourceItem{
			ExternalID: firstNonEmpty(v.GUID, v.Link), URL: strings.TrimSpace(v.Link), Title: cleanText(v.Title),
			Body: cleanText(v.Description), Author: cleanText(v.Author), Language: defaultString(cfg.Language, "en"),
			ContentType: "text/plain", PublishedAt: published, RetrievedAt: retrievedAt,
			Attribution: cfg.Attribution, Licence: cfg.Licence, Sectors: cleanList(v.Categories), Official: cfg.Official,
		}
		if item.URL != "" && item.Title != "" {
			items = append(items, item)
		}
	}
	for _, v := range feed.Entries {
		link := ""
		for _, candidate := range v.Links {
			if candidate.Rel == "" || candidate.Rel == "alternate" {
				link = candidate.Href
				break
			}
		}
		categories := make([]string, 0, len(v.Categories))
		for _, category := range v.Categories {
			categories = append(categories, category.Term)
		}
		item := domain.SourceItem{
			ExternalID: firstNonEmpty(v.ID, link), URL: strings.TrimSpace(link), Title: cleanText(v.Title),
			Body: cleanText(firstNonEmpty(v.Summary, v.Content)), Author: cleanText(v.Author.Name),
			Language: defaultString(cfg.Language, "en"), ContentType: "text/plain",
			PublishedAt: parseFeedTime(firstNonEmpty(v.Published, v.Updated), retrievedAt), RetrievedAt: retrievedAt,
			Attribution: cfg.Attribution, Licence: cfg.Licence, Sectors: cleanList(categories), Official: cfg.Official,
		}
		if item.URL != "" && item.Title != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return nil, errors.New("feed contained no valid items")
	}
	return items, nil
}

func cleanText(value string) string {
	value = html.UnescapeString(htmlTag.ReplaceAllString(value, " "))
	return strings.Join(strings.Fields(value), " ")
}

func cleanList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = cleanText(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func parseFeedTime(value string, fallback time.Time) time.Time {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed.UTC()
		}
	}
	return fallback.UTC()
}

func newestTime(items []domain.SourceItem, fallback time.Time) time.Time {
	latest := fallback
	for _, item := range items {
		if item.PublishedAt.After(latest) {
			latest = item.PublishedAt
		}
	}
	return latest
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

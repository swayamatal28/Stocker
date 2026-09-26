package ingest

import (
	"os"
	"testing"
	"time"
)

func TestParseRSSFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/sample-rss.xml")
	if err != nil {
		t.Fatal(err)
	}
	retrieved := time.Date(2026, 9, 23, 6, 0, 0, 0, time.UTC)
	items, err := parseFeed(fixture, retrieved, RSSAdapterConfig{
		Attribution: "STOCKER fixture", Licence: "Project-owned", Language: "en", Official: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	item := items[0]
	if item.ExternalID != "fixture-1" || item.Title != "INFY files a project-owned test announcement" {
		t.Fatalf("unexpected item: %#v", item)
	}
	if item.Body != "This is saved, synthetic parser-test content." {
		t.Fatalf("HTML was not removed: %q", item.Body)
	}
	if !item.PublishedAt.Equal(time.Date(2026, 9, 23, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected published time: %s", item.PublishedAt)
	}
}

func TestRSSAdapterRejectsUnsafeScheme(t *testing.T) {
	_, err := NewRSSAdapter(RSSAdapterConfig{
		SourceID: "unsafe", FeedURL: "file:///tmp/feed.xml", Attribution: "Fixture", Licence: "Project-owned",
	})
	if err == nil {
		t.Fatal("expected an unsafe URL scheme to be rejected")
	}
}

func TestSourcePolicyRequiresExplicitApproval(t *testing.T) {
	policy := SourcePolicy{
		PollInterval: time.Minute, Timeout: time.Second, RequestsPerMinute: 1, RawRetention: time.Hour,
		Attribution: "Source", Licence: "Licensed", TermsURL: "https://example.invalid/terms",
	}
	if err := policy.Validate(time.Now()); err == nil {
		t.Fatal("expected unapproved automated access to be rejected")
	}
	policy.AutomatedAccessAllowed = true
	if err := policy.Validate(time.Now()); err != nil {
		t.Fatalf("expected approved policy to pass: %v", err)
	}
}

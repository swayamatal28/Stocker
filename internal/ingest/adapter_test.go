package ingest

import (
	"github.com/stocker-app/stocker/internal/domain"
	"testing"
)

func TestCanonicalURL(t *testing.T) {
	got := CanonicalURL("https://EXAMPLE.com/a?utm_source=x&id=7#top")
	if got != "https://example.com/a?id=7" {
		t.Fatalf("got %s", got)
	}
}
func TestContentHashNormalizesWhitespace(t *testing.T) {
	a := ContentHash(domain.SourceItem{Title: "Hello  WORLD"})
	b := ContentHash(domain.SourceItem{Title: "hello world"})
	if a != b {
		t.Fatal("expected normalized hashes")
	}
}

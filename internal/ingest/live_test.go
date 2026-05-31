package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This contract test is opt-in because it contacts real publishers. It never
// runs during the normal deterministic test suite.
func TestConfiguredLiveFeeds(t *testing.T) {
	if os.Getenv("STOCKER_LIVE_FEED_TEST") != "1" {
		t.Skip("set STOCKER_LIVE_FEED_TEST=1 to contact configured publishers")
	}
	raw := os.Getenv("INGEST_SOURCES_JSON")
	if raw == "" && os.Getenv("INGEST_SOURCES_FILE") != "" {
		path := os.Getenv("INGEST_SOURCES_FILE")
		contents, err := os.ReadFile(path)
		if os.IsNotExist(err) && !filepath.IsAbs(path) {
			contents, err = os.ReadFile(filepath.Join("..", "..", path))
		}
		if err != nil {
			t.Fatal(err)
		}
		raw = string(contents)
	}
	configured, err := ParseFeedSources(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range configured {
		source := source
		t.Run(source.Adapter.ID(), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			batch, err := source.Adapter.Fetch(ctx, Cursor{})
			if err != nil {
				t.Fatal(err)
			}
			if len(batch.Items) == 0 {
				t.Fatal("live feed returned no usable items")
			}
			for _, item := range batch.Items {
				if item.Synthetic {
					t.Fatal("live source produced a synthetic item")
				}
			}
		})
	}
}

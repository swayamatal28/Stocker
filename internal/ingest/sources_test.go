package ingest

import "testing"

func TestParseFeedSourcesFailsClosedUntilSourceIsApproved(t *testing.T) {
	raw := `[{"id":"publisher","enabled":true,"feedUrl":"https://example.com/feed.xml","attribution":"Publisher","licence":"Headline and link only","termsUrl":"https://example.com/terms","language":"en","automatedAccessAllowed":false,"robotsChecked":true,"requestsPerMinute":2,"pollInterval":"10m","timeout":"10s","rawRetentionDays":1}]`
	if _, err := ParseFeedSources(raw); err == nil {
		t.Fatal("unapproved source must fail closed")
	}
}

func TestParseFeedSourcesBuildsIndependentPolicies(t *testing.T) {
	raw := `[{"id":"publisher","enabled":true,"feedUrl":"https://example.com/feed.xml","attribution":"Publisher","licence":"Headline and link only","termsUrl":"https://example.com/terms","language":"en","automatedAccessAllowed":true,"robotsChecked":true,"requestsPerMinute":2,"pollInterval":"10m","timeout":"10s","rawRetentionDays":1}]`
	feeds, err := ParseFeedSources(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(feeds) != 1 || feeds[0].Adapter.ID() != "publisher" || feeds[0].Policy.PollInterval.String() != "10m0s" {
		t.Fatalf("unexpected configured feeds: %#v", feeds)
	}
}

func TestParseFeedSourcesRequiresRobotsReview(t *testing.T) {
	raw := `[{"id":"publisher","enabled":true,"feedUrl":"https://example.com/feed.xml","attribution":"Publisher","licence":"Headline and link only","termsUrl":"https://example.com/terms","language":"en","automatedAccessAllowed":true,"robotsChecked":false,"requestsPerMinute":2,"pollInterval":"10m","timeout":"10s","rawRetentionDays":1}]`
	if _, err := ParseFeedSources(raw); err == nil {
		t.Fatal("source without a completed robots review must fail closed")
	}
}

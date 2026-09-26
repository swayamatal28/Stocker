package httpapi

import "testing"

func TestHubPublishAndUnsubscribe(t *testing.T) {
	hub := NewHub()
	ch, unsubscribe := hub.Subscribe()
	if dropped := hub.Publish("news.created", map[string]string{"id": "one"}); dropped != 0 {
		t.Fatalf("unexpected dropped event count: %d", dropped)
	}
	if got := string(<-ch); got != `{"data":{"id":"one"},"event":"news.created"}` {
		t.Fatalf("unexpected event: %s", got)
	}
	unsubscribe()
	if subscribers, _ := hub.Stats(); subscribers != 0 {
		t.Fatalf("expected no subscribers, got %d", subscribers)
	}
}

func TestHubCountsBackpressureDrops(t *testing.T) {
	hub := NewHub()
	_, unsubscribe := hub.Subscribe()
	defer unsubscribe()
	for i := 0; i < 17; i++ {
		hub.Publish("update", i)
	}
	_, dropped := hub.Stats()
	if dropped != 1 {
		t.Fatalf("expected one dropped event, got %d", dropped)
	}
}

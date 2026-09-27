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

func TestHubTargetedPublishIsUserIsolated(t *testing.T) {
	hub := NewHub()
	first, doneFirst := hub.SubscribeFor("user-one")
	defer doneFirst()
	second, doneSecond := hub.SubscribeFor("user-two")
	defer doneSecond()
	hub.PublishTo("user-one", "alert.created", map[string]string{"id": "alert-one"})
	if got := string(<-first); got != `{"data":{"id":"alert-one"},"event":"alert.created"}` {
		t.Fatalf("unexpected targeted event: %s", got)
	}
	select {
	case value := <-second:
		t.Fatalf("targeted event leaked to another user: %s", value)
	default:
	}
}

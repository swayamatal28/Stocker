package httpapi

import (
	"testing"
	"time"
)

func TestHubFanoutLoadPreservesUserIsolation(t *testing.T) {
	hub := NewHub()
	const subscribers = 250
	channels := make([]chan []byte, 0, subscribers)
	done := make([]func(), 0, subscribers)
	for index := 0; index < subscribers; index++ {
		user := "other"
		if index%5 == 0 {
			user = "target"
		}
		channel, unsubscribe := hub.SubscribeFor(user)
		channels = append(channels, channel)
		done = append(done, unsubscribe)
	}
	defer func() {
		for _, unsubscribe := range done {
			unsubscribe()
		}
	}()
	if dropped := hub.PublishTo("target", "alert.created", map[string]string{"id": "load"}); dropped != 0 {
		t.Fatalf("unexpected drops: %d", dropped)
	}
	for index, channel := range channels {
		select {
		case <-channel:
			if index%5 != 0 {
				t.Fatalf("targeted event leaked to subscriber %d", index)
			}
		case <-time.After(5 * time.Millisecond):
			if index%5 == 0 {
				t.Fatalf("target subscriber %d did not receive event", index)
			}
		}
	}
}

func BenchmarkHubFanout(b *testing.B) {
	hub := NewHub()
	done := make([]func(), 0, 1000)
	for index := 0; index < 1000; index++ {
		channel, unsubscribe := hub.SubscribeFor("load-user")
		done = append(done, unsubscribe)
		go func() {
			for range channel {
			}
		}()
	}
	b.Cleanup(func() {
		for _, unsubscribe := range done {
			unsubscribe()
		}
	})
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		hub.PublishTo("load-user", "load", index)
	}
}

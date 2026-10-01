package httpapi

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]string
	dropped atomic.Uint64
}

func NewHub() *Hub { return &Hub{clients: map[chan []byte]string{}} }
func (h *Hub) Subscribe() (chan []byte, func()) {
	return h.SubscribeFor("")
}
func (h *Hub) SubscribeFor(userID string) (chan []byte, func()) {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = userID
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if _, ok := h.clients[ch]; ok {
			delete(h.clients, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}
func (h *Hub) Publish(event string, data any) uint64 {
	return h.publish(event, data, "", false)
}

func (h *Hub) PublishTo(userID, event string, data any) uint64 {
	return h.publish(event, data, userID, true)
}

func (h *Hub) publish(event string, data any, userID string, targeted bool) uint64 {
	b, _ := json.Marshal(map[string]any{"event": event, "data": data})
	var dropped uint64
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch, subscriberUserID := range h.clients {
		if targeted && subscriberUserID != userID {
			continue
		}
		select {
		case ch <- b:
		default:
			dropped++
		}
	}
	h.dropped.Add(dropped)
	return dropped
}

func (h *Hub) Stats() (subscribers int, dropped uint64) {
	h.mu.RLock()
	subscribers = len(h.clients)
	h.mu.RUnlock()
	return subscribers, h.dropped.Load()
}

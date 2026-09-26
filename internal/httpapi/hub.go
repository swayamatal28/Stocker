package httpapi

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
	dropped atomic.Uint64
}

func NewHub() *Hub { return &Hub{clients: map[chan []byte]struct{}{}} }
func (h *Hub) Subscribe() (chan []byte, func()) {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
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
	b, _ := json.Marshal(map[string]any{"event": event, "data": data})
	var dropped uint64
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
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

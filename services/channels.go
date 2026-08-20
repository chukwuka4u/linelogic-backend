package services

import (
	"sync"
)

type Client struct {
	ID   string // queue ID
	Send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client // keyed by client ID
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID] = c
}

func (h *Hub) Unregister(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.clients[id]; ok {
		close(c.Send)
		delete(h.clients, id)
	}
}

// SendTo pushes an event to one specific client (e.g. one admin dashboard)
func (h *Hub) SendTo(id string, data []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.clients[id]
	if !ok {
		return false // client not connected
	}
	select {
	case c.Send <- data:
		return true
	default:
		return false // client too slow, drop
	}
}

// Broadcast to all connected clients (e.g. all admins)
func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.Send <- data:
		default:
		}
	}
}

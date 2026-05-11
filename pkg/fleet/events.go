package fleet

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrEmptyEventNodeID = errors.New("node_id is required")
	ErrEmptyEventType   = errors.New("event type is required")
)

// Event represents a normalized edge event.
type Event struct {
	ID        string         `json:"id"`
	NodeID    string         `json:"node_id"`
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
}

// EventFilter selects events from the in-memory event buffer.
type EventFilter struct {
	NodeID string
	Type   string
	Since  time.Time
}

// EventHub stores recent events in a bounded ring buffer.
type EventHub struct {
	mu     sync.RWMutex
	limit  int
	nextID int
	events []Event
}

// NewEventHub creates a bounded in-memory event hub.
func NewEventHub(limit int) *EventHub {
	if limit <= 0 {
		limit = 1000
	}
	return &EventHub{limit: limit, events: make([]Event, 0, limit)}
}

// Record validates and stores an event.
func (h *EventHub) Record(event Event) (Event, error) {
	if event.NodeID == "" {
		return Event{}, ErrEmptyEventNodeID
	}
	if event.Type == "" {
		return Event{}, ErrEmptyEventType
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Data == nil {
		event.Data = map[string]any{}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	event.ID = fmt.Sprintf("event-%d", h.nextID)
	h.events = append(h.events, event)
	if len(h.events) > h.limit {
		h.events = h.events[len(h.events)-h.limit:]
	}
	return event, nil
}

// Query returns events matching the filter.
func (h *EventHub) Query(filter EventFilter) []Event {
	h.mu.RLock()
	defer h.mu.RUnlock()

	events := make([]Event, 0, len(h.events))
	for _, event := range h.events {
		if filter.NodeID != "" && event.NodeID != filter.NodeID {
			continue
		}
		if filter.Type != "" && event.Type != filter.Type {
			continue
		}
		if !filter.Since.IsZero() && event.Timestamp.Before(filter.Since) {
			continue
		}
		events = append(events, event)
	}
	return events
}

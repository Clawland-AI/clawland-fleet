package fleet

import (
	"testing"
	"time"
)

func TestEventHubStoresBoundedEventsAndFiltersQueries(t *testing.T) {
	hub := NewEventHub(2)
	start := time.Unix(100, 0).UTC()

	if _, err := hub.Record(Event{
		NodeID:    "node-1",
		Type:      "temperature",
		Data:      map[string]any{"temperature_c": 33.1},
		Timestamp: start,
	}); err != nil {
		t.Fatalf("record first event: %v", err)
	}
	if _, err := hub.Record(Event{
		NodeID:    "node-2",
		Type:      "humidity",
		Data:      map[string]any{"humidity_pct": 71.0},
		Timestamp: start.Add(time.Second),
	}); err != nil {
		t.Fatalf("record second event: %v", err)
	}
	if _, err := hub.Record(Event{
		NodeID:    "node-1",
		Type:      "temperature",
		Data:      map[string]any{"temperature_c": 34.2},
		Timestamp: start.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("record third event: %v", err)
	}

	all := hub.Query(EventFilter{})
	if len(all) != 2 {
		t.Fatalf("expected bounded buffer to retain 2 events, got %d", len(all))
	}
	if all[0].NodeID != "node-2" || all[1].NodeID != "node-1" {
		t.Fatalf("expected oldest event to be evicted, got %#v", all)
	}

	filtered := hub.Query(EventFilter{
		NodeID: "node-1",
		Type:   "temperature",
		Since:  start.Add(time.Second),
	})
	if len(filtered) != 1 {
		t.Fatalf("expected one filtered event, got %d", len(filtered))
	}
	if filtered[0].Data["temperature_c"] != 34.2 {
		t.Fatalf("expected latest temperature event, got %#v", filtered[0])
	}
}

func TestEventHubRejectsInvalidEvents(t *testing.T) {
	hub := NewEventHub(10)

	if _, err := hub.Record(Event{Type: "temperature", Data: map[string]any{}}); err == nil {
		t.Fatal("expected missing node id to be rejected")
	}
	if _, err := hub.Record(Event{NodeID: "node-1", Data: map[string]any{}}); err == nil {
		t.Fatal("expected missing event type to be rejected")
	}
}

package fleet

import (
	"testing"
	"time"
)

func TestRegistryUpdatesHeartbeatMetricsAndMarksOffline(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&Node{
		ID:           "node-1",
		Name:         "Rack A1",
		Type:         "picclaw",
		Capabilities: []string{"temperature", "relay"},
	})

	node, ok := registry.UpdateHeartbeat("node-1", "degraded", map[string]float64{"cpu_pct": 72.5})
	if !ok {
		t.Fatal("expected heartbeat to update existing node")
	}
	if node.Status != "degraded" {
		t.Fatalf("expected degraded status, got %q", node.Status)
	}
	if node.Metrics["cpu_pct"] != 72.5 {
		t.Fatalf("expected heartbeat metrics to be stored, got %#v", node.Metrics)
	}

	node.LastSeen = time.Now().Add(-5 * time.Minute)
	offline := registry.MarkOffline(2*time.Minute, time.Now())

	if len(offline) != 1 || offline[0].ID != "node-1" {
		t.Fatalf("expected node-1 to be marked offline, got %#v", offline)
	}
	if node.Status != "offline" {
		t.Fatalf("expected offline status, got %q", node.Status)
	}
}

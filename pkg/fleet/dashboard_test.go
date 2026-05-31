package fleet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDashboardStateIsDeterministic(t *testing.T) {
	service := NewDashboardService()
	fixed := time.Date(2026, 5, 31, 17, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixed }

	service.UpsertNode(DashboardNode{ID: "node-b", Name: "B", Status: "online"})
	service.UpsertNode(DashboardNode{ID: "node-a", Name: "A", Status: "offline"})
	service.ReplaceAlerts([]Alert{
		{ID: "old", CreatedAt: fixed.Add(-2 * time.Hour)},
		{ID: "new", CreatedAt: fixed.Add(-1 * time.Minute)},
	})

	state := service.State()
	if got := state.GeneratedAt; !got.Equal(fixed) {
		t.Fatalf("GeneratedAt = %s, want %s", got, fixed)
	}
	if got := state.Nodes[0].ID; got != "node-a" {
		t.Fatalf("first node = %s, want node-a", got)
	}
	if got := state.Alerts[0].ID; got != "new" {
		t.Fatalf("first alert = %s, want new", got)
	}
}

func TestCommandEndpointQueuesKnownNode(t *testing.T) {
	service := NewDashboardService()
	service.UpsertNode(DashboardNode{ID: "node-a", Name: "A", Status: "online"})
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	body := bytes.NewBufferString(`{"node_id":"node-a","type":"restart","payload":{"reason":"test"}}`)
	response, err := http.Post(server.URL+"/fleet/command", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusAccepted)
	}

	var command Command
	if err := json.NewDecoder(response.Body).Decode(&command); err != nil {
		t.Fatal(err)
	}
	if command.Status != "pending" || command.NodeID != "node-a" || command.Type != "restart" {
		t.Fatalf("unexpected command: %+v", command)
	}
	if got := service.State().Nodes[0].PendingCommands; got != 1 {
		t.Fatalf("pending commands = %d, want 1", got)
	}
}

func TestCommandEndpointRejectsUnknownNode(t *testing.T) {
	service := NewDashboardService()
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	body := bytes.NewBufferString(`{"node_id":"missing","type":"restart"}`)
	response, err := http.Post(server.URL+"/fleet/command", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}

package fleet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerRegistersNodeAndListsOnlineNodes(t *testing.T) {
	handler := NewServer(NewRegistry())

	body := bytes.NewBufferString(`{
		"node_id": "picclaw-1",
		"name": "greenhouse edge",
		"type": "picclaw",
		"capabilities": ["temp", "humidity"],
		"location": "greenhouse-a"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/fleet/register", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var created Node
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if created.ID != "picclaw-1" || created.Status != "online" || created.LastSeen.IsZero() {
		t.Fatalf("unexpected register response: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/fleet/nodes?status=online", nil)
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var nodes []Node
	if err := json.NewDecoder(rec.Body).Decode(&nodes); err != nil {
		t.Fatalf("decode nodes response: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 online node, got %d", len(nodes))
	}
	if nodes[0].ID != "picclaw-1" || nodes[0].Status != "online" {
		t.Fatalf("unexpected node: %+v", nodes[0])
	}
}

func TestServerHeartbeatUpdatesStatusAndMetrics(t *testing.T) {
	handler := NewServer(NewRegistry())
	registerNode(t, handler, `{"node_id":"picclaw-1","type":"picclaw"}`)

	body := bytes.NewBufferString(`{
		"node_id": "picclaw-1",
		"status": "degraded",
		"metrics": {"cpu": 0.71, "battery": 0.42}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/fleet/heartbeat", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/fleet/nodes/picclaw-1", nil)
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var node Node
	if err := json.NewDecoder(rec.Body).Decode(&node); err != nil {
		t.Fatalf("decode node response: %v", err)
	}
	if node.Status != "degraded" {
		t.Fatalf("expected node status degraded, got %q", node.Status)
	}
	if node.Metrics["cpu"] != 0.71 || node.Metrics["battery"] != 0.42 {
		t.Fatalf("unexpected metrics: %+v", node.Metrics)
	}
}

func TestRegistryMarksNodeOfflineAfterHeartbeatTimeout(t *testing.T) {
	registry := NewRegistryWithHeartbeatTimeout(5 * time.Millisecond)
	registry.Register(&Node{ID: "picclaw-1"})

	time.Sleep(10 * time.Millisecond)

	nodes := registry.List()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Status != "offline" {
		t.Fatalf("expected node to be offline after timeout, got %q", nodes[0].Status)
	}
}

func TestServerRejectsInvalidRegisterPayloads(t *testing.T) {
	handler := NewServer(NewRegistry())

	for _, tc := range []struct {
		name    string
		payload string
	}{
		{name: "invalid json", payload: `{`},
		{name: "missing node id", payload: `{"type":"picclaw"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/fleet/register", bytes.NewBufferString(tc.payload))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, rec.Code, rec.Body.String())
			}
		})
	}
}

func registerNode(t *testing.T, handler http.Handler, payload string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/fleet/register", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register node status %d, body %s", rec.Code, rec.Body.String())
	}
}

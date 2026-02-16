package fleet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterHandler(t *testing.T) {
	reg := NewRegistry()
	handler := RegisterHandler(reg)

	req := RegisterRequest{
		NodeID:       "test-node-1",
		NodeType:     "microclaw",
		Capabilities: []string{"dht22"},
		Location:     "office",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp RegisterResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.NodeID != "test-node-1" {
		t.Errorf("Expected node_id 'test-node-1', got '%s'", resp.NodeID)
	}

	if resp.Status != "registered" {
		t.Errorf("Expected status 'registered', got '%s'", resp.Status)
	}

	// Verify node was actually registered
	node := reg.Get("test-node-1")
	if node == nil {
		t.Fatal("Node not found in registry")
	}
}

func TestRegisterHandlerMissingFields(t *testing.T) {
	reg := NewRegistry()
	handler := RegisterHandler(reg)

	req := RegisterRequest{
		NodeType: "microclaw",
		// Missing NodeID
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRegisterHandlerInvalidMethod(t *testing.T) {
	reg := NewRegistry()
	handler := RegisterHandler(reg)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/register", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHeartbeatHandler(t *testing.T) {
	reg := NewRegistry()

	// Register a node first
	node := &Node{
		ID:   "test-node-1",
		Type: "microclaw",
	}
	reg.Register(node)

	handler := HeartbeatHandler(reg)

	req := HeartbeatRequest{
		NodeID:        "test-node-1",
		UptimeSeconds: 3600,
		Status:        "online",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp HeartbeatResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.NodeID != "test-node-1" {
		t.Errorf("Expected node_id 'test-node-1', got '%s'", resp.NodeID)
	}

	if resp.NextCheck != 60 {
		t.Errorf("Expected next_check 60, got %d", resp.NextCheck)
	}
}

func TestHeartbeatHandlerNodeNotFound(t *testing.T) {
	reg := NewRegistry()
	handler := HeartbeatHandler(reg)

	req := HeartbeatRequest{
		NodeID: "non-existent",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestListNodesHandler(t *testing.T) {
	reg := NewRegistry()

	// Register multiple nodes
	for i := 0; i < 3; i++ {
		node := &Node{
			ID:   string(rune('A' + i)),
			Type: "microclaw",
		}
		reg.Register(node)
	}

	handler := ListNodesHandler(reg)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/nodes", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	count := int(resp["count"].(float64))
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestListNodesHandlerWithFilter(t *testing.T) {
	reg := NewRegistry()

	// Register nodes of different types
	reg.Register(&Node{ID: "micro-1", Type: "microclaw"})
	reg.Register(&Node{ID: "pico-1", Type: "picclaw"})
	reg.Register(&Node{ID: "micro-2", Type: "microclaw"})

	handler := ListNodesHandler(reg)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/nodes?node_type=microclaw", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	count := int(resp["count"].(float64))
	if count != 2 {
		t.Errorf("Expected count 2 (filtered microclaw only), got %d", count)
	}
}

func TestGetNodeHandler(t *testing.T) {
	reg := NewRegistry()

	node := &Node{
		ID:       "test-node-1",
		Type:     "microclaw",
		Location: "office",
	}
	reg.Register(node)

	handler := GetNodeHandler(reg)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/nodes/test-node-1", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Node
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != "test-node-1" {
		t.Errorf("Expected ID 'test-node-1', got '%s'", resp.ID)
	}

	if resp.Location != "office" {
		t.Errorf("Expected location 'office', got '%s'", resp.Location)
	}
}

func TestGetNodeHandlerNotFound(t *testing.T) {
	reg := NewRegistry()
	handler := GetNodeHandler(reg)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/nodes/non-existent", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

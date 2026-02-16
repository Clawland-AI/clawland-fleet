package fleet

import (
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if reg.nodes == nil {
		t.Fatal("Registry nodes map is nil")
	}
}

func TestRegister(t *testing.T) {
	reg := NewRegistry()
	node := &Node{
		ID:           "test-node-1",
		Name:         "Test Node",
		Type:         "microclaw",
		Capabilities: []string{"dht22", "mqtt"},
		Location:     "test-location",
	}

	reg.Register(node)

	// Verify node was registered
	if len(reg.nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(reg.nodes))
	}

	retrieved := reg.Get("test-node-1")
	if retrieved == nil {
		t.Fatal("Node not found after registration")
	}

	if retrieved.Status != "online" {
		t.Errorf("Expected status 'online', got '%s'", retrieved.Status)
	}

	if time.Since(retrieved.LastSeen) > time.Second {
		t.Error("LastSeen not set properly")
	}
}

func TestHeartbeat(t *testing.T) {
	reg := NewRegistry()
	node := &Node{
		ID:   "test-node-1",
		Name: "Test Node",
		Type: "microclaw",
	}
	reg.Register(node)

	// Wait a bit then send heartbeat
	time.Sleep(100 * time.Millisecond)
	originalTime := reg.nodes["test-node-1"].LastSeen

	time.Sleep(100 * time.Millisecond)
	success := reg.Heartbeat("test-node-1")
	if !success {
		t.Fatal("Heartbeat failed for existing node")
	}

	newTime := reg.nodes["test-node-1"].LastSeen
	if !newTime.After(originalTime) {
		t.Error("LastSeen not updated after heartbeat")
	}
}

func TestHeartbeatNonExistentNode(t *testing.T) {
	reg := NewRegistry()
	success := reg.Heartbeat("non-existent")
	if success {
		t.Error("Heartbeat should fail for non-existent node")
	}
}

func TestList(t *testing.T) {
	reg := NewRegistry()

	// Register multiple nodes
	for i := 0; i < 3; i++ {
		node := &Node{
			ID:   string(rune('A' + i)),
			Name: string(rune('A' + i)),
			Type: "microclaw",
		}
		reg.Register(node)
	}

	nodes := reg.List()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(nodes))
	}
}

func TestGet(t *testing.T) {
	reg := NewRegistry()
	node := &Node{
		ID:   "test-node-1",
		Name: "Test Node",
		Type: "microclaw",
	}
	reg.Register(node)

	retrieved := reg.Get("test-node-1")
	if retrieved == nil {
		t.Fatal("Get returned nil for existing node")
	}

	if retrieved.ID != "test-node-1" {
		t.Errorf("Expected ID 'test-node-1', got '%s'", retrieved.ID)
	}

	// Test non-existent node
	missing := reg.Get("non-existent")
	if missing != nil {
		t.Error("Get should return nil for non-existent node")
	}
}

func TestMarkOffline(t *testing.T) {
	reg := NewRegistry()

	// Register nodes
	node1 := &Node{
		ID:       "old-node",
		Name:     "Old Node",
		Type:     "microclaw",
		Status:   "online",
		LastSeen: time.Now().Add(-5 * time.Minute),
	}
	node2 := &Node{
		ID:       "new-node",
		Name:     "New Node",
		Type:     "microclaw",
		Status:   "online",
		LastSeen: time.Now(),
	}

	reg.nodes["old-node"] = node1
	reg.nodes["new-node"] = node2

	// Mark nodes offline if no heartbeat for 3 minutes
	count := reg.MarkOffline(3 * time.Minute)

	if count != 1 {
		t.Errorf("Expected 1 node marked offline, got %d", count)
	}

	if reg.nodes["old-node"].Status != "offline" {
		t.Error("Old node should be marked offline")
	}

	if reg.nodes["new-node"].Status != "online" {
		t.Error("New node should still be online")
	}
}

func TestRegisterUpdateExisting(t *testing.T) {
	reg := NewRegistry()

	node1 := &Node{
		ID:       "test-node",
		Name:     "Test Node",
		Type:     "microclaw",
		Location: "location-1",
	}
	reg.Register(node1)

	// Register again with updated location
	node2 := &Node{
		ID:       "test-node",
		Name:     "Test Node",
		Type:     "microclaw",
		Location: "location-2",
	}
	reg.Register(node2)

	// Should still have only 1 node
	if len(reg.nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(reg.nodes))
	}

	// Location should be updated
	retrieved := reg.Get("test-node")
	if retrieved.Location != "location-2" {
		t.Errorf("Expected location 'location-2', got '%s'", retrieved.Location)
	}
}

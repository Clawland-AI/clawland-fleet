// Package fleet provides the core Fleet Manager functionality.
package fleet

import (
	"sync"
	"time"
)

// Node represents a registered edge agent.
type Node struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"` // picclaw, nanoclaw, microclaw
	Capabilities []string          `json:"capabilities"`
	Location     string            `json:"location,omitempty"`
	LastSeen     time.Time         `json:"last_seen"`
	Status       string            `json:"status"` // online, offline, degraded
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Registry manages registered edge nodes.
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]*Node
}

// NewRegistry creates a new node registry.
func NewRegistry() *Registry {
	return &Registry{nodes: make(map[string]*Node)}
}

// Register adds or updates a node in the registry.
func (r *Registry) Register(node *Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	node.LastSeen = time.Now()
	node.Status = "online"
	r.nodes[node.ID] = node
}

// Heartbeat updates the last seen time for a node.
func (r *Registry) Heartbeat(nodeID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[nodeID]; ok {
		n.LastSeen = time.Now()
		n.Status = "online"
		return true
	}
	return false
}

// List returns all registered nodes.
func (r *Registry) List() []*Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nodes := make([]*Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// Get retrieves a specific node by ID.
func (r *Registry) Get(nodeID string) *Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nodes[nodeID]
}

// MarkOffline marks nodes as offline if they haven't sent heartbeat recently.
func (r *Registry) MarkOffline(timeout time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	now := time.Now()
	for _, n := range r.nodes {
		if n.Status == "online" && now.Sub(n.LastSeen) > timeout {
			n.Status = "offline"
			count++
		}
	}
	return count
}
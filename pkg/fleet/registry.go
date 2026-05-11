// Package fleet provides the core Fleet Manager functionality.
package fleet

import (
	"sync"
	"time"
)

// Node represents a registered edge agent.
type Node struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Type         string             `json:"type"` // picclaw, nanoclaw, microclaw
	Capabilities []string           `json:"capabilities"`
	Location     string             `json:"location,omitempty"`
	LastSeen     time.Time          `json:"last_seen"`
	Status       string             `json:"status"` // online, offline, degraded
	Metrics      map[string]float64 `json:"metrics,omitempty"`
	Metadata     map[string]string  `json:"metadata,omitempty"`
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
	_, ok := r.UpdateHeartbeat(nodeID, "online", nil)
	return ok
}

// UpdateHeartbeat updates liveness, status, and metrics for a node.
func (r *Registry) UpdateHeartbeat(nodeID, status string, metrics map[string]float64) (*Node, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[nodeID]; ok {
		n.LastSeen = time.Now()
		if status == "" {
			status = "online"
		}
		n.Status = status
		if metrics != nil {
			n.Metrics = metrics
		}
		return n, true
	}
	return nil, false
}

// Get returns a registered node by ID.
func (r *Registry) Get(nodeID string) (*Node, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	node, ok := r.nodes[nodeID]
	return node, ok
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

// ListByStatus returns nodes that match the given status.
func (r *Registry) ListByStatus(status string) []*Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nodes := make([]*Node, 0)
	for _, n := range r.nodes {
		if n.Status == status {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// MarkOffline marks nodes offline when their last heartbeat is too old.
func (r *Registry) MarkOffline(timeout time.Duration, now time.Time) []*Node {
	r.mu.Lock()
	defer r.mu.Unlock()
	offline := make([]*Node, 0)
	for _, n := range r.nodes {
		if n.Status != "offline" && now.Sub(n.LastSeen) >= timeout {
			n.Status = "offline"
			offline = append(offline, n)
		}
	}
	return offline
}

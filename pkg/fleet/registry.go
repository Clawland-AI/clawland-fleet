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
	mu               sync.RWMutex
	nodes            map[string]*Node
	heartbeatTimeout time.Duration
}

const defaultHeartbeatTimeout = 2 * time.Minute

// NewRegistry creates a new node registry.
func NewRegistry() *Registry {
	return NewRegistryWithHeartbeatTimeout(defaultHeartbeatTimeout)
}

// NewRegistryWithHeartbeatTimeout creates a registry with a custom offline timeout.
func NewRegistryWithHeartbeatTimeout(timeout time.Duration) *Registry {
	if timeout <= 0 {
		timeout = defaultHeartbeatTimeout
	}
	return &Registry{nodes: make(map[string]*Node), heartbeatTimeout: timeout}
}

// Register adds or updates a node in the registry.
func (r *Registry) Register(node *Node) *Node {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored := cloneNode(node)
	stored.LastSeen = time.Now()
	stored.Status = "online"
	r.nodes[stored.ID] = stored
	return cloneNode(stored)
}

// Heartbeat updates the last seen time for a node.
func (r *Registry) Heartbeat(nodeID string) bool {
	_, ok := r.UpdateHeartbeat(nodeID, "online", nil)
	return ok
}

// UpdateHeartbeat updates node status, metrics, and last seen time.
func (r *Registry) UpdateHeartbeat(nodeID string, status string, metrics map[string]float64) (*Node, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[nodeID]; ok {
		n.LastSeen = time.Now()
		if status == "" {
			status = "online"
		}
		n.Status = status
		if metrics != nil {
			n.Metrics = cloneFloatMap(metrics)
		}
		return cloneNode(n), true
	}
	return nil, false
}

// Get returns a registered node by ID.
func (r *Registry) Get(nodeID string) (*Node, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshStatusesLocked(time.Now())
	n, ok := r.nodes[nodeID]
	if !ok {
		return nil, false
	}
	return cloneNode(n), true
}

// List returns all registered nodes.
func (r *Registry) List() []*Node {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshStatusesLocked(time.Now())
	nodes := make([]*Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		nodes = append(nodes, cloneNode(n))
	}
	return nodes
}

func (r *Registry) refreshStatusesLocked(now time.Time) {
	for _, node := range r.nodes {
		if node.Status == "offline" {
			continue
		}
		if now.Sub(node.LastSeen) > r.heartbeatTimeout {
			node.Status = "offline"
		}
	}
}

func cloneNode(node *Node) *Node {
	if node == nil {
		return &Node{}
	}
	clone := *node
	if node.Capabilities != nil {
		clone.Capabilities = append([]string(nil), node.Capabilities...)
	}
	clone.Metrics = cloneFloatMap(node.Metrics)
	if node.Metadata != nil {
		clone.Metadata = make(map[string]string, len(node.Metadata))
		for key, value := range node.Metadata {
			clone.Metadata[key] = value
		}
	}
	return &clone
}

func cloneFloatMap(values map[string]float64) map[string]float64 {
	if values == nil {
		return nil
	}
	clone := make(map[string]float64, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

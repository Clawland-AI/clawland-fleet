// Package fleet provides HTTP handlers for Fleet Manager API.
package fleet

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// RegisterRequest represents the node registration payload.
type RegisterRequest struct {
	NodeID       string            `json:"node_id"`
	NodeType     string            `json:"node_type"`
	Capabilities []string          `json:"capabilities"`
	Location     string            `json:"location,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// RegisterResponse is returned after successful registration.
type RegisterResponse struct {
	NodeID    string    `json:"node_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// HeartbeatRequest represents a heartbeat ping.
type HeartbeatRequest struct {
	NodeID         string            `json:"node_id"`
	UptimeSeconds  int64             `json:"uptime_seconds,omitempty"`
	FreeMemoryKB   int64             `json:"free_memory_kb,omitempty"`
	SensorsActive  int               `json:"sensors_active,omitempty"`
	Status         string            `json:"status,omitempty"`
	Metrics        map[string]string `json:"metrics,omitempty"`
}

// HeartbeatResponse acknowledges the heartbeat.
type HeartbeatResponse struct {
	NodeID    string    `json:"node_id"`
	Received  time.Time `json:"received"`
	NextCheck int       `json:"next_check_seconds"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RegisterHandler handles POST /fleet/register.
func RegisterHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
			return
		}

		// Validate required fields
		if req.NodeID == "" || req.NodeType == "" {
			respondError(w, http.StatusBadRequest, "Missing required fields", "node_id and node_type are required")
			return
		}

		// Create node
		node := &Node{
			ID:           req.NodeID,
			Name:         req.NodeID,
			Type:         req.NodeType,
			Capabilities: req.Capabilities,
			Location:     req.Location,
			Metadata:     req.Metadata,
			Status:       "online",
			LastSeen:     time.Now(),
		}

		registry.Register(node)
		log.Printf("[REGISTER] Node %s (%s) registered from %s", req.NodeID, req.NodeType, r.RemoteAddr)

		resp := RegisterResponse{
			NodeID:    req.NodeID,
			Status:    "registered",
			Message:   "Node successfully registered",
			Timestamp: time.Now(),
		}

		respondJSON(w, http.StatusOK, resp)
	}
}

// HeartbeatHandler handles POST /fleet/heartbeat.
func HeartbeatHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
			return
		}

		if req.NodeID == "" {
			respondError(w, http.StatusBadRequest, "Missing node_id", "node_id is required")
			return
		}

		// Update heartbeat
		found := registry.Heartbeat(req.NodeID)
		if !found {
			respondError(w, http.StatusNotFound, "Node not found", "Node must register first")
			return
		}

		log.Printf("[HEARTBEAT] Node %s (uptime: %ds, status: %s)", req.NodeID, req.UptimeSeconds, req.Status)

		resp := HeartbeatResponse{
			NodeID:    req.NodeID,
			Received:  time.Now(),
			NextCheck: 60, // Expect next heartbeat in 60 seconds
		}

		respondJSON(w, http.StatusOK, resp)
	}
}

// ListNodesHandler handles GET /fleet/nodes.
func ListNodesHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get query parameters for filtering
		nodeType := r.URL.Query().Get("node_type")
		status := r.URL.Query().Get("status")

		nodes := registry.List()

		// Filter by node_type if provided
		if nodeType != "" {
			filtered := make([]*Node, 0)
			for _, n := range nodes {
				if strings.EqualFold(n.Type, nodeType) {
					filtered = append(filtered, n)
				}
			}
			nodes = filtered
		}

		// Filter by status if provided
		if status != "" {
			filtered := make([]*Node, 0)
			for _, n := range nodes {
				if strings.EqualFold(n.Status, status) {
					filtered = append(filtered, n)
				}
			}
			nodes = filtered
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"nodes": nodes,
			"count": len(nodes),
		})
	}
}

// GetNodeHandler handles GET /fleet/nodes/{node_id}.
func GetNodeHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract node_id from path
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/fleet/nodes/")
		nodeID := strings.TrimSpace(path)

		if nodeID == "" {
			respondError(w, http.StatusBadRequest, "Missing node_id", "node_id must be provided in path")
			return
		}

		node := registry.Get(nodeID)
		if node == nil {
			respondError(w, http.StatusNotFound, "Node not found", "No node with ID "+nodeID)
			return
		}

		respondJSON(w, http.StatusOK, node)
	}
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error JSON response.
func respondError(w http.ResponseWriter, status int, error string, message string) {
	respondJSON(w, status, ErrorResponse{
		Error:   error,
		Message: message,
	})
}

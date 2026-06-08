package fleet

import (
	"encoding/json"
	"net/http"
	"strings"
)

// NewServer builds the Fleet Manager HTTP API.
func NewServer(registry *Registry) http.Handler {
	if registry == nil {
		registry = NewRegistry()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/fleet/register", handleRegister(registry))
	mux.HandleFunc("/fleet/heartbeat", handleHeartbeat(registry))
	mux.HandleFunc("/fleet/nodes", handleListNodes(registry))
	mux.HandleFunc("/fleet/nodes/", handleGetNode(registry))
	return mux
}

type registerRequest struct {
	NodeID       string            `json:"node_id"`
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Capabilities []string          `json:"capabilities"`
	Location     string            `json:"location"`
	Metadata     map[string]string `json:"metadata"`
}

type heartbeatRequest struct {
	NodeID  string             `json:"node_id"`
	ID      string             `json:"id"`
	Status  string             `json:"status"`
	Metrics map[string]float64 `json:"metrics"`
}

func handleRegister(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}

		id := strings.TrimSpace(req.NodeID)
		if id == "" {
			id = strings.TrimSpace(req.ID)
		}
		if id == "" {
			writeError(w, http.StatusBadRequest, "node_id is required")
			return
		}

		node := &Node{
			ID:           id,
			Name:         req.Name,
			Type:         req.Type,
			Capabilities: req.Capabilities,
			Location:     req.Location,
			Metadata:     req.Metadata,
		}
		registered := registry.Register(node)

		writeJSON(w, http.StatusCreated, registered)
	}
}

func handleHeartbeat(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		var req heartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}

		id := strings.TrimSpace(req.NodeID)
		if id == "" {
			id = strings.TrimSpace(req.ID)
		}
		if id == "" {
			writeError(w, http.StatusBadRequest, "node_id is required")
			return
		}

		node, ok := registry.UpdateHeartbeat(id, strings.TrimSpace(req.Status), req.Metrics)
		if !ok {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}

		writeJSON(w, http.StatusOK, node)
	}
}

func handleListNodes(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		status := strings.TrimSpace(r.URL.Query().Get("status"))
		nodes := registry.List()
		if status != "" {
			filtered := nodes[:0]
			for _, node := range nodes {
				if node.Status == status {
					filtered = append(filtered, node)
				}
			}
			nodes = filtered
		}

		writeJSON(w, http.StatusOK, nodes)
	}
}

func handleGetNode(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		nodeID := strings.TrimPrefix(r.URL.Path, "/fleet/nodes/")
		nodeID = strings.TrimSpace(nodeID)
		if nodeID == "" {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}

		node, ok := registry.Get(nodeID)
		if !ok {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}

		writeJSON(w, http.StatusOK, node)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

package fleet

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Server exposes Fleet Manager HTTP endpoints.
type Server struct {
	registry *Registry
	events   *EventHub
	mux      *http.ServeMux
}

// NewServer creates a Fleet Manager HTTP server.
func NewServer(registry *Registry, events *EventHub) *Server {
	if registry == nil {
		registry = NewRegistry()
	}
	if events == nil {
		events = NewEventHub(1000)
	}
	server := &Server{
		registry: registry,
		events:   events,
		mux:      http.NewServeMux(),
	}
	server.mux.HandleFunc("/fleet/register", server.handleRegister)
	server.mux.HandleFunc("/fleet/heartbeat", server.handleHeartbeat)
	server.mux.HandleFunc("/fleet/events", server.handleEvents)
	server.mux.HandleFunc("/fleet/nodes/", server.handleNode)
	server.mux.HandleFunc("/fleet/nodes", server.handleNodes)
	return server
}

// ServeHTTP routes Fleet Manager API requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

type registerRequest struct {
	NodeID       string            `json:"node_id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Capabilities []string          `json:"capabilities"`
	Location     string            `json:"location"`
	Metadata     map[string]string `json:"metadata"`
}

type heartbeatRequest struct {
	NodeID  string             `json:"node_id"`
	Status  string             `json:"status"`
	Metrics map[string]float64 `json:"metrics"`
}

type heartbeatResponse struct {
	OK   bool  `json:"ok"`
	Node *Node `json:"node"`
}

type nodesResponse struct {
	Nodes []*Node `json:"nodes"`
}

type eventsResponse struct {
	Events []Event `json:"events"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if request.NodeID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "node_id is required"})
		return
	}
	node := &Node{
		ID:           request.NodeID,
		Name:         request.Name,
		Type:         request.Type,
		Capabilities: request.Capabilities,
		Location:     request.Location,
		Metadata:     request.Metadata,
	}
	s.registry.Register(node)
	writeJSON(w, http.StatusCreated, node)
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var request heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if request.NodeID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "node_id is required"})
		return
	}
	node, ok := s.registry.UpdateHeartbeat(request.NodeID, request.Status, request.Metrics)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "node not found"})
		return
	}
	writeJSON(w, http.StatusOK, heartbeatResponse{OK: true, Node: node})
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" {
		writeJSON(w, http.StatusOK, nodesResponse{Nodes: s.registry.ListByStatus(status)})
		return
	}
	writeJSON(w, http.StatusOK, nodesResponse{Nodes: s.registry.List()})
}

func (s *Server) handleNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	nodeID := strings.TrimPrefix(r.URL.Path, "/fleet/nodes/")
	if nodeID == "" || strings.Contains(nodeID, "/") {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "node not found"})
		return
	}
	node, ok := s.registry.Get(nodeID)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "node not found"})
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handlePostEvent(w, r)
	case http.MethodGet:
		s.handleListEvents(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
	}
}

func (s *Server) handlePostEvent(w http.ResponseWriter, r *http.Request) {
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if event.NodeID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrEmptyEventNodeID.Error()})
		return
	}
	if event.Type == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrEmptyEventType.Error()})
		return
	}
	if _, ok := s.registry.Get(event.NodeID); !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "node not found"})
		return
	}
	stored, err := s.events.Record(event)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, stored)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	filter := EventFilter{
		NodeID: r.URL.Query().Get("node_id"),
		Type:   r.URL.Query().Get("type"),
	}
	if since := r.URL.Query().Get("since"); since != "" {
		parsed, err := time.Parse(time.RFC3339, since)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid since timestamp"})
			return
		}
		filter.Since = parsed
	}
	writeJSON(w, http.StatusOK, eventsResponse{Events: s.events.Query(filter)})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

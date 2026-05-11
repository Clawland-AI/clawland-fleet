package fleet

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Server exposes Fleet Manager HTTP endpoints.
type Server struct {
	registry   *Registry
	dispatcher *Dispatcher
	mux        *http.ServeMux
}

// NewServer creates a Fleet Manager HTTP server.
func NewServer(registry *Registry, dispatcher *Dispatcher) *Server {
	if registry == nil {
		registry = NewRegistry()
	}
	if dispatcher == nil {
		dispatcher = NewDispatcher()
	}

	server := &Server{
		registry:   registry,
		dispatcher: dispatcher,
		mux:        http.NewServeMux(),
	}
	server.mux.HandleFunc("/fleet/command", server.handleCommand)
	server.mux.HandleFunc("/fleet/command/status", server.handleCommandStatus)
	server.mux.HandleFunc("/fleet/heartbeat", server.handleHeartbeat)
	server.mux.HandleFunc("/fleet/register", server.handleRegister)
	return server
}

// ServeHTTP routes Fleet Manager API requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

type commandRequest struct {
	NodeID  string         `json:"node_id"`
	Type    CommandType    `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

type heartbeatRequest struct {
	NodeID string `json:"node_id"`
}

type commandStatusRequest struct {
	CommandID string        `json:"command_id"`
	Status    CommandStatus `json:"status"`
	Error     string        `json:"error,omitempty"`
}

type heartbeatResponse struct {
	OK       bool      `json:"ok"`
	Commands []Command `json:"commands"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleGetCommand(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var request commandRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if request.NodeID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrEmptyNodeID.Error()})
		return
	}
	if !validCommandType(request.Type) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrInvalidCommandType.Error()})
		return
	}
	if _, ok := s.registry.Get(request.NodeID); !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "node not found"})
		return
	}

	command, err := s.dispatcher.Dispatch(request.NodeID, request.Type, request.Payload)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, ErrEmptyNodeID) && !errors.Is(err, ErrInvalidCommandType) {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, command)
}

func (s *Server) handleGetCommand(w http.ResponseWriter, r *http.Request) {
	commandID := r.URL.Query().Get("id")
	if commandID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "command id is required"})
		return
	}
	command, ok := s.dispatcher.Command(commandID)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: ErrCommandNotFound.Error()})
		return
	}

	writeJSON(w, http.StatusOK, command)
}

func (s *Server) handleCommandStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var request commandStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if request.CommandID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "command_id is required"})
		return
	}
	if err := s.dispatcher.UpdateStatus(request.CommandID, request.Status, request.Error); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrCommandNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, errorResponse{Error: err.Error()})
		return
	}

	command, _ := s.dispatcher.Command(request.CommandID)
	writeJSON(w, http.StatusOK, command)
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
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrEmptyNodeID.Error()})
		return
	}
	if ok := s.registry.Heartbeat(request.NodeID); !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "node not found"})
		return
	}

	writeJSON(w, http.StatusOK, heartbeatResponse{
		OK:       true,
		Commands: s.dispatcher.DeliverPending(request.NodeID),
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var node Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if node.ID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}

	s.registry.Register(&node)
	writeJSON(w, http.StatusCreated, node)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

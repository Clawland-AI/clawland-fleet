package fleet

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// GeoLocation describes where a fleet node appears on the dashboard map.
type GeoLocation struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	MapX int     `json:"map_x"`
	MapY int     `json:"map_y"`
}

// NodeMetrics contains the small health snapshot shown in the dashboard.
type NodeMetrics struct {
	Battery     int     `json:"battery"`
	Temperature float64 `json:"temperature_c"`
	SignalDBM   int     `json:"signal_dbm"`
}

// DashboardNode is the JSON shape consumed by the web dashboard.
type DashboardNode struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Type            string      `json:"type"`
	Status          string      `json:"status"`
	PendingCommands int         `json:"pending_commands"`
	Capabilities    []string    `json:"capabilities"`
	Location        GeoLocation `json:"location"`
	Metrics         NodeMetrics `json:"metrics"`
	LastSeen        time.Time   `json:"last_seen"`
}

// Alert represents an open Fleet Manager incident.
type Alert struct {
	ID        string    `json:"id"`
	NodeID    string    `json:"node_id"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Command tracks a cloud-to-edge command queued from the dashboard.
type Command struct {
	ID        string          `json:"id"`
	NodeID    string          `json:"node_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

// DashboardState is returned by GET /fleet/dashboard/state.
type DashboardState struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Nodes       []DashboardNode `json:"nodes"`
	Alerts      []Alert         `json:"alerts"`
	Commands    []Command       `json:"commands"`
}

// DashboardService stores the in-memory data needed by the dashboard preview.
type DashboardService struct {
	mu       sync.RWMutex
	nodes    map[string]DashboardNode
	alerts   []Alert
	commands []Command
	now      func() time.Time
}

// NewDashboardService creates an empty dashboard service.
func NewDashboardService() *DashboardService {
	return &DashboardService{
		nodes: make(map[string]DashboardNode),
		now:   time.Now,
	}
}

// NewDemoDashboardService returns a service seeded with realistic sample data.
func NewDemoDashboardService() *DashboardService {
	service := NewDashboardService()
	now := time.Now().UTC()
	for _, node := range []DashboardNode{
		{
			ID:              "pond-a-picoclaw",
			Name:            "Pond A Gateway",
			Type:            "picclaw",
			Status:          "online",
			Capabilities:    []string{"aquaculture-monitoring", "mqtt", "edge-api"},
			Location:        GeoLocation{Name: "North pond", Lat: 37.781, Lng: -122.404, MapX: 32, MapY: 48},
			Metrics:         NodeMetrics{Battery: 92, Temperature: 24.1, SignalDBM: -54},
			LastSeen:        now.Add(-22 * time.Second),
			PendingCommands: 0,
		},
		{
			ID:              "greenhouse-3-nanoclaw",
			Name:            "Greenhouse 3",
			Type:            "nanoclaw",
			Status:          "degraded",
			Capabilities:    []string{"greenhouse", "relay-control", "camera"},
			Location:        GeoLocation{Name: "West greenhouse", Lat: 37.786, Lng: -122.411, MapX: 58, MapY: 36},
			Metrics:         NodeMetrics{Battery: 41, Temperature: 31.6, SignalDBM: -82},
			LastSeen:        now.Add(-2 * time.Minute),
			PendingCommands: 2,
		},
		{
			ID:              "cold-chain-van-12",
			Name:            "Van 12",
			Type:            "picclaw",
			Status:          "online",
			Capabilities:    []string{"cold-chain", "gps", "audit-log"},
			Location:        GeoLocation{Name: "Downtown route", Lat: 37.774, Lng: -122.419, MapX: 44, MapY: 66},
			Metrics:         NodeMetrics{Battery: 77, Temperature: 3.8, SignalDBM: -63},
			LastSeen:        now.Add(-47 * time.Second),
			PendingCommands: 1,
		},
		{
			ID:              "warehouse-door-2",
			Name:            "Warehouse Door 2",
			Type:            "microclaw",
			Status:          "offline",
			Capabilities:    []string{"door-sensor", "pir"},
			Location:        GeoLocation{Name: "Loading dock", Lat: 37.769, Lng: -122.398, MapX: 72, MapY: 55},
			Metrics:         NodeMetrics{Battery: 12, Temperature: 18.4, SignalDBM: -96},
			LastSeen:        now.Add(-12 * time.Minute),
			PendingCommands: 0,
		},
	} {
		service.UpsertNode(node)
	}
	service.ReplaceAlerts([]Alert{
		{ID: "alert-001", NodeID: "warehouse-door-2", Severity: "critical", Title: "Node missed three heartbeats", CreatedAt: now.Add(-12 * time.Minute)},
		{ID: "alert-002", NodeID: "greenhouse-3-nanoclaw", Severity: "warning", Title: "Signal quality below deployment threshold", CreatedAt: now.Add(-6 * time.Minute)},
		{ID: "alert-003", NodeID: "cold-chain-van-12", Severity: "info", Title: "Firmware update waiting for maintenance window", CreatedAt: now.Add(-98 * time.Minute)},
	})
	return service
}

// UpsertNode adds or replaces a node snapshot.
func (s *DashboardService) UpsertNode(node DashboardNode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node.ID] = node
}

// ReplaceAlerts swaps the currently open alert list.
func (s *DashboardService) ReplaceAlerts(alerts []Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append([]Alert(nil), alerts...)
}

// State returns a deterministic dashboard snapshot.
func (s *DashboardService) State() DashboardState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]DashboardNode, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	alerts := append([]Alert(nil), s.alerts...)
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].CreatedAt.After(alerts[j].CreatedAt)
	})

	commands := append([]Command(nil), s.commands...)
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].CreatedAt.After(commands[j].CreatedAt)
	})

	return DashboardState{
		GeneratedAt: s.now().UTC(),
		Nodes:       nodes,
		Alerts:      alerts,
		Commands:    commands,
	}
}

// QueueCommand records a command for delivery to an edge node.
func (s *DashboardService) QueueCommand(nodeID, commandType string, payload json.RawMessage) (Command, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nodes[nodeID]; !ok {
		return Command{}, false
	}

	now := s.now().UTC()
	command := Command{
		ID:        "cmd-" + now.Format("20060102150405.000000000"),
		NodeID:    nodeID,
		Type:      commandType,
		Payload:   payload,
		Status:    "pending",
		CreatedAt: now,
	}
	s.commands = append(s.commands, command)
	node := s.nodes[nodeID]
	node.PendingCommands++
	s.nodes[nodeID] = node
	return command, true
}

// Handler returns the API routes used by the dashboard.
func (s *DashboardService) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/fleet/dashboard/state", s.handleState)
	mux.HandleFunc("/fleet/nodes", s.handleNodes)
	mux.HandleFunc("/fleet/alerts", s.handleAlerts)
	mux.HandleFunc("/fleet/command", s.handleCommand)
	return mux
}

func (s *DashboardService) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, s.State())
}

func (s *DashboardService) handleNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, s.State().Nodes)
}

func (s *DashboardService) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, s.State().Alerts)
}

func (s *DashboardService) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var request struct {
		NodeID  string          `json:"node_id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid command payload")
		return
	}

	request.NodeID = strings.TrimSpace(request.NodeID)
	request.Type = strings.TrimSpace(request.Type)
	if request.NodeID == "" || request.Type == "" {
		writeError(w, http.StatusBadRequest, "node_id and type are required")
		return
	}

	command, ok := s.QueueCommand(request.NodeID, request.Type, request.Payload)
	if !ok {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	writeJSON(w, http.StatusAccepted, command)
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

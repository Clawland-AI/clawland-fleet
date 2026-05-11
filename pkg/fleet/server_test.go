package fleet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerHandlesNodeLifecycleAndEvents(t *testing.T) {
	server := NewServer(NewRegistry(), NewEventHub(10))

	register := httptest.NewRequest(http.MethodPost, "/fleet/register", bytes.NewBufferString(`{
		"node_id":"node-1",
		"name":"Rack A1",
		"type":"picclaw",
		"capabilities":["temperature","relay"],
		"location":"rack-a"
	}`))
	register.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()

	server.ServeHTTP(registerResponse, register)

	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d: %s", http.StatusCreated, registerResponse.Code, registerResponse.Body.String())
	}

	heartbeat := httptest.NewRequest(http.MethodPost, "/fleet/heartbeat", bytes.NewBufferString(`{
		"node_id":"node-1",
		"status":"online",
		"metrics":{"cpu_pct":13.5,"queue_depth":1}
	}`))
	heartbeat.Header.Set("Content-Type", "application/json")
	heartbeatResponse := httptest.NewRecorder()

	server.ServeHTTP(heartbeatResponse, heartbeat)

	if heartbeatResponse.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status %d, got %d: %s", http.StatusOK, heartbeatResponse.Code, heartbeatResponse.Body.String())
	}
	var heartbeatBody struct {
		OK   bool `json:"ok"`
		Node Node `json:"node"`
	}
	if err := json.NewDecoder(heartbeatResponse.Body).Decode(&heartbeatBody); err != nil {
		t.Fatalf("decode heartbeat response: %v", err)
	}
	if !heartbeatBody.OK || heartbeatBody.Node.Metrics["cpu_pct"] != 13.5 {
		t.Fatalf("expected heartbeat metrics in response, got %#v", heartbeatBody)
	}

	event := httptest.NewRequest(http.MethodPost, "/fleet/events", bytes.NewBufferString(`{
		"node_id":"node-1",
		"type":"temperature_warning",
		"data":{"temperature_c":38.4}
	}`))
	event.Header.Set("Content-Type", "application/json")
	eventResponse := httptest.NewRecorder()

	server.ServeHTTP(eventResponse, event)

	if eventResponse.Code != http.StatusAccepted {
		t.Fatalf("expected event status %d, got %d: %s", http.StatusAccepted, eventResponse.Code, eventResponse.Body.String())
	}

	listNodes := httptest.NewRequest(http.MethodGet, "/fleet/nodes?status=online", nil)
	nodesResponse := httptest.NewRecorder()

	server.ServeHTTP(nodesResponse, listNodes)

	if nodesResponse.Code != http.StatusOK {
		t.Fatalf("expected list nodes status %d, got %d: %s", http.StatusOK, nodesResponse.Code, nodesResponse.Body.String())
	}
	var nodesBody struct {
		Nodes []Node `json:"nodes"`
	}
	if err := json.NewDecoder(nodesResponse.Body).Decode(&nodesBody); err != nil {
		t.Fatalf("decode nodes response: %v", err)
	}
	if len(nodesBody.Nodes) != 1 || nodesBody.Nodes[0].ID != "node-1" {
		t.Fatalf("expected node-1 in nodes response, got %#v", nodesBody.Nodes)
	}

	listEvents := httptest.NewRequest(http.MethodGet, "/fleet/events?node_id=node-1&type=temperature_warning", nil)
	eventsResponse := httptest.NewRecorder()

	server.ServeHTTP(eventsResponse, listEvents)

	if eventsResponse.Code != http.StatusOK {
		t.Fatalf("expected list events status %d, got %d: %s", http.StatusOK, eventsResponse.Code, eventsResponse.Body.String())
	}
	var eventsBody struct {
		Events []Event `json:"events"`
	}
	if err := json.NewDecoder(eventsResponse.Body).Decode(&eventsBody); err != nil {
		t.Fatalf("decode events response: %v", err)
	}
	if len(eventsBody.Events) != 1 || eventsBody.Events[0].Type != "temperature_warning" {
		t.Fatalf("expected temperature warning event, got %#v", eventsBody.Events)
	}
}

func TestServerRejectsHeartbeatForUnknownNode(t *testing.T) {
	server := NewServer(NewRegistry(), NewEventHub(10))
	request := httptest.NewRequest(http.MethodPost, "/fleet/heartbeat", bytes.NewBufferString(`{"node_id":"missing"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, response.Code, response.Body.String())
	}
}

func TestServerRejectsInvalidEvents(t *testing.T) {
	server := NewServer(NewRegistry(), NewEventHub(10))
	request := httptest.NewRequest(http.MethodPost, "/fleet/events", bytes.NewBufferString(`{"type":"temperature","data":{}}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
	}
}

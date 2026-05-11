package fleet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerDispatchesCommandAndDeliversOnHeartbeat(t *testing.T) {
	dispatcher := NewDispatcher()
	server := NewServer(NewRegistry(), dispatcher)

	register := httptest.NewRequest(http.MethodPost, "/fleet/register", bytes.NewBufferString(`{"id":"node-1","name":"PicClaw 1","type":"picclaw"}`))
	register.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()

	server.ServeHTTP(registerResponse, register)

	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d: %s", http.StatusCreated, registerResponse.Code, registerResponse.Body.String())
	}

	body := bytes.NewBufferString(`{"node_id":"node-1","type":"restart","payload":{"reason":"test"}}`)
	request := httptest.NewRequest(http.MethodPost, "/fleet/command", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}
	rawCommand := response.Body.String()
	if strings.Contains(rawCommand, "delivered_at") || strings.Contains(rawCommand, "executed_at") {
		t.Fatalf("expected pending command to omit unset timestamps, got %s", rawCommand)
	}
	var command Command
	if err := json.NewDecoder(strings.NewReader(rawCommand)).Decode(&command); err != nil {
		t.Fatalf("decode command response: %v", err)
	}
	if command.NodeID != "node-1" {
		t.Fatalf("expected node_id node-1, got %q", command.NodeID)
	}
	if command.Type != CommandTypeRestart {
		t.Fatalf("expected command type %q, got %q", CommandTypeRestart, command.Type)
	}
	if command.Status != CommandStatusPending {
		t.Fatalf("expected status %q, got %q", CommandStatusPending, command.Status)
	}

	heartbeat := httptest.NewRequest(http.MethodPost, "/fleet/heartbeat", bytes.NewBufferString(`{"node_id":"node-1"}`))
	heartbeat.Header.Set("Content-Type", "application/json")
	heartbeatResponse := httptest.NewRecorder()

	server.ServeHTTP(heartbeatResponse, heartbeat)

	if heartbeatResponse.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status %d, got %d: %s", http.StatusOK, heartbeatResponse.Code, heartbeatResponse.Body.String())
	}
	var heartbeatBody struct {
		OK       bool      `json:"ok"`
		Commands []Command `json:"commands"`
	}
	if err := json.NewDecoder(heartbeatResponse.Body).Decode(&heartbeatBody); err != nil {
		t.Fatalf("decode heartbeat response: %v", err)
	}
	if !heartbeatBody.OK {
		t.Fatal("expected heartbeat ok")
	}
	if len(heartbeatBody.Commands) != 1 {
		t.Fatalf("expected 1 delivered command, got %d", len(heartbeatBody.Commands))
	}
	if heartbeatBody.Commands[0].ID != command.ID {
		t.Fatalf("expected delivered command %q, got %q", command.ID, heartbeatBody.Commands[0].ID)
	}
	if heartbeatBody.Commands[0].Status != CommandStatusDelivered {
		t.Fatalf("expected delivered status %q, got %q", CommandStatusDelivered, heartbeatBody.Commands[0].Status)
	}

	stored, ok := dispatcher.Command(command.ID)
	if !ok {
		t.Fatalf("expected command %q to be stored", command.ID)
	}
	if stored.Status != CommandStatusDelivered {
		t.Fatalf("expected stored status %q, got %q", CommandStatusDelivered, stored.Status)
	}
}

func TestServerUpdatesAndReadsCommandStatus(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&Node{ID: "node-1", Name: "PicClaw 1", Type: "picclaw"})
	dispatcher := NewDispatcher()
	server := NewServer(registry, dispatcher)
	command, err := dispatcher.Dispatch("node-1", CommandTypeExecuteSkill, map[string]any{"skill": "inspect"})
	if err != nil {
		t.Fatalf("dispatch command: %v", err)
	}

	body := bytes.NewBufferString(`{"command_id":"` + command.ID + `","status":"failed","error":"skill not found"}`)
	request := httptest.NewRequest(http.MethodPost, "/fleet/command/status", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status update code %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var updated Command
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if updated.Status != CommandStatusFailed {
		t.Fatalf("expected status %q, got %q", CommandStatusFailed, updated.Status)
	}
	if updated.Error != "skill not found" {
		t.Fatalf("expected failure error to be recorded, got %q", updated.Error)
	}

	get := httptest.NewRequest(http.MethodGet, "/fleet/command?id="+command.ID, nil)
	getResponse := httptest.NewRecorder()

	server.ServeHTTP(getResponse, get)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d: %s", http.StatusOK, getResponse.Code, getResponse.Body.String())
	}
	var got Command
	if err := json.NewDecoder(getResponse.Body).Decode(&got); err != nil {
		t.Fatalf("decode command response: %v", err)
	}
	if got.Status != CommandStatusFailed || got.Error != "skill not found" {
		t.Fatalf("expected failed command with error, got %#v", got)
	}
}

func TestServerRejectsCommandForUnknownNode(t *testing.T) {
	server := NewServer(NewRegistry(), NewDispatcher())

	request := httptest.NewRequest(http.MethodPost, "/fleet/command", bytes.NewBufferString(`{"node_id":"missing","type":"restart"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, response.Code, response.Body.String())
	}
}

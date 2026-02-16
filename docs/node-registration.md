# Node Registration & Heartbeat Implementation

This document describes the Fleet Manager's node registration and heartbeat monitoring system.

## Overview

The node registration and heartbeat system allows edge agents (MicroClaw, PicClaw, NanoClaw) to register with the Fleet Manager and maintain their online status through periodic heartbeat signals.

## Architecture

```
┌────────────────┐         ┌─────────────────┐
│  Edge Agent    │  POST   │ Fleet Manager   │
│  (MicroClaw)   ├────────►│  /api/v1/fleet/ │
│                │         │   register      │
└────────────────┘         └─────────────────┘
        │                           │
        │  Heartbeat every 60s      │
        │  POST /api/v1/fleet/      │
        │       heartbeat           │
        └──────────────────────────►│
                                    │
                           ┌────────▼────────┐
                           │  Node Registry  │
                           │  (In-Memory)    │
                           └─────────────────┘
```

## API Endpoints

### 1. POST /api/v1/fleet/register

Register a new edge agent with the Fleet Manager.

**Request:**
```json
{
  "node_id": "microclaw-esp32-f4cfa210",
  "node_type": "microclaw",
  "capabilities": ["dht22", "mqtt"],
  "location": "office-floor-2",
  "metadata": {
    "gateway": "nanoclaw-pi-01",
    "firmware": "v1.0.0"
  }
}
```

**Response (200 OK):**
```json
{
  "node_id": "microclaw-esp32-f4cfa210",
  "status": "registered",
  "message": "Node successfully registered",
  "timestamp": "2026-02-16T08:30:00Z"
}
```

**Errors:**
- `400 Bad Request` - Missing required fields or invalid JSON
- `409 Conflict` - Node already registered (can re-register to update)

### 2. POST /api/v1/fleet/heartbeat

Send periodic heartbeat to indicate the node is alive and operational.

**Request:**
```json
{
  "node_id": "microclaw-esp32-f4cfa210",
  "uptime_seconds": 86400,
  "free_memory_kb": 200,
  "sensors_active": 2,
  "status": "online"
}
```

**Response (200 OK):**
```json
{
  "node_id": "microclaw-esp32-f4cfa210",
  "received": "2026-02-16T08:31:00Z",
  "next_check_seconds": 60
}
```

**Errors:**
- `400 Bad Request` - Missing node_id or invalid JSON
- `404 Not Found` - Node not registered (must call /register first)

### 3. GET /api/v1/fleet/nodes

List all registered nodes with optional filtering.

**Query Parameters:**
- `node_type` - Filter by type (microclaw, picclaw, nanoclaw, moltclaw)
- `status` - Filter by status (online, offline)
- `location` - Filter by location metadata

**Response (200 OK):**
```json
{
  "nodes": [
    {
      "id": "microclaw-esp32-f4cfa210",
      "name": "microclaw-esp32-f4cfa210",
      "type": "microclaw",
      "capabilities": ["dht22", "mqtt"],
      "location": "office-floor-2",
      "last_seen": "2026-02-16T08:31:00Z",
      "status": "online",
      "metadata": {
        "gateway": "nanoclaw-pi-01"
      }
    }
  ],
  "count": 1
}
```

### 4. GET /api/v1/fleet/nodes/{node_id}

Get detailed information about a specific node.

**Response (200 OK):**
```json
{
  "id": "microclaw-esp32-f4cfa210",
  "name": "microclaw-esp32-f4cfa210",
  "type": "microclaw",
  "capabilities": ["dht22", "mqtt"],
  "location": "office-floor-2",
  "last_seen": "2026-02-16T08:31:00Z",
  "status": "online",
  "metadata": {
    "gateway": "nanoclaw-pi-01",
    "firmware": "v1.0.0"
  }
}
```

**Errors:**
- `404 Not Found` - Node does not exist

## Heartbeat Monitoring

The Fleet Manager automatically monitors node health:

- **Heartbeat Interval:** Edge agents should send heartbeat every **60 seconds**
- **Timeout:** If no heartbeat received for **3 minutes**, node is marked as **offline**
- **Background Task:** Runs every 30 seconds to check for stale nodes

## Node States

| State | Description |
|-------|-------------|
| `online` | Node is actively sending heartbeats |
| `offline` | No heartbeat for >3 minutes |
| `degraded` | (Future) Partial functionality or warnings |

## Implementation Details

### In-Memory Registry

The current implementation uses an in-memory map with mutex protection:

```go
type Registry struct {
    mu    sync.RWMutex
    nodes map[string]*Node
}
```

**Future Enhancement:** Replace with persistent storage (PostgreSQL, Redis) for production deployment.

### Concurrency Safety

All registry operations are protected by `sync.RWMutex`:
- Read operations use `RLock()` for concurrent reads
- Write operations use `Lock()` for exclusive access

### Graceful Shutdown

The Fleet Manager supports graceful shutdown:
1. Stop accepting new connections
2. Wait up to 10 seconds for in-flight requests to complete
3. Clean shutdown

## Testing

Run the test suite:

```bash
go test ./pkg/fleet/... -v
```

**Test Coverage:**
- Registry operations (Register, Heartbeat, List, Get)
- HTTP handlers (all endpoints)
- Error cases (missing fields, non-existent nodes)
- Filtering (by node_type, status)
- Offline detection (MarkOffline timeout logic)

## Usage Example

### Edge Agent (MicroClaw)

```go
// Register once at startup
resp, _ := http.Post("http://fleet:8080/api/v1/fleet/register", 
    "application/json",
    bytes.NewReader([]byte(`{
        "node_id": "microclaw-esp32-f4cfa210",
        "node_type": "microclaw",
        "capabilities": ["dht22"]
    }`)))

// Send heartbeat every 60 seconds
ticker := time.NewTicker(60 * time.Second)
for range ticker.C {
    http.Post("http://fleet:8080/api/v1/fleet/heartbeat",
        "application/json",
        bytes.NewReader([]byte(`{
            "node_id": "microclaw-esp32-f4cfa210",
            "uptime_seconds": 3600
        }`)))
}
```

### Dashboard Query

```bash
# List all online nodes
curl http://fleet:8080/api/v1/fleet/nodes?status=online

# Get specific node details
curl http://fleet:8080/api/v1/fleet/nodes/microclaw-esp32-f4cfa210
```

## Security Considerations

**Current Implementation:**
- No authentication (MVP phase)

**Future Enhancements:**
- JWT tokens (issued during registration)
- API key authentication
- TLS/HTTPS support
- Rate limiting

## Performance

**Expected Load:**
- 100 nodes × 1 heartbeat/60s = ~1.67 req/s
- 1,000 nodes × 1 heartbeat/60s = ~16.7 req/s
- 10,000 nodes × 1 heartbeat/60s = ~167 req/s

**Current Capacity:**
- In-memory registry can handle 10,000+ nodes
- HTTP server configured with 15s read/write timeout
- 60s idle connection timeout

## Next Steps

1. ✅ Node registration and heartbeat (this PR)
2. 🚧 Event hub for sensor data collection (#3)
3. 🚧 Command dispatch to edge nodes (#5)
4. 📋 Persistent storage (PostgreSQL migration)
5. 📋 JWT authentication
6. 📋 WebSocket support for real-time updates

## References

- [OpenAPI Specification](../openapi.yaml)
- [Clawland Fleet README](../README.md)
- [Contributor Revenue Share](https://github.com/Clawland-AI/.github/blob/main/CONTRIBUTOR-REVENUE-SHARE.md)

# Clawland Fleet

**Cloud-Edge orchestration platform — Fleet Manager, Edge API Server, and Edge Reporter for managing distributed Claw agents.**

> Part of the [Clawland](https://github.com/Clawland-AI) ecosystem.

---

## Overview

Clawland Fleet is the **nervous system** connecting cloud and edge. It provides a unified control plane to deploy, monitor, update, and orchestrate hundreds of Claw agents across distributed locations.

## Components

### Fleet Manager (Cloud)
- **Node Registry** — Track all edge nodes: status, location, capabilities, firmware version
- **Task Dispatcher** — Push commands and configurations to edge nodes
- **Health Dashboard** — Real-time monitoring with alerting
- **OTA Orchestrator** — Rolling firmware/skill updates with rollback

### Edge API Server (runs on PicClaw)
- **REST/gRPC API** — Receive commands from cloud
- **Local Task Queue** — Buffer commands during offline periods
- **State Reporter** — Periodic heartbeat with metrics

### Edge Reporter (runs on PicClaw)
- **Event Streaming** — Push alerts and anomalies to cloud in real-time
- **Batch Upload** — Compress and send sensor data on schedule
- **Offline Buffer** — Store-and-forward when connectivity is lost

## Fleet Manager API

Start the HTTP service:

```bash
go run ./cmd/fleet
```

Set `PORT` to listen on a different port:

```bash
PORT=9090 go run ./cmd/fleet
```

Register an edge node:

```bash
curl -X POST http://localhost:8080/fleet/register \
  -H 'Content-Type: application/json' \
  -d '{
    "node_id": "picclaw-1",
    "type": "picclaw",
    "capabilities": ["temperature", "humidity"],
    "location": "greenhouse-a"
  }'
```

Send a heartbeat with status and metrics:

```bash
curl -X POST http://localhost:8080/fleet/heartbeat \
  -H 'Content-Type: application/json' \
  -d '{
    "node_id": "picclaw-1",
    "status": "online",
    "metrics": {"cpu": 0.41, "battery": 0.87}
  }'
```

List all nodes, or filter by status:

```bash
curl http://localhost:8080/fleet/nodes
curl http://localhost:8080/fleet/nodes?status=online
curl http://localhost:8080/fleet/nodes?status=offline
```

Read a single node:

```bash
curl http://localhost:8080/fleet/nodes/picclaw-1
```

## Architecture

```
┌─────────────────────────────────────────┐
│           Fleet Manager (Cloud)          │
│  ┌──────────┐ ┌────────┐ ┌───────────┐ │
│  │ Registry │ │Dashboard│ │OTA Manager│ │
│  └──────────┘ └────────┘ └───────────┘ │
└─────┬──────────────┬──────────────┬─────┘
      │              │              │
  ┌───▼────┐    ┌────▼───┐    ┌────▼───┐
  │Edge API│    │Edge API│    │Edge API│
  │PicClaw │    │PicClaw │    │PicClaw │
  │Node 1  │    │Node 2  │    │Node N  │
  └───┬────┘    └────┬───┘    └────┬───┘
      │              │              │
  ┌───▼────┐    ┌────▼───┐    ┌────▼───┐
  │Micro x3│    │Micro x5│    │Micro x2│
  └────────┘    └────────┘    └────────┘
```

## Status

🚧 **Pre-Alpha** — Architecture design phase. Looking for contributors!

## License

**Business Source License 1.1** (BSL 1.1)

- **Additional Use Grant:** You may use this software for any purpose **except** operating a commercial SaaS that competes with Clawland Fleet's hosted offering.
- **Change Date:** 4 years from each release date.
- **Change License:** Apache License 2.0.

See [LICENSE](LICENSE) for full terms.

## Contributing

See the [Clawland Contributing Guide](https://github.com/Clawland-AI/.github/blob/main/CONTRIBUTING.md).

**Core contributors share 20% of product revenue.** Read the [Contributor Revenue Share](https://github.com/Clawland-AI/.github/blob/main/CONTRIBUTOR-REVENUE-SHARE.md) terms.

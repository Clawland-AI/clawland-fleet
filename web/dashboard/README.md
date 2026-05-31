# Fleet Dashboard

This directory contains a dependency-free Fleet Manager dashboard for the Clawland bounty board task.

## What it covers

- Real-time style node status summary for online, degraded, and offline edge nodes.
- Alert aggregation with critical, warning, and info severity states.
- Command dispatch form for restart, config update, skill execution, and firmware update operations.
- Geolocated node map using fixture coordinates.
- Command queue preview backed by `POST /fleet/command` when the Fleet Manager server is running.
- API-backed data loading from `GET /fleet/dashboard/state`, with fixture fallback for static previews.

## Local preview

With the Fleet Manager API and dashboard server:

```sh
go run ./cmd/fleet
```

Then open `http://127.0.0.1:8080`.

For a static-only preview:

```sh
python3 -m http.server 8088 -d web/dashboard
```

Then open `http://127.0.0.1:8088`.

## API integration notes

The dashboard first loads `GET /fleet/dashboard/state`, then falls back to `fixtures/fleet-state.json` for static previews. The state response is:

```json
{
  "generated_at": "2026-05-31T17:00:00Z",
  "nodes": [],
  "alerts": []
}
```

Commands are queued in the browser session today. When the command dispatch API is available, the submit handler in `app.js` can post `{ node_id, type, payload }` to `POST /fleet/command` and then render the returned command status.
Commands are submitted to `POST /fleet/command`:

```json
{
  "node_id": "pond-a-picoclaw",
  "type": "restart",
  "payload": { "reason": "operator requested" }
}
```

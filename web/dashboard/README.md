# Fleet Dashboard

This directory contains a dependency-free Fleet Manager dashboard for the Clawland bounty board task.

## What it covers

- Real-time style node status summary for online, degraded, and offline edge nodes.
- Alert aggregation with critical, warning, and info severity states.
- Command dispatch form for restart, config update, skill execution, and firmware update operations.
- Geolocated node map using fixture coordinates.
- Session-local command queue preview.
- Fixture-backed data loading that can be replaced with Fleet Manager API responses later.

## Local preview

From the repository root:

```sh
python3 -m http.server 8088 -d web/dashboard
```

Then open `http://127.0.0.1:8088`.

## API integration notes

The dashboard currently loads `fixtures/fleet-state.json`. A production Fleet Manager can replace that fixture with an endpoint returning:

```json
{
  "generated_at": "2026-05-31T17:00:00Z",
  "nodes": [],
  "alerts": []
}
```

Commands are queued in the browser session today. When the command dispatch API is available, the submit handler in `app.js` can post `{ node_id, type, payload }` to `POST /fleet/command` and then render the returned command status.

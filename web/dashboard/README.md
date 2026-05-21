# Fleet Manager Dashboard

Static dashboard for the Clawland Fleet Manager bounty. It is dependency-free
vanilla HTML/CSS/JavaScript so it can be served by the existing Go binary, a CDN,
or any local static server.

## Features

- Real-time style node status board with online, degraded, and offline nodes.
- Alert aggregation by severity and source node.
- Command dispatch panel with safe command templates.
- Map view for geolocated nodes using CSS positioning and latitude/longitude.
- Mock SSE refresh loop for local demos; replace with `/api/events` when the
  Fleet Manager exposes a live stream.
- Responsive layout for laptop and tablet operations rooms.

## Local Preview

```bash
cd web/dashboard
python3 -m http.server 8088
```

Open `http://localhost:8088`.

## Integration Points

The dashboard expects the following JSON shape from a future Fleet Manager API:

- `GET /api/fleet/status` -> full snapshot matching `fixtures/fleet-state.json`
- `GET /api/fleet/events` -> Server-Sent Events stream with node or alert deltas
- `POST /api/fleet/commands` -> command dispatch body:

```json
{
  "node_id": "pond-guardian-01",
  "command": "skill.restart",
  "payload": {
    "skill": "aquaculture-monitoring"
  }
}
```

The local demo keeps command history in browser memory and does not contact a
backend.

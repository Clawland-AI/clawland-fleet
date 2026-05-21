const state = {
  snapshot: null,
  filter: 'all',
  commands: []
};

const elements = {
  lastUpdated: document.querySelector('#lastUpdated'),
  totalNodes: document.querySelector('#totalNodes'),
  onlineNodes: document.querySelector('#onlineNodes'),
  degradedNodes: document.querySelector('#degradedNodes'),
  openAlerts: document.querySelector('#openAlerts'),
  criticalCount: document.querySelector('#criticalCount'),
  queuedCount: document.querySelector('#queuedCount'),
  nodeList: document.querySelector('#nodeList'),
  alertList: document.querySelector('#alertList'),
  commandList: document.querySelector('#commandList'),
  commandNode: document.querySelector('#commandNode'),
  commandType: document.querySelector('#commandType'),
  commandPayload: document.querySelector('#commandPayload'),
  commandForm: document.querySelector('#commandForm'),
  map: document.querySelector('#map'),
  refreshButton: document.querySelector('#refreshButton'),
  filterButtons: document.querySelectorAll('[data-filter]')
};

async function loadSnapshot() {
  const response = await fetch('./fixtures/fleet-state.json', { cache: 'no-store' });
  if (!response.ok) {
    throw new Error(`Unable to load fleet snapshot: ${response.status}`);
  }
  const snapshot = await response.json();
  state.snapshot = simulateRealtime(snapshot);
  state.commands = [...state.snapshot.commands, ...state.commands.filter((command) => command.local)];
  render();
}

function simulateRealtime(snapshot) {
  const now = new Date();
  const copy = structuredClone(snapshot);
  copy.generated_at = now.toISOString();
  copy.nodes = copy.nodes.map((node, index) => {
    if (node.status === 'offline') return node;
    return {
      ...node,
      last_seen: new Date(now.getTime() - index * 13000).toISOString(),
      metrics: {
        ...node.metrics,
        latency_ms: Math.max(80, Math.round((node.metrics.latency_ms || 220) * (0.9 + Math.random() * 0.2))),
        cpu_percent: Math.min(96, Math.max(4, Math.round(node.metrics.cpu_percent + Math.random() * 8 - 4)))
      }
    };
  });
  return copy;
}

function render() {
  if (!state.snapshot) return;
  const { nodes, alerts } = state.snapshot;
  const openAlerts = alerts.filter((alert) => !alert.acknowledged);
  const criticalAlerts = openAlerts.filter((alert) => alert.severity === 'critical');

  elements.lastUpdated.textContent = `Updated ${formatTime(state.snapshot.generated_at)}`;
  elements.totalNodes.textContent = String(nodes.length);
  elements.onlineNodes.textContent = String(nodes.filter((node) => node.status === 'online').length);
  elements.degradedNodes.textContent = String(nodes.filter((node) => node.status === 'degraded').length);
  elements.openAlerts.textContent = String(openAlerts.length);
  elements.criticalCount.textContent = `${criticalAlerts.length} critical`;
  elements.queuedCount.textContent = `${state.commands.filter((command) => command.status === 'queued').length} queued`;

  renderNodes(nodes);
  renderAlerts(alerts);
  renderCommands(state.commands);
  renderNodeOptions(nodes);
  renderMap(nodes);
}

function renderNodes(nodes) {
  const filtered = state.filter === 'all' ? nodes : nodes.filter((node) => node.status === state.filter);
  elements.nodeList.innerHTML = filtered.map((node) => `
    <article class="node-card">
      <div>
        <div class="node-title">
          <span class="status-dot status-${escapeHtml(node.status)}"></span>
          <strong>${escapeHtml(node.name)}</strong>
        </div>
        <div class="node-meta">
          <span>${escapeHtml(node.id)}</span>
          <span>${escapeHtml(node.type)}</span>
          <span>${escapeHtml(node.location)}</span>
          <span>Seen ${formatTime(node.last_seen)}</span>
        </div>
        <div class="chips" aria-label="Capabilities">
          ${node.capabilities.map((capability) => `<span class="chip">${escapeHtml(capability)}</span>`).join('')}
        </div>
      </div>
      <div class="metric-grid">
        <span><strong>${node.metrics.cpu_percent}%</strong>CPU</span>
        <span><strong>${node.metrics.memory_mb} MB</strong>Memory</span>
        <span><strong>${node.metrics.battery_percent}%</strong>Battery</span>
        <span><strong>${node.metrics.latency_ms ?? 'n/a'} ms</strong>Latency</span>
      </div>
    </article>
  `).join('');
}

function renderAlerts(alerts) {
  elements.alertList.innerHTML = alerts.map((alert) => `
    <article class="alert-item alert-${escapeHtml(alert.severity)}">
      <strong>${escapeHtml(alert.title)}</strong>
      <p>${escapeHtml(alert.message)}</p>
      <div class="node-meta">
        <span>${escapeHtml(alert.node_id)}</span>
        <span>${escapeHtml(alert.severity)}</span>
        <span>${formatTime(alert.timestamp)}</span>
        <span>${alert.acknowledged ? 'acknowledged' : 'open'}</span>
      </div>
    </article>
  `).join('');
}

function renderCommands(commands) {
  elements.commandList.innerHTML = commands.map((command) => `
    <article class="command-item">
      <strong>${escapeHtml(command.command)}</strong>
      <div class="node-meta">
        <span>${escapeHtml(command.node_id)}</span>
        <span>${escapeHtml(command.status)}</span>
        <span>${formatTime(command.created_at)}</span>
      </div>
    </article>
  `).join('');
}

function renderNodeOptions(nodes) {
  const selected = elements.commandNode.value;
  elements.commandNode.innerHTML = nodes.map((node) => `
    <option value="${escapeHtml(node.id)}">${escapeHtml(node.name)} (${escapeHtml(node.status)})</option>
  `).join('');
  if (nodes.some((node) => node.id === selected)) {
    elements.commandNode.value = selected;
  }
}

function renderMap(nodes) {
  const bounds = nodes.reduce((acc, node) => ({
    minLat: Math.min(acc.minLat, node.latitude),
    maxLat: Math.max(acc.maxLat, node.latitude),
    minLon: Math.min(acc.minLon, node.longitude),
    maxLon: Math.max(acc.maxLon, node.longitude)
  }), { minLat: 90, maxLat: -90, minLon: 180, maxLon: -180 });

  elements.map.innerHTML = nodes.map((node) => {
    const x = scale(node.longitude, bounds.minLon, bounds.maxLon, 12, 88);
    const y = scale(node.latitude, bounds.minLat, bounds.maxLat, 84, 16);
    return `
      <div class="map-node" data-status="${escapeHtml(node.status)}" style="left:${x}%;top:${y}%">
        <strong>${escapeHtml(node.name)}</strong>
        <div class="node-meta">${escapeHtml(node.location)}</div>
      </div>
    `;
  }).join('');
}

function scale(value, min, max, outMin, outMax) {
  if (min === max) return (outMin + outMax) / 2;
  return outMin + ((value - min) / (max - min)) * (outMax - outMin);
}

function formatTime(value) {
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(new Date(value));
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}

elements.filterButtons.forEach((button) => {
  button.addEventListener('click', () => {
    state.filter = button.dataset.filter;
    elements.filterButtons.forEach((item) => item.classList.toggle('is-active', item === button));
    render();
  });
});

elements.refreshButton.addEventListener('click', () => {
  loadSnapshot().catch((error) => {
    elements.lastUpdated.textContent = error.message;
  });
});

elements.commandForm.addEventListener('submit', (event) => {
  event.preventDefault();
  let payload;
  try {
    payload = JSON.parse(elements.commandPayload.value);
  } catch {
    elements.commandPayload.focus();
    return;
  }
  state.commands.unshift({
    id: `local-${Date.now()}`,
    node_id: elements.commandNode.value,
    command: elements.commandType.value,
    payload,
    status: 'queued',
    local: true,
    created_at: new Date().toISOString()
  });
  renderCommands(state.commands);
  elements.queuedCount.textContent = `${state.commands.filter((command) => command.status === 'queued').length} queued`;
});

loadSnapshot().catch((error) => {
  elements.lastUpdated.textContent = error.message;
});

setInterval(() => {
  loadSnapshot().catch(() => {});
}, 30000);

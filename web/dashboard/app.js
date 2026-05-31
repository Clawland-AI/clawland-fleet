const state = {
  filter: "all",
  data: null,
  commands: [],
  pollTimer: null,
};

const els = {
  onlineCount: document.querySelector("#onlineCount"),
  degradedCount: document.querySelector("#degradedCount"),
  offlineCount: document.querySelector("#offlineCount"),
  criticalCount: document.querySelector("#criticalCount"),
  lastUpdated: document.querySelector("#lastUpdated"),
  nodeList: document.querySelector("#nodeList"),
  alertList: document.querySelector("#alertList"),
  mapCanvas: document.querySelector("#mapCanvas"),
  commandNode: document.querySelector("#commandNode"),
  commandType: document.querySelector("#commandType"),
  commandPayload: document.querySelector("#commandPayload"),
  commandForm: document.querySelector("#commandForm"),
  commandQueue: document.querySelector("#commandQueue"),
  liveToggle: document.querySelector("#liveToggle"),
  refreshButton: document.querySelector("#refreshButton"),
};

async function loadFleetState() {
  const response = await fetch("./fixtures/fleet-state.json", { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`Unable to load fleet fixture: ${response.status}`);
  }
  state.data = await response.json();
  render();
}

function byStatus(status) {
  return (node) => status === "all" || node.status === status;
}

function render() {
  const data = state.data;
  if (!data) return;

  const nodes = data.nodes ?? [];
  const alerts = data.alerts ?? [];
  const counts = nodes.reduce(
    (acc, node) => {
      acc[node.status] = (acc[node.status] ?? 0) + 1;
      return acc;
    },
    { online: 0, degraded: 0, offline: 0 },
  );

  els.onlineCount.textContent = counts.online;
  els.degradedCount.textContent = counts.degraded;
  els.offlineCount.textContent = counts.offline;
  els.criticalCount.textContent = alerts.filter((alert) => alert.severity === "critical").length;
  els.lastUpdated.textContent = `Updated ${formatRelative(data.generated_at)}`;

  renderNodes(nodes.filter(byStatus(state.filter)));
  renderAlerts(alerts);
  renderMap(nodes);
  renderCommandTargets(nodes);
  renderCommandQueue();
}

function renderNodes(nodes) {
  els.nodeList.replaceChildren(
    ...(nodes.length
      ? nodes.map((node) => {
          const card = document.createElement("article");
          card.className = "node-card";
          card.innerHTML = `
            <div>
              <div class="node-title">
                <span>${escapeHtml(node.name)}</span>
                <span class="pill ${node.status}">${node.status}</span>
              </div>
              <div class="node-meta">
                <span>${escapeHtml(node.type)}</span>
                <span>${escapeHtml(node.location.name)}</span>
                <span>${node.metrics.battery}% battery</span>
              </div>
            </div>
            <div class="node-meta">
              <span>${node.metrics.temperature_c}C</span>
              <span>${node.metrics.signal_dbm} dBm</span>
              <span>${node.pending_commands} queued</span>
            </div>
          `;
          return card;
        })
      : [emptyMessage("No nodes match this filter")]),
  );
}

function renderAlerts(alerts) {
  els.alertList.replaceChildren(
    ...(alerts.length
      ? alerts.map((alert) => {
          const card = document.createElement("article");
          card.className = "alert-card";
          card.innerHTML = `
            <div class="alert-title">
              <span>${escapeHtml(alert.title)}</span>
              <span class="pill ${alert.severity}">${alert.severity}</span>
            </div>
            <p class="alert-time">${escapeHtml(alert.node_id)} - ${formatRelative(alert.created_at)}</p>
          `;
          return card;
        })
      : [emptyMessage("No open alerts")]),
  );
}

function renderMap(nodes) {
  els.mapCanvas.replaceChildren(
    ...nodes.map((node) => {
      const pin = document.createElement("button");
      pin.type = "button";
      pin.className = `pin ${node.status}`;
      pin.style.left = `${node.location.map_x}%`;
      pin.style.top = `${node.location.map_y}%`;
      pin.title = `${node.name}: ${node.status}`;
      pin.innerHTML = `<span>${escapeHtml(node.name)}</span>`;
      pin.addEventListener("click", () => {
        state.filter = node.status;
        updateSegments();
        render();
      });
      return pin;
    }),
  );
}

function renderCommandTargets(nodes) {
  const current = els.commandNode.value;
  els.commandNode.replaceChildren(
    ...nodes.map((node) => {
      const option = document.createElement("option");
      option.value = node.id;
      option.textContent = `${node.name} (${node.status})`;
      option.disabled = node.status === "offline";
      return option;
    }),
  );
  if (current) els.commandNode.value = current;
}

function renderCommandQueue() {
  els.commandQueue.replaceChildren(
    ...(state.commands.length
      ? state.commands.map((command) => {
          const item = document.createElement("div");
          item.className = "queue-item";
          item.textContent = `${command.type} queued for ${command.nodeId} at ${command.createdAt}`;
          return item;
        })
      : [emptyMessage("No commands queued in this browser session")]),
  );
}

function emptyMessage(text) {
  const node = document.createElement("div");
  node.className = "empty";
  node.textContent = text;
  return node;
}

function setFilter(filter) {
  state.filter = filter;
  updateSegments();
  render();
}

function updateSegments() {
  document.querySelectorAll(".segment").forEach((button) => {
    button.classList.toggle("active", button.dataset.filter === state.filter);
  });
}

function startPolling() {
  stopPolling();
  state.pollTimer = window.setInterval(loadFleetState, 15000);
}

function stopPolling() {
  if (state.pollTimer) {
    window.clearInterval(state.pollTimer);
    state.pollTimer = null;
  }
}

function formatRelative(value) {
  const timestamp = new Date(value).getTime();
  const diffSeconds = Math.max(0, Math.round((Date.now() - timestamp) / 1000));
  if (diffSeconds < 60) return `${diffSeconds}s ago`;
  const diffMinutes = Math.round(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  return `${Math.round(diffMinutes / 60)}h ago`;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

document.querySelectorAll(".segment").forEach((button) => {
  button.addEventListener("click", () => setFilter(button.dataset.filter));
});

els.refreshButton.addEventListener("click", loadFleetState);

els.liveToggle.addEventListener("change", () => {
  if (els.liveToggle.checked) {
    startPolling();
  } else {
    stopPolling();
  }
});

els.commandForm.addEventListener("submit", (event) => {
  event.preventDefault();
  state.commands.unshift({
    nodeId: els.commandNode.value,
    type: els.commandType.value,
    payload: els.commandPayload.value,
    createdAt: new Date().toLocaleTimeString(),
  });
  renderCommandQueue();
});

await loadFleetState();
startPolling();

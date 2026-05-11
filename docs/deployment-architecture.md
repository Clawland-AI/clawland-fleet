# Clawland Fleet Deployment Architecture

This document maps the production topology for a Clawland installation, from
sensor-level agents through the regional gateway and up to the cloud Fleet
Manager.

## L1 → L2 → L3 data flow

```mermaid
flowchart TB
  subgraph L1["L1 edge and sensor layer"]
    Micro["MicroClaw MCU\nDHT22, dissolved oxygen,\nvibration, smoke"]
    Pico["PicoClaw edge agent\nlocal rules + GPIO/relay"]
  end

  subgraph L2["L2 site gateway"]
    Nano["NanoClaw Raspberry Pi / SBC\naggregation + offline decisions"]
  end

  subgraph L3["L3 cloud control plane"]
    Fleet["Clawland Fleet Manager\nregistry, events, commands"]
    Molt["MoltClaw cloud gateway\nLLM routing + orchestration"]
    Dash["Dashboard / alerts\nGrafana, Feishu, Telegram"]
  end

  Micro -->|"sensor frames\nMQTT/serial/LoRa"| Pico
  Pico -->|"events + heartbeat\nHTTP/MQTT"| Nano
  Nano -->|"normalized events\n/fleet/events"| Fleet
  Fleet -->|"command queue\n/fleet/command"| Nano
  Nano -->|"local dispatch"| Pico
  Fleet --> Molt
  Fleet --> Dash
```

## Network topology options

```mermaid
flowchart LR
  subgraph Site["Factory, data center, pond, or greenhouse"]
    Sensors["Sensors + relays"]
    Micro["MicroClaw"]
    Pico["PicoClaw"]
    Nano["NanoClaw gateway"]
    Sensors --- Micro
    Micro -->|"UART/I2C/SPI"| Pico
    Pico -->|"LAN Wi-Fi/Ethernet"| Nano
  end

  subgraph Backhaul["Backhaul choices"]
    LAN["LAN/VPN"]
    Cell["4G/5G router"]
    LoRa["LoRa gateway\nlow bandwidth fallback"]
  end

  subgraph Cloud["Cloud region"]
    Fleet["Fleet Manager"]
    Store["event store / metrics"]
    Alert["alert channels"]
  end

  Nano --> LAN --> Fleet
  Nano --> Cell --> Fleet
  Micro -. "critical telemetry" .-> LoRa -.-> Fleet
  Fleet --> Store
  Fleet --> Alert
```

## Component interaction sequence

```mermaid
sequenceDiagram
  participant Edge as PicoClaw / NanoClaw
  participant Fleet as Fleet Manager
  participant Operator as Operator dashboard

  Edge->>Fleet: POST /fleet/register
  Fleet-->>Edge: 201 node registered
  loop every heartbeat interval
    Edge->>Fleet: POST /fleet/heartbeat
    Fleet-->>Edge: 200 ok + pending commands
  end
  Edge->>Fleet: POST /fleet/events
  Fleet-->>Edge: 202 accepted
  Operator->>Fleet: POST /fleet/command
  Fleet-->>Operator: 202 command queued
  Edge->>Fleet: POST /fleet/heartbeat
  Fleet-->>Edge: pending command payload
  Edge->>Fleet: POST /fleet/command/status
  Fleet-->>Edge: 200 status recorded
```

## Failure scenarios and fallback paths

| Scenario | Detection | Fallback behavior | Recovery signal |
| --- | --- | --- | --- |
| L1 sensor stops reporting | PicoClaw misses sensor read deadline | PicoClaw reports a `sensor_stale` event and keeps last known safe actuator state | Sensor read succeeds again |
| PicoClaw loses WAN access | Fleet heartbeat timeout marks node offline | NanoClaw stores events locally and applies offline decision rules | Next successful heartbeat |
| NanoClaw gateway fails | Fleet stops seeing all site nodes | PicoClaw continues local threshold actions and buffers events | Gateway registration refresh |
| Cloud Fleet Manager unavailable | Edge HTTP requests fail or time out | Edge agents keep a bounded store-and-forward queue and retry with backoff | `/fleet/heartbeat` returns 200 |
| Command delivery fails | Command remains pending past TTL | Fleet marks command failed and emits an operator alert | Command status update or replacement command |

```mermaid
flowchart TD
  Healthy["Normal operation"] --> MissedHeartbeat{"heartbeat timeout?"}
  MissedHeartbeat -- "no" --> Healthy
  MissedHeartbeat -- "yes" --> Offline["mark node offline"]
  Offline --> LocalRules["edge runs local rules\nand buffers events"]
  LocalRules --> Retry{"backoff retry succeeds?"}
  Retry -- "no" --> LocalRules
  Retry -- "yes" --> Reconcile["flush buffered events\nand reconcile command status"]
  Reconcile --> Healthy
```

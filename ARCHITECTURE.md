# Clawland Fleet Architecture

Complete deployment architecture and data flow diagrams for the Clawland edge AI network.

---

## Table of Contents

- [Overview](#overview)
- [Three-Tier Architecture](#three-tier-architecture)
- [Data Flow Diagram](#data-flow-diagram)
- [Network Topology](#network-topology)
- [Component Interaction](#component-interaction)
- [Failure Scenarios](#failure-scenarios)
- [Deployment Examples](#deployment-examples)

---

## Overview

Clawland Fleet orchestrates a three-tier edge AI network:

| Layer | Agent | Hardware | Role |
|-------|-------|----------|------|
| **L1** | MicroClaw, PicoClaw | $2-10 MCU/SBC | Sensor reading, local rules |
| **L2** | NanoClaw | $50 SBC (Raspberry Pi) | Regional gateway, aggregation |
| **L3** | MoltClaw, Fleet Manager | Cloud / Mac Mini | Orchestration, global state |

---

## Three-Tier Architecture

```mermaid
graph TB
    subgraph "L3: Cloud / Datacenter"
        FM[Fleet Manager<br/>Orchestration]
        MC[MoltClaw<br/>AI Gateway]
        DB[(PostgreSQL<br/>State DB)]
        EH[Event Hub<br/>MQTT Broker]
        
        FM --> DB
        FM --> EH
        MC --> FM
    end
    
    subgraph "L2: Regional Gateway (Raspberry Pi)"
        NC1[NanoClaw-01<br/>Office]
        NC2[NanoClaw-02<br/>Factory]
        NC3[NanoClaw-03<br/>Warehouse]
        
        NC1 --> EH
        NC2 --> EH
        NC3 --> EH
    end
    
    subgraph "L1: Edge Sensors"
        subgraph "Office Sensors"
            PC1[PicoClaw<br/>Door Monitor]
            MC1[MicroClaw<br/>Temp+Humidity]
        end
        
        subgraph "Factory Sensors"
            PC2[PicoClaw<br/>Motion Detector]
            MC2[MicroClaw<br/>CO2 Monitor]
            MC3[MicroClaw<br/>Vibration]
        end
        
        subgraph "Warehouse Sensors"
            PC3[PicoClaw<br/>RFID Reader]
            MC4[MicroClaw<br/>Water Leak]
        end
        
        PC1 --> NC1
        MC1 --> NC1
        
        PC2 --> NC2
        MC2 --> NC2
        MC3 --> NC2
        
        PC3 --> NC3
        MC4 --> NC3
    end
    
    style FM fill:#667eea
    style MC fill:#764ba2
    style NC1 fill:#48bb78
    style NC2 fill:#48bb78
    style NC3 fill:#48bb78
    style PC1 fill:#ed8936
    style PC2 fill:#ed8936
    style PC3 fill:#ed8936
    style MC1 fill:#e53e3e
    style MC2 fill:#e53e3e
    style MC3 fill:#e53e3e
    style MC4 fill:#e53e3e
```

---

## Data Flow Diagram

### Sensor Data Upload (L1 → L2 → L3)

```mermaid
sequenceDiagram
    participant MC as MicroClaw (L1)
    participant NC as NanoClaw (L2)
    participant EH as Event Hub (L3)
    participant FM as Fleet Manager (L3)
    
    Note over MC: Sensor reads<br/>temperature: 25°C
    
    MC->>MC: Validate reading<br/>(range check)
    MC->>NC: MQTT Publish<br/>clawland/sensor-01/temp<br/>{"value": 25.0}
    
    NC->>NC: Aggregate data<br/>(last 1 min)
    NC->>EH: MQTT Publish<br/>clawland/gateway-01/batch<br/>[{"sensor": "01", ...}]
    
    EH->>FM: Event notification<br/>(new sensor data)
    FM->>FM: Store in DB<br/>Run alerts/rules
    
    alt Temperature > 30°C
        FM->>NC: Command<br/>{"action": "alert", "target": "sensor-01"}
        NC->>MC: MQTT Subscribe<br/>clawland/sensor-01/cmd
        MC->>MC: Trigger local alarm
    end
```

### Command Dispatch (L3 → L2 → L1)

```mermaid
sequenceDiagram
    participant User as User/API
    participant FM as Fleet Manager
    participant EH as Event Hub
    participant NC as NanoClaw
    participant PC as PicoClaw
    
    User->>FM: POST /api/nodes/sensor-01/restart
    FM->>FM: Validate permission
    FM->>EH: MQTT Publish<br/>clawland/sensor-01/cmd<br/>{"action": "restart"}
    
    EH->>NC: Route to gateway<br/>(knows sensor-01 on NC)
    NC->>PC: MQTT Subscribe<br/>clawland/sensor-01/cmd
    PC->>PC: Execute restart<br/>esp_restart()
    
    PC->>NC: ACK<br/>{"status": "restarted"}
    NC->>EH: Forward ACK
    EH->>FM: Command result
    FM->>User: Response<br/>{"result": "success"}
```

---

## Network Topology

### Option 1: LAN + WiFi

```mermaid
graph LR
    subgraph "Local Network (192.168.1.x)"
        R[WiFi Router<br/>Gateway]
        
        subgraph "L2"
            NC[NanoClaw<br/>Pi 4<br/>ETH: 192.168.1.100]
        end
        
        subgraph "L1"
            MC1[MicroClaw-01<br/>WiFi: 192.168.1.201]
            MC2[MicroClaw-02<br/>WiFi: 192.168.1.202]
            PC1[PicoClaw-01<br/>WiFi: 192.168.1.101]
        end
        
        R --> NC
        R --> MC1
        R --> MC2
        R --> PC1
    end
    
    R -->|Internet| Cloud[MoltClaw<br/>Fleet Manager]
```

### Option 2: 4G/LTE (Remote Deployment)

```mermaid
graph TD
    subgraph "Remote Site"
        NC[NanoClaw<br/>USB 4G Modem]
        
        MC1[MicroClaw-01]
        MC2[MicroClaw-02]
        PC1[PicoClaw-01]
        
        MC1 -->|UART/Serial| NC
        MC2 -->|UART/Serial| NC
        PC1 -->|WiFi Hotspot| NC
    end
    
    NC -->|4G LTE| Tower[Cell Tower]
    Tower -->|Internet| Cloud[MoltClaw<br/>Fleet Manager]
```

### Option 3: LoRa (Ultra-Low Power)

```mermaid
graph TD
    subgraph "Farm Deployment"
        NC[NanoClaw<br/>LoRa Gateway]
        
        MC1[MicroClaw-01<br/>LoRa 868MHz]
        MC2[MicroClaw-02<br/>LoRa 868MHz]
        MC3[MicroClaw-03<br/>LoRa 868MHz]
        
        MC1 -.->|LoRa<br/>2km range| NC
        MC2 -.->|LoRa<br/>2km range| NC
        MC3 -.->|LoRa<br/>5km range| NC
    end
    
    NC -->|WiFi/Ethernet| Cloud[MoltClaw]
    
    style MC1 fill:#e53e3e
    style MC2 fill:#e53e3e
    style MC3 fill:#e53e3e
```

---

## Component Interaction

### Fleet Manager Components

```mermaid
graph TB
    subgraph "Fleet Manager (Go)"
        API[REST API Server<br/>:8080]
        Reg[Node Registry<br/>Service]
        HB[Heartbeat<br/>Monitor]
        Cmd[Command<br/>Dispatcher]
        Evt[Event<br/>Processor]
        
        API --> Reg
        API --> Cmd
        HB --> Reg
        Evt --> Reg
    end
    
    subgraph "Data Layer"
        PG[(PostgreSQL)]
        RD[(Redis Cache)]
    end
    
    subgraph "Message Layer"
        MQTT[MQTT Broker<br/>Mosquitto]
    end
    
    Reg --> PG
    Reg --> RD
    HB --> RD
    Cmd --> MQTT
    Evt --> MQTT
```

### NanoClaw Regional Gateway

```mermaid
graph TB
    subgraph "NanoClaw (Python)"
        AGG[Data Aggregator]
        FWD[Cloud Forwarder]
        LOC[Local Cache]
        DEC[Decision Engine<br/>Offline Rules]
        
        AGG --> LOC
        AGG --> FWD
        DEC --> LOC
    end
    
    subgraph "Upstream"
        MQTT[MQTT to Fleet]
    end
    
    subgraph "Downstream"
        L1[L1 Sensors<br/>MicroClaw/PicoClaw]
    end
    
    L1 --> AGG
    FWD --> MQTT
    DEC -.->|Offline Mode| L1
```

---

## Failure Scenarios

### Scenario 1: Cloud Connection Lost

```mermaid
sequenceDiagram
    participant MC as MicroClaw
    participant NC as NanoClaw
    participant Cloud as Fleet Manager
    
    MC->>NC: Sensor data<br/>(temp: 32°C)
    NC->>Cloud: Forward data<br/>(MQTT)
    
    Note over Cloud: ❌ Network outage
    
    NC->>NC: Detect connection loss<br/>(heartbeat timeout)
    NC->>NC: Enter offline mode<br/>(activate local rules)
    
    MC->>NC: Sensor data<br/>(temp: 35°C)
    NC->>NC: Check local threshold<br/>(>30°C = alert)
    NC->>MC: Send alert command<br/>(no cloud dependency)
    
    Note over Cloud: ✅ Network restored
    
    NC->>Cloud: Reconnect<br/>(resync buffered data)
    NC->>Cloud: Upload cached readings<br/>(catchup mode)
```

### Scenario 2: Regional Gateway Failure

```mermaid
graph TD
    subgraph "Before Failure"
        NC[NanoClaw<br/>Primary Gateway<br/>✅ Active]
        MC1[MicroClaw-01] --> NC
        MC2[MicroClaw-02] --> NC
    end
    
    subgraph "After Failure"
        NC2[NanoClaw<br/>Primary Gateway<br/>❌ Down]
        NCF[NanoClaw Fallback<br/>Secondary Gateway<br/>✅ Active]
        
        MC1F[MicroClaw-01] -.->|Auto-discover| NCF
        MC2F[MicroClaw-02] -.->|Auto-discover| NCF
    end
    
    style NC2 fill:#e53e3e
    style NCF fill:#48bb78
```

**Fallback Mechanism**:
1. L1 nodes send mDNS/beacon every 30s
2. If primary gateway silent >90s, L1 scans for fallback
3. Fallback gateway announces: `_clawland._tcp.local`
4. L1 reconnects to fallback, resumes operation

### Scenario 3: Sensor Node Failure

```mermaid
sequenceDiagram
    participant MC as MicroClaw
    participant NC as NanoClaw
    participant FM as Fleet Manager
    
    MC->>NC: Heartbeat<br/>(every 60s)
    NC->>FM: Node status<br/>(healthy)
    
    Note over MC: ❌ Power loss
    
    NC->>NC: Wait for next heartbeat<br/>(timeout: 180s)
    NC->>NC: Mark node as DOWN
    NC->>FM: Alert: Node offline<br/>{"sensor": "01", "status": "down"}
    
    FM->>FM: Trigger notification<br/>(email/Slack)
    
    Note over MC: ✅ Power restored
    
    MC->>NC: Reconnect + backlog<br/>(cached readings)
    NC->>FM: Alert: Node recovered<br/>{"sensor": "01", "status": "up"}
```

---

## Deployment Examples

### Example 1: Office Building (Small)

```
L3 (Cloud): MoltClaw on AWS t3.small ($10/month)
L2 (Gateway): 1x NanoClaw (Pi 4) per floor
L1 (Sensors): 10x MicroClaw per floor
- Temperature/humidity in each room
- Motion sensors on doors
- CO2 monitors in meeting rooms

Total: 3 floors = 3 gateways + 30 sensors
Cost: $10/month (cloud) + $150 (Pi) + $60 (sensors)
```

### Example 2: Factory (Medium)

```
L3 (Cloud): MoltClaw on DigitalOcean droplet ($20/month)
L2 (Gateway): 5x NanoClaw (Pi 4) per production zone
L1 (Sensors): 50x MicroClaw across zones
- Vibration sensors on machinery
- Temperature probes on critical equipment
- Door sensors on storage rooms
- Water leak detectors

Total: 5 zones = 5 gateways + 50 sensors
Cost: $20/month (cloud) + $250 (Pi) + $100 (sensors)
```

### Example 3: Smart Farm (Large, LoRa)

```
L3 (Cloud): MoltClaw on VPS ($15/month)
L2 (Gateway): 10x NanoClaw (Pi 4 + LoRa) covering 100 hectares
L1 (Sensors): 200x MicroClaw with LoRa modules
- Soil moisture sensors (every 0.5 hectare)
- Weather stations (every 10 hectares)
- Livestock tracking (GPS collars)

Total: 10 gateways + 200 sensors
Cost: $15/month (cloud) + $500 (Pi + LoRa) + $400 (sensors)
LoRa range: Up to 5km in open terrain
```

---

## Scaling Characteristics

| Metric | Small | Medium | Large |
|--------|-------|--------|-------|
| **L1 Nodes** | 10-50 | 50-500 | 500+ |
| **L2 Gateways** | 1-3 | 3-10 | 10-100 |
| **L3 Cloud** | Single instance | Load balanced | Multi-region |
| **Data Rate** | <1 MB/day | 1-100 MB/day | >100 MB/day |
| **Cost/Month** | <$50 | $50-500 | >$500 |

---

## Technology Stack

| Component | Technology | Port/Protocol |
|-----------|-----------|---------------|
| **Fleet Manager** | Go, Gin framework | 8080 (HTTP) |
| **Event Hub** | Mosquitto MQTT | 1883 (MQTT), 8883 (MQTTS) |
| **Database** | PostgreSQL 15+ | 5432 |
| **Cache** | Redis | 6379 |
| **NanoClaw** | Python 3.9+ | MQTT client |
| **PicoClaw** | Go | MQTT client |
| **MicroClaw** | C/C++ (Arduino) | MQTT client |

---

## Security Considerations

### Authentication
- **L1 → L2**: Pre-shared keys (PSK) in MQTT
- **L2 → L3**: TLS client certificates
- **API**: JWT tokens with role-based access

### Network Isolation
- L1 sensors on isolated VLAN (no internet access)
- L2 gateways have dual NICs (sensor LAN + uplink WAN)
- L3 Fleet Manager exposed via reverse proxy (nginx)

### Data Encryption
- MQTT over TLS (port 8883)
- At-rest encryption for PostgreSQL
- End-to-end encryption for sensitive sensor data

---

**Built for Clawland Fleet Issue #4** 🚀

For implementation details, see:
- [Fleet API Spec](../docs/api-spec.md) (Issue #1)
- [Node Registration Service](../pkg/registry/) (Issue #2)
- [Event Hub Implementation](../pkg/eventhub/) (Issue #3)

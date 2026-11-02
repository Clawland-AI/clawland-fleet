# Clawland Fleet

**云边端一体化协同编排平台 —— 包含集群管理器 (Fleet Manager)、边缘 API 服务端 (Edge API Server) 与边缘上报器 (Edge Reporter)，专为大规模分布式 Claw 智能体集群设计。**

[English](README.md) | [简体中文](README_CN.md)

> [Clawland](https://github.com/Clawland-AI) 开源生态核心成员。

---

## 概述 (Overview)

Clawland Fleet 是连接云端控制与物理边缘节点的**中枢神经系统**。它提供统一的云边控制平面，用于在成百上千个分布式现场中部署、监控、升级和编排 Claw 智能体网络。

## 核心组件架构 (Components)

### 1. 集群管理器 (Fleet Manager - 云端)
- **节点注册中心 (Node Registry)** — 全局追踪所有边缘节点：运行状态、地理位置、外设能力集、固件版本
- **任务分发器 (Task Dispatcher)** — 批量下发控制指令与动态配置到边缘节点
- **健康监控看板 (Health Dashboard)** — 实时集群拓扑大屏与异常事件智能预警
- **OTA 编排器 (OTA Orchestrator)** — 支持滚动平滑固件/技能升级与故障自动回滚

### 2. 边缘 API 服务端 (Edge API Server - 运行于 PicClaw)
- **REST/gRPC 接口** — 高性能接收并验证来自云端下发的即时控制指令
- **本地任务缓冲队列 (Local Task Queue)** — 在断网离线期间安全暂存待执行命令
- **节点状态报告器 (State Reporter)** — 定时回传心跳信号与关键运行指标

### 3. 边缘上报器 (Edge Reporter - 运行于 PicClaw)
- **实时事件流 (Event Streaming)** — 亚秒级向云端推送紧急告警与异常遥测数据
- **批量压缩上传 (Batch Upload)** — 按计划压缩、打包并批量回传传感器历史时序数据
- **断网持久化缓冲 (Offline Buffer)** — 网络中断时触发本地“存储-待发”机制，恢复后断点续传

## 架构拓扑 (Architecture)

```
┌─────────────────────────────────────────┐
│          集群管理器 (Fleet Manager)     │
│  ┌──────────┐ ┌────────┐ ┌───────────┐  │
│  │ 节点注册 │ │监控看板│ │OTA 编排器 │  │
│  └──────────┘ └────────┘ └───────────┘  │
└─────┬──────────────┬──────────────┬─────┘
      │              │              │
  ┌───▼────┐    ┌────▼───┐    ┌────▼───┐
  │边缘 API│    │边缘 API│    │边缘 API│
  │PicClaw │    │PicClaw │    │PicClaw │
  │节点 1  │    │节点 2  │    │节点 N  │
  └───┬────┘    └────┬───┘    └────┬───┘
      │              │              │
  ┌───▼────┐    ┌────▼───┐    ┌────▼───┐
  │Micro x3│    │Micro x5│    │Micro x2│
  └────────┘    └────────┘    └────────┘
```

## 项目状态 (Status)

🚧 **Pre-Alpha** — 处于系统架构与接口规范设计阶段。热烈欢迎开源贡献者加入！

## 开源授权 (License)

**商业源码协议 1.1 (Business Source License 1.1 - BSL 1.1)**

- **附加使用许可 (Additional Use Grant):** 您可以出于任何目的使用本软件，**唯一的例外是**：不得利用本软件运营与 Clawland Fleet 官方托管云服务构成直接商业竞争的 SaaS 产品。
- **开源变更期 (Change Date):** 自每个版本发布之日起 4 年。
- **变更后协议 (Change License):** 自动转为完全开放的 Apache License 2.0 协议。

详情请参阅 [LICENSE](LICENSE) 完整条款。

## 参与贡献 (Contributing)

详情请参阅 [Clawland 贡献指南](https://github.com/Clawland-AI/.github/blob/main/CONTRIBUTING.md)。

**核心贡献者享有产品收入 20% 的收益分成。** 查阅 [贡献者收益分成条款](https://github.com/Clawland-AI/.github/blob/main/CONTRIBUTOR-REVENUE-SHARE.md) 了解激励细节。

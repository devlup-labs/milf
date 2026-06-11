# ☁️ Pocket Cloud: Distributed Serverless Compute Platform

> *Turn any mobile device into a compute node. Execute code at the edge in microseconds.*

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Backend](https://img.shields.io/badge/Backend-Go-blue?logo=go)](#3-central-server)
[![React Client](https://img.shields.io/badge/Frontend-React-cyan?logo=react)](#4-developer-dashboard)
[![Flutter Android](https://img.shields.io/badge/Mobile-Flutter-blue?logo=flutter)](#5-consumer-node)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [System Architecture Overview](#2-system-architecture-overview)
3. [Central Server](#3-central-server)
4. [Developer Dashboard (Client)](#4-developer-dashboard)
5. [Consumer Node (Edge Worker)](#5-consumer-node)
6. [WASM Compilation Engine](#6-wasm-compilation-engine)
7. [Security & Threat Model](#7-security--threat-model)
8. [Database Architecture](#8-database-architecture)
9. [Infrastructure & Cost Analysis (India)](#9-infrastructure--cost-analysis-india)
10. [Current Implementation Status](#10-current-implementation-status)
11. [Documentation Index](#11-documentation-index)

---

## 1. Executive Summary

**Pocket Cloud** is a distributed serverless platform that transforms standard mobile devices into powerful compute nodes by executing **WebAssembly (WASM)** lambdas in a secure, sandboxed environment.

### The Problem
Traditional serverless platforms (AWS Lambda, Google Cloud Functions) suffer from:
- **Cold starts of 300ms–3s** due to microVM (Firecracker) boot times and heavy language runtimes.
- **Vendor lock-in** to centralized cloud regions.
- **High egress costs** that make edge computing prohibitively expensive.

### The Pocket Cloud Solution
Instead of spinning up entire virtual machines, Pocket Cloud compiles user code into tiny WebAssembly binaries **once** on the Central Server, then distributes them to edge nodes running the WebAssembly Micro Runtime (WAMR). This achieves:
- **< 5ms cold starts** (vs. 800ms+ on AWS Lambda).
- **Near-native execution speed** for compiled languages (C, C++, Rust, Go).
- **True edge portability** — any Android device becomes a compute node.

### Key Features
| Feature | Description |
| :--- | :--- |
| **Real-time Tasking** | WebSocket-based push architecture for instant task execution on mobile nodes. |
| **On-the-fly Compilation** | Submit C/Go/Rust code via the dashboard; compiled to WASM in seconds. |
| **Secure Sandbox** | Powered by WAMR with WASI support, process isolation, and Seccomp filters. |
| **Full-Stack Auth** | Google OAuth 2.0 + JWT for developer onboarding and secure communication. |
| **Live Monitoring** | Track execution logs, outputs, and status in real-time from the React dashboard. |

---

## 2. System Architecture Overview

Pocket Cloud is composed of three major subsystems that work in concert.

| Module | Technology | Role |
| :--- | :--- | :--- |
| **Central Server** | Go, PostgreSQL, Redis | The orchestrator. Handles Auth, WASM compilation, Task Queues, and WebSocket hubs. |
| **Developer Dashboard** | React, Vite, Tailwind | The developer interface. Create, manage, invoke lambdas, and view execution results. |
| **Consumer Node** | Flutter, Kotlin, C++, WAMR | The execution engine. Connects via WebSocket to receive and run WASM tasks on Android devices. |

### End-to-End Data Flow

```
Developer writes C++ function
        │
        ▼
┌─────────────────────┐
│  Developer Dashboard │  ── POST /api/v1/functions/upload ──►
│  (React/Vite SPA)   │
└─────────────────────┘
        │                              ┌──────────────────────┐
        │                              │    Central Server     │
        │                              │  ┌────────────────┐  │
        │                              │  │  API Gateway    │  │
        │                              │  └───────┬────────┘  │
        │                              │          │           │
        │                              │  ┌───────▼────────┐  │
        │                              │  │  Compiler Node  │  │
        │                              │  │  (C++ → WASM)   │  │
        │                              │  └───────┬────────┘  │
        │                              │          │           │
        │                              │  ┌───────▼────────┐  │
        │                              │  │  Object Store   │  │
        │                              │  │  (.wasm binary) │  │
        │                              │  └───────┬────────┘  │
        │                              │          │           │
        │    User invokes function     │  ┌───────▼────────┐  │
        │ ─── POST /api/v1/invoke ───► │  │  Orchestrator   │  │
        │                              │  └───────┬────────┘  │
        │                              │          │           │
        │                              │  ┌───────▼────────┐  │
        │                              │  │ Worker Manager  │──┼──► WebSocket Push
        │                              │  └────────────────┘  │
        │                              └──────────────────────┘
        │                                         │
        │                                         ▼
        │                              ┌──────────────────────┐
        │    ◄── Result ──────────     │   Consumer Node       │
        │                              │  ┌────────────────┐  │
        │                              │  │ Flutter (Dart)  │  │
        │                              │  └───────┬────────┘  │
        │                              │  ┌───────▼────────┐  │
        │                              │  │ Kotlin (JNI)   │  │
        │                              │  └───────┬────────┘  │
        │                              │  ┌───────▼────────┐  │
        │                              │  │ WAMR (C++)     │  │
        │                              │  │ Sandboxed Exec │  │
        │                              │  └────────────────┘  │
        │                              └──────────────────────┘
```

---

## 3. Central Server

The Central Server is a Go-based backend that acts as the control plane for the entire platform.

### Core Modules

| Module | Responsibility |
| :--- | :--- |
| **Authentication** | Google OAuth 2.0, JWT issuance, token refresh, and user management. |
| **API Gateway** | SSL termination, rate-limiting, and routing to internal services. |
| **Compiler Module** | Receives raw source code, injects ABI wrappers, and compiles to `wasm32-wasi`. |
| **Orchestrator** | Manages function lifecycle. Routes invocation triggers to the correct queue. |
| **Worker Manager** | Tracks consumer node health via heartbeats and dispatches tasks over WebSockets. |

### API Surface

**Public APIs:**
- `POST /api/v1/auth/signup` — Register a new developer.
- `POST /api/v1/auth/login` — Authenticate and receive JWT tokens.
- `POST /api/v1/functions/upload` — Upload source code for compilation.
- `POST /api/v1/invoke/{id}` — Trigger a lambda execution.
- `GET /api/v1/lambdas/{id}/wasm` — Download the compiled WASM binary.
- `POST /api/v1/files` — Upload execution output files.

**Internal APIs (Worker Fleet):**
- `POST /internal/worker/register` — Consumer node joins the execution pool.
- `POST /internal/worker/heartbeat` — High-frequency health reports.
- `POST /internal/worker/result` — Consumer returns execution output.

> **Detailed Reference:** [CentralServer/architecture.md](CentralServer/architecture.md) · [CentralServer/apis_and_infrastructure_cost.md](CentralServer/apis_and_infrastructure_cost.md)

---

## 4. Developer Dashboard

A React/Vite Single Page Application (SPA) that serves as the developer's primary interface.

### Technology Stack
- **Framework:** React 18 + Vite
- **Styling:** Tailwind CSS + ShadcnUI
- **State:** `@tanstack/react-query` with IndexedDB for local-first caching
- **Auth:** Google OAuth 2.0 → JWT stored in `HttpOnly` cookies (proposed)
- **Editor:** Monaco Editor (VS Code engine) for in-browser code editing and syntax highlighting

### Core Screens
| Screen | Purpose |
| :--- | :--- |
| **Login/Register** | Google OAuth-based onboarding. |
| **Dashboard** | Overview of all deployed lambdas and their status. |
| **Function Editor** | Monaco-powered editor for writing/editing lambda code. |
| **Execution Logs** | Real-time log streaming from consumer nodes. |
| **Settings** | API key management, resource quotas, billing overview. |

> **Detailed Reference:** [Client/architecture.md](Client/architecture.md)

---

## 5. Consumer Node (Edge Worker)

The Consumer Node is a hybrid Android application that turns any mobile device into a secure compute worker.

### Three-Layer Architecture

```
┌────────────────────────────────────────────┐
│          Flutter Orchestrator (Dart)        │
│  Cloud Connect │ Policy Checker │ Output   │
├────────────────────────────────────────────┤
│          Kotlin Orchestrator (JNI)          │
│  OS Stats │ Process Manager │ IPC Bridge   │
├────────────────────────────────────────────┤
│          WAMR Sandbox (C/C++)              │
│  Loader │ Executor │ Memory Accountant     │
└────────────────────────────────────────────┘
```

**Layer 1 — Flutter (Control Plane):**
- Manages the WebSocket connection to the Central Server.
- Implements the `PolicyChecker` to accept/reject jobs based on device health.
- Formats and sends execution results back to the cloud.

**Layer 2 — Kotlin (System Plane):**
- Reads real-time OS metrics via `android.os.Debug.MemoryInfo`.
- Manages the `isolatedProcess` lifecycle and Android WakeLocks.
- Bridges Flutter ↔ C++ via JNI (`MethodChannel`).

**Layer 3 — WAMR (Execution Plane):**
- Initializes the WAMR runtime with configurable memory quotas (512MB heap / 16MB stack).
- Validates WASM bytecode, resolves imports, and executes functions.
- Captures outputs and safely traps crashes without affecting the host app.

### Consumer-Cloud Communication Protocol

1. **Register:** `POST /api/v1/sinks/register` → Receives a `sink_id`.
2. **Connect:** Opens a persistent WebSocket at `ws://<server>/api/v1/sinks/ws?sinkId=<sink_id>`.
3. **Heartbeat:** Sends resource snapshots (RAM, CPU) every 30 seconds.
4. **Receive Task:** Server pushes `task_assignment` with WASM binary + input payload.
5. **Execute:** Flutter → Kotlin → C++ (WAMR) pipeline runs the function.
6. **Return Result:** Output sent back via WebSocket as `task_result`.

> **Detailed Reference:** [ConsumerNode/architecture.md](ConsumerNode/architecture.md) · [ConsumerNode/native_connect_architecture.md](ConsumerNode/native_connect_architecture.md) · [ConsumerNode/cloud_communication.md](ConsumerNode/cloud_communication.md)

---

## 6. WASM Compilation Engine

The compilation engine is the core innovation. It shifts the "heavy lifting" from the edge to the server.

### Pipeline

```
Source Code (C++/Go/Rust)
    │
    ▼
┌──────────────┐
│ Intake Layer │ ── Determines payload type (File/Zip/Docker)
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│ ABI Wrapper Gen  │ ── Auto-generates the byte-to-type translation layer
└──────┬───────────┘
       │
       ▼
┌──────────────────────┐
│ Language Compiler     │
│ • C/C++: clang       │
│ • Rust:  rustc       │
│ • Go:   tinygo       │
│ Target: wasm32-wasi  │
└──────┬───────────────┘
       │
       ▼
┌──────────────────┐
│ Object Store     │ ── SHA256 hash → wasm/fn_ab12cd/function.wasm
└──────────────────┘
```

### Performance Comparison

| Scenario | Compiler | Server Build Time | Cold Start | Execution Speed |
| :--- | :--- | :--- | :--- | :--- |
| **C++ (Image → PDF)** | `clang` + `wasi-sdk` | 2–5s | **< 5ms** | Near-native |
| **Rust (Crypto Hash)** | `rustc` + `wasm32-wasi` | 5–15s | **< 5ms** | Near-native |
| **Go (Data Parsing)** | `tinygo` | 1–3s | **< 5ms** | Fast |
| **Python (Scripts)** | Py2Wasm | Varies | ~50ms | Moderate |
| *AWS Lambda (Node.js)* | *N/A* | *0s* | *300–800ms* | *Fast (V8 JIT)* |
| *AWS Lambda (Python)* | *N/A* | *0s* | *800ms–1.2s* | *Moderate* |

> **Detailed Reference:** [Engine/compiler_consumer.md](Engine/compiler_consumer.md) · [Engine/server_side_compiler_analysis.md](Engine/server_side_compiler_analysis.md)

---

## 7. Security & Threat Model

Pocket Cloud operates on a **Zero-Trust** model. All user-submitted WASM code is treated as potentially malicious.

### Defense-in-Depth

| Layer | Technology | Protection |
| :--- | :--- | :--- |
| **User** | Flutter Auth | Prevents unauthorized job dispatch. |
| **Middleware** | Kotlin Admission Controller | Denies execution if device state is unstable. |
| **Runtime** | WAMR Bytecode Validator | Prevents malformed or illegal module loading. |
| **Logic** | Memory Accountant | Prevents heap/stack overflows and memory bombs. |
| **Process** | Android `isolatedProcess` | Zero permissions — no camera, contacts, or storage access. |
| **Kernel** | Seccomp + Cgroups | Whitelist-only syscalls; hard CPU/RAM/IO quotas. |

### Key Threat Mitigations

- **Sandbox Escape:** Even if WAMR is exploited, the `isolatedProcess` has a unique UID with zero permissions. Seccomp filters block `execve`, `open`, and other dangerous syscalls.
- **Resource Exhaustion (DoS):** `MemoryAccountant` enforces hard caps. Execution timeouts kill infinite loops. `OomPredictor` kills modules before the Android LMK intervenes.
- **Data Exfiltration:** Network namespace is restricted by default. High-resolution timers are disabled to mitigate timing side-channels.

> **Detailed Reference:** [Security/threat_model.md](Security/threat_model.md)

---

## 8. Database Architecture

The data layer is segmented across three storage technologies optimized for their access patterns.

### Storage Strategy

| Store | Technology | Purpose |
| :--- | :--- | :--- |
| **Relational** | PostgreSQL | Users, Lambdas, Executions, Consumer Nodes |
| **Object** | Cloudflare R2 (S3-compatible) | `.wasm` binaries, source code, large payloads |
| **Cache** | Redis (Upstash) | Orchestrator maps, Worker heartbeats, task queues |

### Core Data Models

**Users:** `id`, `email`, `password_hash`, `tier` (free/pro)
**Lambdas:** `id`, `owner_id`, `name`, `runtime`, `wasm_ref`, `memory_limit`, `status`
**Executions:** `id`, `lambda_id`, `status`, `duration_ms`, `output_ref`
**Consumers:** `id`, `ip_address`, `max_memory`, `status`, `last_heartbeat`

> **Detailed Reference:** [CentralServer/database_strategy.md](CentralServer/database_strategy.md)

---

## 9. Infrastructure & Cost Analysis (India)

All costs are estimated for **10,000 active users** with deployment in India (Mumbai/Bangalore).

### Compute Infrastructure (₹/month)

| Component | Sizing | Cost (INR) |
| :--- | :--- | :--- |
| Load Balancer | DigitalOcean LB | ₹1,000–₹2,000 |
| API Gateway + Orchestrator | 2x (2 vCPU, 4GB RAM) | ₹4,000 |
| Compiler Nodes | 2x (4 vCPU, 8GB RAM) Compute-Optimized | ₹9,000 |
| Worker Manager | 1x (2 vCPU, 2GB RAM) | ₹1,500 |
| Network Egress | ~500 GB API traffic | ₹500–₹4,500 |
| **Subtotal (Compute)** | | **₹16,000–₹21,000** |

### Database Infrastructure (₹/month)

| Component | Platform | Cost (INR) |
| :--- | :--- | :--- |
| PostgreSQL | Neon.tech / Supabase | ₹2,500–₹5,000 |
| Object Store | Cloudflare R2 (600GB + 2TB egress) | ₹750 ($0 egress) |
| Redis Cache | Upstash | ₹1,700–₹3,300 |
| **Subtotal (Data)** | | **₹5,000–₹9,000** |

### Total Monthly Cost

| | Optimized (R2 + DO) | AWS-Heavy |
| :--- | :--- | :--- |
| **10,000 Users** | **₹21,000–₹30,000/mo** | ₹45,000+/mo |

> **Critical Decision:** Using **Cloudflare R2** (zero egress) instead of AWS S3 saves ~₹15,000/month at this scale.

> **Detailed Reference:** [CentralServer/apis_and_infrastructure_cost.md](CentralServer/apis_and_infrastructure_cost.md) · [CentralServer/database_strategy.md](CentralServer/database_strategy.md)

---

## 10. Current Implementation Status

| Component | Status | What's Working |
| :--- | :--- | :--- |
| **Central Server** | ✅ Functional | Auth, WASM compilation (C), WebSocket hub, REST APIs |
| **Developer Dashboard** | ✅ Functional | Google OAuth, Lambda CRUD, Live execution logs |
| **Consumer Node (Execution)** | ✅ Functional | WAMR C++ engine, JNI/AIDL bridge, Process Isolation |
| **Consumer Node (Networking)** | ✅ Functional | WebSocket connect, task receive, result send |
| **Admission Controller** | 🔲 Proposed | Mock heartbeat (hardcoded 2048MB RAM) |
| **Billing Module** | 🔲 Proposed | Execution times logged but not monetized |
| **Seccomp Filters** | 🔲 Proposed | Architecture defined, not yet applied |
| **Zero-Copy Memory** | 🔲 Proposed | Currently using standard `ByteArray` copy |
| **AOT Cache** | 🔲 Proposed | No local WASM caching on consumer |

---

## 11. Documentation Index

All detailed architectural documents are located in the `docs/` directory.

### Central Server
- [architecture.md](CentralServer/architecture.md) — Module design, Eraser diagrams, and data flow.
- [database_strategy.md](CentralServer/database_strategy.md) — Database types, schemas, and cost analysis.
- [apis_and_infrastructure_cost.md](CentralServer/apis_and_infrastructure_cost.md) — API routes, compute sizing, and INR cost breakdown.

### Developer Dashboard (Client)
- [architecture.md](Client/architecture.md) — React/Vite SPA design and module overview.

### Consumer Node
- [architecture.md](ConsumerNode/architecture.md) — Three-layer (Flutter/Kotlin/C++) architecture overview.
- [lib_architecture.md](ConsumerNode/lib_architecture.md) — Flutter Dart module design (current vs. proposed).
- [native_connect_architecture.md](ConsumerNode/native_connect_architecture.md) — 12-step execution lifecycle of the Kotlin-WAMR bridge.
- [wasm_runtime_architecture.md](ConsumerNode/wasm_runtime_architecture.md) — Deep dive into the WAMR sandbox runtime.
- [cloud_communication.md](ConsumerNode/cloud_communication.md) — Consumer-Server communication protocol.

### Engine
- [compiler_consumer.md](Engine/compiler_consumer.md) — Compilation pipeline and multi-language support.
- [server_side_compiler_analysis.md](Engine/server_side_compiler_analysis.md) — Performance comparison vs. AWS Lambda.

### Security
- [threat_model.md](Security/threat_model.md) — Attack vectors, mitigations, and defense-in-depth analysis.

---

## License

This project is licensed under the **Apache License 2.0**.

Developed with ❤️ by Devlup Labs / [adarshxsh](https://github.com/adarshxsh)

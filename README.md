# 🚀 MILF: Mobile Infra for Lambdas and Files

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Backend](https://img.shields.io/badge/Backend-Go-blue?logo=go)](./CentralServer)
[![React Client](https://img.shields.io/badge/Frontend-React-cyan?logo=react)](./Client)
[![Flutter Android](https://img.shields.io/badge/Mobile-Flutter-blue?logo=flutter)](./consumeronlywamr)

**MILF** (Mobile Infra for Lambdas and Files) is a state-of-the-art distributed serverless platform. It transforms standard mobile devices into powerful compute nodes by executing **WebAssembly (WASM)** lambdas in a secure, sandboxed environment.

## 🌟 Key Features
- **Real-time Tasking**: WebSocket-based push architecture for instant task execution on mobile nodes.
* **On-the-fly Compilation**: Submit C code via the dashboard and have it compiled to WASM in milliseconds.
* **Secure Sandbox**: Powered by WAMR (WebAssembly Micro Runtime) with WASI support for isolated execution.
* **Full-Stack Auth**: Google OAuth 2.0 integration for seamless developer onboarding.
* **Live Monitoring**: Track execution logs and outputs in real-time from the React dashboard.

---

## 🏗️ Project Structure

| Module | Technology | Description |
| :--- | :--- | :--- |
| **`CentralServer`** | Go, PostgreSQL | The orchestrator. Handles Auth, WASM compilation, and WebSocket hubs. |
| **`Client`** | React, Vite, Tailwind | The developer dashboard. Create, manage, and invoke lambdas. |
| **`consumeronlywamr`** | Flutter, C++, WAMR | The execution node. Connects via WS to run WASM tasks on Android. |

---

## 🛠️ Getting Started

### Prerequisites
- **Go** 1.21+ (for backend)
- **Node.js** 18+ (for frontend)
- **Flutter** (for android app)
- **PostgreSQL** (with a database named `central_server_db`)
- **WASI SDK** (Installed at `/opt/wasi-sdk` for C-to-WASM compilation)

### 1. Central Server (Backend)
```bash
cd CentralServer
cp .env.example .env
# Edit .env with your Google Client ID and DB credentials
go run cmd/server/main.go
```

### 2. Client Dashboard (Frontend)
```bash
cd Client
cp .env.example .env
# Set VITE_GOOGLE_CLIENT_ID and VITE_API_BASE_URL
npm install
npm run dev
```

### 3. Mobile Sink (Android)
```bash
cd consumeronlywamr
flutter pub get
flutter run
```

---

## 🤝 Contributing & Implementation Plan

We welcome contributions! This project is a modular ecosystem. Please follow these steps to contribute:

### 1. Understand the Flow
Review the [Implementation Plan](./implementation_plan.md) and the architectural blueprint in `plan.json` to understand how data flows from the React Client -> Go Server -> Android Sink.

### 2. Setting Up for Contribution
1. **Fork the Repository**: Create your own feature branch.
2. **Setup Env**: Never push your `.env` files. Ensure you update `.env.example` if you add new configurations.
3. **WASM Toolchain**: Ensure you have the `wasi-sdk` installed if you are working on the compiler module.

### 3. Development Workflow
- **Backend**: Focus on `internal/` directories following clean architecture.
- **Frontend**: Components are built using ShadcnUI and Tailwind. Use `src/lib/api.ts` for all server interactions.
- **Mobile**: Logic is primarily in `lib/modules/cloud_connect` (WS orchestration) and `lib/modules/native_bridge` (WASM runner).

### 4. Pull Requests
- Ensure code is linted and formatted.
- Update documentation if you add new API endpoints.
- Keep commits descriptive (e.g., `feat: add realtime log streaming`).

---

## 🛡️ Security & Isolation
- **WebAssembly Isolation**: Functions run in a zero-trust per-execution sandbox.
- **Google OAuth**: Secure identity management via verified ID tokens.
- **JWT Protection**: All inter-module communication is protected via short-lived JWTs.

---

## 📄 License
This project is licensed under the Apache License 2.0.

Developed with ❤️ by Devlup Labs[adarshxsh](https://github.com/adarshxsh)

# 🚀 MILF: Mobile Infra for Lambdas and Files

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Backend](https://img.shields.io/badge/Backend-Go-blue?logo=go)](./CentralServer)
[![React Client](https://img.shields.io/badge/Frontend-React-cyan?logo=react)](./Client)
[![Flutter Android](https://img.shields.io/badge/Mobile-Flutter-blue?logo=flutter)](./consumeronlywamr)

**MILF** (Mobile Infra for Lambdas and Files) is a distributed serverless platform that transforms standard mobile devices into powerful compute nodes by executing **WebAssembly (WASM)** lambdas in a secure, sandboxed environment.

## 🌟 Key Features
- **Real-time Tasking**: WebSocket-based push architecture for instant task execution on mobile nodes.
- **On-the-fly Compilation**: Submit C code via the dashboard and have it compiled to WASM in milliseconds.
- **Secure Sandbox**: Powered by WAMR (WebAssembly Micro Runtime) with WASI support for isolated execution.
- **Full-Stack Auth**: Google OAuth 2.0 integration and JWT for seamless developer onboarding and secure communication.
- **Live Monitoring**: Track execution logs, outputs, and status in real-time from the React dashboard.

---

## 🏗️ Project Structure

| Module | Technology | Description |
| :--- | :--- | :--- |
| **`CentralServer`** | Go, PostgreSQL | The orchestrator. Handles Auth, WASM compilation, Task Queues, and WebSocket hubs. |
| **`Client`** | React, Vite, Tailwind | The developer dashboard. Create, manage, invoke lambdas, and view execution results. |
| **`consumeronlywamr`** | Flutter, C++, WAMR | The execution node. Connects via WS to receive and run WASM tasks on Android devices. |

---

## 🛠️ Getting Started (Local Setup)

Welcome to the team! Follow these steps to set up the project locally on your machine.

### Prerequisites
- **Go** 1.21+
- **Node.js** 18+ & **npm**
- **Flutter** SDK (and Android Studio/Emulator for testing)
- **PostgreSQL** 14+
- **WASI SDK**: Required for compiling C code into WASM.
  - Download from [WebAssembly/wasi-sdk](https://github.com/WebAssembly/wasi-sdk)
  - Extract it (e.g., to `/opt/wasi-sdk`).
  - Set the `CLANG_PATH` environment variable in `CentralServer/.env` to point to the `bin/clang` inside the SDK.

### 1. Database Setup
1. Start PostgreSQL.
2. Create the required databases. By default, the backend expects `milf_db` or similar depending on your `.env`.
   ```bash
   psql -U postgres -c "CREATE DATABASE milf_functions;"
   ```

### 2. Central Server (Backend)
The backend handles the REST API, WASM compilation, and websocket connections.
```bash
cd CentralServer
# 1. Copy the example environment file
cp .env.example .env

# 2. Update .env with your credentials:
# - Set DB_DSN (e.g., postgres://postgres:password@localhost:5432/milf_functions?sslmode=disable)
# - Set GOOGLE_CLIENT_ID for Auth
# - Set JWT_SECRET to a secure random string

# 3. Download Go modules and run the server
go mod tidy
# - Set CLANG_PATH to your wasi-sdk clang binary
#   Example: CLANG_PATH=/opt/wasi-sdk/bin/clang
go run cmd/server/main.go
```
*The server will typically start on `http://localhost:8080`.*

### 3. Client Dashboard (Frontend)
The web interface for creating and dispatching WASM functions.
```bash
cd Client

# 1. Copy the example environment file
cp .env.example .env

# 2. Update .env:
# - VITE_API_BASE_URL=http://localhost:8080
# - VITE_GOOGLE_CLIENT_ID=<Your Google Client ID>

# 3. Install dependencies and start the dev server
npm install
npm run dev
```
*The frontend will start on `http://localhost:5173`.*

### 4. Mobile Sink (Android App)
The Flutter application that acts as the compute node.
```bash
cd consumeronlywamr

# 1. Fetch Flutter dependencies
flutter pub get

# 2. Add your server's IP address
# If running on an Android Emulator, localhost is usually 10.0.2.2.
# If testing on a physical device, ensure it's on the same Wi-Fi network and update the API URL in the Dart code to your local machine's IP (e.g., 192.168.x.x:8080).

# 3. Run the app
flutter run
```

---

## 🤝 Contributing Guidelines

We welcome contributions! Please follow this workflow to ensure smooth collaboration:

### 1. Branching Strategy
- Create a new branch for your feature or bugfix:
  ```bash
  git checkout -b feature/your-feature-name
  # or
  git checkout -b fix/your-bug-name
  ```

### 2. Development Workflow
- **Backend (`CentralServer`)**: Follow clean architecture principles. Place core business logic in `internal/core`, handlers in `internal/handler`, and database interactions in `internal/storage`.
- **Frontend (`Client`)**: Built heavily using ShadcnUI and Tailwind CSS. Use `src/lib/api.ts` for all backend interactions. Keep components modular in `src/components`.
- **Mobile (`consumeronlywamr`)**:
  - `lib/node_controller.dart`: Handles incoming tasks and JSON payloads.
  - `android/app/src/main/cpp/`: Contains the WAMR native C++ bridge for executing the WebAssembly binaries.

### 3. Submitting a Pull Request
1. Ensure your code is properly formatted (e.g., `go fmt`, `npm run lint`).
2. Do not commit your `.env` files. If you add a new environment variable, make sure to add it to `.env.example`.
3. Provide a clear description of what your PR solves. Include screenshots if it involves UI changes.
4. Request a review from a teammate.

---

## 🛡️ Security & Architecture Notes
- **WebAssembly Isolation**: Functions run in a zero-trust per-execution sandbox on the mobile device.
- **Payload Handling**: Always ensure payloads are correctly encoded as `TaskEnvelope` JSON strings when communicating over WebSockets to avoid casting crashes on the Dart side.
- **Google OAuth**: Used for secure identity management. Make sure your Google Cloud Console has authorized your local dev URIs (`http://localhost:5173`).

---

## 📄 License
This project is licensed under the Apache License 2.0.

Developed with ❤️ by Devlup Labs / [adarshxsh](https://github.com/adarshxsh)

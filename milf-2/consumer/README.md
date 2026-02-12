# MILF: Consumer Module 🚀
**Mobile Infra for Lambdas and Files**

The **Consumer Module** is a high-performance orchestration node designed to receive, validate, and execute WebAssembly (WASM) binaries on mobile devices. It acts as the execution edge for the MILF ecosystem.

## 🏗️ Architecture

This module follows a **Modular Monolithic** design, bridging Flutter for orchestration and cross-platform UI with native Kotlin for low-level system interactions and high-performance WASM execution.

### Logic Flow:
1.  **Flutter Orchestrator**: Receives jobs from the cloud, checks admission policies, and manages the lifecycle of the consumer node.
2.  **Kotlin Orchestrator**: Handles OS-level resource monitoring (CPU/RAM) and manages isolated WASM execution slots.
3.  **WASM Runtime**: Executes binaries in a safe, non-blocking background thread with strict time and resource limits.

## 📦 Project Structure

```text
lib/
├── config/             # Environment and dependency injection
├── core/               # Shared models and error handling
└── modules/
    ├── cloud_connect/  # API/Websocket communication
    ├── policy_checker/ # Admission control and validation
    └── native_bridge/  # MethodChannel bridge to Kotlin

android/.../consumer/
└── features/
    ├── orchestrator/   # Command handling from Flutter
    ├── os_stats/       # Hardware resource telemetry
    ├── process_manager/# Execution slot lifecycle
    └── wasm_runtime/   # WASM engine integration
```

## 🚀 Getting Started

### Prerequisites
- Flutter SDK `^3.10.7`
- Android SDK with NDK support

### Initial Setup
1.  **Install Dependencies**:
    ```bash
    flutter pub get
    ```
2.  **Configure Environment**:
    Copy the example environment file and fill in your keys.
    ```bash
    cp .env.example .env
    ```
3.  **Run the App**:
    ```bash
    flutter run
    ```

## 🔒 Security & Performance
- **Isolation**: Each WASM job runs in a background thread isolated from the main UI process.
- **Throttling Protection**: Execution is managed via Coroutines and `Dispatchers.Default` to prevent UI lag.
- **Resource Limits**: Configurable timeouts (default 5s) protect the node from runaway processes.
- **Secret Management**: Environment variables are handled via `flutter_dotenv` and are git-ignored to prevent key leakage.

## 🤝 Contributing
This is an open-source project. Contributions to the WASM runtime integration or the policy engine are highly welcomed!

---
*Built with ❤️ for the Devlup Community.*

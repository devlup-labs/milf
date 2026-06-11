# Flutter Architecture Documentation (Consumer Node)

This document provides a comprehensive overview of the **Consumer Node** Flutter application. It breaks down the full architectural proposal (Future State) and analyzes the current implementation (Initial Reference).

---

## 1. Full Architectural Proposal (Future State)

The following structure represents the target architecture for a production-grade, modular, and scalable mobile compute node.

### Core Structure (`lib/core/`)
Focuses on cross-cutting concerns and global configurations.
*   **`theme/`**: Centralized design system. Ensures a premium, consistent dark-mode aesthetic across all screens.
*   **`router/`**: Declarative routing (e.g., using `go_router`) to handle navigation between auth, dashboard, and settings.
*   **`constants/`**: Environment-specific URLs and hardcoded configuration values.
*   **`widgets/`**: Reusable atomic UI components (Buttons, TextFields, Cards) following the design system.

### Features Layer (`lib/features/`)
Domain-driven modules that encapsulate specific business logic.
*   **`auth/`**: Manages the user lifecycle.
    *   **Login/Register**: Views and Controllers handling Google OAuth and JWT acquisition.
*   **`dashboard/`**: The main interface showing the "Node Heartbeat" and active execution status.
*   **`logs/`**: Detailed telemetry. Displays real-time WASM logs and system events.
*   **`billing/`**: **[Proposed]** Visualizes earnings from providing compute resources. Tracks "Compute Time" converted to currency.
*   **`settings/`**: Allows users to set resource quotas (e.g., "Only run on Wi-Fi," "Max RAM: 2GB").

### Service Layer (`lib/services/`)
Handles raw data and external communication.
*   **`api_service.dart`**: Centralized HTTP client for REST requests.
*   **`websocket_service.dart`**: Manages the persistent duplex connection for receiving low-latency WASM tasks.
*   **`auth_service.dart`**: Specifically handles token refresh logic and session persistence.

### Models & Storage (`lib/models/` & `lib/storage/`)
*   **Models**: Strongly typed Data Transfer Objects (DTOs) for Users, Logs, and Billing records.
*   **Secure Storage**: Uses `flutter_secure_storage` (Keychain/Keystore) to save sensitive JWTs and private keys.

---

## 2. Actual Implementation Analysis (Initial Reference)

The current implementation is a **simplified, high-performance flat structure** designed to validate the core mission: **Executing WASM on Android.**

### Why is it simplified?
The project is currently in the **Execution Validation Phase**. Adding complex multi-file routing and billing features before the WAMR JNI bridge is stable would increase technical debt.

### What is actually implemented?
| File | Role | Actual Feature Set |
| :--- | :--- | :--- |
| **`main.dart`** | Entry Point | Bootstraps the app with `ChangeNotifierProvider` for global state. |
| **`node_controller.dart`** | Logic Center | **Implemented:** JNI MethodChannel bridges, JSON payload parsing, and task routing. |
| **`cloud_sync.dart`** | Connection | **Implemented:** WebSocket management, automatic reconnection, and node registration. |
| **`node_screen.dart`** | UI | Real-time log visualization and connection toggle. |

### Current "Billing & Policy" State
*   **Policy**: Currently hardcoded to accept all incoming tasks if the WAMR runtime is initialized.
*   **RAM Management**: The `CloudSync` service currently sends a **mock heartbeat** (`ram_available_mb: 2048`).
*   **Billing**: Not yet implemented. All "Compute Time" is logged in the `ExecutionRecord` history but not converted to a financial model.

### Next Steps for Implementation:
1.  **Refactor to `features/`**: Move the current `node_screen.dart` into `features/dashboard/`.
2.  **Telemetry Analysis**: Expand `ExecutionRecord` to include peak RSS memory and CPU cycles (read from the C++ layer) to provide data for the **Billing** module.
3.  **Real OS Stats**: Replace the mock 2048MB RAM heartbeat with the real values from the `OSStatsReader` documented in the Consumer Architecture.

---
**Summary:** The current `lib` is a "Vertical Slice"—it does the hardest part (Task -> WAMR -> Output) perfectly but leaves the "Lateral Features" (Auth, Billing, Themes) for the next development sprint.

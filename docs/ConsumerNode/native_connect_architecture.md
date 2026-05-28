# Native Kotlin Connect: Flutter-WAMR Bridge Architecture

This document specifies the architecture, execution flow, and module design for the **Native Kotlin Connect** layer. This bridge is responsible for the secure, high-performance orchestration of WebAssembly (WASM) modules within an Android isolated process.

---

## 1. Step-by-Step Execution Lifecycle

The lifecycle of a WASM task is divided into 12 discrete steps to ensure deterministic behavior and host safety.

### Phase 1: Initiation & Intent
1.  **User Initiates Execution (Flutter):**
    *   The user triggers an action in the Flutter UI.
    *   Flutter builds a `RuntimeConfig` (specifying memory limits, timeouts, and sandbox permissions).
    *   The request is sent through the `MethodChannel`. No code is executed yet; this is purely an **intent**.

### Phase 2: Command Entry
2.  **Native Bridge Intake (Android):**
    *   `WamrBridgePlugin.kt` receives the call and forwards it to the `WamrBridgeService`.
    *   `CommandHandler.kt` parses and validates the parameters, converting the Flutter JSON config into a native-friendly object. No memory is allocated for the WASM module yet.

### Phase 3: Admission & Sandbox
3.  **Admission Control:**
    *   `AdmissionController.kt` queries `SlotManager` to check if enough RAM is available and if the concurrency limit has been reached.
    *   If rejected, Flutter receives a failure. If accepted, a **resource slot** is reserved.
4.  **Sandbox Initialization:**
    *   `IsolationManager.kt` launches an `isolatedProcess`.
    *   Android namespaces (PID, Mount, Network) are applied to harden the process.
    *   Inside the runtime, `seccomp` syscall filters are applied, and sandbox policies are loaded.

### Phase 4: Runtime & Loading
5.  **WAMR Core Initialization:**
    *   `runtime_init.c` initializes the WAMR VM and registers **only safe host functions**. No WASM is loaded yet.
6.  **WASM Loading & Linking:**
    *   `wasm_loader.c` parses and validates the WASM bytecode.
    *   `module_linker.c` resolves imports. If any disallowed imports are found, the execution is aborted and the slot is released.

### Phase 5: Enforcement & Execution
7.  **Memory Setup:**
    *   `linear_memory.c` and `heap_allocator.c` allocate bounded memory segments.
    *   `stack_manager.c` enables stack guards. `memory_accountant.c` becomes the source of truth for usage tracking.
8.  **Execution Loop:**
    *   `executor.c` starts instruction execution using either the Interpreter or AOT loader.
9.  **Real-time Monitoring:**
    *   `sampler_loop.c` periodically samples RSS, Heap, and Stack usage.
    *   If `threshold_watcher.c` predicts an Out-Of-Memory (OOM) event, it kills the execution **before** the host OS intervenes.

### Phase 6: Observability & Cleanup
10. **IPC & Streaming:**
    *   Live metrics and execution logs are streamed through `metrics_channel` and `logs_channel` back to Flutter via an `EventChannel`.
11. **Termination:**
    *   **Normal:** `stop.c` terminates the instance and `SlotManager` frees the resource slot.
    *   **Forced:** `forced_kill.c` triggers an immediate halt and `crash_handler.c` dumps diagnostics to Flutter.
12. **Snapshot & Recovery (Optional):**
    *   If enabled, the linear memory is dumped to `memory_snapshot.bin` for pause/resume or stateful workload recovery.

---

## 2. Directory Structure (`flutter_wamr_bridge/`)

```text
flutter_wamr_bridge/
└── native/android/
    ├── entry/          # Plugin entry & Background Service
    ├── ipc/            # MethodChannel/EventChannel Handlers
    ├── lifecycle/      # Process & Namespace management
    ├── scheduler/      # Priority-based task queuing
    ├── slot/           # Resource slot & Admission control
    ├── instance/       # Single WASM unit management
    ├── memory/         # Samplers, Accountants, & Enforcers
    ├── logs/           # Log & Crash dispatchers
    ├── wamr/           # JNI links to C++ WAMR
    └── binder/         # AIDL files for Service IPC
```

---

## 3. Module Breakdown

### A. Flutter (Dart) Layer
*   **`RuntimeAPI`**: High-level interface for developers (`start()`, `stop()`, `pause()`).
*   **`MetricsAPI`**: Stream-based abstraction for real-time memory/CPU charts.
*   **`RuntimeConfig`**: Model for memory (in MB), timeouts (in ms), and permission bits.

### B. Android (Kotlin) Native Bridge
*   **`WamrBridgeService`**: A Foreground Service (to prevent Doze mode) that manages the lifecycle of the execution environment.
*   **`IsolationManager`**: Implements Android `isolatedProcess="true"` logic.
*   **`MemorySampler`**: Uses `android.os.Debug` to collect real-time RSS stats.

### C. Common Layer (Protocol & Security)
*   **`Command.proto`**: Shared Protobuf schema for all control messages.
*   **`PermissionGuard`**: Validates every command against the user's signed policy before execution.
*   **`InputSanitizer`**: Prevents "Zip Bombs" or malformed WASM binaries from being loaded.

---

## 4. Why This Design is Correct
1.  **Deterministic Execution**: All resources are bounded before the first instruction runs.
2.  **Hard Isolation**: Uses OS-level namespaces; WAMR code cannot see the host filesystem.
3.  **Pre-OOM Protection**: Monitoring usage *inside* the bridge prevents the entire app from being killed by the Android LMK (Low Memory Killer).
4.  **Pure Runtime**: WAMR stays a "dumb" execution engine; all policy and admission logic is handled in the safer Kotlin/Java layer.

**Summary:** This bridge lets **Policy** decide *if* execution happens, the **Sandbox** decide *where* it happens, and **WAMR** decide *how* it executes—while every byte is continuously monitored.

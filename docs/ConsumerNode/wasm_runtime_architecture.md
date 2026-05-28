# WASM Sandbox Runtime: Deep Dive Architecture

This document provides a detailed technical specification of the **WASM Sandbox Runtime**. It is structured to mirror the system's modular design as visualized in the Eraser architectural diagram, detailing the interactions between security, execution, and monitoring layers.

---

## 1. Security Boundary (The Outer Shell)
The Security Boundary is the primary defense layer. It ensures that any WASM execution is strictly contained and cannot impact the host system.

### A. Configuration (`Config`)
*   **`RuntimeConfig`**: Defines the versioning and feature toggles (e.g., enabling/disabling WASI or specific host functions).
*   **`MemoryPolicy`**: Sets hard caps on linear memory, maximum heap growth, and stack depth.
*   **`SandboxPolicy`**: A declarative ruleset defining what the sandbox is allowed to do (e.g., "Allow read-only access to /tmp").

### B. Isolation (Linux Namespaces)
*   **`PID Namespace`**: Virtualizes the process tree. The WASM process sees itself as PID 1 and cannot see or signal other host processes.
*   **`Mount Namespace`**: Creates a private file system view. The process typically sees an empty root or a restricted virtual filesystem.
*   **`Network Namespace`**: Either provides a private network stack or completely severs network access to prevent data exfiltration.

### C. Seccomp (Syscall Filtering)
*   **`Security Filter`**: Enforces a strict whitelist of allowed Linux syscalls.
*   **`Seccomp Profile`**: A JSON-based configuration that determines exactly which syscalls are permitted based on the active policy.

### D. Resource Limits (Cgroups)
*   **`CgroupMemory`**: Enforces the hard RAM limit at the kernel level.
*   **`CgroupCpu`**: Manages CPU shares and throttling to prevent a single lambda from starving the host device.
*   **`CgroupIO`**: Limits disk/network IO bandwidth to prevent resource exhaustion.

---

## 2. Core Execution Engine
This block represents the actual WAMR integration and instruction processing.

### A. Loader & Linker
*   **`WasmLoader`**: Parses the binary, validates the module structure, and checks for bytecode integrity.
*   **`ModuleLinker`**: Resolves imports and binds the allowed host functions defined in the `Data Adapter`.
*   **`Executor`**: The main driver that manages the instruction execution loop.

### B. Engine Types
*   **`Interpreter`**: Executes instructions directly from bytecode. Best for low-latency startup.
*   **`AotLoader`**: Loads pre-compiled machine code for near-native performance.
*   **`JITGuard`**: A specialized security module that monitors and restricts dynamic code generation to prevent JIT-based exploits.

---

## 3. Memory System & Real-Time Tracking
The memory system is the most complex part of the runtime, requiring constant accounting and enforcement.

### A. Memory Layout
*   **`LinearMemory`**: The raw WASM linear memory block.
*   **`HeapManager`**: A deterministic allocator that manages memory requests *inside* the WASM environment.
*   **`StackManager`**: Allocates per-thread stacks and manages stack-pointer guards to prevent overflows.

### B. Accounting & High-Watermark
*   **`MemoryAccountant`**: Acts as the single source of truth for total memory usage (Linear + Heap + Stack).
*   **`HighWatermark`**: Records the peak memory usage during an execution's lifecycle for billing and optimization.

### C. Real-Time Tracking (The Watchdog)
*   **`SampleLoop`**: A periodic background thread that samples the `MemoryAccountant`.
*   **`DeltaCalculator`**: Computes the rate of change in memory usage between samples.
*   **`OomPredictor`**: Analyzes the delta to detect memory leaks or "runaway" allocations. It can trigger an emergency shutdown *before* the OS kills the process.

---

## 4. System Management & Lifecycle
Manages the "Birth, Life, and Death" of a WASM instance.

### A. Lifecycle Controller
*   **`SandboxInit`**: Initializes the Security Boundary (Namespaces/Seccomp) **before** any code is loaded.
*   **`RuntimeInit`**: Initializes the WAMR core and registers the host function table.
*   **`CrashHandler`**: Captures diagnostics, stack traces, and violation reasons if an execution fails.

### B. Storage & Snapshots
*   **`MemorySnapshot`**: Serializes the linear memory and heap state to disk.
*   **`RuntimeState`**: Stores the CPU state (registers/program counter) for pause and resume functionality.
*   **`AotCache`**: Maintains a local cache of verified AOT binaries to speed up subsequent executions.

---

## 5. External Interface (External Communication)
The gateway between the sandboxed process and the Outside World (Kotlin Connect).

### A. Data Adapter
*   **`SystemCallDispatcher`**: Intercepts WASI or custom calls and validates them against the `SandboxPolicy`.
*   **`HostFunctions`**: The implementation of safe host APIs (e.g., `milf_pdf_generate`) that WASM is allowed to call.

### B. IPC & Observability
*   **`Control/Data Plane`**: The primary communication channel (Socket/Pipe) for receiving commands (Start/Stop/Pause) and streaming logs.
*   **`ExecutionLogger`**: Captures `stdout`/`stderr` from the WASM runtime and routes them to the `Data Plane`.
*   **`LiveMetricsWS`**: A high-frequency WebSocket-capable stream that sends real-time RAM/CPU charts to the dashboard.

---

## 6. Documented Flows (As per Eraser Diagram)

1.  **Initialization Flow**: `LIFECYCLE` (SandboxInit) → `SECURITY BOUNDARY` (Isolation/Config) → `CORE EXECUTION` (Loader).
2.  **Execution Heartbeat**: `CORE EXECUTION` (Engine) → `MEMORY SYSTEM` (Accounting) → `REALTIME TRACKING` (SampleLoop).
3.  **Policy Enforcement**: `REALTIME TRACKING` (OomPredictor) → `LIFECYCLE` (ForcedKill/CrashHandler).
4.  **Data Feedback**: `CORE EXECUTION` (Engine) → `DATA ADAPTER` (HostFunctions) → `EXTERNAL INTERFACE` (Observability/Logs).

---
**Conclusion:** This design ensures that **Execution** is always subordinate to **Policy**. By decoupling memory accounting from the execution engine, the system remains observable even if the WASM module becomes unresponsive or enters an infinite loop.

# Consumer Node Architecture (consumeronlywamr)

This document outlines the first-look architecture for the **Consumer Node** (mobile worker node). The system is a hybrid application featuring a UI/Control plane written in Flutter, a system-level Orchestrator in Kotlin, and a high-performance execution engine running WAMR in C/C++.

> [!IMPORTANT]
> **Current Implementation Status**
> Please note that while this document maps out the complete target architecture, the *current* implementation (as reflected in the PPT and READMEs) is focused heavily on the **execution engine and output sending**. 
> - **Implemented:** WAMR C++ compilation, basic Process Isolation (`isolatedProcess="true"`), JNI/AIDL bridging, executing WASM, and sending the output payload back to the cloud via WebSockets.
> - **Future Work (Not Yet Implemented):** Detailed Billing metrics, the Advanced Policy Checker (thermal throttling/deep OS resource checks), Output/Error Analyzers, and Zero-Copy Ashmem optimization. These remain architectural proposals.

## Eraser.io Diagram Code (Function Accept Case)

You can copy this code into [Eraser.io](https://www.eraser.io/) to reproduce the exact "Function Accept Case" flow you proposed:

```eraser
// Nodes
CentralServer [icon: server, label: "Central Server"]

group FlutterOrchestrator [label: "Flutter Orchestrator"] {
  CloudConnect [icon: cloud, label: "Cloud Connect"]
  PolicyChecker [icon: check, label: "Policy Checker"]
  KotlinConnect_F [icon: activity, label: "Kotlin Connect"]
  OutputLayer_F [icon: log-out, label: "Output Layer"]
}

group KotlinOrchestrator [label: "Kotlin Orchestrator"] {
  FlutterConnect [icon: activity, label: "Flutter Connect"]
  OSStatsReader [icon: cpu, label: "OS Stats Reader"]
  ProcessManager [icon: settings, label: "Process Manager"]
  OutputLayer_K [icon: log-out, label: "Output Layer"]
}

group NativeSandbox [label: "Native Sandbox Initializer (C++)"] {
  KotlinConnect_C [icon: code, label: "Kotlin Connect (JNI)"]
  SandboxInitializer [icon: box, label: "Sandbox Initializer"]
  OutputAnalyzer [icon: alert-circle, label: "Output/Error Analyzer"]
}

WAMR [icon: terminal, label: "WAMR Runtime (Sandboxed Exec)"]

// Edges - Admission Flow
CentralServer > CloudConnect: 1) Job (jobID, funcID, metadata, input, wasm)
CloudConnect > PolicyChecker: 2) Send check req
PolicyChecker > KotlinConnect_F: 3) Send resource check req
KotlinConnect_F > FlutterConnect: 4) Ask OS resources (MethodChannel)
FlutterConnect > OSStatsReader: Request stats
OSStatsReader > FlutterConnect: 5) OS Resources (CPU, Mem, Temp)
FlutterConnect > KotlinConnect_F: 6) Resource Details
KotlinConnect_F > PolicyChecker: 7) Send OS Resource details

// Edges - Execution Flow
PolicyChecker > ProcessManager: 8) Process Allocation Req (Persistent Connection)
ProcessManager > KotlinConnect_F: 9) Process confirmation
KotlinConnect_F > ProcessManager: 10) Function details (wasm_binary, input)
ProcessManager > KotlinConnect_C: 11) Sandbox details (JNI - binary, mem limit)
KotlinConnect_C > SandboxInitializer: 12) Sandbox initialize details
SandboxInitializer > WAMR: 13) Runtime Initialized (binary execution)

// Edges - Output Flow
WAMR > OutputAnalyzer: 14) Output details (result/error)
OutputAnalyzer > KotlinConnect_C: 15) Send output details
KotlinConnect_C > OutputLayer_K: 16) Send output back to Kotlin
OutputLayer_K > KotlinConnect_F: 17) Output details (PID, funcID, output)
KotlinConnect_F > OutputLayer_F: Route to UI/Cloud
OutputLayer_F > CloudConnect: 18) Prepare Cloud Payload
CloudConnect > CentralServer: 19) Output to cloud
```

---

## 1. Flutter Orchestrator (Control Plane)
The Flutter layer acts as the high-level brain, handling external networking and user interface (if any).

*   **Cloud Connect:** Manages the WebSockets/HTTP polling with the Central Server. Implements `receiveJob()`, `heartBeat()`, and `postResult()`. It is strictly network-focused.
*   **Policy Checker:** The gateway. Before accepting a job, it calls `checkAdmission()`. It queries the Kotlin layer for current hardware capacity and rejects jobs if the device is overheating or out of memory.
*   **Kotlin Connect:** Uses Flutter `MethodChannel` or `EventChannel` (for persistent streams) to communicate down to the native OS layer.
*   **Output Layer:** Formats the raw byte results returned from the WASM execution into JSON/Payloads for the Central Server.

## 2. Kotlin/Java Orchestrator (System Plane)
The Kotlin layer has access to the low-level Android OS APIs necessary for resource monitoring and JNI (Java Native Interface) bridging.

*   **OS Stats Reader:** 
    *   Implements `getOSResources()` using `android.os.Debug.MemoryInfo` and the `ActivityManager` to read device PSS (Proportional Set Size) memory. 
    *   Reads `/proc/stat` or uses Android's `HardwarePropertiesManager` to get CPU frequencies and thermal throttling status.
*   **Process Manager:**
    *   Tracks active execution slots (`checkAvailableProcesses()`). 
    *   Responsible for waking the device or managing Android WakeLocks if the screen is off (Hibernation logic).
*   **JNI Bridge:** Passes the `.wasm` bytes down to the C++ runtime.

## 3. Native Sandbox Initializer (C/C++ & WAMR)
This layer runs the actual WebAssembly Micro Runtime (WAMR). It must be written in C/C++ (`native-lib.cpp`) using Android's NDK.

*   **Sandbox Initializer:** Configures the WAMR environment. It takes the limits passed from Kotlin (e.g., max memory footprint) and enforces them on the WASM module.
*   **WAMR Execution:** Instantiates the WASM module and passes the input payload bytes to the compiled wrapper function.
*   **Output/Error Analyzer:** Captures standard output, return values, and memory segmentation faults (if the WASM binary crashes), safely packaging them back to Kotlin without crashing the host Android app.

---

## 4. Advanced Technical Proposals (Memory & Performance)

Based on the research links provided, here are critical optimizations for the Android Consumer Node:

### A. Memory Management (Zero-Copy Transfer)
*   **The Problem:** Passing a 5MB `.wasm` binary from Flutter -> Kotlin -> C++ normally involves copying the byte array twice. In a constrained mobile environment, this causes massive Garbage Collection (GC) spikes and out-of-memory crashes.
*   **The Solution (Ashmem / Direct ByteBuffer):**
    *   Instead of standard byte arrays, use **Android Shared Memory (Ashmem)** or **JNI Direct ByteBuffers** (`ByteBuffer.allocateDirect()`).
    *   This allows the C++ WAMR layer to read the exact same memory address space that Kotlin allocated, achieving a **Zero-Copy** transfer. The binary is passed instantly without duplicating RAM usage.

### B. OS Metrics Reliability
*   Monitoring CPU accurately on modern Android versions (Android 8+) is restricted. You cannot easily read other apps' `/proc/` stats. 
*   **Resolution:** The `OS Stats Reader` should focus strictly on the current app's memory profile using `Debug.getMemoryInfo()` and the system's thermal status. If the Android OS reports thermal throttling, the `Policy Checker` should reject incoming jobs to prevent the OS from killing the Consumer App.

### C. Hibernation & Background Execution
*   Android aggressively kills background processes to save battery (Doze Mode).
*   **Resolution:** The Kotlin layer must start a **Foreground Service** with an ongoing Notification if it intends to act as a worker node while the app is closed. Otherwise, the Central Server's `jobID` will fail midway through execution when the OS suspends the app.

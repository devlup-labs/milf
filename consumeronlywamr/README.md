# Wasm-Android-Sandbox (W.A.S.)

[![Security: Sandbox](https://img.shields.io/badge/Security-Isolated-blue.svg)](https://developer.android.com/guide/components/processes-and-threads#IPC)

A high-performance, secure sandbox for executing WebAssembly (WASM) modules on Android. Built with **WAMR (WebAssembly Micro Runtime)** and **C++/JNI**, this project focuses on strict resource isolation and security.

## 🚀 Key Features

- **Process Isolation**: Leverages Android's `isolatedProcess="true"` service. Execution happens in a separate UID/process, protecting the main UI and sensitive data.
- **Strict Resource Quotas**:
  - **Memory Tracking**: Real-time RSS monitoring and configurable heap/stack limits (currently set to 512MB/16MB).
  - **CPU Monitoring**: Execution tracking to prevent "busy-wait" or infinite loop attacks.
- **Dynamic Invocation**: Supports JNI-bridged calls to exported WASM functions with dynamic parameter mapping.
- **Direct IPC**: Uses AIDL (Android Interface Definition Language) for low-latency communication between the application and the sandbox.

## 🛠 Tech Stack

- **Runtime**: WebAssembly Micro Runtime (WAMR)
- **Language**: C++ (Core Engine), Kotlin (Android Glue), Flutter (Demo UI)
- **Interface**: JNI & AIDL

## 🏗 Architecture

The sandbox is designed for environments where untrusted code must be executed locally (e.g., Bitcoin script validation, private computation).

1. **Host App**: A Flutter/Kotlin interface targets the `IWasmService`.
2. **Secure Service**: A background service running in an isolated process.
3. **Native Engine**: The C++ layer initializes the WAMR interpreter, sets up memory trackers, and executes the WASM bytecode.

## 📊 Performance & Monitoring

Built-in `ExecutionMonitor` collects:
- Total execution time (ms)
- Peak memory usage
- Instruction counts (via WAMR hooks)

## 🏗 Getting Started

1. Clone the repository.
2. Initialize WAMR submodule in `android/app/src/main/cpp/wamr`.
3. Build using Android Studio (requires NDK).

---
*Developed as part of a mission to enable secure client-side execution in Bitcoin-centric mobile applications.*

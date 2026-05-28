# Security & Threat Model: Hardened WASM Sandboxing

This document provides a deep-dive security analysis of the MILF platform. Given that our core value proposition is executing untrusted code on user devices, we maintain a **Zero-Trust** posture regarding the WebAssembly (WASM) modules.

---

## 1. The Trust Model

*   **Trusted Components:** Central Server, Kotlin Orchestrator, Native Sandbox Initializer.
*   **Untrusted Components:** User-submitted WASM binaries, Input Payloads, External Network Data.
*   **Goal:** Ensure that an exploit in an **Untrusted Component** cannot compromise the integrity, availability, or privacy of the **Trusted Components** or the host Android OS.

---

## 2. Attack Vectors & Deep Analysis

### A. Sandbox Escape (Runtime Exploits)
*   **Threat:** A malicious WASM module exploits a vulnerability in the WAMR runtime (e.g., a buffer overflow in the interpreter) to execute arbitrary machine code on the host.
*   **Analysis:** If the runtime is compromised, the attacker gains the privileges of the process running the runtime.
*   **Mitigation:** 
    *   **Layer 1 (Bytecode Validation):** WAMR performs strict validation of the module structure, types, and function signatures before instantiation.
    *   **Layer 2 (Isolated Process):** The runtime executes within an Android `isolatedProcess="true"`. This process has a unique UID and **zero permissions**. It cannot access the camera, contacts, or storage, even if the runtime is escaped.
    *   **Layer 3 (Seccomp Filters):** Even inside the isolated process, we apply a Seccomp whitelist. If an attacker tries to call `execve` or `open` on a sensitive path, the kernel immediately kills the process.

### B. Resource Exhaustion (DoS)
*   **Threat:** A "Zip Bomb" WASM binary or an infinite loop (`while(true)`) designed to freeze the host device or drain the battery.
*   **Analysis:** CPU and Memory are finite. On mobile, excessive usage triggers the Low Memory Killer (LMK) or thermal throttling.
*   **Mitigation:**
    *   **Memory Quotas:** The `MemoryAccountant` enforces hard caps on the Linear Memory and Heap. This is backed by **Cgroup Memory limits** at the OS level.
    *   **Execution Monitoring:** We use WAMR's "Instruction Counting" or periodic watchdog timers. If a function exceeds its `timeout_ms`, the `WasmController` interrupts the execution loop.

### C. Data Exfiltration & Side-Channels
*   **Threat:** A module uses timing attacks (Spectre/Meltdown style) or unauthorized network calls to steal data from other executions or the host.
*   **Analysis:** WASM memory is isolated, but shared hardware resources (cache, branch predictors) can leak information.
*   **Mitigation:**
    *   **Deterministic Execution:** We disable unsafe JIT paths and high-resolution timers inside the sandbox to mitigate timing attacks.
    *   **Network Siloing:** By default, the `Network Namespace` is restricted. Any network call must go through the `HostInterface` dispatcher, which validates the target URL against a whitelist.

### D. Malicious Input Payloads
*   **Threat:** Providing a specially crafted JSON payload that causes the WASM-Host boundary (JNI) to crash.
*   **Mitigation:** 
    *   **Safe Serialization:** Use `DirectByteBuffers` with explicit length checks. 
    *   **Input Sanitization:** The `CommandHandler` validates the payload size and schema before passing it to the C++ layer.

---

## 3. Defense-in-Depth Summary

| Defense Layer | Technology | Primary Protection |
| :--- | :--- | :--- |
| **User Land** | Flutter Auth | Prevents unauthorized job dispatch. |
| **Middleware** | Kotlin Admission Controller | Denies execution if device state is unstable. |
| **Runtime** | WAMR Validator | Prevents malformed/illegal bytecode execution. |
| **Logic** | Memory Accountant | Prevents Heap/Stack overflows. |
| **Process** | Android `isolatedProcess` | Prevents access to Android System Services/Hardware. |
| **Kernel** | Seccomp & Cgroups | Restricts Syscalls and enforces hard hardware quotas. |

---

## 4. Residual Risks & Ongoing Research

1.  **Zero-Day Runtime Vulnerabilities:** While WAMR is highly audited, new exploits are possible. Our defense relies on the OS-level `isolatedProcess` to contain these.
2.  **Thermal Throttling:** A high volume of small, safe tasks can still heat the device. We are implementing a **Cool-down Policy** in the `AdmissionController` to reject tasks if the battery temperature exceeds 40°C.
3.  **Kernel Vulnerabilities:** The "last mile" of security is the Android Kernel. We keep the system calls limited to the absolute minimum required for WASI to minimize the attack surface.

---

**Deep Analysis Conclusion:** The MILF security model is built on the principle of **Redundant Containment**. We do not rely on any single technology (like WAMR) to be perfect. Instead, we stack independent security technologies such that an attacker must break at least three different layers (WASM, JNI, and Kernel) to achieve a meaningful compromise.

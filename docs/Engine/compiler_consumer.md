# Compiler & Consumer (Execution Engine)

This document details the inner workings of the two most critical low-level components of the architecture: The **Compiler Module** (which transforms arbitrary user code) and the **Consumer Node** (which executes the code on the edge).

## 1. Working Principle

The core philosophy of this engine is **Ahead-of-Time (AOT) Standardization**. Instead of maintaining bloated containers with Python, Node, Go, and C++ runtimes on every worker node, the system uses WebAssembly (WASM) as the universal execution format.

### The Pipeline:
1. **Intake & Build Context Setup:** The Compiler Module receives the raw code (File, Zip, or Docker context). It provisions an isolated `/tmp/job-id/src/` workspace.
2. **Dynamic ABI Wrapping:** The compiler analyzes the function signature and auto-generates a "Wrapper". The runtime does not know about the user's `add(int a, int b)` function. It only knows `run(input_ptr, input_len, output_ptr, output_cap)`. The auto-generated wrapper translates the runtime's raw byte arrays into the types the user's function expects.
3. **Targeted Compilation:** The code (User Code + Wrapper) is compiled targeting the `wasm32-wasi` architecture.
4. **Distribution & Execution:** The resulting `.wasm` binary is stored. When invoked, the **Consumer Node** (running WAMR - WebAssembly Micro Runtime) downloads the binary. The consumer allocates a linear memory sandbox, injects the input payload into the memory pointer, and calls the `run` wrapper function.

---

## 2. Ease to Support Multiple Languages

The architecture makes adding support for new languages incredibly simple, provided the language can compile to the WebAssembly System Interface (WASI) target.

### Why it's easy:
- **The Consumer Node Never Changes:** The consumer (worker node) is completely language agnostic. It runs a WAMR instance. It doesn't care if the binary was written in Rust, C++, or Go. It only cares that it's a valid WASM binary exposing the standard `run` entrypoint.
- **Pluggable Toolchains:** To add a new language (e.g., Swift or Zig), you only need to modify the Central Server's Compiler Module:
  1. Add a new `case "zig": compileZig(ctx)` block.
  2. Install the Zig toolchain on the Compiler server.
  3. Write a small string-template generator for the ABI Wrapper in Zig.
- **Zero Runtime Bloat:** Traditional serverless platforms (like AWS Lambda) have to distribute heavy runtime layers (e.g., an 80MB Python environment) to worker nodes. Here, the language toolchain is only required *once* on the Central Compiler. The edge nodes remain extremely lightweight.

---

## 3. Runtime Analysis Over Real-World Function Types

Using WASM (WAMR) on consumer nodes provides unique performance profiles compared to traditional Docker-based serverless functions.

### A. Compute-Heavy Functions (e.g., Image Processing, Cryptography)
- **Performance:** Near-native speeds. Languages like C, C++, and Rust compile down directly to highly optimized WASM instructions. WAMR can further utilize AOT (Ahead-Of-Time) or JIT (Just-In-Time) compilation on the edge node to translate WASM to native machine code (x86/ARM) for maximum throughput.
- **Memory Footprint:** Extremely deterministic. A C++ function processing an image will only use exactly the memory allocated to its linear WASM memory block (e.g., 64MB). There is no garbage collector overhead.

### B. "Cold Start" Scenarios (e.g., Infrequent Triggers)
- **Performance:** **Exceptional.** Traditional Docker lambdas suffer from 500ms to 2-second cold starts because they must spin up a container namespace, cgroups, and start a runtime process. 
- **WASM Advantage:** WAMR can instantiate a WASM module in **microseconds** (often < 5ms). This makes the platform ideal for hyper-bursty, high-concurrency workloads.

### C. State/Event-Driven Functions (e.g., Data Transformation, JSON Parsing)
- **Performance:** Highly efficient, provided the ABI Wrapper serializes and deserializes data quickly. 
- **Limitation & Mitigation:** Moving large JSON payloads back and forth across the WASM boundary (from host memory to WASM linear memory) can incur a serialization cost. 
- **Real-World Impact:** For small event payloads (like IoT triggers or webhook payloads), the parsing cost is negligible. For massive datasets, utilizing shared memory protocols or chunked streams across the boundary is required.

### D. I/O Bound Functions (e.g., Fetching from external APIs)
- **Performance:** Reliant on WASI capabilities. Because WASM runs in a strict sandbox, network I/O is not inherently native to raw WASM. It requires the WASI (WebAssembly System Interface) socket extensions or host-callbacks.
- **Real-World Impact:** If the consumer node supports WASI networking, the performance is standard. However, the true strength of this architecture shines in data-processing and computation rather than heavy network orchestrations.

## Summary Conclusion
The Compiler -> WAMR Consumer pipeline achieves what Docker-based FaaS platforms struggle with: **True Microsecond Cold Starts** and **Universal Edge Portability**. By forcing the complexity onto the Central Compiler (toolchains and wrapper generation), the edge consumers remain dumb, lightning-fast, and universally compatible with any compiled language.

# Advanced Architecture Decisions

This document captures the design philosophy and technical decisions regarding the hybrid testing module, file handling, and queue management for the Central Server and Client integration.

## 1. Hybrid Testing Module (Client + Server)

The hybrid approach correctly splits responsibilities to maximize user experience while maintaining security.

*   **Client Side (The IDE Experience):** Integrating the **Monaco Editor** is the industry standard (used by VS Code, AWS Lambda Console, CodeSandbox). It handles:
    *   Syntax and color highlighting.
    *   Bracket matching and auto-completion.
    *   Basic linting without heavy computation.
*   **Server Side (The Source of Truth):** The server must handle the actual "Validation Module". It compiles the code, resolves dependencies, and runs unit tests in an isolated sandbox. The client cannot be trusted to validate its own code for execution safety.

## 2. Docker Based Input

*   **Observation:** Flattening a Docker image to send over raw HTTPS is inefficient and anti-pattern.
*   **Resolution:** Functions deployed via Docker should follow standard OCI (Open Container Initiative) workflows. The client shouldn't upload a raw Dockerfile and context directly to the execution engine. Instead, the user pushes to a **Container Registry** (like Docker Hub or a private registry), and the FaaS platform pulls the image URI. 
*   **Alternative:** If direct upload is required, the client uploads a `.zip` file of the build context, and the server's build-runner daemon builds the image remotely.

## 3. Chunking & File Security

You identified a critical tradeoff in file handling: **Platform Features vs. Zero-Knowledge Privacy.**

*   **Chunk → Hash (Standard Protocol):** 
    *   *How it works:* The file is chunked and uploaded over standard TLS/HTTPS. The server hashes the chunks for integrity checks.
    *   *Benefits:* Allows the FaaS platform to index the files, provide previews, and run server-side search.
*   **Hash/Encrypt → Chunk (Client-Side Encryption):**
    *   *How it works:* The client encrypts the file before chunking and uploading.
    *   *Cons (as noted):* 
        *   **Editing is hard (❎):** You cannot edit the file server-side.
        *   **Sharing is complex (🤔):** To share the file, you must securely distribute the decryption keys using Public Key Infrastructure (PKI), meaning the FaaS platform acts only as a dumb blob store.
*   **Decision:** For a Serverless platform where code needs to be executed, the **Chunk → Hash** method is required for source code. End-to-end encryption (Hash -> Chunk) should only be an option for arbitrary data/storage buckets that the functions consume, not the functions themselves.

## 4. Queue Management (Resource & Priority Based)

*   **Proposed Architecture:** Multiple queues sorted by resource requirements, delay time, and a calculated score.
*   **The Risk (Starvation):** Your intuition is 100% correct. This is a classic operating systems problem. If you prioritize fast, low-resource tasks, a heavy task might sit in the queue forever if lightweight tasks keep arriving (Starvation).
*   **The Solution:** 
    1.  **Aging:** Gradually increase the priority score of a task the longer it sits in the queue, guaranteeing it eventually executes.
    2.  **Dedicated Resource Pools:** Instead of purely priority-based routing, route tasks to different *worker pools* (e.g., a fast-lane pool for lightweight functions, and a heavy-compute pool for long-running tasks). This isolates the workloads so they don't starve each other.

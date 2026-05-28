# Central Server Architecture & Design

This document covers the architectural layout, modules, and component interactions of the Central Server, based on the provided design pseudo-code and flow diagrams.

## Eraser.io Diagram Code

You can copy and paste the following code into [Eraser.io](https://www.eraser.io/) to generate the exact architectural flow diagram provided in your sketch.

```eraser
// Nodes
Client [icon: monitor, label: "Client's Client"]
User [icon: user, label: "Developer/User"]
AccountsPlane [icon: user-cog, label: "Accounts Plane"]
UserDB [(icon: database, label: "User Database")]
FuncGateway [icon: server, label: "Func Gateway"]
RawFuncStore [(icon: file, label: "Raw Func Data Store (File Level)")]
CompilationUnit [icon: tool, label: "Compilation Unit"]
ObjectDB [(icon: database, label: "Object Database")]
Orchestrator [icon: activity, label: "Orchestrator"]
ServiceLayer [icon: list, label: "Service Layer (Queue Pool)"]
WorkerManager [icon: settings, label: "Worker Manager"]
WorkerNode [icon: box, label: "Worker Node"]
MonitoringUnit [icon: activity, label: "Monitoring Unit"]

// Edges - Authentication & Setup
User > AccountsPlane: 1) User signup
AccountsPlane > UserDB: 2) Database entries / account check
AccountsPlane > User: 3) Session tokens

// Edges - Function Upload & Compilation Flow
User > FuncGateway: 1) First time func give (new func endpoint, run immediate=True)
FuncGateway > User: port/subdomain for that user
FuncGateway > RawFuncStore: Store raw func data
FuncGateway > CompilationUnit: 2) Send for comp (ref)
CompilationUnit > FuncGateway: (If any compilation error before stage 3, send the error)
CompilationUnit > ObjectDB: Func binary, ref, userID, metadata, trigger info
CompilationUnit > Orchestrator: activateImmediate

// Edges - Invocation Flow
Client > FuncGateway: 1) (user secret, input, funcID) Input based trigger
FuncGateway > Orchestrator: 2') (ref ID, input)
ObjectDB > ServiceLayer: 5) Func metadata using the refID
Orchestrator > ServiceLayer: 6) Binary, metadata, input
ServiceLayer > WorkerManager: 7) Pulls the job according to resources

// Edges - Execution & Monitoring
WorkerManager > WorkerNode: (Func binary, objID, funcID, input)
WorkerNode > MonitoringUnit: (Heartbeat)
MonitoringUnit > WorkerManager: (Updates the cache of worker manager with heartbeats)
WorkerNode > FuncGateway: Output, jobID, funcID
```

---

## 1. Services / API Layer
The API layer acts as the entry point for both developers (managing functions) and end-users (invoking functions). 

**Authentication API**
- `POST /signup` -> `signup(name, email, password, metadata)`
- `POST /login` -> `login(email, password)`
- `POST /refresh` -> `refresh_token(refresh_token)`

**Gateway API**
- `POST /lambda/upload` -> `upload_lambda(source_code, runtime, config, trigger_type, runImmediate)`
- `GET /lambda/list` -> `get_lambda()`
- `POST /lambda/invoke` -> `request_trigger(lambda_id, input, user_secret)`
- `POST /file/upload/chunk` -> `upload_file_chunk(file_id, chunk_index, encrypted_chunk_data)`
- `POST /file/upload/finalize` -> `finalize_file_upload(file_id, total_chunks, metadata)`

## 2. Databases
The system utilizes multiple data stores optimized for different types of data:

1. **Relational Database (User DB):**
   - **Purpose:** Stores relational client information.
   - **Tables/Data:** `client_id`, `email`, `password_hash`, `token_revocation_info`, `owner_id`.
2. **Relational Database (Lambda Metadata DB):**
   - **Purpose:** Manages lambda configurations and state.
   - **Tables/Data:** `owner_id`, `lambda_id`, `runtime`, `config`, `trigger_type`, `wasm_reference`, `execution_id`, `file_id`, `chunk_index`.
3. **Object Storage / Blob Store (Object DB):**
   - **Purpose:** Stores the actual large binaries.
   - **Data:** Compiled `.wasm` binaries, raw zipped source code, and large file chunks.
4. **In-Memory Cache (e.g., Redis) - Optional but implied:**
   - **Purpose:** Orchestrator maps for active function references and Worker Manager heartbeat caching.

## 3. Core Modules

1. **Client Authentication Module:** Handles JWT/Session issuance, validation, and user identity checking.
2. **Gateway Module:** The main traffic router. It handles incoming triggers, validates user secrets, and coordinates file chunking/uploads.
3. **Compiler Module:** An asynchronous worker that pulls raw code from the object store, compiles it to `wasm32-wasi` based on the language, injects ABI wrappers, and stores the resulting binary.
4. **Orchestrator Module:** Manages the lifecycle of a function execution. It holds in-memory maps of active functions and pushes valid execution requests to the Service queue.
5. **Service Module (Queue Layer):** Manages the priority and routing queues. Implements algorithms (like round-robin or resource-based scoring) to route tasks (`enqueue_task`, `select_queue`).
6. **Worker Manager Module:** Tracks consumer nodes. Handles registration (`register_consumer`), health checks (`heartbeat`), and task delivery (`deliver_task_to_consumer`).

## 4. System Design (Data Flow)
**Deployment Flow:**
1. Developer calls `/lambda/upload`.
2. Gateway stores the raw file in the Raw Func Data Store.
3. Gateway triggers the Compiler Module.
4. Compiler generates `.wasm`, stores it in Object DB, and notifies the Orchestrator (`activateImmediate`).

**Execution Flow:**
1. End-user calls `/lambda/invoke` with a secret and payload.
2. Gateway validates the secret and forwards the trigger to the Orchestrator.
3. Orchestrator fetches metadata and enqueues the task in the Service Layer.
4. Worker Manager pulls the task from the queue based on available node resources and dispatches it.
5. The Worker Node executes the `.wasm` binary and returns the output directly to the Gateway (or via the Worker Manager).

## 5. External Components Used
- **WASM Runtimes (e.g., WAMR, Wasmtime):** For securely executing the compiled binaries on edge nodes.
- **WASI SDK / Clang / TinyGo / Rustup:** Toolchains required by the Compiler module to target `wasm32-wasi`.
- **Database Engines:** PostgreSQL/MySQL for relational data; AWS S3/MinIO for Object Storage.
- **Queueing Engine:** RabbitMQ, Kafka, or Redis Streams for the Service Module's `queue pool`.

## 6. Compiler Details
The compiler is a highly specialized intake pipeline designed to standardize arbitrary user code into WebAssembly.

**Architecture:**
- **Intake Layer:** Determines the payload type (File, Zip, Docker). It extracts the source into a temporary workspace (`/tmp/job-id/src/`).
- **Branching Layer:** Identifies the language (C, C++, Rust, Go) and routes to the specific toolchain.
- **Language Compilers:**
  - *C/C++:* Uses `clang` with `--target=wasm32-wasi`.
  - *Rust:* Uses `cargo build --target wasm32-wasi`.
  - *Go:* Uses `tinygo build -target=wasi`.
- **ABI Wrapper Generation:** Crucial step where the system auto-generates a translation layer (`wrapper.go`/`wrapper.c`). This wrapper takes the raw byte input from the WAMR runtime, parses it into the function's expected types (e.g., `int`, `string`), calls the user's function, and formats the output bytes.
- **Storage:** Hashes the final `.wasm` binary (SHA256) and stores it in the Object DB under `wasm/fn_ab12cd/function.wasm` with status `"ready"`.

# Central Server Modules Report

Date: 2026-02-12

## 1) Overview
The Central Server is a Go service that wires together authentication, a function gateway, a compiler pipeline, an orchestrator, a queue manager, and a sink manager. It exposes HTTP APIs for user auth, function lifecycle, execution triggers, sink registration, and task results. Storage is backed by PostgreSQL and in-memory repositories for development or testing.

Key runtime wiring happens in cmd/server/main.go, which builds the database connection, initializes services, and mounts HTTP routes.

## 2) Entry Points and Wiring
- cmd/server/main.go: server bootstrap, environment variables, DB connection, service wiring, HTTP routes, and CORS.
- app/bootstrap.go: placeholder for dependency wiring (currently empty).
- utils/logger.go: simple logging helper with INFO/ERROR helpers.

## 3) Modules

### A) Auth Module (internal/auth)
**Purpose:** User registration, login, and JWT authentication.

**Core logic:**
- AuthService handles registration (bcrypt hashing), login (bcrypt check), and JWT issuance/verification.
- JWT claims include user ID and username; token expires after 24 hours.

**HTTP layer:**
- POST /api/v1/auth/register
- POST /api/v1/auth/login
- AuthMiddleware validates Bearer tokens and stores user info in context.

**Key types:**
- User, Claims, LoginRequest, AuthResponse.
- Context helpers: WithAuthContext, UserIDFromContext, UsernameFromContext.

**Dependencies:**
- interfaces.UserRepository (Postgres or in-memory implementation).

### B) Gateway Module (internal/gateway)
**Purpose:** Function lifecycle management and execution triggers.

**Core logic (LambdaService):**
- StoreandQueue stores a lambda, validates input, and enqueues compilation.
- TriggerLambda creates an execution record and sends a trigger to the orchestrator.
- Activate/Deactivate sends activation requests to orchestrator.
- ListLambdas, GetLambda, DeleteLambda, ListExecutions.

**Domain model:**
- Lambda (function definition), Execution (invocation status).
- CompilationQueue with blocking dequeue for compiler worker.

**HTTP layer:**
- Clean APIs: /api/v1/lambdas, /api/v1/executions, /api/v1/execute/{id}.
- Compat APIs for legacy client: /functions/* and /invocations.

**Dependencies:**
- FuncGatewayDB (function repository)
- CompilerDB (compile status)
- OrchestratorService (activate/trigger/deactivate)
- ExecutionRepository (execution records)

### C) Compiler Module (internal/compiler)
**Purpose:** Build functions into WASM and store artifacts.

**Core logic:**
- Compiler.Compile fetches CompilationRequest, validates, compiles, stores WASM, stores metadata, and optionally triggers run.
- CompilerWorker loops over compilation queue jobs and activates the service through orchestrator after success.

**Current runtime support:**
- C runtime is implemented (clang to WASM/WASI).
- Go/Rust/Cpp stubs return "not implemented" errors.

**Dependencies:**
- ObjectStore (fetch sources and store WASM/metadata)
- RunTrigger (optional immediate execution)
- CompilationQueue
- OrchestratorService

### D) Orchestrator Module (internal/orchestrator)
**Purpose:** Activation of compiled functions and routing triggers into the queue service.

**Core logic:**
- ActivateService checks metadata status and calls gateway activation.
- ReceiveTrigger dispatches a job to QueueService (or enqueues).
- Tracks activated functions and their metadata in memory for fast lookup.

**Dependencies:**
- Database (GetLambdaMetadata)
- FuncGateway (ActivateJob, DeactivateJob)
- QueueService (DispatchOrEnqueue)

### E) Queue Service Module (internal/queueService)
**Purpose:** Queue jobs based on resource requirements and dispatch to sinks.

**Core logic:**
- QueuePool: multiple queues with RAM ranges.
- QueueSelector picks a queue by max RAM.
- Enqueue and ClaimNextJob for sink dispatching.
- DispatchOrEnqueue sends directly to active sink when possible; otherwise queues.

**Dependencies:**
- SinkManagerService (to check active sinks and deliver tasks).

### F) Sink Manager Module (internal/sinkManager)
**Purpose:** Manage sink registration, heartbeats, task delivery, and results.

**Core logic:**
- Register/login sinks (bcrypt + JWT).
- Process heartbeat and mark sink online/offline.
- Deliver tasks to sinks and record task status.
- Process task results and call optional result callback.
- Stale detector marks sinks offline after heartbeat timeout.

**HTTP layer:**
- POST /api/v1/sinks/register
- POST /api/v1/sinks/login
- POST /api/v1/sinks/heartbeat
- POST /api/v1/sinks/result
- GET /api/v1/sinks
- GET /api/v1/sinks/{id}
- DELETE /api/v1/sinks/{id}
- GET /api/v1/tasks/{id}/result

**Dependencies:**
- SinkRepository, TaskRepository, TaskResultRepository
- SinkClient (HTTP client to sinks)
- QueueService (for dispatching from queues)

### G) Storage Module (internal/storage)
**Purpose:** Data persistence and object storage for compilation.

**PostgreSQL repositories:**
- PostgresUserRepo: users table.
- PostgresFunctionRepo: functions table, metadata for orchestrator.
- PostgresExecutionRepo: executions table.
- PostgresObjectStore: fetches compilation request and stores WASM refs.

**In-memory repositories (dev/testing):**
- MemoryLambdaRepo, MemoryExecutionRepo, MemoryUserRepo.
- MemorySinkRepo, MemoryTaskRepo, MemoryTaskResultRepo.
- MemoryObjectStore for compiler tests.

**Notes:**
- Object store currently stores WASM bytes directly in DB (placeholder for external object storage).

### H) Events Module (internal/events)
**Purpose:** Placeholder for eventing (folder is empty except .gitkeep).

### I) Utils Module (utils)
**Purpose:** Shared logging helpers.

### J) Database Schema (schema.sql)
**Purpose:** PostgreSQL schema for users, functions, executions, and logs.

## 4) High-Level Flow (Typical Path)
1. User registers and logs in (Auth module returns JWT).
2. User creates a function (Gateway stores lambda and enqueues compile job).
3. Compiler worker dequeues job, compiles to WASM, stores metadata, and calls Orchestrator to activate.
4. Orchestrator activates the function and dispatches triggers to QueueService.
5. QueueService routes the task to an available sink or queues it.
6. SinkManager delivers task to sink and processes results.
7. Gateway exposes execution status and results through its APIs.

## 5) Presentation Highlights (Suggested Talking Points)
- Modular design with clear domain, handler, and interface layers.
- JWT-based auth and sink authentication with bcrypt.
- Compiler pipeline with queue-based worker model.
- Orchestrator as the control plane for activation and dispatch.
- QueueService supports resource-aware routing (RAM-based queues).
- SinkManager provides heartbeats, task delivery, and result collection.
- PostgreSQL-backed persistence with in-memory fallbacks for testing.

## 6) Gaps / TODOs Visible in Code
- app/bootstrap.go is a placeholder for dependency wiring.
- Compiler only supports C runtime currently.
- Events module is empty (future event bus).
- Object store uses DB for WASM bytes (consider external storage).

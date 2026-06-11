# Consumer-Cloud Communication Protocol

This document specifies how the **Consumer Node** (Mobile/Edge) interacts with the **Central Server**. It details the lifecycle of a node from initial registration to task execution and result reporting.

---

## 1. Node Registration & Authentication

Before a node can accept jobs, it must identify itself and establish a secure session.

### A. Registration Flow
When the app starts or the user clicks "Connect":
1.  **Request:** The node sends a `POST` request to `/api/v1/sinks/register`.
2.  **Payload:**
    ```json
    {
      "email": "node_identity@milf.local",
      "password": "...", 
      "endpoint": "ws-node"
    }
    ```
3.  **Response:** The server returns a unique `sink_id`.
    ```json
    {
      "sink_id": "snk_82934abc..."
    }
    ```
4.  **Persistence:** The `sink_id` is stored in the node's local memory to maintain the session.

### B. Authentication
All subsequent requests (REST and WebSocket) are authenticated using a **Bearer Token** in the `Authorization` header.

---

## 2. Persistent Task Channel (WebSockets)

The node maintains a long-lived WebSocket connection to receive real-time task assignments.

### A. Connection Handshake
*   **Endpoint:** `ws://<server>/api/v1/sinks/ws?sinkId=<sink_id>`
*   **Heartbeat:** The node sends a `type: heartbeat` message every 30 seconds to prevent connection timeout and report local resource availability (RAM/CPU).

### B. Task Assignment (`task_assignment`)
When a job is dispatched, the server pushes an assignment packet:
```json
{
  "type": "task_assignment",
  "payload": {
    "execution_id": "exec_123",
    "lambda_id": "lam_456",
    "wasm_base64": "...", // Optional: Inlined binary for small functions
    "payload": {
       "type": "json",
       "data": { "arg1": 10, "arg2": 20 }
    }
  }
}
```

---

## 3. Resource Acquisition

If the WASM binary is not inlined in the WebSocket message, the node fetches it via a dedicated REST endpoint.

1.  **Endpoint:** `GET /api/v1/lambdas/{lambda_id}/wasm`
2.  **Strategy:** The node checks its local `AOT Cache` first. If missing, it downloads the binary, validates the hash, and passes the bytes to the Native Bridge.

---

## 4. Result Reporting & File Uploads

After execution, the node must return the output to the cloud.

### A. Immediate Result
Small outputs (strings/JSON/ints) are sent directly through the WebSocket:
```json
{
  "type": "task_result",
  "payload": {
    "execution_id": "exec_123",
    "success": true,
    "output": { "result": 30 }
  }
}
```

### B. Large File Uploads (e.g., Image to PDF)
If the WASM function generates a file:
1.  The Native Bridge saves the file to a secure local directory.
2.  The Bridge returns a `FILE:<filename>` reference to Flutter.
3.  **Upload:** Flutter reads the file and performs a `Multipart POST` to `/api/v1/files`.
4.  **Reference:** The resulting `file_id` is sent as the final execution output via the WebSocket.

---

## 5. Telemetry & Observability

The node streams real-time status back to the server for the developer dashboard.

1.  **Heartbeats:** Includes `ram_available_mb` and `storage_available_mb`.
2.  **Live Logs:** (Proposed) If requested by the server, the node streams `stdout` and `stderr` from the WASM runtime via the WebSocket `logs` channel.
3.  **Metrics:** (Proposed) Detailed execution metrics (CPU cycles, peak RSS) are sent upon task completion for billing and performance analysis.

---

**Summary:** This protocol ensures the Consumer Node remains "passive" until a task is pushed, minimizing battery consumption while allowing for near-instant execution of distributed compute jobs.

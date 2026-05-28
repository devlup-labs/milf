# Central Server APIs & Compute Infrastructure (India Deployment)

This document expands on the non-database components of the Central Server: the API Gateway, the Orchestrator, and the Compiler nodes. It details their working principles and provides an infrastructure cost analysis in **INR (₹)** assuming the deployment is hosted in India (e.g., AWS `ap-south-1` Mumbai or local providers like DigitalOcean BLR / E2E Networks).

---

## 1. Core Compute Modules & APIs

### A. API Gateway Layer (Nginx / Traefik / AWS ALB)
The Gateway is the front door. It handles SSL termination, rate-limiting, and routes traffic to the correct internal microservices.

**Key API Routes:**
*   `POST /api/v1/auth/*` -> Routes to the Authentication Service.
*   `POST /api/v1/functions/upload` -> Proxies raw code to the Compiler intake.
*   `POST /api/v1/invoke/{id}` -> Fast-path API. Must have extremely low latency. Validates the `user_secret` and immediately passes the payload to the Orchestrator via gRPC or Redis queue.

### B. The Orchestrator Module
The Orchestrator acts as the central brain managing executions.
*   **Trigger Management:** Receives the trigger from the Gateway.
*   **State Management:** Checks an in-memory Redis map to ensure the function is marked as `ready` and not currently `compiling`.
*   **Queueing:** Pushes the `{ lambda_id, payload, execution_id }` into the Service Layer (RabbitMQ or Redis Stream).

### C. Worker Manager (Internal Control Plane)
The Worker Manager handles the fleet of Consumer Nodes. It exposes an internal API (not accessible to the public web) for node coordination.

**Key Internal APIs:**
*   `POST /internal/worker/register` -> `register_consumer(metadata)`: A new consumer node comes online and requests authorization to join the execution pool.
*   `POST /internal/worker/heartbeat` -> `heartbeat(consumer_id, status)`: A high-frequency (UDP/fast TCP) endpoint where consumers ping their health, available RAM, and CPU usage. Used to build the routing table.
*   `POST /internal/worker/result` -> `receive_execution_result(task_id, result)`: A consumer node pushes the final payload/output of a compiled function back to the central plane to be stored or forwarded to the waiting client.
*   **Task Dispatch:** Instead of consumers polling for work, the Worker Manager actively pushes tasks (`deliver_task_to_consumer`) via established long-lived connections (gRPC streams or WebSockets) based on the heartbeat data.

### D. Compiler Nodes (The Heavy Lifters)
*   **Role:** While the Orchestrator handles millions of tiny API requests, the Compiler Nodes do the heavy CPU-bound work of transforming C++/Go/Rust code into WebAssembly.
*   **Architecture Requirement:** These nodes must be compute-optimized (High CPU). They do not need much memory or network speed, but they need raw processing power to keep compilation times under 3 seconds.

---

## 2. Infrastructure Cost Analysis (₹ INR) for 10,000 Users

Assuming the system is deployed in **India (Mumbai/BLR)** and handles **10,000 developers** (approx. 15 million API requests and 100,000 compilations per month).

*Note: Database costs (~₹8,000) are excluded here as they were covered in the Database Strategy document.*

| Infrastructure Component | Purpose / Sizing | Monthly Cost Estimate (INR) |
| :--- | :--- | :--- |
| **Load Balancer** | Distributes traffic across Gateway nodes (e.g., DigitalOcean LB or AWS ALB). | **₹1,000 - ₹2,000** |
| **API Gateway & Orchestrator Nodes** | 2x Standard Instances (2 vCPU, 4GB RAM). High concurrency, low CPU. Handles all web traffic and Redis queue pushing. | 2 x ₹2,000 = **₹4,000** |
| **Compiler Nodes (Build Servers)** | 2x Compute-Optimized Instances (4 vCPU, 8GB RAM). Scales up dynamically when a batch of users deploy new code. | 2 x ₹4,500 = **₹9,000** |
| **Worker Manager Node** | 1x Standard Instance (2 vCPU, 2GB RAM). Tracks consumer heartbeats. | **₹1,500** |
| **Network Egress (API Traffic)** | ~500 GB of pure API JSON payload traffic out to the internet (₹9/GB on AWS, much cheaper on DO/Linode). | **₹500 - ₹4,500** |
| **Total Server Infrastructure** | For a highly available, robust control plane handling 10k users. | **₹16,000 - ₹21,000 / month** |

### Strategic Tips for Indian Deployment:
1. **Avoid AWS Bandwidth Costs:** AWS (`ap-south-1`) charges roughly ₹9 per GB for data transfer out to the internet. For heavy API/File platforms, this becomes the largest bill. Consider hosting the compute layer on **DigitalOcean (Bangalore)**, **Linode/Akamai**, or Indian cloud providers like **E2E Networks**, which offer massive free bandwidth pools (e.g., 2TB - 4TB free per server).
2. **Auto-Scaling the Compiler:** The Compiler Nodes cost ₹9,000/mo if left running 24/7. However, developers mostly deploy code during the day. By using Kubernetes (EKS/DOKS) or AWS Auto Scaling Groups, you can scale the Compiler nodes to `0` at night and save 40% on compute costs.
3. **Consumer Node Costs:** Note that the above costs do *not* include the actual Consumer Edge Nodes executing the WASM code, as those are often decentralized, run on user devices, or deployed on cheap spot instances.

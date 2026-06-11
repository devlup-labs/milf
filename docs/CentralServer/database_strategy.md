# Database Strategy & Cost Analysis

To support a highly distributed FaaS platform like MILF, the database architecture must be segmented. You cannot store large compiled `.wasm` binaries in the same database you use to rapidly query user account status.

Here is the recommended database strategy, data models, and a cost analysis tailored for an active scale of **10,000 users**.

---

## 1. Required Database Types & Favorable Platforms

### A. Primary Relational Database (SQL)
*   **Purpose:** ACID-compliant storage for Users, Authentication, Lambda Metadata, and Execution Logs.
*   **Recommended Database:** **PostgreSQL**. (Highly robust, supports `JSONB` for flexible lambda configurations).
*   **Favorable Platform:** 
    *   *Neon.tech or Supabase:* Best for modern serverless architectures. They offer auto-scaling compute and scale-to-zero capabilities.
    *   *AWS RDS (Aurora Serverless v2):* Best for enterprise-grade high availability, though slightly more expensive.

### B. Object / Blob Storage
*   **Purpose:** Storing large immutable files. (Raw source code uploads, compiled `.wasm` binaries, and large payload chunks).
*   **Recommended Protocol:** S3-Compatible API.
*   **Favorable Platform:** 
    *   *Cloudflare R2:* **Highly Recommended.** They charge $0 for egress (bandwidth) fees. Since Consumer nodes will constantly be downloading `.wasm` binaries, zero egress fees will save you massive amounts of money compared to AWS.
    *   *AWS S3:* Industry standard, but bandwidth egress to your consumer edge nodes will become expensive at scale.

### C. In-Memory Cache / Message Broker
*   **Purpose:** Managing the active execution queues, tracking Worker Node heartbeats, and Orchestrator mapping.
*   **Recommended Database:** **Redis**.
*   **Favorable Platform:**
    *   *Upstash:* Serverless Redis with per-request pricing.
    *   *AWS ElastiCache:* If deploying entirely within the AWS ecosystem.

---

## 2. Core Data Models (PostgreSQL)

### `Users` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `email` | VARCHAR | Unique, Indexed |
| `password_hash`| VARCHAR | Security |
| `tier` | VARCHAR | e.g., 'free', 'pro' (Billing limits) |

### `Lambdas` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `owner_id` | UUID | Foreign Key -> Users |
| `name` | VARCHAR | e.g., 'image-resizer' |
| `runtime` | VARCHAR | e.g., 'c++', 'go', 'rust' |
| `wasm_ref` | VARCHAR | Pointer to the object store URI |
| `memory_limit`| INT | Memory allocated for Consumer node |
| `status` | VARCHAR | 'pending_compile', 'ready', 'error' |

### `Executions` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `lambda_id` | UUID | Foreign Key -> Lambdas |
| `status` | VARCHAR | 'queued', 'running', 'success', 'failed' |
| `duration_ms` | INT | Used for billing |
| `output_ref` | VARCHAR | Pointer to S3 if output payload is large |

### `Consumers` (Worker Nodes) Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key (consumer_id) |
| `owner_id` | UUID | Optional: If workers are privately hosted by users |
| `ip_address`| VARCHAR | Network location of the node |
| `max_memory`| INT | Total RAM capacity of the node |
| `status` | VARCHAR | 'active', 'offline', 'drained' |
| `last_heartbeat`| TIMESTAMP | Used to evict dead nodes |

---

## 3. Cost Analysis (Estimation for 10,000 Users)

**Assumptions for 10,000 Users:**
*   **Active Users:** ~2,000 daily active developers.
*   **Lambdas Created:** ~50,000 functions total (5 per user).
*   **Executions:** ~500,000 invocations per day (15 million/month).
*   **Storage Profiles:**
    *   Metadata (PostgreSQL): ~50 GB.
    *   WASM Binaries (Object Storage): 50,000 * 2MB = 100 GB.
    *   Execution Payload Inputs/Outputs: ~500 GB rolling storage.
    *   Egress Bandwidth (Downloads by Consumers): ~2 TB / month.

### Monthly Cost Breakdown (Cloud Provider Estimates)

| Service | Platform Choice | Sizing / Usage | Estimated Monthly Cost |
| :--- | :--- | :--- | :--- |
| **Relational DB** | Neon.tech / Supabase | 50 GB Storage + Moderate Compute (2-4 vCPU) | **$30 - $60** |
| **Object Store** | Cloudflare R2 | 600 GB Storage + 2TB Egress | **$9.00** *(Storage)* + **$0** *(Egress)* = **$9** |
| **Object Store** *(Alt)*| AWS S3 | 600 GB Storage + 2TB Egress | $13 *(Storage)* + $180 *(Egress)* = **$193** |
| **Cache / Queue** | Upstash (Redis) | 2 GB Memory, High throughput | **$20 - $40** |
| **Total DB Ops** | **Optimized Route (R2)** | Supporting 15M Executions / Month | **~$60 - $110 / month** |

### Cost Analysis Summary:
At 10,000 users, the database infrastructure is surprisingly affordable (**under $150/month**) *if* you avoid cloud-provider traps like AWS Egress fees. 

Because your Consumer Nodes are pulling `.wasm` files constantly, bandwidth out of your Object Store will be your largest hidden cost. Using a zero-egress provider like **Cloudflare R2** for your Object Storage is the single most important financial decision you can make for this architecture. The relational queries (Postgres) and queueing (Redis) scale very predictably and cheaply.

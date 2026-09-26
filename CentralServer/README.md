# 🖥️ CentralServer (MILF Backend Orchestrator)

The central orchestration backend for **MILF** (Mobile Infra for Lambdas and Files). Written in Go, it manages authentication, C-to-WASM compilation, task queueing, database persistence (PostgreSQL), and real-time WebSocket dispatch to mobile worker nodes.

---

## 🚀 Running Locally

You can run CentralServer locally either using **Docker** (recommended for Windows & macOS) or **Native Go**.

### Option A: Local with Docker (Recommended)

Docker provides an isolated Linux container with the `wasi-sdk` clang compiler already bundled and configured.

#### 1. Build the Docker Image
```bash
# From CentralServer directory:
docker build -t milf-server .
```

#### 2. Run Container
* **If connecting to PostgreSQL on Windows/macOS host machine**:
  ```bash
  docker run -d --name milf-central-server \
    -p 8080:8080 \
    -e DB_HOST=host.docker.internal \
    -e DB_PORT=5432 \
    -e DB_NAME=central_server_db \
    -e DB_USER=postgres \
    -e DB_PASSWORD=your_password \
    -e JWT_SECRET=your_jwt_secret_here \
    milf-server
  ```
  *(On Windows PowerShell, replace `\` with backticks `` ` ``)*.

* **If connecting to Supabase / Cloud Postgres**:
  ```bash
  docker run -d --name milf-central-server \
    -p 8080:8080 \
    -e DATABASE_URL="postgresql://postgres:password@your-db-host:5432/postgres?sslmode=require" \
    -e JWT_SECRET=your_jwt_secret_here \
    milf-server
  ```

---

### Option B: Native Go Setup

#### 1. Prerequisites
* **Go 1.22+**: `go version`
* **PostgreSQL**: Running locally on port `5432`
* **WASI SDK 24.0**: Required for C to WASM compilation. Download from [WebAssembly/wasi-sdk](https://github.com/WebAssembly/wasi-sdk/releases/tag/wasi-sdk-24).

#### 2. Initialize Database & Environment
1. Run schema migrations:
   ```bash
   psql -U postgres -d central_server_db -f schema.sql
   ```
2. Create `.env`:
   ```bash
   cp .env.example .env
   ```
   Set `DATABASE_URL`, `JWT_SECRET`, and `CLANG_PATH` pointing to your local `clang` binary.

#### 3. Run Server
```bash
go mod download
go run cmd/server/main.go
```

---

## 🌐 Deploying to Render (Render.com)

Render allows you to host the server using Docker in a few clicks:

### 1. Create a New Web Service
1. Go to your [Render Dashboard](https://dashboard.render.com).
2. Click **New +** -> **Web Service**.
3. Connect your GitHub repository containing the MILF project.

### 2. Configure Service Settings
| Field | Value |
| :--- | :--- |
| **Name** | `milf-central-server` |
| **Region** | Choose nearest region (e.g., Oregon, Frankfurt, Singapore) |
| **Root Directory** | `CentralServer` |
| **Environment / Runtime** | **Docker** |
| **Dockerfile Path** | `./Dockerfile` |

### 3. Add Environment Variables on Render
Under **Environment Variables** in Render, add:

| Key | Example Value | Description |
| :--- | :--- | :--- |
| `DATABASE_URL` | `postgresql://postgres:pass@ep-xyz.supabase.co:5432/postgres?sslmode=require` | Connection string to Supabase or Render PostgreSQL |
| `PORT` | `8080` | Server listening port |
| `JWT_SECRET` | `your-secure-random-secret-key` | Secret key for signing auth tokens |
| `GOOGLE_CLIENT_ID` | `xyz.apps.googleusercontent.com` | Google OAuth Client ID |

*(Note: `CLANG_PATH` is already pre-configured to `/opt/wasi-sdk/bin/clang` inside the `Dockerfile`, so no manual path is needed on Render!)*

### 4. Deploy & Verify
Click **Create Web Service**. Render will:
1. Build the Docker image (downloading Go dependencies & WASI-SDK).
2. Start the server on port `8080`.
3. Provide an HTTPS endpoint (e.g., `https://milf-central-server.onrender.com`).

Verify deployment with:
```bash
curl https://milf-central-server.onrender.com/health
```

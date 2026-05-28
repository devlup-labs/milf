# Client Architecture & Design

This document outlines the architectural structure of the Client application in a clean, standardized format, mirroring the Central Server architecture breakdown.

## Eraser.io Diagram Code

You can copy and paste the following code into [Eraser.io](https://www.eraser.io/) to generate a high-level view of the Client-Side data flow.

```eraser
// Nodes
User [icon: user, label: "Developer/User"]
UI [icon: monitor, label: "React View Layer (Pages)"]
Monaco [icon: code, label: "Monaco Editor (Testing Module)"]
StateCache [(icon: database, label: "React Query / IndexedDB")]
APIClient [icon: send, label: "API Client (fetch)"]
CentralServer [icon: server, label: "Central Server (Gateway)"]

// Edges - Code Editing Flow
User > UI: Interacts with Dashboard
UI > Monaco: Writes Function Code
Monaco > UI: Syntax Highlighting / Basic Linting (Client-side)

// Edges - Data Flow
User > UI: Triggers Action (e.g., Invoke)
UI > StateCache: Checks for cached data
StateCache > APIClient: Cache miss / Mutation request
APIClient > CentralServer: HTTP REST / WebSocket connection
CentralServer > APIClient: Returns JSON / Stream
APIClient > StateCache: Updates Cache
StateCache > UI: Triggers Re-render
```

---

## 1. Services / API Layer
The Client interacts with the Central Server through a centralized API service (`src/lib/api.ts`). This layer acts as the unified bridge for all network requests.

**Authentication APIs**
- `login(username, password)` -> Trades credentials for HttpOnly Session Cookies.
- `googleLogin(idToken)` -> OAuth2 integration.

**Function Management APIs**
- `createFunction(data)` -> Submits runtime, memory, and source code payload.
- `listFunctions(search)` -> Fetches available lambda metadata.
- `invokeFunction(id, input)` -> Submits an execution payload to the Orchestrator.

**Observability APIs**
- `listInvocations(query)` -> Fetches execution history.
- `listLogs(query)` -> Polls (or streams via WebSockets) execution logs.

## 2. Databases (Client-Side Storage)
The client relies on local storage mechanisms to achieve a "Local-First" feel and manage session states securely.

1. **In-Memory Cache (React Query):**
   - **Purpose:** Primary state management. Stores API responses temporarily to prevent redundant network calls.
2. **IndexedDB:**
   - **Purpose:** Persists the React Query cache across page reloads. Allows the dashboard to render instantly while background fetching (stale-while-revalidate).
3. **Browser Cookies (HttpOnly):**
   - **Purpose:** Securely stores authentication tokens (Access/Refresh). Prevents XSS vulnerabilities compared to `localStorage`.

## 3. Core Modules
1. **View / Routing Module (`src/pages`):** Defines the UI screens (Dashboard, Functions, Invocations) and orchestrates URL-based navigation using `react-router-dom`.
2. **State & Query Module (`src/hooks`):** Wraps raw API calls with `@tanstack/react-query` to handle loading states, errors, caching, and background refetching.
3. **Authentication Module (`AuthContext.tsx`):** A global React Context that monitors cookie validity, manages the active user session, and guards protected routes.
4. **Component Library (`src/components`):** Reusable UI blocks separated into Layouts (Navbar/Sidebar), Shared Domain widgets (DataTables, FileExplorer), and Base UI primitives (Buttons, Dialogs).
5. **File Handling Module:** Responsible for reading local files, preparing multipart chunking for large assets, and managing client-side encryption (if required).

## 4. System Design (Data Flow)
**Data Fetching Flow (e.g., Loading Dashboard):**
1. User navigates to `/dashboard`.
2. The View component calls `useFunctions()`.
3. React Query checks IndexedDB/Memory for existing function data.
4. If cached, UI renders instantly. React Query fires a background `fetch` to `api.ts`.
5. `api.ts` requests `/lambda/list` from the Central Server.
6. Central Server responds; React Query updates the cache and re-renders the UI with fresh data.

**Code Deployment Flow:**
1. User writes code in the **Monaco Editor** component.
2. User clicks "Deploy".
3. The View collects the source code string and metadata, calling the `createFunction` mutation.
4. React Query sends a POST request via `api.ts` to the Gateway.
5. Upon success, React Query invalidates the function list cache, triggering an automatic refresh of the UI.

## 5. External Components Used
- **React 18 & Vite:** Core rendering engine and rapid build tool.
- **Tailwind CSS & Shadcn UI:** For styling and accessible, headless UI components.
- **@tanstack/react-query:** For asynchronous state management and caching.
- **React Router DOM:** For client-side navigation.
- **@monaco-editor/react:** For embedding the VS Code editor experience directly in the browser.

## 6. Hybrid Testing Module (Client-Side)
The client does **not** compile or execute code, but it provides a rich IDE experience to catch errors early.

**Architecture:**
- **Editor Engine:** Monaco Editor.
- **Capabilities:**
  - *Syntax Highlighting:* Maps language semantics (C, Go, Rust) to colors.
  - *Bracket Matching & Formatting:* Auto-closes parentheses and formats code blocks.
  - *Basic Linting:* Catches syntax errors (e.g., missing semicolons in JS/C) before sending the payload over the network.
- **Boundary:** It intentionally stops at semantic analysis. It does not attempt to resolve module dependencies or validate against execution constraints—that is offloaded to the Central Server's Compiler Module.

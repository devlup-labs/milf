# Future Implementation Plan (Client Side)

This document details the roadmap and implementation steps required to transition the Client application to the production-ready architecture designed for the Central Server integration.

## Phase 1: Security & Authentication Overhaul

Currently, authentication tokens are stored in `localStorage`, which presents a security risk (XSS).

**Implementation Steps:**
1. **HttpOnly Cookies:** Update the `AuthContext.tsx` and `api.ts` modules to no longer expect a JWT token in the response payload to be stored manually. The Central Server will set an `HttpOnly, Secure, SameSite=Strict` cookie containing the access and refresh tokens.
2. **Credentialed Fetch:** Update all `fetch` calls in `src/lib/api.ts` to include `credentials: "include"`. This ensures the browser automatically sends the HttpOnly cookies with every request.
3. **CSRF Protection:** The backend should issue a CSRF token (e.g., via a non-HttpOnly cookie or a meta tag). The client must intercept mutating requests (POST, PUT, DELETE) and append this token to an `X-CSRF-Token` header.
4. **Auth Keys UI:** Add a new tab in the `Settings.tsx` page to allow developers to generate, view, and revoke API Auth Keys used for programmatic access to their lambdas.

## Phase 2: File Upload Chunking Implementation

To reliably handle large files (e.g., heavy WASM binaries or Docker images) over unreliable mobile/client networks, we will implement the multipart chunking logic outlined in the architecture design.

**Implementation Steps:**
1. **Chunking Utility (`src/lib/chunking.ts`):** 
   - Create a utility that takes a JavaScript `File` object and slices it into chunks (e.g., 5MB chunks) using `File.prototype.slice()`.
2. **Encryption:**
   - Integrate Web Crypto API to encrypt each chunk client-side before transmission if end-to-end security is required.
3. **Upload Coordinator (`useMultipartUpload.ts`):**
   - Build a custom hook that orchestrates the upload:
     - Initiates the multipart upload on the backend (gets an Upload ID).
     - Uploads each chunk concurrently (using `Promise.all` with a concurrency limit).
     - Finalizes the upload by sending the backend the list of successful chunk IDs/ETags.

## Phase 3: Real-Time Observability (WebSockets/SSE)

Polling for logs and execution statuses is inefficient. We will migrate to real-time event streaming.

**Implementation Steps:**
1. **WebSocket Client (`src/lib/ws.ts`):**
   - Implement a robust WebSocket client wrapper that handles auto-reconnection, exponential backoff, and heartbeat pings.
2. **Live Logs Component:**
   - Update `Logs.tsx` to subscribe to a specific function's log stream.
   - Use a sliding window array in React state to render the incoming stream without overwhelming the DOM.
3. **Execution Status:**
   - On the `FunctionDetail.tsx` page, when a function is invoked, immediately open a channel to stream the execution status (`queued` -> `running` -> `completed`) rather than polling the `/api/v1/invocations` endpoint.

## Phase 4: Offline Support & Advanced Caching

To achieve the "Local-First" architecture goal, the UI should feel instantaneous and work across network drops.

**Implementation Steps:**
1. **Persist React Query:**
   - Install `@tanstack/react-query-persist-client` and `idb-keyval`.
   - Configure the query client to persist its cache to IndexedDB. This allows the dashboard to render immediately from cache on a fresh load while refetching in the background (stale-while-revalidate).
2. **Optimistic Updates:**
   - Update mutations in `src/hooks/useQueries.ts` to implement optimistic updates. For example, if a user clicks "Pause Schedule", immediately update the local React Query cache so the UI toggles instantly, and rollback if the server request fails.

## Conclusion

By executing these four phases, the Client side of the application will graduate from a development-focused SPA to a secure, resilient, and highly performant dashboard capable of integrating smoothly with the distributed Central Server architecture.

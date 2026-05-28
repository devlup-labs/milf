// Real API service connecting to Go backend
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

export interface ApiResponse<T> {
  ok: boolean;
  data?: T;
  error?: string;
}

/* Auth */
export async function login(username: string, password: string): Promise<{ token: string, username?: string }> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) throw new Error("Login failed");
  return res.json();
}

export async function googleLogin(idToken: string): Promise<{ token: string, username?: string }> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/google`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id_token: idToken }),
  });
  if (!res.ok) throw new Error("Google login failed");
  return res.json();
}

export async function register(username: string, password: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) throw new Error("Registration failed");
}

/* Functions (CREATE, GET, DELETE) */
export async function createFunction(data: any, token: string): Promise<any> {
  // Runtime values now match backend directly
  const mapRuntime = (runtime: string) => {
    return runtime;
  };

  const res = await fetch(`${API_BASE_URL}/api/v1/functions/create`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
    },
    body: JSON.stringify({
      name: data.name,
      runtime: mapRuntime(data.runtime),
      memory: data.memory,
      sourceCode: data.source.type === "inline" ? data.source.code : "// uploaded",
      run_type: data.runType || "on_command",
      cron_expression: data.cronExpression || "",
      metadata: {},
    }),
  });
  if (!res.ok) throw new Error("Create function failed");
  return res.json();
}

// Transform backend Lambda to frontend FunctionEntity
function transformLambda(fn: any) {
  return {
    id: fn.id,
    name: fn.name,
    runtime: fn.runtime,
    status: "active" as const,
    memory: fn.memory_mb || 128,
    timeout: 30,
    tags: [],
    envVars: [],
    createdAt: fn.created_at,
    updatedAt: fn.updated_at,
    lastRunAt: undefined,
    lastRunStatus: undefined,
    avgDurationMs: undefined,
    invocations24h: 0,
    errors24h: 0,
    runType: fn.run_type || "on_command",
    cronExpression: fn.cron_expression || "",
    source: { type: "inline" as const, code: fn.source_code ? atob(fn.source_code) : "" },
  };
}

export async function listFunctions(token: string, search?: string): Promise<any[]> {
  const url = new URL(`${API_BASE_URL}/api/v1/lambdas`);
  if (search) url.searchParams.set("q", search);

  const res = await fetch(url, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Fetch functions failed");
  const body = await res.json();
  const functions = Array.isArray(body) ? body : body.functions || [];
  return functions.map(transformLambda);
}

export async function getFunction(id: string, token: string): Promise<any> {
  const res = await fetch(`${API_BASE_URL}/api/v1/lambdas/${id}`, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Function not found");
  const lambda = await res.json();
  return transformLambda(lambda);
}

export async function deleteFunction(id: string, token: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/lambdas/${id}`, {
    method: "DELETE",
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Delete failed");
}

/* Invoke */
export async function invokeFunction(id: string, input: any, token: string): Promise<any> {
  // input is already a TaskEnvelope JSON string like: '{"type":"json","data":{"a":10}}'  
  // Parse it so we can send it as a proper JSON object, not a double-encoded string
  let inputObj: unknown;
  if (typeof input === "string") {
    try {
      inputObj = JSON.parse(input);
    } catch {
      inputObj = { type: "json", data: input };
    }
  } else {
    inputObj = input;
  }

  const res = await fetch(`${API_BASE_URL}/api/v1/functions/invoke`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
    },
    body: JSON.stringify({ id, input: inputObj }),
  });
  if (!res.ok) throw new Error(await res.text() || "Invoke failed");
  return res.json();
}

/* Executions/Invocations */
export async function listInvocations(token: string, query?: { q?: string; status?: string }): Promise<any[]> {
  const url = new URL(`${API_BASE_URL}/api/v1/invocations`);
  if (query?.q) url.searchParams.set("q", query.q);
  if (query?.status) url.searchParams.set("status", query.status);

  const res = await fetch(url, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) return [];
  const body = await res.json();
  const invocations = Array.isArray(body) ? body : body.invocations || [];

  // Transform execution response to invocation format
  return invocations.map((exec: any) => ({
    id: exec.id,
    requestId: exec.id,
    functionId: exec.functionId || exec.lambda_id,
    functionName: exec.functionName || exec.functionId || exec.lambda_id || "Unknown",
    status: exec.status === "completed" ? "success" : exec.status === "failed" ? "error" : exec.status === "running" ? "running" : "pending",
    durationMs: 0, // Backend doesn't track this yet
    memoryUsedMb: 0, // Backend doesn't track this yet
    timestamp: exec.startedAt || new Date().toISOString(),
    output: exec.output,
    error: exec.error,
  }));
}

export async function getExecution(id: string, token: string): Promise<any> {
  const res = await fetch(`${API_BASE_URL}/api/v1/executions/${id}`, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Execution not found");
  const data = await res.json();
  // Normalise to camelCase fields the frontend uses 
  return {
    id: data.id,
    lambdaId: data.lambda_id,
    status: data.status,      // "pending" | "running" | "completed" | "failed"
    output: data.output,
    error: data.error,
    startedAt: data.started_at,
    finishedAt: data.finished_at,
  };
}

/* Logs (Phase 4: Real Observability) */
export async function listLogs(token: string, query?: { q?: string; level?: string }): Promise<any[]> {
  const url = new URL(`${API_BASE_URL}/api/v1/logs`);
  if (query?.level && query.level !== "all") url.searchParams.set("level", query.level);

  const res = await fetch(url, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) return [];
  const body = await res.json();
  const logs = Array.isArray(body) ? body : [];

  // Transform to frontend LogEntity format & apply client-side search
  return logs
    .map((log: any) => ({
      id: log.id,
      requestId: log.request_id || log.id,
      timestamp: log.timestamp,
      functionName: log.function_name || "Unknown",
      level: log.level || "info",
      message: log.message || "",
      details: log.details || undefined,
    }))
    .filter((log: any) => {
      if (!query?.q) return true;
      const q = query.q.toLowerCase();
      return (
        log.message.toLowerCase().includes(q) ||
        log.functionName.toLowerCase().includes(q) ||
        log.requestId.toLowerCase().includes(q)
      );
    });
}

/* Scheduler */
export async function pauseSchedule(id: string, token: string): Promise<any> {
  const res = await fetch(`${API_BASE_URL}/api/v1/lambdas/${id}/schedule/pause`, {
    method: "POST",
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to pause schedule");
  return res.json();
}

export async function resumeSchedule(id: string, token: string): Promise<any> {
  const res = await fetch(`${API_BASE_URL}/api/v1/lambdas/${id}/schedule/resume`, {
    method: "POST",
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to resume schedule");
  return res.json();
}

export async function getScheduleStatus(id: string, token: string): Promise<{ id: string; paused: boolean }> {
  const res = await fetch(`${API_BASE_URL}/api/v1/lambdas/${id}/schedule/status`, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to get schedule status");
  return res.json();
}

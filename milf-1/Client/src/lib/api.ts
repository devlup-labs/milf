// Real API service connecting to Go backend
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

export interface ApiResponse<T> {
  ok: boolean;
  data?: T;
  error?: string;
}

/* Auth */
export async function login(username: string, password: string): Promise<{token: string}> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) throw new Error("Login failed");
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
  // Map frontend runtime to backend runtime
  const mapRuntime = (runtime: string) => {
    if (runtime.startsWith('go')) return 'go';
    if (runtime.startsWith('node')) return 'javascript';
    if (runtime.startsWith('python')) return 'python';
    if (runtime.startsWith('java')) return 'java';
    if (runtime.startsWith('dotnet')) return 'javascript';
    return 'go';
  };

  const res = await fetch(`${API_BASE_URL}/functions/create`, {
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
  const res = await fetch(`${API_BASE_URL}/functions/invoke`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
    },
    body: JSON.stringify({ 
      id, 
      input: typeof input === "string" ? JSON.parse(input) : input 
    }),
  });
  if (!res.ok) throw new Error("Invoke failed");
  return res.json();
}

/* Executions/Invocations */
export async function listInvocations(token: string, query?: {q?: string; status?: string}): Promise<any[]> {
  const url = new URL(`${API_BASE_URL}/invocations`);
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
    status: exec.status === "completed" || exec.status === "running" ? "success" : "error",
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
  return res.json();
}

/* Logs */
export async function listLogs(token: string, query?: {q?: string; level?: string}): Promise<any[]> {
  // Use invocations as logs proxy until dedicated logs endpoint
  const invocations = await listInvocations(token, { q: query?.q });
  const logs = invocations.map((inv: any) => {
    const isError = inv.status === "error";
    return {
      id: inv.id,
      requestId: inv.requestId,
      timestamp: inv.timestamp,
      functionName: inv.functionName,
      level: isError ? "error" : "info",
      message: isError ? "Invocation failed" : "Invocation completed",
      details: inv.error ? String(inv.error) : inv.output ? JSON.stringify(inv.output, null, 2) : undefined,
    };
  });
  if (query?.level && query.level !== "all") {
    return logs.filter((log: any) => log.level === query.level);
  }
  return logs;
}

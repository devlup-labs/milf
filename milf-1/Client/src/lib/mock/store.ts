import type { FileEntity, FunctionEntity, InvocationEntity, LogEntity, Session, UserEntity } from "./types";
import { nowIso, randomId } from "./utils";

interface Db {
  functions: FunctionEntity[];
  files: FileEntity[];
  invocations: InvocationEntity[];
  logs: LogEntity[];
  session?: Session;
  project: {
    name: string;
    id: string;
    settings: {
      autoDeploy: boolean;
      logRetention30d: boolean;
    };
  };
  users: UserEntity[];
}

const STORAGE_KEY = "sl_demo_db_v1";

function seed(): Db {
  const t = nowIso();

  const fn1: FunctionEntity = {
    id: "fn-1",
    name: "process-payment",
    runtime: "go1.21",
    status: "active",
    memory: 256,
    timeout: 30,
    tags: ["payments", "production"],
    envVars: [
      { key: "STRIPE_API_KEY", value: "sk_test_***" },
      { key: "WEBHOOK_URL", value: "https://api.example.com/webhooks" },
    ],
    createdAt: t,
    updatedAt: t,
    lastRunAt: t,
    lastRunStatus: "success",
    avgDurationMs: 124,
    invocations24h: 1432,
    errors24h: 2,
    source: {
      type: "inline",
      code: `package main\n\nimport (\n  \"context\"\n  \"encoding/json\"\n)\n\nfunc Handler(ctx context.Context, event json.RawMessage) (interface{}, error) {\n  return map[string]string{\"message\": \"Hello, World!\"}, nil\n}`,
    },
  };

  const fn2: FunctionEntity = {
    id: "fn-2",
    name: "user-auth",
    runtime: "go1.21",
    status: "active",
    memory: 128,
    timeout: 15,
    tags: ["auth"],
    envVars: [],
    createdAt: t,
    updatedAt: t,
    lastRunAt: t,
    lastRunStatus: "success",
    avgDurationMs: 89,
    invocations24h: 892,
    errors24h: 0,
    source: {
      type: "inline",
      code: `package main\n\nimport (\n  \"context\"\n  \"encoding/json\"\n)\n\ntype Login struct { Email string \`json:\"email\"\` }\n\nfunc Handler(ctx context.Context, event json.RawMessage) (interface{}, error) {\n  return map[string]any{\"ok\": true}, nil\n}`,
    },
  };

  const invocations: InvocationEntity[] = [
    {
      id: "inv-001",
      requestId: "req-abc123",
      functionId: "fn-1",
      functionName: "process-payment",
      status: "success",
      durationMs: 124,
      memoryUsedMb: 45,
      timestamp: t,
    },
    {
      id: "inv-002",
      requestId: "req-def456",
      functionId: "fn-1",
      functionName: "process-payment",
      status: "error",
      durationMs: 30012,
      memoryUsedMb: 128,
      timestamp: t,
    },
  ];

  const logs: LogEntity[] = [
    {
      id: randomId("log"),
      requestId: "req-abc123",
      timestamp: t,
      functionName: "process-payment",
      level: "info",
      message: "Function invoked",
      details: "Incoming request from 192.168.1.100",
    },
    {
      id: randomId("log"),
      requestId: "req-def456",
      timestamp: t,
      functionName: "send-notification",
      level: "error",
      message: "Failed to send notification",
      details: "Error: Connection timeout after 30s\nStack trace:\n  at sendEmail (handler.go:45)\n  at main (main.go:12)",
    },
  ];

  const files: FileEntity[] = [
    {
      id: "file-1",
      name: "config.json",
      path: "/config",
      sizeBytes: 2457,
      type: "application/json",
      modifiedAt: t,
      kind: "file",
      textContent: `{"version":"1.0.0","environment":"production"}`,
    },
    {
      id: "file-2",
      name: "README.md",
      path: "/",
      sizeBytes: 1200,
      type: "text/markdown",
      modifiedAt: t,
      kind: "file",
      textContent: "# Example file\n\nThis is a demo artifact.",
    },
  ];

  return {
    functions: [fn1, fn2],
    files,
    invocations,
    logs,
    project: {
      name: "my-project",
      id: "prj_abc123def456",
      settings: { autoDeploy: true, logRetention30d: true },
    },
    users: [
      {
        id: "user-default",
        email: "demo@example.com",
        password: "password",
        name: "Demo User",
        createdAt: t,
        avatarUrl: "https://api.dicebear.com/7.x/avataaars/svg?seed=Felix",
      },
    ],
  };
}

function safeParse(raw: string | null): Db | null {
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Db;
  } catch {
    return null;
  }
}

export function loadDb(): Db {
  const parsed = safeParse(localStorage.getItem(STORAGE_KEY));
  if (parsed) {
    // Migration: ensure users exist if loading from old schema
    if (!parsed.users) {
      parsed.users = seed().users;
      saveDb(parsed);
    }
    return parsed;
  }
  const db = seed();
  localStorage.setItem(STORAGE_KEY, JSON.stringify(db));
  return db;
}

export function saveDb(db: Db) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(db));
}

export function updateDb(mutator: (db: Db) => void) {
  const db = loadDb();
  mutator(db);
  saveDb(db);
  return db;
}

export function clearSession() {
  updateDb((db) => {
    delete db.session;
  });
}

import type { FileEntity, FunctionEntity, InvocationEntity, LogEntity, Session, SourceFile } from "./types";
import { clearSession, loadDb, saveDb, updateDb } from "./store";
import { isTextLike, nowIso, randomId, simulate } from "./utils";
import JSZip from "jszip";

function requireSession(): Session {
  const db = loadDb();
  const session = db.session;
  if (!session) {
    const err = new Error("Session expired. Please sign in again.");
    (err as any).code = "SESSION_EXPIRED";
    throw err;
  }
  if (new Date(session.expiresAt).getTime() < Date.now()) {
    clearSession();
    const err = new Error("Session expired. Please sign in again.");
    (err as any).code = "SESSION_EXPIRED";
    throw err;
  }
  return session;
}

function normalizePath(p: string) {
  return p.replace(/\\/g, "/").replace(/^\//, "");
}

function guessEntry(files: SourceFile[], preferredNames: string[] = ["main.go", "handler.go"]) {
  const set = new Set(files.map((f) => f.path.toLowerCase()));
  for (const name of preferredNames) {
    const found = Array.from(set).find((p) => p.endsWith(name));
    if (found) return files.find((f) => f.path.toLowerCase() === found)?.path;
  }
  return undefined;
}

export async function parseZipFile(file: File, onProgress?: (pct: number) => void): Promise<{ files: SourceFile[]; entryPath?: string }> {
  const zip = await JSZip.loadAsync(file);
  const entries = Object.values(zip.files).filter((f) => !f.dir);
  const total = entries.length || 1;
  const files: SourceFile[] = [];

  let i = 0;
  for (const entry of entries) {
    i++;
    const path = normalizePath(entry.name);
    // Best-effort: treat unknown as text for common extensions.
    const asText = isTextLike("", path);
    const sizeBytes = (entry as any)._data?.uncompressedSize ?? 0;
    const text = asText ? await entry.async("string") : undefined;
    files.push({ path, sizeBytes, type: "file", text });
    onProgress?.(Math.round((i / total) * 100));
  }

  const entryPath = guessEntry(files);
  return { files, entryPath };
}

export async function parseDirectoryFiles(
  fileList: FileList,
  onProgress?: (pct: number) => void,
): Promise<{ rootName: string; files: SourceFile[]; entryPath?: string }> {
  const arr = Array.from(fileList);
  const rootName = (arr[0] as any)?.webkitRelativePath?.split("/")?.[0] ?? "directory";
  const total = arr.length || 1;
  const files: SourceFile[] = [];

  for (let i = 0; i < arr.length; i++) {
    const f = arr[i];
    const rel = (f as any).webkitRelativePath ? (f as any).webkitRelativePath : f.name;
    const path = normalizePath(rel).replace(new RegExp(`^${rootName}/`), "");
    const asText = isTextLike(f.type, f.name);
    const text = asText ? await f.text() : undefined;
    files.push({ path, sizeBytes: f.size, type: "file", text });
    onProgress?.(Math.round(((i + 1) / total) * 100));
  }

  const entryPath = guessEntry(files);
  return { rootName, files, entryPath };
}

export const mockApi = {
  auth: {
    async login(email: string, password: string) {
      return simulate(() => {
        const db = loadDb();
        const user = db.users.find((u) => u.email === email);

        if (!user || user.password !== password) {
          const err = new Error("Invalid credentials");
          (err as any).code = "INVALID_CREDENTIALS";
          throw err;
        }

        const session: Session = {
          token: randomId("sess"),
          email: user.email,
          expiresAt: new Date(Date.now() + 1000 * 60 * 60 * 24).toISOString(), // 24h
        };

        db.session = session;
        saveDb(db);
        return session;
      }, { failureRate: 0.04 });
    },
    async logout() {
      return simulate(() => {
        clearSession();
        return true;
      }, { failureRate: 0.02 });
    },
    async getSession() {
      return simulate(() => {
        const db = loadDb();
        return db.session ?? null;
      }, { failureRate: 0.02 });
    },
  },

  project: {
    async getProject() {
      requireSession();
      return simulate(() => loadDb().project, { failureRate: 0.02 });
    },
    async updateProjectName(name: string) {
      requireSession();
      return simulate(() => {
        if (!name.trim()) throw new Error("Project name is required");
        return updateDb((db) => {
          db.project.name = name.trim();
        }).project;
      });
    },
    async updateSettings(settings: { autoDeploy: boolean; logRetention30d: boolean }) {
      requireSession();
      return simulate(() => {
        return updateDb((db) => {
          db.project.settings = settings;
        }).project;
      });
    },
  },

  functions: {
    async list(query?: { q?: string }) {
      requireSession();
      return simulate(() => {
        const db = loadDb();
        const q = query?.q?.trim().toLowerCase();
        if (!q) return db.functions;
        return db.functions.filter((f) => f.name.toLowerCase().includes(q));
      });
    },
    async get(id: string) {
      requireSession();
      return simulate(() => {
        const fn = loadDb().functions.find((f) => f.id === id);
        if (!fn) throw new Error("Function not found");
        return fn;
      });
    },
    async create(input: Omit<FunctionEntity, "id" | "createdAt" | "updatedAt" | "invocations24h" | "errors24h">) {
      requireSession();
      return simulate(() => {
        const t = nowIso();
        const entity: FunctionEntity = {
          ...input,
          id: randomId("fn"),
          createdAt: t,
          updatedAt: t,
          invocations24h: 0,
          errors24h: 0,
        };
        updateDb((db) => {
          db.functions.unshift(entity);
        });
        return entity;
      }, { failureRate: 0.06 });
    },
    async update(id: string, patch: Partial<FunctionEntity>) {
      requireSession();
      return simulate(() => {
        const t = nowIso();
        const db = updateDb((db0) => {
          const idx = db0.functions.findIndex((f) => f.id === id);
          if (idx < 0) throw new Error("Function not found");
          db0.functions[idx] = { ...db0.functions[idx], ...patch, updatedAt: t };
        });
        return db.functions.find((f) => f.id === id)!;
      });
    },
    async duplicate(id: string) {
      requireSession();
      return simulate(() => {
        const src = loadDb().functions.find((f) => f.id === id);
        if (!src) throw new Error("Function not found");
        const t = nowIso();
        const entity: FunctionEntity = {
          ...src,
          id: randomId("fn"),
          name: `${src.name}-copy`,
          createdAt: t,
          updatedAt: t,
          invocations24h: 0,
          errors24h: 0,
        };
        updateDb((db) => {
          db.functions.unshift(entity);
        });
        return entity;
      });
    },
    async remove(id: string) {
      requireSession();
      return simulate(() => {
        updateDb((db) => {
          db.functions = db.functions.filter((f) => f.id !== id);
        });
        return true;
      }, { failureRate: 0.06 });
    },
    async invoke(id: string, jsonInput: string) {
      requireSession();
      return simulate(() => {
        const fn = loadDb().functions.find((f) => f.id === id);
        if (!fn) throw new Error("Function not found");
        try {
          JSON.parse(jsonInput || "{}");
        } catch {
          const err = new Error("Invalid JSON input");
          (err as any).code = "INVALID_JSON";
          throw err;
        }

        const durationMs = Math.max(20, Math.round((fn.avgDurationMs ?? 120) * (0.7 + Math.random() * 0.9)));
        const memoryUsedMb = Math.min(fn.memory, Math.max(16, Math.round(fn.memory * (0.15 + Math.random() * 0.35))));
        const ok = Math.random() > 0.12;
        const t = nowIso();
        const requestId = `req-${Math.random().toString(36).slice(2, 8)}`;

        const inv: InvocationEntity = {
          id: randomId("inv"),
          requestId,
          functionId: fn.id,
          functionName: fn.name,
          status: ok ? "success" : "error",
          durationMs,
          memoryUsedMb,
          timestamp: t,
        };

        const log: LogEntity = {
          id: randomId("log"),
          requestId,
          timestamp: t,
          functionName: fn.name,
          level: ok ? "info" : "error",
          message: ok ? "Invocation completed" : "Invocation failed",
          details: ok
            ? `durationMs=${durationMs} memoryUsedMb=${memoryUsedMb}`
            : `Error: simulated failure\nrequestId=${requestId}`,
        };

        updateDb((db) => {
          db.invocations.unshift(inv);
          db.logs.unshift(log);
          const idx = db.functions.findIndex((f) => f.id === id);
          if (idx >= 0) {
            const prev = db.functions[idx];
            db.functions[idx] = {
              ...prev,
              lastRunAt: t,
              lastRunStatus: ok ? "success" : "error",
              invocations24h: prev.invocations24h + 1,
              errors24h: prev.errors24h + (ok ? 0 : 1),
              avgDurationMs: Math.round((prev.avgDurationMs ?? durationMs) * 0.8 + durationMs * 0.2),
            };
          }
        });

        return {
          ok,
          requestId,
          response: ok
            ? { success: true, requestId }
            : { success: false, error: "simulated failure", requestId },
          durationMs,
          memoryUsedMb,
        };
      }, { failureRate: 0.05, minMs: 450, maxMs: 1400 });
    },
  },

  invocations: {
    async list(query?: { q?: string; status?: "success" | "error" | "all" }) {
      requireSession();
      return simulate(() => {
        const db = loadDb();
        const q = query?.q?.trim().toLowerCase();
        const status = query?.status ?? "all";
        return db.invocations.filter((inv) => {
          const matchesSearch = !q || inv.functionName.toLowerCase().includes(q) || inv.requestId.toLowerCase().includes(q);
          const matchesStatus = status === "all" || inv.status === status;
          return matchesSearch && matchesStatus;
        });
      });
    },
  },

  logs: {
    async list(query?: { q?: string; level?: "info" | "warn" | "error" | "all" }) {
      requireSession();
      return simulate(() => {
        const db = loadDb();
        const q = query?.q?.trim().toLowerCase();
        const level = query?.level ?? "all";
        return db.logs.filter((log) => {
          const matchesSearch =
            !q ||
            log.message.toLowerCase().includes(q) ||
            log.functionName.toLowerCase().includes(q) ||
            log.requestId.toLowerCase().includes(q);
          const matchesLevel = level === "all" || log.level === level;
          return matchesSearch && matchesLevel;
        });
      });
    },
  },

  files: {
    async list(query?: { q?: string }) {
      requireSession();
      return simulate(() => {
        const q = query?.q?.trim().toLowerCase();
        const db = loadDb();
        if (!q) return db.files;
        return db.files.filter((f) => f.name.toLowerCase().includes(q));
      });
    },
    async upload(files: File[], onProgress?: (pct: number) => void) {
      requireSession();

      // simulate chunked progress across all files
      const totalBytes = files.reduce((acc, f) => acc + f.size, 0) || 1;
      let sent = 0;

      return simulate(async () => {
        const created: FileEntity[] = [];

        for (const f of files) {
          // Simulated chunking
          const chunks = Math.max(3, Math.ceil(f.size / (256 * 1024)));
          for (let i = 0; i < chunks; i++) {
            const chunkBytes = f.size / chunks;
            sent += chunkBytes;
            onProgress?.(Math.min(100, Math.round((sent / totalBytes) * 100)));
            // small delay between chunks
            await new Promise((r) => setTimeout(r, 80));
          }

          const isZip = f.name.toLowerCase().endsWith(".zip");
          const t = nowIso();
          let archive: FileEntity["archive"] | undefined;
          if (isZip) {
            // Parse zip after upload finishes (still client-side).
            const parsed = await parseZipFile(f);
            archive = { files: parsed.files, entryPath: parsed.entryPath };
          }
          const textContent = isTextLike(f.type, f.name) && f.size < 200_000 ? await f.text() : undefined;

          const entity: FileEntity = {
            id: randomId("file"),
            name: f.name,
            path: "/",
            sizeBytes: f.size,
            type: f.type || (isZip ? "application/zip" : "application/octet-stream"),
            modifiedAt: t,
            kind: isZip ? "archive" : "file",
            archive,
            textContent,
          };
          created.push(entity);
        }

        updateDb((db) => {
          db.files = [...created, ...db.files];
        });

        onProgress?.(100);
        return created;
      }, { failureRate: 0.07, minMs: 350, maxMs: 900 });
    },
    async remove(id: string) {
      requireSession();
      return simulate(() => {
        updateDb((db) => {
          db.files = db.files.filter((f) => f.id !== id);
        });
        return true;
      }, { failureRate: 0.06 });
    },
  },
};

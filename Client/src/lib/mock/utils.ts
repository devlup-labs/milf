export function nowIso() {
  return new Date().toISOString();
}

export function randomId(prefix: string) {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}${Math.random()
    .toString(36)
    .slice(2, 6)}`;
}

export function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function simulate<T>(
  fn: () => T | Promise<T>,
  opts?: { minMs?: number; maxMs?: number; failureRate?: number },
) {
  const minMs = opts?.minMs ?? 250;
  const maxMs = opts?.maxMs ?? 850;
  const failureRate = opts?.failureRate ?? 0.08;

  const ms = Math.floor(minMs + Math.random() * (maxMs - minMs));
  await sleep(ms);

  if (Math.random() < failureRate) {
    const err = new Error("Request failed. Please retry.");
    (err as any).code = "NETWORK_ERROR";
    throw err;
  }
  return await fn();
}

export function isTextLike(mime: string, name: string) {
  const lower = name.toLowerCase();
  if (
    mime.startsWith("text/") ||
    mime.includes("json") ||
    mime.includes("xml") ||
    mime.includes("yaml") ||
    mime.includes("javascript") ||
    mime.includes("typescript")
  ) {
    return true;
  }
  return [
    ".go",
    ".js",
    ".ts",
    ".tsx",
    ".jsx",
    ".json",
    ".md",
    ".txt",
    ".yml",
    ".yaml",
    ".toml",
    ".env",
    ".css",
    ".html",
    ".sql",
    ".csv",
  ].some((ext) => lower.endsWith(ext));
}

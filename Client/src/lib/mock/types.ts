export type Runtime = "go1.21" | "go1.20" | "go1.19";

export type FunctionStatus = "active" | "inactive" | "error";

export interface FunctionEntity {
  id: string;
  name: string;
  runtime: Runtime;
  status: FunctionStatus;
  memory: number;
  timeout: number;
  tags: string[];
  envVars: { key: string; value: string }[];
  createdAt: string; // ISO
  updatedAt: string; // ISO
  lastRunAt?: string; // ISO
  lastRunStatus?: "success" | "error";
  avgDurationMs?: number;
  invocations24h: number;
  errors24h: number;
  source:
  | { type: "inline"; code: string }
  | { type: "zip"; fileName: string; files: SourceFile[]; entryPath?: string }
  | { type: "directory"; rootName: string; files: SourceFile[]; entryPath?: string };
}

export interface InvocationEntity {
  id: string;
  functionId: string;
  functionName: string;
  requestId: string;
  status: "success" | "error";
  durationMs: number;
  memoryUsedMb: number;
  timestamp: string; // ISO
}

export interface LogEntity {
  id: string;
  requestId: string;
  timestamp: string; // ISO
  functionName: string;
  level: "info" | "warn" | "error";
  message: string;
  details?: string;
}

export type FileKind = "file" | "archive";

export interface FileEntity {
  id: string;
  name: string;
  path: string; // folder path, leading slash
  sizeBytes: number;
  type: string;
  modifiedAt: string; // ISO
  kind: FileKind;
  // If uploaded zip, keep parsed listing to enable explorer preview.
  archive?: {
    files: SourceFile[];
    entryPath?: string;
  };
  // For small text files, store content to enable preview.
  textContent?: string;
}

export interface SourceFile {
  path: string; // posix relative
  sizeBytes: number;
  type: "file";
  text?: string; // present for text files
}

export interface Session {
  token: string;
  email: string;
  expiresAt: string; // ISO
}

export interface UserEntity {
  id: string;
  email: string;
  password: string; // Plain text for mock simplicity, or hash if desired
  name: string;
  avatarUrl?: string;
  createdAt: string;
}

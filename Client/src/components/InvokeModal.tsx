import { useState, useEffect } from "react";
import { Play, AlertTriangle, CheckCircle2, X, Loader2, Info } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { inferParams, type ParamType } from "@/lib/inferParams";

// ─── Type guides (validation + display) ─────────────────────────────────────

const TYPE_GUIDES: Record<ParamType, { label: string; hint: string; validate: (v: string) => boolean }> = {
  int:    { label: "int32",    hint: "32-bit integer  ·  −2,147,483,648 to 2,147,483,647  ·  e.g. 42",  validate: v => /^-?\d+$/.test(v.trim()) && Math.abs(Number(v)) <= 2147483647 },
  long:   { label: "int64",   hint: "64-bit integer  ·  e.g. 9876543210",                               validate: v => /^-?\d+$/.test(v.trim()) },
  float:  { label: "float32", hint: "32-bit decimal  ·  e.g. 3.14",                                    validate: v => /^-?\d+(\.\d+)?([eE][+-]?\d+)?$/.test(v.trim()) },
  double: { label: "float64", hint: "64-bit decimal  ·  higher precision  ·  e.g. 3.14159265",         validate: v => /^-?\d+(\.\d+)?([eE][+-]?\d+)?$/.test(v.trim()) },
  string: { label: "string",  hint: "Plain text  ·  e.g. hello world",                                  validate: v => v.length > 0 },
  bool:   { label: "bool",    hint: "Must be exactly  true  or  false",                                 validate: v => v.trim() === "true" || v.trim() === "false" },
  bytes:  { label: "bytes",   hint: "Base64-encoded binary data",                                       validate: v => { try { return btoa(atob(v)) === v; } catch { return false; } } },
};

// ─── Component ────────────────────────────────────────────────────────────────

interface InvokeModalProps {
  open: boolean;
  onClose: () => void;
  functionName: string;
  runtime: string;
  sourceCode: string;
  onInvoke: (inputPayload: string) => Promise<void>;
  isLoading: boolean;
}

interface FieldState {
  value: string;
  touched: boolean;
  valid: boolean | null;
}

export function InvokeModal({
  open,
  onClose,
  functionName,
  runtime,
  sourceCode,
  onInvoke,
  isLoading,
}: InvokeModalProps) {
  const params = inferParams(sourceCode, runtime);
  const [fields, setFields] = useState<Record<string, FieldState>>({});
  const [rawJson, setRawJson] = useState("");
  const [rawJsonError, setRawJsonError] = useState("");
  const [mode, setMode] = useState<"guided" | "raw">("guided");

  // Reset fields when modal opens
  useEffect(() => {
    if (open) {
      const initial: Record<string, FieldState> = {};
      for (const p of params) {
        initial[p.name] = { value: "", touched: false, valid: null };
      }
      setFields(initial);
      // Build a readable default raw-JSON example from inferred params
      const exampleData: Record<string, unknown> = {};
      for (const p of params) {
        exampleData[p.name] = p.type === "int" || p.type === "long" ? 42
          : p.type === "float" || p.type === "double" ? 3.14
          : p.type === "bool" ? true
          : "value";
      }
      setRawJson(JSON.stringify({ type: "json", data: exampleData }, null, 2));
      setMode(params.length > 0 ? "guided" : "raw");
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  if (!open) return null;

  const setField = (name: string, value: string) => {
    const param = params.find(p => p.name === name)!;
    const guide = TYPE_GUIDES[param.type];
    setFields(prev => ({
      ...prev,
      [name]: { value, touched: true, valid: value === "" ? null : guide.validate(value) },
    }));
  };

  const allValid = params.every(p => {
    const f = fields[p.name];
    if (!f) return true;
    return f.valid !== false;
  });

  const buildPayload = (): string => {
    if (mode === "raw") {
      try {
        JSON.parse(rawJson);
        setRawJsonError("");
        return rawJson;
      } catch {
        setRawJsonError("Invalid JSON — check for missing commas or quotes");
        return "";
      }
    }

    const data: Record<string, unknown> = {};
    for (const p of params) {
      const raw = fields[p.name]?.value ?? "";
      if (raw === "") continue;
      if (p.type === "int" || p.type === "long") data[p.name] = parseInt(raw);
      else if (p.type === "float" || p.type === "double") data[p.name] = parseFloat(raw);
      else if (p.type === "bool") data[p.name] = raw === "true";
      else data[p.name] = raw;
    }
    return JSON.stringify({ type: "json", data });
  };

  const handleSubmit = async () => {
    const payload = buildPayload();
    if (!payload) return;
    await onInvoke(payload);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70" onClick={onClose}>
      <div
        className="bg-[#0f0f1a] border border-white/10 rounded-xl shadow-2xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/10">
          <div>
            <h2 className="text-sm font-semibold text-white flex items-center gap-2">
              <Play className="h-3.5 w-3.5 text-blue-400" />
              Invoke <span className="font-mono text-blue-400">{functionName}</span>
            </h2>
            <p className="text-xs text-white/40 mt-0.5">
              Provide the inputs before dispatching to the mobile WASM node
            </p>
          </div>
          <button onClick={onClose} className="text-white/40 hover:text-white transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Mode toggle */}
        <div className="flex items-center gap-1 px-5 pt-3">
          <button
            className={cn(
              "text-xs px-3 py-1.5 rounded-md font-medium transition-colors",
              mode === "guided" ? "bg-blue-600 text-white" : "text-white/50 hover:text-white hover:bg-white/5"
            )}
            onClick={() => setMode("guided")}
          >
            Guided Input
          </button>
          <button
            className={cn(
              "text-xs px-3 py-1.5 rounded-md font-medium transition-colors",
              mode === "raw" ? "bg-blue-600 text-white" : "text-white/50 hover:text-white hover:bg-white/5"
            )}
            onClick={() => setMode("raw")}
          >
            Raw JSON
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
          {mode === "guided" ? (
            params.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <Info className="h-8 w-8 text-white/20 mb-3" />
                <p className="text-sm text-white/50">Could not infer parameters from source code.</p>
                <p className="text-xs text-white/30 mt-1">Switch to Raw JSON to provide input manually.</p>
                <button
                  className="mt-3 text-xs text-blue-400 hover:underline"
                  onClick={() => setMode("raw")}
                >
                  Switch to Raw JSON →
                </button>
              </div>
            ) : (
              params.map(param => {
                const guide = TYPE_GUIDES[param.type];
                const field = fields[param.name];
                const showValid = field?.touched && field?.valid !== null;
                return (
                  <div key={param.name} className="space-y-1.5">
                    <div className="flex items-center justify-between">
                      <Label className="text-xs text-white/80 font-mono">
                        {param.name}
                        <span className="ml-2 text-[10px] px-1.5 py-0.5 bg-white/10 text-white/40 rounded font-sans">
                          {guide.label}
                        </span>
                      </Label>
                      {showValid && (
                        field.valid
                          ? <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
                          : <AlertTriangle className="h-3.5 w-3.5 text-amber-400" />
                      )}
                    </div>
                    <Input
                      value={field?.value ?? ""}
                      onChange={e => setField(param.name, e.target.value)}
                      placeholder={`e.g. ${param.example}`}
                      className={cn(
                        "h-8 bg-white/5 border text-sm font-mono text-white placeholder:text-white/20",
                        "transition-colors focus:ring-1 focus:ring-blue-500",
                        showValid && !field.valid
                          ? "border-amber-500/50 focus:ring-amber-500"
                          : "border-white/10"
                      )}
                    />
                    <p className="text-[10px] text-white/30">{guide.hint}</p>
                    {showValid && !field.valid && (
                      <p className="text-[10px] text-amber-400 flex items-center gap-1">
                        <AlertTriangle className="h-2.5 w-2.5" />
                        Expected {guide.label}: {guide.hint}
                      </p>
                    )}
                  </div>
                );
              })
            )
          ) : (
            <div className="space-y-1.5">
              <Label className="text-xs text-white/60">Input Payload (TaskEnvelope JSON)</Label>
              <div className="relative">
                <textarea
                  value={rawJson}
                  onChange={e => { setRawJson(e.target.value); setRawJsonError(""); }}
                  rows={8}
                  className={cn(
                    "w-full bg-white/5 border rounded-md text-xs font-mono text-white",
                    "p-3 resize-none focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors",
                    rawJsonError ? "border-red-500/50" : "border-white/10"
                  )}
                />
                {rawJsonError && (
                  <p className="text-[10px] text-red-400 mt-1 flex items-center gap-1">
                    <AlertTriangle className="h-2.5 w-2.5" /> {rawJsonError}
                  </p>
                )}
              </div>
              <p className="text-[10px] text-white/30">
                Use <code className="bg-white/10 px-1 rounded">"type": "binary"</code> for base64 data,{" "}
                <code className="bg-white/10 px-1 rounded">"type": "json"</code> for structured input, or{" "}
                <code className="bg-white/10 px-1 rounded">"type": "null"</code> for no input.
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-5 py-4 border-t border-white/10 bg-black/20">
          <p className="text-[10px] text-white/30">
            Task → Android WASM node → result polls back here
          </p>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={onClose} className="text-white/60">
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleSubmit}
              disabled={isLoading || !allValid}
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              {isLoading
                ? <><Loader2 className="h-3.5 w-3.5 mr-2 animate-spin" /> Dispatching...</>
                : <><Play className="h-3.5 w-3.5 mr-2" /> Invoke</>
              }
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

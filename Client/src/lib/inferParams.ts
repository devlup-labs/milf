/**
 * Utility: infer WASI function parameters from source code.
 * Kept in a separate file so component files don't export non-component values
 * (which breaks Vite Fast Refresh).
 */

export type ParamType = "int" | "long" | "float" | "double" | "string" | "bool" | "bytes";

export interface FunctionParam {
  name: string;
  type: ParamType;
  description: string;
  example: string;
}

function mapCType(typ: string): ParamType {
  const t = typ.trim().toLowerCase().replace(/\s+/g, " ");
  if (t === "int" || t === "int32_t" || t === "uint32_t") return "int";
  if (t === "long" || t === "long long" || t === "int64_t" || t === "uint64_t") return "long";
  if (t === "float") return "float";
  if (t === "double") return "double";
  if (t === "bool" || t === "_bool") return "bool";
  if (t.startsWith("char") || t.includes("*")) return "string";
  return "int";
}

function parseParamList(paramStr: string): FunctionParam[] {
  return paramStr
    .split(",")
    .map(seg => {
      seg = seg.trim();
      if (!seg || seg === "void") return null;
      const tokens = seg.split(/\s+/);
      if (tokens.length < 2) return null;
      const name = tokens[tokens.length - 1].replace(/^\*+/, "");
      const typePart = tokens.slice(0, -1).join(" ");
      const t = mapCType(typePart);
      return {
        name,
        type: t,
        description: `C parameter (${typePart})`,
        example:
          t === "int" || t === "long" ? "42"
          : t === "float" || t === "double" ? "3.14"
          : t === "bool" ? "true"
          : "hello",
      } as FunctionParam;
    })
    .filter(Boolean) as FunctionParam[];
}

/**
 * Infer parameters from a WASI-exported function.
 *
 * Handles the canonical form:
 *   __attribute__((visibility("default"))) __attribute__((used))
 *   int main(int a, int b) { ... }
 */
export function inferParams(sourceCode: string, runtime: string): FunctionParam[] {
  if (!sourceCode || sourceCode.includes("// File-based")) return [];

  // ── C / C++ ───────────────────────────────────────────────────────────────
  if (runtime === "c" || runtime === "cpp") {
    const collapsed = sourceCode.replace(/\r?\n/g, " ").replace(/\s+/g, " ");

    // Match __attribute__((visibility("default"))) ... rettype name ( params )
    const attrRegex =
      /__attribute__\s*\(\s*\(\s*visibility\s*\(\s*"default"\s*\)\s*\)\s*\)[^(]*\(\s*([^)]*)\)/g;

    let match: RegExpExecArray | null;
    while ((match = attrRegex.exec(collapsed)) !== null) {
      const paramStr = match[1].trim();
      if (!paramStr || paramStr === "void") continue;
      const params = parseParamList(paramStr);
      if (params.length > 0) return params;
    }

    // Fallback: plain int main(int a, int b)
    const plainMain = collapsed.match(/\bint\s+main\s*\(\s*([^)]+)\s*\)/);
    if (plainMain) {
      const params = parseParamList(plainMain[1]);
      if (params.length > 0) return params;
    }
    return [];
  }

  // ── Go ────────────────────────────────────────────────────────────────────
  if (runtime === "go") {
    const goMatch = sourceCode.match(/func\s+(?:invoke|main|handler|Handler)\s*\(([^)]+)\)/);
    if (goMatch) {
      return goMatch[1]
        .split(",")
        .map(arg => {
          const parts = arg.trim().split(/\s+/);
          if (parts.length < 2) return null;
          const [name, typ] = parts;
          return {
            name,
            type:
              typ.includes("int64") ? "long"
              : typ.includes("int") ? "int"
              : typ.includes("float64") ? "double"
              : typ.includes("float32") ? "float"
              : typ === "bool" ? "bool"
              : "string",
            description: `Go parameter (${typ})`,
            example: typ.includes("int") ? "42" : typ === "bool" ? "true" : "value",
          } as FunctionParam;
        })
        .filter(Boolean) as FunctionParam[];
    }
    return [];
  }

  // ── Python ────────────────────────────────────────────────────────────────
  if (runtime === "python") {
    const pyMatch = sourceCode.match(/def\s+(?:handler|main|invoke)\s*\(([^)]+)\)/);
    if (pyMatch) {
      return pyMatch[1]
        .split(",")
        .map(arg => {
          const name = arg.trim().split(":")[0].split("=")[0].trim();
          const ann = (arg.includes(":") ? arg.split(":")[1] : "").trim().split("=")[0].trim();
          return {
            name,
            type:
              ann === "int" ? "int"
              : ann === "float" ? "float"
              : ann === "bool" ? "bool"
              : "string",
            description: "Python parameter",
            example: ann === "int" ? "42" : ann === "float" ? "3.14" : ann === "bool" ? "true" : "value",
          } as FunctionParam;
        })
        .filter(p => p.name && p.name !== "self");
    }
    return [];
  }

  return [];
}

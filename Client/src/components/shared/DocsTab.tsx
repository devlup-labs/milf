import { useState } from "react";
import { Link } from "react-router-dom";
import {
  Tag, Cpu, Clock, Code2, Play, Download, AlertTriangle, CheckCircle2,
  ChevronDown, ChevronUp, ArrowRight, Zap, Globe, FileText, Terminal,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

// ─── Reusable helpers ────────────────────────────────────────────────────────

function SectionHeader({ icon: Icon, title, subtitle }: { icon: React.ElementType; title: string; subtitle?: string }) {
  return (
    <div className="flex items-start gap-4 pb-4 border-b border-border mb-6">
      <div className="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0">
        <Icon className="h-5 w-5 text-primary" />
      </div>
      <div>
        <h2 className="text-base font-semibold text-foreground">{title}</h2>
        {subtitle && <p className="text-sm text-muted-foreground mt-0.5">{subtitle}</p>}
      </div>
    </div>
  );
}

function CodeBlock({ code, lang = "c" }: { code: string; lang?: string }) {
  return (
    <div className="bg-black/40 border border-border/60 rounded-lg overflow-hidden my-3">
      <div className="flex items-center gap-1.5 px-3 py-2 bg-black/20 border-b border-border/30">
        <div className="w-2.5 h-2.5 rounded-full bg-red-500/70" />
        <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/70" />
        <div className="w-2.5 h-2.5 rounded-full bg-green-500/70" />
        <span className="text-[10px] font-mono text-muted-foreground ml-2">{lang}</span>
      </div>
      <pre className="p-4 text-xs font-mono text-foreground/90 leading-relaxed overflow-x-auto">
        <code>{code}</code>
      </pre>
    </div>
  );
}

function Callout({
  type,
  children,
}: {
  type: "tip" | "warning" | "info";
  children: React.ReactNode;
}) {
  const styles = {
    tip: "bg-green-500/8 border-green-500/25 text-green-400",
    warning: "bg-yellow-500/8 border-yellow-500/25 text-yellow-400",
    info: "bg-blue-500/8 border-blue-500/25 text-blue-400",
  };
  const Icon = type === "tip" ? CheckCircle2 : type === "warning" ? AlertTriangle : Zap;
  return (
    <div className={cn("flex items-start gap-3 border rounded-lg p-3.5 text-sm", styles[type])}>
      <Icon className="h-4 w-4 mt-0.5 shrink-0" />
      <div className="leading-relaxed text-foreground/80">{children}</div>
    </div>
  );
}

function Accordion({ title, children }: { title: string; children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-foreground hover:bg-muted/20 transition-colors"
      >
        {title}
        {open ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
      </button>
      {open && <div className="px-4 pb-4 pt-2 text-sm text-muted-foreground leading-relaxed border-t border-border">{children}</div>}
    </div>
  );
}

// ─── Docs sections ────────────────────────────────────────────────────────────

function QuickStart({ onCreateClick }: { onCreateClick: () => void }) {
  const steps = [
    {
      n: "1",
      icon: Code2,
      color: "text-blue-400",
      bg: "bg-blue-400/10 border-blue-400/20",
      title: "Pick or write a function",
      desc: "Browse Demo Functions and click \"Use This Template\" — or click Create Function and write your own C code.",
    },
    {
      n: "2",
      icon: Zap,
      color: "text-yellow-400",
      bg: "bg-yellow-400/10 border-yellow-400/20",
      title: "Name it and deploy",
      desc: "Give your function a unique slug-style name. Leave Memory and Timeout at defaults. Hit Deploy.",
    },
    {
      n: "3",
      icon: Play,
      color: "text-green-400",
      bg: "bg-green-400/10 border-green-400/20",
      title: "Invoke and see results",
      desc: "Open the Invoke / Test tab, type a JSON payload, and watch your code execute on a real Android node.",
    },
  ];

  return (
    <section className="space-y-5">
      <SectionHeader icon={Zap} title="Quick Start" subtitle="From zero to running WASM in 3 steps" />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {steps.map(({ n, icon: Icon, color, bg, title, desc }) => (
          <div key={n} className="bg-surface border border-border rounded-xl p-4 relative">
            <div className={cn("w-9 h-9 rounded-lg border flex items-center justify-center mb-3", bg)}>
              <Icon className={cn("h-4.5 w-4.5", color)} />
            </div>
            <span className="absolute top-3 right-3 text-[10px] font-bold text-muted-foreground/50">STEP {n}</span>
            <h3 className="text-sm font-semibold text-foreground mb-1">{title}</h3>
            <p className="text-xs text-muted-foreground leading-relaxed">{desc}</p>
          </div>
        ))}
      </div>

      <div className="flex items-center gap-3">
        <Button asChild className="gap-2">
          <Link to="/functions/create">
            <Code2 className="h-4 w-4" />
            Create Your First Function
          </Link>
        </Button>
        <ArrowRight className="h-4 w-4 text-muted-foreground" />
        <span className="text-xs text-muted-foreground">Takes about 2 minutes</span>
      </div>
    </section>
  );
}

function NamingGuide() {
  return (
    <section className="space-y-5">
      <SectionHeader
        icon={Tag}
        title="Naming Your Function"
        subtitle="A good name makes your function easy to find and invoke"
      />

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="space-y-2">
          <p className="text-xs font-semibold text-green-400 uppercase tracking-wider">✅ Good names</p>
          <div className="space-y-1">
            {["echo-json", "fetch-weather", "pdf-generator", "image-resize", "crypto-hash"].map((n) => (
              <div key={n} className="flex items-center gap-2 bg-green-500/5 border border-green-500/15 rounded px-3 py-2">
                <span className="font-mono text-sm text-green-400">{n}</span>
              </div>
            ))}
          </div>
        </div>
        <div className="space-y-2">
          <p className="text-xs font-semibold text-red-400 uppercase tracking-wider">❌ Avoid these</p>
          <div className="space-y-1">
            {[
              ["My Function", "spaces not allowed"],
              ["fn1", "too short, not descriptive"],
              ["UPPER_CASE", "use lowercase-hyphens"],
              ["my.function.v2", "dots cause routing issues"],
              ["a".repeat(65), "max 64 characters"],
            ].map(([n, reason]) => (
              <div key={n} className="flex items-center justify-between bg-red-500/5 border border-red-500/15 rounded px-3 py-2">
                <span className="font-mono text-sm text-red-400 truncate max-w-[120px]">{n}</span>
                <span className="text-[10px] text-muted-foreground">{reason}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      <Callout type="tip">
        <strong>Rule of thumb:</strong> Name it after what it <em>does</em>, not what it is. <code className="bg-muted px-1 rounded text-xs">fetch-stock-price</code> is better than <code className="bg-muted px-1 rounded text-xs">my-wasm-function</code>.
      </Callout>
    </section>
  );
}

function ExecutionSettings() {
  const settings = [
    {
      icon: Cpu,
      label: "Memory",
      defaultVal: "128 MB",
      range: "128 MB — 3008 MB",
      why: "WASM linear memory is pre-allocated at startup. 128MB is enough for most compute-heavy tasks including image processing. Only increase this if you're seeing out-of-memory errors.",
      when: "Bump to 512MB+ if you're manipulating large files (> 5MB images, video frames).",
    },
    {
      icon: Clock,
      label: "Timeout",
      defaultVal: "30 seconds",
      range: "1s — 900s",
      why: "The scheduler kills the execution if it runs longer than this. Most WASM functions complete in under 500ms. 30s is very generous.",
      when: "Increase only for batch operations (downloading many large files, heavy PDF rendering).",
    },
  ];

  return (
    <section className="space-y-5">
      <SectionHeader
        icon={Cpu}
        title="Execution Settings"
        subtitle="Why the defaults work perfectly for 95% of use cases"
      />

      <div className="space-y-4">
        {settings.map(({ icon: Icon, label, defaultVal, range, why, when }) => (
          <div key={label} className="bg-surface border border-border rounded-xl p-5">
            <div className="flex items-start gap-4">
              <div className="w-9 h-9 bg-primary/10 border border-primary/20 rounded-lg flex items-center justify-center shrink-0">
                <Icon className="h-4 w-4 text-primary" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3 flex-wrap">
                  <h3 className="text-sm font-semibold text-foreground">{label}</h3>
                  <span className="text-[10px] font-bold px-2 py-0.5 bg-primary/10 text-primary rounded-full border border-primary/20">
                    Default: {defaultVal}
                  </span>
                  <span className="text-[10px] text-muted-foreground font-mono">{range}</span>
                </div>
                <p className="text-xs text-muted-foreground mt-2 leading-relaxed">{why}</p>
                <p className="text-xs text-primary/70 mt-1.5">
                  <strong className="text-primary">When to change:</strong> {when}
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>

      <Callout type="tip">
        You can always change these later from the <strong>Configuration</strong> tab inside your function's detail page — no need to redeploy.
      </Callout>
    </section>
  );
}

function WritingCode() {
  return (
    <section className="space-y-5">
      <SectionHeader
        icon={Terminal}
        title="Writing Your wasm_main"
        subtitle="Two function signatures — pick the right one for your use case"
      />

      <div className="space-y-6">
        {/* Signature 1 */}
        <div className="bg-surface border border-border rounded-xl overflow-hidden">
          <div className="px-5 pt-5 pb-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-[10px] font-bold uppercase tracking-wider text-yellow-400 bg-yellow-400/10 border border-yellow-400/20 px-2 py-0.5 rounded-full">Beginner</span>
              <h3 className="text-sm font-semibold text-foreground">Primitive types — integers, floats</h3>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Use this when your function takes and returns simple numbers. No buffers, no JSON.
            </p>
          </div>
          <CodeBlock
            lang="c · signature 1"
            code={`#include <stdint.h>

__attribute__((visibility("default"))) __attribute__((used))
int wasm_main(int a, int b) {
  return a + b;   // Return value is the output
}`}
          />
          <div className="px-5 pb-5 grid grid-cols-2 gap-3">
            <div className="bg-muted/20 rounded-lg p-3">
              <p className="text-[9px] font-bold uppercase text-muted-foreground mb-1">Input (payload)</p>
              <code className="text-xs font-mono text-foreground">{`{"a": 3, "b": 7}`}</code>
            </div>
            <div className="bg-emerald-500/5 rounded-lg p-3">
              <p className="text-[9px] font-bold uppercase text-emerald-400 mb-1">Output</p>
              <code className="text-xs font-mono text-emerald-400">10</code>
            </div>
          </div>
        </div>

        {/* Signature 2 */}
        <div className="bg-surface border border-border rounded-xl overflow-hidden">
          <div className="px-5 pt-5 pb-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-[10px] font-bold uppercase tracking-wider text-blue-400 bg-blue-400/10 border border-blue-400/20 px-2 py-0.5 rounded-full">Standard</span>
              <h3 className="text-sm font-semibold text-foreground">JSON / String — the standard ABI</h3>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Use this for all complex inputs/outputs. <code className="bg-muted px-1 rounded text-xs">payload</code> is your JSON input,{" "}
              <code className="bg-muted px-1 rounded text-xs">out_buf</code> is where you write the result. Return the number of bytes written.
            </p>
          </div>
          <CodeBlock
            lang="c · signature 2 (recommended)"
            code={`#include <milf.h>   // milf_memcpy, milf_strlen, milf_stream_*

MILF_EXPORT int wasm_main(
    char* payload,    // Your JSON input, not null-terminated
    int   payload_len,// Byte length of payload
    char* out_buf,    // Write your output here
    int   out_max     // Maximum bytes you can write
) {
    const char* msg = "{\\"status\\": \\"ok\\"}";
    milf_memcpy(out_buf, msg, milf_strlen(msg));
    return milf_strlen(msg);  // ← always return bytes written
}`}
          />
          <div className="px-5 pb-5">
            <Callout type="info">
              <code className="bg-muted px-1 rounded text-xs">milf.h</code> replaces the C standard library inside the sandbox. Use{" "}
              <code className="bg-muted px-1 rounded text-xs">milf_memcpy</code> instead of <code className="bg-muted px-1 rounded text-xs">memcpy</code>,{" "}
              <code className="bg-muted px-1 rounded text-xs">milf_strlen</code> instead of <code className="bg-muted px-1 rounded text-xs">strlen</code>.
            </Callout>
          </div>
        </div>

        {/* Network signature */}
        <div className="bg-surface border border-border rounded-xl overflow-hidden">
          <div className="px-5 pt-5 pb-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-[10px] font-bold uppercase tracking-wider text-emerald-400 bg-emerald-400/10 border border-emerald-400/20 px-2 py-0.5 rounded-full">Network</span>
              <h3 className="text-sm font-semibold text-foreground">Fetch from the internet</h3>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Use <code className="bg-muted px-1 rounded text-xs">milf_stream_open</code> / <code className="bg-muted px-1 rounded text-xs">milf_stream_read</code> /
              {" "}<code className="bg-muted px-1 rounded text-xs">milf_stream_close</code> to make HTTP requests inside your function.
            </p>
          </div>
          <CodeBlock
            lang="c · network fetch"
            code={`#include <milf.h>

MILF_EXPORT int wasm_main(char* payload, int payload_len,
                           char* out_buf, int out_max) {
    int handle = milf_stream_open("https://api.example.com/data");
    if (handle < 0) { /* handle error */ return 0; }

    int bytes = milf_stream_read(handle, out_buf, out_max - 1);
    milf_stream_close(handle);
    out_buf[bytes] = '\\0';
    return bytes;
}`}
          />
        </div>
      </div>

      <div className="space-y-2">
        <h3 className="text-sm font-medium text-foreground">Common Questions</h3>
        <div className="space-y-2">
          <Accordion title="Can I use printf or stdlib?">
            No. The sandbox does not link against glibc. Use <code className="bg-muted px-1 rounded text-xs">milf_memcpy</code>,{" "}
            <code className="bg-muted px-1 rounded text-xs">milf_strlen</code>, <code className="bg-muted px-1 rounded text-xs">milf_memset</code> for memory,
            and <code className="bg-muted px-1 rounded text-xs">snprintf</code> is available if you need formatted strings.
          </Accordion>
          <Accordion title="Can I use malloc / free?">
            Yes — but sparingly. For performance-critical code, use a bump allocator (static pool + offset counter) like the multi-images-to-pdf demo does. Every function call resets the WASM memory space.
          </Accordion>
          <Accordion title="How do I return a file (PDF, image, etc.)?">
            Call <code className="bg-muted px-1 rounded text-xs">milf_storage_save("filename.pdf", data_ptr, data_len)</code> to store the file, then write{" "}
            <code className="bg-muted px-1 rounded text-xs">FILE:filename.pdf</code> into <code className="bg-muted px-1 rounded text-xs">out_buf</code>.
            The dashboard will show a download button automatically.
          </Accordion>
        </div>
      </div>
    </section>
  );
}

function InvocationGuide() {
  return (
    <section className="space-y-5">
      <SectionHeader
        icon={Play}
        title="The Invoke / Test Tab"
        subtitle="How to run your function and read the output"
      />

      {/* What is invocation */}
      <div className="bg-gradient-to-br from-blue-500/5 to-violet-500/5 border border-blue-500/15 rounded-xl p-5 space-y-3">
        <h3 className="text-sm font-semibold text-foreground">What does Invoke actually do?</h3>
        <div className="flex items-start gap-3">
          <div className="flex flex-col items-center gap-1 shrink-0 pt-1">
            {["Dashboard", "Central Server", "Android Node", "Result"].map((label, i, arr) => (
              <div key={label} className="flex flex-col items-center">
                <div className={cn(
                  "w-28 text-center text-[10px] font-semibold px-2 py-1.5 rounded-lg border",
                  i === 0 ? "bg-primary/10 text-primary border-primary/20" :
                  i === arr.length - 1 ? "bg-green-500/10 text-green-400 border-green-500/20" :
                  "bg-muted/40 text-muted-foreground border-border"
                )}>
                  {label}
                </div>
                {i < arr.length - 1 && <div className="h-3 w-px bg-border" />}
              </div>
            ))}
          </div>
          <div className="space-y-3 text-xs text-muted-foreground leading-relaxed pt-1 flex-1">
            <p>You click <strong className="text-foreground">Invoke</strong> with a JSON payload.</p>
            <p>The Central Server finds a connected Android node, sends it your WASM binary + your payload.</p>
            <p>WAMR executes your <code className="bg-muted px-1 rounded">wasm_main</code> in a sandboxed process. The entire run happens on a real phone.</p>
            <p>The result is sent back to the dashboard and shown in the output panel — usually within 500ms.</p>
          </div>
        </div>
      </div>

      {/* Step by step */}
      <div className="space-y-3">
        <h3 className="text-sm font-medium text-foreground">Step-by-step walkthrough</h3>

        {[
          {
            step: "1",
            title: "Open your function",
            desc: <>Go to <strong>My Functions</strong> → click the function name → you land on the detail page.</>,
          },
          {
            step: "2",
            title: "Click \"Invoke / Test\" tab",
            desc: <>You'll see the invocation panel with a blue <strong>Invoke Function</strong> button.</>,
          },
          {
            step: "3",
            title: "Type your payload",
            desc: (
              <>
                <p className="mb-2">
                  Click <strong>Invoke Function</strong> — a modal opens with a JSON payload editor. The editor is
                  pre-filled based on your function's code signature.
                </p>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <p className="text-[9px] uppercase font-bold text-muted-foreground mb-1.5">Signature 1 (ints)</p>
                    <div className="bg-black/30 rounded-lg p-3 font-mono text-xs text-foreground">
                      {`{\n  "a": 3,\n  "b": 7\n}`}
                    </div>
                  </div>
                  <div>
                    <p className="text-[9px] uppercase font-bold text-muted-foreground mb-1.5">Signature 2 (JSON)</p>
                    <div className="bg-black/30 rounded-lg p-3 font-mono text-xs text-foreground">
                      {`{\n  "message": "hello"\n}`}
                    </div>
                  </div>
                </div>
              </>
            ),
          },
          {
            step: "4",
            title: "Hit \"Invoke\" and wait",
            desc: <>You'll see <em>"Task dispatched to mobile node. Waiting for result..."</em> — this means the job is queued. If no Android node is connected, it will wait. With a node connected, you get the result in &lt; 1 second.</>,
          },
          {
            step: "5",
            title: "Read the output",
            desc: (
              <div className="space-y-2">
                <p>The result appears in the output panel below the button.</p>
                <div className="space-y-1.5">
                  {[
                    { type: "JSON / text", example: '{ "status": "ok", "result": 10 }', color: "text-foreground" },
                    { type: "Error", example: '{ "error": "Empty payload" }', color: "text-red-400" },
                    { type: "File output", example: "FILE:output.pdf → Download button appears", color: "text-emerald-400" },
                  ].map(({ type, example, color }) => (
                    <div key={type} className="flex items-center gap-3 bg-muted/20 border border-border/50 rounded-lg px-3 py-2">
                      <span className="text-[10px] font-bold text-muted-foreground w-20 shrink-0">{type}</span>
                      <code className={cn("font-mono text-xs", color)}>{example}</code>
                    </div>
                  ))}
                </div>
              </div>
            ),
          },
        ].map(({ step, title, desc }) => (
          <div key={step} className="flex items-start gap-4 bg-surface border border-border rounded-xl p-4">
            <div className="w-7 h-7 rounded-full bg-primary text-primary-foreground text-xs font-bold flex items-center justify-center shrink-0">
              {step}
            </div>
            <div className="flex-1">
              <h4 className="text-sm font-semibold text-foreground mb-1.5">{title}</h4>
              <div className="text-xs text-muted-foreground leading-relaxed">{desc}</div>
            </div>
          </div>
        ))}
      </div>

      <Callout type="warning">
        <strong>No output?</strong> This usually means no Android consumer node is currently online. Check that the Pocket Cloud mobile app is running and connected. The function is deployed — it just needs an executor.
      </Callout>

      {/* File outputs */}
      <div className="bg-surface border border-border rounded-xl p-5">
        <div className="flex items-center gap-2 mb-3">
          <Download className="h-4 w-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-foreground">File Outputs (PDFs, Images)</h3>
        </div>
        <p className="text-xs text-muted-foreground leading-relaxed mb-3">
          When your function returns <code className="bg-muted px-1 rounded text-xs">FILE:filename.ext</code>, the dashboard automatically
          shows a download card with the file size, MIME type, and a big green Download button. For PDFs, there's also
          an inline preview.
        </p>
        <div className="bg-emerald-500/5 border border-emerald-500/20 rounded-lg p-3 font-mono text-xs text-emerald-400">
          {`FILE:output.pdf  →  📥 Download output.pdf (142 KB)`}
        </div>
      </div>
    </section>
  );
}

// ─── Main DocsTab export ─────────────────────────────────────────────────────

const NAV_ITEMS = [
  { id: "quick-start", label: "Quick Start", icon: Zap },
  { id: "naming", label: "Naming Functions", icon: Tag },
  { id: "execution", label: "Execution Settings", icon: Cpu },
  { id: "code", label: "Writing Code", icon: Terminal },
  { id: "invoke", label: "Invoke / Test", icon: Play },
];

export function DocsTab() {
  const [active, setActive] = useState("quick-start");

  const scrollTo = (id: string) => {
    setActive(id);
    document.getElementById(`docs-section-${id}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  return (
    <div className="flex gap-6 min-h-[600px]">
      {/* Sticky sidebar nav */}
      <nav className="w-44 shrink-0 hidden md:block">
        <div className="sticky top-4 space-y-0.5">
          <p className="text-[9px] font-bold uppercase tracking-widest text-muted-foreground px-3 mb-3">On this page</p>
          {NAV_ITEMS.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => scrollTo(id)}
              className={cn(
                "w-full flex items-center gap-2 px-3 py-2 rounded-lg text-xs transition-colors text-left",
                active === id
                  ? "bg-primary/10 text-primary font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/30"
              )}
            >
              <Icon className="h-3.5 w-3.5 shrink-0" />
              {label}
            </button>
          ))}
        </div>
      </nav>

      {/* Scrollable content */}
      <div className="flex-1 min-w-0 space-y-16 pb-12">
        <div id="docs-section-quick-start"><QuickStart onCreateClick={() => {}} /></div>
        <div id="docs-section-naming"><NamingGuide /></div>
        <div id="docs-section-execution"><ExecutionSettings /></div>
        <div id="docs-section-code"><WritingCode /></div>
        <div id="docs-section-invoke"><InvocationGuide /></div>
      </div>
    </div>
  );
}

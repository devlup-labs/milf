import { useState, useEffect, useLayoutEffect, useRef } from "react";
import { X, ChevronRight, ChevronLeft, Sparkles, MapPin } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const TOUR_KEY = "milf-tour-v1-done";

interface TourStep {
  targetId?: string;           // element ID to spotlight; undefined = centred modal
  title: string;
  body: React.ReactNode;
  placement?: "bottom" | "top" | "right" | "left";
  cta?: string;                // override the "Next" label on the last step
}

const STEPS: TourStep[] = [
  {
    title: "👋 Welcome to Pocket Cloud",
    body: (
      <div className="space-y-2 text-sm text-muted-foreground leading-relaxed">
        <p>
          <strong className="text-foreground">Pocket Cloud</strong> lets you write a C function, compile it
          to WebAssembly, and run it on real Android devices in{" "}
          <span className="text-primary font-semibold">&lt; 5ms</span>.
        </p>
        <p>
          This quick tour will show you exactly how to deploy your first function in under 3 minutes.
        </p>
      </div>
    ),
    cta: "Let's go →",
  },
  {
    targetId: "tab-demo",
    placement: "bottom",
    title: "🧪 Demo Functions — explore first",
    body: (
      <p className="text-sm text-muted-foreground leading-relaxed">
        Start here. Six real working templates — from a simple integer adder to a multi-image PDF renderer —
        with full source code, sample inputs, and expected outputs. Click{" "}
        <strong className="text-foreground">Use This Template</strong> to pre-fill the editor instantly.
      </p>
    ),
  },
  {
    targetId: "tab-docs",
    placement: "bottom",
    title: "📖 Docs — learn the platform",
    body: (
      <p className="text-sm text-muted-foreground leading-relaxed">
        The <strong className="text-foreground">Docs</strong> tab lives right here. It covers how to name
        your function, why the default execution settings are fine, how to write a valid{" "}
        <code className="bg-muted px-1 rounded text-xs">wasm_main</code>, and exactly how to use the
        Invocation tab to test your code end-to-end.
      </p>
    ),
  },
  {
    targetId: "create-function-btn",
    placement: "bottom",
    title: "✏️ Create your first function",
    body: (
      <div className="space-y-2 text-sm text-muted-foreground leading-relaxed">
        <p>
          When you're ready, click <strong className="text-foreground">Create Function</strong>. You'll land
          in the editor with a working template already loaded.
        </p>
        <ol className="list-decimal list-inside space-y-1 text-xs">
          <li>Give it a name (e.g. <code className="bg-muted px-1 rounded">my-first-fn</code>)</li>
          <li>Leave Memory &amp; Timeout at their defaults</li>
          <li>Hit <strong className="text-foreground">Deploy Function</strong></li>
        </ol>
      </div>
    ),
  },
  {
    targetId: "invoke-tab-hint",
    placement: "top",
    title: "⚡ Invoke / Test — run it live",
    body: (
      <div className="space-y-2 text-sm text-muted-foreground leading-relaxed">
        <p>
          Once deployed, open your function and switch to the{" "}
          <strong className="text-foreground">Invoke / Test</strong> tab. Type a JSON payload — for example{" "}
          <code className="bg-muted px-1 rounded text-xs">{`{"a":3,"b":7}`}</code> — and click{" "}
          <strong className="text-foreground">Invoke Function</strong>.
        </p>
        <p>
          Your code executes on a real Android node and the result appears live right here.
        </p>
      </div>
    ),
    cta: "Got it 🎉",
  },
];

// ─── Spotlight overlay ──────────────────────────────────────────────────────

interface SpotlightRect {
  top: number; left: number; width: number; height: number;
}

function Spotlight({ rect }: { rect: SpotlightRect | null }) {
  if (!rect) return null;
  const PAD = 8;
  const r = {
    top: rect.top - PAD,
    left: rect.left - PAD,
    width: rect.width + PAD * 2,
    height: rect.height + PAD * 2,
  };
  return (
    <div
      className="fixed inset-0 z-[9998] pointer-events-none"
      style={{
        background: `radial-gradient(ellipse at ${r.left + r.width / 2}px ${r.top + r.height / 2}px, transparent ${Math.max(r.width, r.height) * 0.55}px, rgba(0,0,0,0.72) ${Math.max(r.width, r.height) * 0.7}px)`,
      }}
    >
      {/* Highlighted ring */}
      <div
        className="absolute rounded-lg ring-2 ring-primary ring-offset-2 ring-offset-transparent animate-pulse"
        style={{
          top: r.top,
          left: r.left,
          width: r.width,
          height: r.height,
          boxShadow: "0 0 0 9999px rgba(0,0,0,0.60)",
        }}
      />
    </div>
  );
}

// ─── Tooltip card ────────────────────────────────────────────────────────────

interface TooltipProps {
  step: TourStep;
  stepIndex: number;
  total: number;
  spotRect: SpotlightRect | null;
  onNext: () => void;
  onBack: () => void;
  onSkip: () => void;
}

function TourTooltip({ step, stepIndex, total, spotRect, onNext, onBack, onSkip }: TooltipProps) {
  const tooltipRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState<React.CSSProperties>({ top: "50%", left: "50%", transform: "translate(-50%,-50%)" });

  useLayoutEffect(() => {
    if (!spotRect || !tooltipRef.current) {
      setPos({ top: "50%", left: "50%", transform: "translate(-50%,-50%)" });
      return;
    }
    const TW = tooltipRef.current.offsetWidth || 320;
    const TH = tooltipRef.current.offsetHeight || 200;
    const GAP = 16;
    const placement = step.placement ?? "bottom";
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    let top = 0, left = 0;
    if (placement === "bottom") {
      top = spotRect.top + spotRect.height + GAP;
      left = spotRect.left + spotRect.width / 2 - TW / 2;
    } else if (placement === "top") {
      top = spotRect.top - TH - GAP;
      left = spotRect.left + spotRect.width / 2 - TW / 2;
    } else if (placement === "right") {
      top = spotRect.top + spotRect.height / 2 - TH / 2;
      left = spotRect.left + spotRect.width + GAP;
    } else {
      top = spotRect.top + spotRect.height / 2 - TH / 2;
      left = spotRect.left - TW - GAP;
    }

    // Clamp to viewport
    left = Math.max(12, Math.min(left, vw - TW - 12));
    top = Math.max(12, Math.min(top, vh - TH - 12));

    setPos({ position: "fixed", top, left, transform: "none" });
  }, [spotRect, step.placement, stepIndex]);

  return (
    <div
      ref={tooltipRef}
      style={pos}
      className="fixed z-[9999] w-[320px] bg-background border border-border rounded-xl shadow-2xl shadow-black/40 overflow-hidden"
    >
      {/* Top accent */}
      <div className="h-1 bg-gradient-to-r from-primary via-violet-500 to-pink-500" />

      <div className="p-5">
        {/* Header */}
        <div className="flex items-start justify-between mb-3">
          <h3 className="text-sm font-semibold text-foreground leading-snug pr-4">{step.title}</h3>
          <button onClick={onSkip} className="text-muted-foreground hover:text-foreground transition-colors shrink-0 mt-0.5">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body */}
        <div className="mb-5">{step.body}</div>

        {/* Progress dots */}
        <div className="flex items-center gap-1.5 mb-4">
          {Array.from({ length: total }).map((_, i) => (
            <div
              key={i}
              className={cn(
                "h-1.5 rounded-full transition-all",
                i === stepIndex ? "bg-primary w-5" : "bg-muted w-1.5"
              )}
            />
          ))}
        </div>

        {/* Controls */}
        <div className="flex items-center gap-2">
          {stepIndex > 0 && (
            <Button variant="ghost" size="sm" className="gap-1 h-8" onClick={onBack}>
              <ChevronLeft className="h-3.5 w-3.5" />
              Back
            </Button>
          )}
          <div className="flex-1" />
          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-muted-foreground text-xs"
            onClick={onSkip}
          >
            Skip tour
          </Button>
          <Button size="sm" className="h-8 gap-1 px-4" onClick={onNext}>
            {step.cta ?? "Next"}
            {!step.cta && <ChevronRight className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function GuidedTour({ forceStart = false, onDone }: { forceStart?: boolean; onDone?: () => void }) {
  const [active, setActive] = useState(false);
  const [stepIndex, setStepIndex] = useState(0);
  const [spotRect, setSpotRect] = useState<SpotlightRect | null>(null);

  // Determine whether to auto-start
  useEffect(() => {
    if (forceStart) { setActive(true); setStepIndex(0); return; }
    const done = localStorage.getItem(TOUR_KEY);
    if (!done) setActive(true);
  }, [forceStart]);

  const step = STEPS[stepIndex];

  // Update spotlight target whenever step changes
  useLayoutEffect(() => {
    if (!active) return;
    if (!step.targetId) { setSpotRect(null); return; }

    const measure = () => {
      const el = document.getElementById(step.targetId!);
      if (!el) { setSpotRect(null); return; }
      const r = el.getBoundingClientRect();
      setSpotRect({ top: r.top, left: r.left, width: r.width, height: r.height });
    };

    measure();
    // Re-measure after a brief delay in case of layout shifts
    const t = setTimeout(measure, 150);
    window.addEventListener("resize", measure);
    return () => { clearTimeout(t); window.removeEventListener("resize", measure); };
  }, [active, stepIndex, step.targetId]);

  const finish = () => {
    localStorage.setItem(TOUR_KEY, "1");
    setActive(false);
    onDone?.();
  };

  const next = () => {
    if (stepIndex >= STEPS.length - 1) { finish(); return; }
    setStepIndex((i) => i + 1);
  };

  const back = () => setStepIndex((i) => Math.max(0, i - 1));

  if (!active) return null;

  return (
    <>
      {/* Dark overlay — pointer-events on to block clicks except our tooltip */}
      <div className="fixed inset-0 z-[9997]" onClick={(e) => e.stopPropagation()} />
      <Spotlight rect={spotRect} />
      <TourTooltip
        step={step}
        stepIndex={stepIndex}
        total={STEPS.length}
        spotRect={spotRect}
        onNext={next}
        onBack={back}
        onSkip={finish}
      />
    </>
  );
}

/** Small "Restart tour" trigger button */
export function TourRestartButton({ className }: { className?: string }) {
  const [show, setShow] = useState(false);

  return (
    <>
      <button
        onClick={() => { localStorage.removeItem(TOUR_KEY); setShow(true); }}
        className={cn(
          "flex items-center gap-1.5 text-xs text-muted-foreground hover:text-primary transition-colors",
          className
        )}
      >
        <MapPin className="h-3 w-3" />
        Restart tour
      </button>
      {show && <GuidedTour forceStart onDone={() => setShow(false)} />}
    </>
  );
}

import { useState, useEffect } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import { ArrowLeft, Play, Settings2, Code, Activity, ScrollText, Copy, Download, Trash2, Loader2 } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { AppLayout } from "@/components/layout";
import { StatusBadge } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { useFunction, useInvokeFunction, useLogs, useDeleteFunction } from "@/hooks/useQueries";
import { useToast } from "@/hooks/use-toast";

// Safe date formatter that handles invalid dates
function formatSafeTimestamp(timestamp?: string): string {
  if (!timestamp) return "-";
  try {
    const date = new Date(timestamp);
    if (isNaN(date.getTime())) return "-";
    return formatDistanceToNow(date, { addSuffix: true });
  } catch {
    return "-";
  }
}

export default function FunctionDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [searchParams] = useSearchParams();
  const initialTab = searchParams.get("tab") || "overview";

  const [activeTab, setActiveTab] = useState(initialTab);
  const [testInput, setTestInput] = useState(`{
  "amount": 99.00,
  "currency": "INR",
  "method": "card"
}`);
  const [testOutput, setTestOutput] = useState("");

  const { data: functionData, isLoading, error } = useFunction(id!);
  const invokeFunction = useInvokeFunction();
  const deleteFunction = useDeleteFunction();

  // Fetch logs related to this function (filtering by name for mock purposes)
  const { data: logs = [] } = useLogs({ q: functionData?.name });

  const handleInvoke = async () => {
    if (!id || !functionData) return;
    setTestOutput("");
    try {
      const result = await invokeFunction.mutateAsync({ id, input: testInput });
      setTestOutput(JSON.stringify(result, null, 2));
      toast({
        title: "Function Invoked",
        description: `Status: ${result.ok ? "Success" : "Error"}`,
        variant: result.ok ? "default" : "destructive",
      });
    } catch (e) {
      toast({
        title: "Invocation Failed",
        description: (e as Error).message,
        variant: "destructive",
      });
    }
  };

  const handleDelete = async () => {
    if (!id) return;
    if (confirm("Are you sure you want to delete this function?")) {
      try {
        await deleteFunction.mutateAsync(id);
        toast({ title: "Function deleted" });
        navigate("/functions");
      } catch (e) {
        toast({ title: "Delete failed", description: (e as Error).message });
      }
    }
  };

  if (isLoading) {
    return (
      <AppLayout>
        <div className="flex items-center justify-center h-[50vh]">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      </AppLayout>
    );
  }

  if (error || !functionData) {
    return (
      <AppLayout>
        <div className="flex flex-col items-center justify-center h-[50vh] gap-4">
          <h2 className="text-xl font-semibold">Function not found</h2>
          <Button onClick={() => navigate("/functions")}>Back to Functions</Button>
        </div>
      </AppLayout>
    );
  }

  // Determine code content for display
  const displayCode = functionData.source.type === "inline"
    ? functionData.source.code
    : (functionData.source.type === "zip" || functionData.source.type === "directory" ? "// File-based source" : "// Unknown source");

  return (
    <AppLayout>
      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate("/functions")}
            className="h-8 w-8"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-xl font-semibold font-mono">{functionData.name}</h1>
              <StatusBadge status={functionData.status === "active" ? "success" : "error"}>
                {functionData.status}
              </StatusBadge>
            </div>
            <p className="text-sm text-muted-foreground mt-1">
              {functionData.runtime} · {functionData.memory} MB · {functionData.timeout}s timeout
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="secondary" size="sm">
            <Download className="h-3.5 w-3.5 mr-2" />
            Export
          </Button>
          <Button variant="destructive" size="sm" onClick={handleDelete}>
            <Trash2 className="h-3.5 w-3.5 mr-2" />
            Delete
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="bg-transparent border-b border-border rounded-none w-full justify-start gap-0 p-0 h-auto">
          {[
            { value: "overview", label: "Overview", icon: Activity },
            { value: "code", label: "Code", icon: Code },
            { value: "config", label: "Configuration", icon: Settings2 },
            { value: "invoke", label: "Invoke / Test", icon: Play },
            { value: "logs", label: "Logs", icon: ScrollText },
          ].map((tab) => (
            <TabsTrigger
              key={tab.value}
              value={tab.value}
              className={cn(
                "flex items-center gap-2 px-4 py-2 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent",
                "text-muted-foreground data-[state=active]:text-foreground"
              )}
            >
              <tab.icon className="h-3.5 w-3.5" />
              {tab.label}
            </TabsTrigger>
          ))}
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="mt-6">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <div className="bg-surface border border-border rounded-md p-4">
              <p className="text-xs text-muted-foreground uppercase">Invocations (24h)</p>
              <p className="text-2xl font-semibold mt-1">{functionData.invocations24h.toLocaleString()}</p>
            </div>
            <div className="bg-surface border border-border rounded-md p-4">
              <p className="text-xs text-muted-foreground uppercase">Avg Duration</p>
              <p className="text-2xl font-semibold mt-1">{functionData.avgDurationMs ? Math.round(functionData.avgDurationMs) + "ms" : "-"}</p>
            </div>
            <div className="bg-surface border border-border rounded-md p-4">
              <p className="text-xs text-muted-foreground uppercase">Memory</p>
              <p className="text-2xl font-semibold mt-1">{functionData.memory} MB</p>
            </div>
            <div className="bg-surface border border-border rounded-md p-4">
              <p className="text-xs text-muted-foreground uppercase">Last Run</p>
              <p className="text-2xl font-semibold mt-1">
                {functionData.lastRunAt ? formatDistanceToNow(new Date(functionData.lastRunAt), { addSuffix: true }) : "-"}
              </p>
            </div>
          </div>

          <div className="bg-surface border border-border rounded-md p-4">
            <h3 className="text-sm font-medium mb-4">Function Details</h3>
            <dl className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <dt className="text-muted-foreground">Function ID</dt>
                <dd className="font-mono mt-1">{functionData.id}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Created</dt>
                <dd className="mt-1">{formatDistanceToNow(new Date(functionData.createdAt), { addSuffix: true })}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Runtime</dt>
                <dd className="mt-1">{functionData.runtime}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Timeout</dt>
                <dd className="mt-1">{functionData.timeout} seconds</dd>
              </div>
            </dl>
          </div>
        </TabsContent>

        {/* Code Tab */}
        <TabsContent value="code" className="mt-6">
          <div className="bg-terminal border border-border rounded-md">
            <div className="flex items-center justify-between px-4 py-2 border-b border-border">
              <span className="text-sm text-muted-foreground font-mono">source</span>
              <Button variant="ghost" size="sm" className="h-7">
                <Copy className="h-3.5 w-3.5 mr-2" />
                Copy
              </Button>
            </div>
            <pre className="p-4 overflow-auto text-sm font-mono">
              <code className="text-foreground">{displayCode}</code>
            </pre>
          </div>
        </TabsContent>

        {/* Configuration Tab */}
        <TabsContent value="config" className="mt-6">
          <div className="space-y-6">
            <div className="bg-surface border border-border rounded-md p-4">
              <h3 className="text-sm font-medium mb-4">Environment Variables</h3>
              <div className="space-y-2">
                {functionData.envVars.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No environment variables set.</p>
                ) : (
                  functionData.envVars.map((env, i) => (
                    <div key={i} className="flex items-center gap-4 py-2 border-b border-border last:border-0">
                      <span className="font-mono text-sm text-foreground">{env.key}</span>
                      <span className="text-muted-foreground">=</span>
                      <span className="font-mono text-sm text-muted-foreground">{env.value}</span>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </TabsContent>

        {/* Invoke Tab */}
        <TabsContent value="invoke" className="mt-6">
          <div className="grid md:grid-cols-2 gap-4">
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium">Input (JSON)</h3>
                <Button
                  onClick={handleInvoke}
                  disabled={invokeFunction.isPending}
                  size="sm"
                >
                  {invokeFunction.isPending ? <Loader2 className="h-3.5 w-3.5 mr-2 animate-spin" /> : <Play className="h-3.5 w-3.5 mr-2" />}
                  {invokeFunction.isPending ? "Running..." : "Run"}
                </Button>
              </div>
              <Textarea
                value={testInput}
                onChange={(e) => setTestInput(e.target.value)}
                className="font-mono text-sm h-64 bg-terminal resize-none"
                placeholder='{"key": "value"}'
              />
            </div>

            <div className="space-y-3">
              <h3 className="text-sm font-medium">Output</h3>
              <div className="bg-terminal border border-border rounded-md h-64 overflow-auto">
                {testOutput ? (
                  <pre className="p-4 text-sm font-mono text-foreground">{testOutput}</pre>
                ) : (
                  <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
                    {invokeFunction.isPending ? "Executing..." : "Run the function to see output"}
                  </div>
                )}
              </div>
            </div>
          </div>
        </TabsContent>

        {/* Logs Tab */}
        <TabsContent value="logs" className="mt-6">
          <div className="bg-terminal border border-border rounded-md">
            <div className="px-4 py-2 border-b border-border">
              <span className="text-sm text-muted-foreground">Recent logs</span>
            </div>
            <div className="divide-y divide-border">
              {logs.length === 0 ? (
                <div className="p-4 text-sm text-muted-foreground">No logs found.</div>
              ) : (
                logs.map((log, i) => (
                  <div key={i} className="px-4 py-2 font-mono text-xs flex items-start gap-4 hover:bg-surface-hover">
                    <span className="text-muted-foreground shrink-0">
                      {formatSafeTimestamp(log.timestamp)}
                    </span>
                    <span
                      className={cn(
                        "shrink-0 uppercase w-12",
                        log.level === "error" ? "text-status-error" : "text-muted-foreground"
                      )}
                    >
                      {log.level}
                    </span>
                    <span className="text-foreground">{log.message}</span>
                    <span className="text-muted-foreground ml-auto">{log.requestId}</span>
                  </div>
                ))
              )}
            </div>
          </div>
        </TabsContent>
      </Tabs>
    </AppLayout>
  );
}

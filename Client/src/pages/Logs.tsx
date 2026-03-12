import { useMemo, useState } from "react";
import { Search, Copy, ChevronDown, ChevronRight } from "lucide-react";
import { AppLayout } from "@/components/layout";
import { PageHeader } from "@/components/shared";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { useLogs } from "@/hooks/useQueries";
import { LogEntity } from "@/lib/mock/types";

export default function Logs() {
  const [searchQuery, setSearchQuery] = useState("");
  const [levelFilter, setLevelFilter] = useState("all");
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const { data: logs = [], isLoading } = useLogs({ q: searchQuery, level: levelFilter });

  const toggleRow = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const filteredLogs = useMemo(() => {
    return logs.filter((log: LogEntity) => {
      const matchesSearch =
        log.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
        log.functionName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        log.requestId.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesLevel = levelFilter === "all" || log.level === levelFilter;
      return matchesSearch && matchesLevel;
    });
  }, [logs, searchQuery, levelFilter]);

  const getLevelColor = (level: string) => {
    switch (level) {
      case "error":
        return "text-status-error";
      case "warn":
        return "text-status-warning";
      case "info":
        return "text-muted-foreground";
      default:
        return "text-muted-foreground";
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title="Logs"
        description="View execution logs across all functions"
      />

      {/* Filters */}
      <div className="flex items-center gap-4 mb-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search logs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 bg-background"
          />
        </div>
        <Select value={levelFilter} onValueChange={setLevelFilter}>
          <SelectTrigger className="w-32 h-8 bg-background">
            <SelectValue placeholder="Level" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Levels</SelectItem>
            <SelectItem value="info">Info</SelectItem>
            <SelectItem value="warn">Warning</SelectItem>
            <SelectItem value="error">Error</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Log List */}
      <div className="bg-terminal border border-border rounded-md overflow-hidden">
        <div className="divide-y divide-border">
          {isLoading ? (
            <div className="px-4 py-6 text-sm text-muted-foreground">Loading logs...</div>
          ) : (
            filteredLogs.map((log) => {
            const isExpanded = expandedRows.has(log.id);
            return (
              <div key={log.id} className="group">
                <button
                  onClick={() => toggleRow(log.id)}
                  className="w-full flex items-start gap-3 px-4 py-2 text-left hover:bg-surface-hover micro-transition"
                >
                  <span className="mt-0.5 text-muted-foreground">
                    {isExpanded ? (
                      <ChevronDown className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5" />
                    )}
                  </span>
                  <span className="text-xs font-mono text-muted-foreground shrink-0 w-44">
                    {log.timestamp}
                  </span>
                  <span
                    className={cn(
                      "text-xs font-mono uppercase shrink-0 w-12",
                      getLevelColor(log.level)
                    )}
                  >
                    {log.level}
                  </span>
                  <span className="text-xs font-mono text-primary shrink-0 w-32 truncate">
                    {log.functionName}
                  </span>
                  <span className="text-sm text-foreground flex-1 truncate">
                    {log.message}
                  </span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 opacity-0 group-hover:opacity-100"
                    onClick={(e) => {
                      e.stopPropagation();
                      navigator.clipboard.writeText(log.message);
                    }}
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                </button>
                
                {isExpanded && (
                  <div className="px-4 pb-3 pl-12 animate-fade-in">
                    <div className="bg-background/50 rounded-md p-3 text-xs font-mono">
                      <div className="flex items-center gap-4 mb-2 text-muted-foreground">
                        <span>Request ID: {log.requestId}</span>
                      </div>
                      <pre className="whitespace-pre-wrap text-foreground">{log.details || "No details"}</pre>
                    </div>
                  </div>
                )}
              </div>
            );
          })
          )}
        </div>
      </div>
    </AppLayout>
  );
}

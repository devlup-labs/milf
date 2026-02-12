import { useState } from "react";
import { Search, ExternalLink, Loader2 } from "lucide-react";
import { Link } from "react-router-dom";
import { formatDistanceToNow } from "date-fns";
import { AppLayout } from "@/components/layout";
import { PageHeader, DataTable, StatusBadge } from "@/components/shared";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useInvocations } from "@/hooks/useQueries";
import { InvocationEntity } from "@/lib/mock/types";

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

export default function Invocations() {
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "success" | "error">("all");

  const { data: invocations = [], isLoading } = useInvocations({
    q: searchQuery,
    status: statusFilter,
  });

  const columns = [
    {
      key: "requestId",
      header: "Request ID",
      render: (inv: InvocationEntity) => (
        <span className="font-mono text-xs text-muted-foreground">{inv.requestId}</span>
      ),
    },
    {
      key: "function",
      header: "Function",
      render: (inv: InvocationEntity) => (
        <Link
          to={`/functions/${inv.functionId}`}
          className="font-mono text-sm text-foreground hover:text-primary hover:underline inline-flex items-center gap-1"
        >
          {inv.functionName}
          <ExternalLink className="h-3 w-3" />
        </Link>
      ),
    },
    {
      key: "status",
      header: "Status",
      render: (inv: InvocationEntity) => (
        <StatusBadge status={inv.status === "success" ? "success" : "error"}>
          {inv.status === "success" ? "Success" : "Failed"}
        </StatusBadge>
      ),
    },
    {
      key: "duration",
      header: "Duration",
      className: "text-right",
      render: (inv: InvocationEntity) => (
        <span className="text-muted-foreground font-mono text-xs">{inv.durationMs}ms</span>
      ),
    },
    {
      key: "memory",
      header: "Memory",
      className: "text-right",
      render: (inv: InvocationEntity) => (
        <span className="text-muted-foreground">{inv.memoryUsedMb} MB</span>
      ),
    },
    {
      key: "timestamp",
      header: "Time",
      className: "text-right",
      render: (inv: InvocationEntity) => (
        <span className="text-muted-foreground text-xs">
          {formatSafeTimestamp(inv.timestamp)}
        </span>
      ),
    },
  ];

  return (
    <AppLayout>
      <PageHeader
        title="Invocations"
        description="View all function invocations"
      />

      {/* Filters */}
      <div className="flex items-center gap-4 mb-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search invocations..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 bg-background"
          />
        </div>
        <Select value={statusFilter} onValueChange={(v: any) => setStatusFilter(v)}>
          <SelectTrigger className="w-32 h-8 bg-background">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="success">Success</SelectItem>
            <SelectItem value="error">Failed</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Invocations Table */}
      <div className="bg-surface border border-border rounded-md">
        {isLoading ? (
          <div className="p-8 flex items-center justify-center text-muted-foreground">
            <Loader2 className="h-6 w-6 animate-spin mr-2" />
            Loading invocations...
          </div>
        ) : (
          <DataTable
            columns={columns}
            data={invocations}
            emptyMessage="No invocations found"
          />
        )}
      </div>
    </AppLayout>
  );
}

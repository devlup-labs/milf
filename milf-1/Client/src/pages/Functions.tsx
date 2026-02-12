import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Plus, Search, MoreVertical, Play, Trash2, Copy, ExternalLink, Loader2 } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { AppLayout } from "@/components/layout";
import { PageHeader, DataTable, StatusBadge, EmptyState } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useFunctions, useDeleteFunction } from "@/hooks/useQueries";
import { FunctionEntity } from "@/lib/mock/types";

export default function Functions() {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState("");
  const { data: functions = [], isLoading, error } = useFunctions(searchQuery);
  const deleteFunction = useDeleteFunction();

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this function?")) {
      deleteFunction.mutate(id);
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case "active":
        return "Active";
      case "error":
        return "Error";
      case "inactive":
        return "Inactive";
      default:
        return status;
    }
  };

  const columns = [
    {
      key: "name",
      header: "Name",
      render: (fn: FunctionEntity) => (
        <Link
          to={`/functions/${fn.id}`}
          className="font-mono text-sm text-foreground hover:text-primary hover:underline"
        >
          {fn.name}
        </Link>
      ),
    },
    {
      key: "runtime",
      header: "Runtime",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground font-mono text-xs">{fn.runtime}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      render: (fn: FunctionEntity) => (
        <StatusBadge
          status={fn.status === "active" ? "success" : fn.status === "error" ? "error" : "inactive"}
        >
          {getStatusLabel(fn.status)}
        </StatusBadge>
      ),
    },
    {
      key: "invocations",
      header: "Invocations",
      className: "text-right",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground">{fn.invocations24h.toLocaleString()}</span>
      ),
    },
    {
      key: "avgDuration",
      header: "Avg Duration",
      className: "text-right",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground">
          {fn.avgDurationMs ? `${Math.round(fn.avgDurationMs)}ms` : "-"}
        </span>
      ),
    },
    {
      key: "lastRun",
      header: "Last Run",
      className: "text-right",
      render: (fn: FunctionEntity) => (
        <span className="text-muted-foreground">
          {fn.lastRunAt ? formatDistanceToNow(new Date(fn.lastRunAt), { addSuffix: true }) : "-"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      className: "w-10",
      render: (fn: FunctionEntity) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-7 w-7">
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => navigate(`/functions/${fn.id}`)}>
              <ExternalLink className="h-3.5 w-3.5 mr-2" />
              View Details
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => navigate(`/functions/${fn.id}?tab=invoke`)}>
              <Play className="h-3.5 w-3.5 mr-2" />
              Invoke
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Copy className="h-3.5 w-3.5 mr-2" />
              Duplicate
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive" onClick={() => handleDelete(fn.id)}>
              <Trash2 className="h-3.5 w-3.5 mr-2" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  return (
    <AppLayout>
      <PageHeader
        title="Functions"
        description="Manage your serverless functions"
        actions={
          <Button asChild>
            <Link to="/functions/create">
              <Plus className="h-4 w-4 mr-2" />
              Create Function
            </Link>
          </Button>
        }
      />

      {/* Search and filters */}
      <div className="flex items-center gap-4 mb-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search functions..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-8 bg-background"
          />
        </div>
      </div>

      {/* Functions Table */}
      <div className="bg-surface border border-border rounded-md">
        {isLoading ? (
          <div className="p-8 flex items-center justify-center text-muted-foreground">
            <Loader2 className="h-6 w-6 animate-spin mr-2" />
            Loading functions...
          </div>
        ) : functions.length === 0 && searchQuery === "" ? (
          <EmptyState
            icon={<Plus className="h-6 w-6 text-muted-foreground" />}
            title="No functions yet"
            description="Create your first serverless function to get started"
            action={
              <Button asChild>
                <Link to="/functions/create">Create Function</Link>
              </Button>
            }
          />
        ) : (
          <DataTable
            columns={columns}
            data={functions}
            emptyMessage="No functions match your search"
          />
        )}
      </div>
    </AppLayout>
  );
}

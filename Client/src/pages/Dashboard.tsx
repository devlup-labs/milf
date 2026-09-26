import { useMemo } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Plus, Zap, FolderOpen, Activity, AlertTriangle, DollarSign, Clock, Sparkles, ArrowRight } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { AppLayout } from "@/components/layout";
import { PageHeader, StatsCard, DataTable, StatusBadge } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { useFunctions, useInvocations } from "@/hooks/useQueries";

type ActivityItem = {
  id: string;
  type: "invocation";
  name: string;
  status: "success" | "error";
  timestamp: string;
  duration: string;
};

export default function Dashboard() {
  const navigate = useNavigate();
  const { data: functions = [], isLoading: functionsLoading } = useFunctions();
  const { data: invocations = [], isLoading: invocationsLoading } = useInvocations();

  const stats = useMemo(() => {
    const invocationCount = invocations.length;
    const errorCount = invocations.filter((inv) => inv.status === "error").length;
    const estimatedCost = invocationCount * 0.0000002;

    return {
      functions: functions.length,
      files: 0,
      invocations: invocationCount,
      errors: errorCount,
      estimatedCost,
    };
  }, [functions, invocations]);

  const recentActivity: ActivityItem[] = useMemo(() => {
    return invocations
      .slice()
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, 5)
      .map((inv) => ({
        id: inv.id,
        type: "invocation",
        name: inv.functionName,
        status: inv.status,
        timestamp: inv.timestamp,
        duration: inv.durationMs ? `${inv.durationMs}ms` : "-",
      }));
  }, [invocations]);

  const activityColumns = [
    {
      key: "type",
      header: "Type",
      render: (item: ActivityItem) => (
        <span className="text-muted-foreground capitalize">{item.type}</span>
      ),
    },
    {
      key: "name",
      header: "Resource",
      render: (item: ActivityItem) => (
        <span className="font-mono text-sm">{item.name}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      render: (item: ActivityItem) => (
        <StatusBadge status={item.status}>
          {item.status === "success" ? "Success" : "Failed"}
        </StatusBadge>
      ),
    },
    {
      key: "duration",
      header: "Duration",
      className: "text-right",
      render: (item: ActivityItem) => (
        <span className="text-muted-foreground">{item.duration}</span>
      ),
    },
    {
      key: "timestamp",
      header: "Time",
      className: "text-right",
      render: (item: ActivityItem) => (
        <span className="text-muted-foreground">
          {item.timestamp
            ? formatDistanceToNow(new Date(item.timestamp), { addSuffix: true })
            : "-"}
        </span>
      ),
    },
  ];

  const handleActivityClick = (item: ActivityItem) => {
    if (item.type === "invocation") {
      navigate("/invocations");
    } else {
      navigate(`/functions/${item.name}`);
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title="Dashboard"
        description="Overview of your serverless infrastructure"
        actions={
          <div className="flex items-center gap-3">
            <Button asChild variant="outline" className="h-9">
              <Link to="/functions/create">
                <Plus className="h-4 w-4 mr-2" />
                Create Function
              </Link>
            </Button>
            <Button asChild className="h-9 bg-primary hover:bg-primary/90 text-primary-foreground shadow-md shadow-primary/20">
              <Link to="/functions?tab=demo">
                <Zap className="h-4 w-4 mr-2" />
                Explore Functions
              </Link>
            </Button>
          </div>
        }
      />

      {/* Quick Start Guidance Banner */}
      <div className="relative overflow-hidden bg-gradient-to-br from-primary/10 via-purple-500/5 to-transparent border border-primary/25 rounded-2xl p-6 mb-8 flex flex-col md:flex-row items-start md:items-center justify-between gap-6 group">
        {/* Decorative background glow */}
        <div className="absolute -right-16 -top-16 w-48 h-48 bg-primary/15 rounded-full blur-3xl pointer-events-none group-hover:bg-primary/20 transition-colors duration-500" />
        
        <div className="space-y-2 max-w-2xl relative z-10">
          <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-primary/15 border border-primary/30 text-[11px] font-semibold text-primary">
            <Sparkles className="h-3 w-3 animate-spin" style={{ animationDuration: '6s' }} />
            Getting Started
          </div>
          <h2 className="text-xl font-bold tracking-tight text-foreground">
            Explore and deploy pre-built WASM functions
          </h2>
          <p className="text-sm text-muted-foreground leading-relaxed">
            Check out our 6 interactive C/WASM templates inside the demo tab (JSON echo, Network fetchers, PDF engines) running on Android edge nodes at sub-5ms latency.
          </p>
        </div>

        <Button asChild size="lg" className="relative z-10 shrink-0 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold px-6 shadow-lg shadow-primary/25 active:scale-95 transition-all">
          <Link to="/functions?tab=demo" className="flex items-center gap-2">
            <Zap className="h-4 w-4" />
            Explore Functions
            <ArrowRight className="h-4 w-4" />
          </Link>
        </Button>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4 mb-8">
        <Link to="/functions" className="block transition-transform hover:scale-[1.02] relative group">
          <div className="absolute -inset-0.5 bg-gradient-to-r from-primary to-purple-600 rounded-xl blur opacity-30 group-hover:opacity-60 transition duration-300" />
          <StatsCard
            title="Functions"
            value={functionsLoading ? "-" : stats.functions}
            icon={Zap}
            className="h-full cursor-pointer border-primary/40 bg-surface relative"
          />
          <span className="absolute -top-2 -right-1.5 px-2.5 py-0.5 rounded-full text-[9px] font-bold bg-primary text-primary-foreground border border-background shadow-md uppercase tracking-wider animate-bounce">
            Start Here
          </span>
        </Link>
        <Link to="/files" className="block transition-transform hover:scale-[1.02]">
          <StatsCard
            title="Files"
            value={functionsLoading ? "-" : stats.files}
            icon={FolderOpen}
            className="h-full cursor-pointer hover:border-primary/50"
          />
        </Link>
        <Link to="/invocations" className="block transition-transform hover:scale-[1.02]">
          <StatsCard
            title="Invocations (24h)"
            value={invocationsLoading ? "-" : stats.invocations.toLocaleString()}
            icon={Activity}
            className="h-full cursor-pointer hover:border-primary/50"
          />
        </Link>
        <Link to="/logs" className="block transition-transform hover:scale-[1.02]">
          <StatsCard
            title="Errors"
            value={invocationsLoading ? "-" : stats.errors}
            icon={AlertTriangle}
            className="h-full cursor-pointer hover:border-primary/50"
          />
        </Link>
        <Link to="/billing" className="block transition-transform hover:scale-[1.02]">
          <StatsCard
            title="Est. Cost"
            value={invocationsLoading ? "-" : `$${stats.estimatedCost.toFixed(2)}`}
            subtitle="This month"
            icon={DollarSign}
            className="h-full cursor-pointer hover:border-primary/50"
          />
        </Link>
      </div>

      {/* Recent Activity */}
      <div className="bg-surface border border-border rounded-md">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-sm font-medium text-foreground">Recent Activity</h2>
          </div>
          <Link
            to="/invocations"
            className="text-xs text-primary hover:underline"
          >
            View all
          </Link>
        </div>
        <DataTable
          columns={activityColumns}
          data={recentActivity}
          onRowClick={handleActivityClick}
          emptyMessage={invocationsLoading ? "Loading activity..." : "No recent activity"}
        />
      </div>
    </AppLayout>
  );
}

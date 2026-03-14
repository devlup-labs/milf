import { AppLayout } from "@/components/layout";
import { PageHeader, StatsCard, DataTable } from "@/components/shared";
import { DollarSign, Zap, Clock, Database } from "lucide-react";

// Mock billing data
const billingStats = {
  currentMonth: 4.32,
  projected: 12.50,
  invocations: 45234,
  computeTime: "12.5 hours",
};

const costBreakdown = [
  { resource: "Function Invocations", usage: "45,234", rate: "$0.0000002/inv", cost: "$0.90" },
  { resource: "Compute Time", usage: "12.5 hours", rate: "$0.0000166667/GB-s", cost: "$2.50" },
  { resource: "Storage", usage: "2.3 GB", rate: "$0.023/GB", cost: "$0.05" },
  { resource: "Data Transfer", usage: "15.2 GB", rate: "$0.09/GB", cost: "$0.87" },
];

const columns = [
  {
    key: "resource",
    header: "Resource",
    render: (item: typeof costBreakdown[0]) => (
      <span className="font-medium">{item.resource}</span>
    ),
  },
  {
    key: "usage",
    header: "Usage",
    render: (item: typeof costBreakdown[0]) => (
      <span className="text-muted-foreground">{item.usage}</span>
    ),
  },
  {
    key: "rate",
    header: "Rate",
    render: (item: typeof costBreakdown[0]) => (
      <span className="text-muted-foreground font-mono text-xs">{item.rate}</span>
    ),
  },
  {
    key: "cost",
    header: "Cost",
    className: "text-right",
    render: (item: typeof costBreakdown[0]) => (
      <span className="font-mono">{item.cost}</span>
    ),
  },
];

export default function Billing() {
  return (
    <AppLayout>
      <PageHeader
        title="Billing"
        description="View your usage and costs"
      />

      {/* Summary Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <StatsCard
          title="Current Month"
          value={`$${billingStats.currentMonth.toFixed(2)}`}
          icon={DollarSign}
        />
        <StatsCard
          title="Projected"
          value={`$${billingStats.projected.toFixed(2)}`}
          subtitle="End of month"
          icon={DollarSign}
        />
        <StatsCard
          title="Invocations"
          value={billingStats.invocations.toLocaleString()}
          icon={Zap}
        />
        <StatsCard
          title="Compute Time"
          value={billingStats.computeTime}
          icon={Clock}
        />
      </div>

      {/* Usage Graph Placeholder */}
      <div className="bg-surface border border-border rounded-md p-6 mb-8">
        <h3 className="text-sm font-medium mb-4">Daily Usage (Last 30 days)</h3>
        <div className="h-32 flex items-end gap-1">
          {Array.from({ length: 30 }).map((_, i) => {
            const height = Math.random() * 80 + 20;
            return (
              <div
                key={i}
                className="flex-1 bg-primary/30 hover:bg-primary/50 micro-transition rounded-t"
                style={{ height: `${height}%` }}
                title={`Day ${i + 1}`}
              />
            );
          })}
        </div>
        <div className="flex justify-between mt-2 text-xs text-muted-foreground">
          <span>30 days ago</span>
          <span>Today</span>
        </div>
      </div>

      {/* Cost Breakdown */}
      <div className="bg-surface border border-border rounded-md">
        <div className="px-4 py-3 border-b border-border">
          <h3 className="text-sm font-medium">Cost Breakdown</h3>
        </div>
        <DataTable columns={columns} data={costBreakdown} />
        <div className="px-4 py-3 border-t border-border flex justify-between">
          <span className="font-medium">Total</span>
          <span className="font-mono font-medium">$4.32</span>
        </div>
      </div>
    </AppLayout>
  );
}

import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  status: "success" | "warning" | "error" | "pending" | "inactive";
  children: React.ReactNode;
  className?: string;
}

const statusClasses = {
  success: "status-success",
  warning: "status-warning",
  error: "status-error",
  pending: "status-pending",
  inactive: "status-inactive",
};

export function StatusBadge({ status, children, className }: StatusBadgeProps) {
  return (
    <span className={cn("status-badge", statusClasses[status], className)}>
      {children}
    </span>
  );
}

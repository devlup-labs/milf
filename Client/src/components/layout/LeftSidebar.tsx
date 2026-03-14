import { NavLink, useLocation } from "react-router-dom";
import { 
  Home, 
  Zap, 
  FolderOpen, 
  Activity, 
  ScrollText, 
  CreditCard, 
  Settings,
  ChevronLeft,
  ChevronRight
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { useState } from "react";

const navItems = [
  { to: "/", icon: Home, label: "Home" },
  { to: "/functions", icon: Zap, label: "Functions" },
  { to: "/files", icon: FolderOpen, label: "Files" },
  { to: "/invocations", icon: Activity, label: "Invocations" },
  { to: "/logs", icon: ScrollText, label: "Logs" },
  { to: "/billing", icon: CreditCard, label: "Billing" },
  { to: "/settings", icon: Settings, label: "Settings" },
];

export function LeftSidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const location = useLocation();

  return (
    <aside
      className={cn(
        "fixed left-0 top-12 bottom-0 z-40 bg-sidebar border-r border-sidebar-border transition-all duration-200",
        collapsed ? "w-12" : "w-48"
      )}
    >
      <nav className="flex flex-col h-full py-2">
        <div className="flex-1 space-y-0.5 px-2">
          {navItems.map((item) => {
            const isActive = location.pathname === item.to || 
              (item.to !== "/" && location.pathname.startsWith(item.to));
            
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={cn(
                  "flex items-center gap-3 px-2 py-1.5 rounded-md text-sm micro-transition",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground"
                )}
              >
                <item.icon className="h-4 w-4 shrink-0" />
                {!collapsed && <span>{item.label}</span>}
              </NavLink>
            );
          })}
        </div>

        {/* Collapse toggle */}
        <div className="px-2 mt-auto">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setCollapsed(!collapsed)}
            className="w-full justify-center h-7 text-muted-foreground hover:text-foreground"
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <>
                <ChevronLeft className="h-4 w-4 mr-2" />
                <span>Collapse</span>
              </>
            )}
          </Button>
        </div>
      </nav>
    </aside>
  );
}

export function useSidebarWidth() {
  // This could be a context in a real app
  return { collapsed: false, width: "w-48", marginLeft: "ml-48" };
}

import { ReactNode, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { TopNavbar } from "./TopNavbar";
import { LeftSidebar } from "./LeftSidebar";
import { useAuth } from "@/contexts/AuthContext";

interface AppLayoutProps {
  children: ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const navigate = useNavigate();
  const { session, isLoading, logout } = useAuth();

  useEffect(() => {
    if (!isLoading && !session) {
      navigate("/login");
    }
  }, [isLoading, session, navigate]);

  const handleLogout = async () => {
    await logout();
    navigate("/login");
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  // Prevent flash of content before redirect
  if (!session) return null;

  return (
    <div className="min-h-screen bg-background">
      <TopNavbar onLogout={handleLogout} />
      <LeftSidebar />
      <main className="pt-12 pl-48 min-h-screen">
        <div className="p-6">
          {children}
        </div>
      </main>
    </div>
  );
}

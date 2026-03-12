import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "@/contexts/AuthContext";

import { GoogleOAuthProvider } from "@react-oauth/google";

// Pages
import Signup from "./pages/Signup";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Functions from "./pages/Functions";
import CreateFunction from "./pages/CreateFunction";
import FunctionDetail from "./pages/FunctionDetail";
import Files from "./pages/Files";
import Logs from "./pages/Logs";
import Invocations from "./pages/Invocations";
import Billing from "./pages/Billing";
import Settings from "./pages/Settings";
import NotFound from "./pages/NotFound";

const queryClient = new QueryClient();

// Use an environment variable for the Google Client ID
const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || "";

const App = () => (
  <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <TooltipProvider>
          <Toaster />
          <Sonner />
          <BrowserRouter>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/signup" element={<Signup />} />
              <Route path="/" element={<Dashboard />} />
              <Route path="/functions" element={<Functions />} />
              <Route path="/functions/create" element={<CreateFunction />} />
              <Route path="/functions/:id" element={<FunctionDetail />} />
              <Route path="/files" element={<Files />} />
              <Route path="/logs" element={<Logs />} />
              <Route path="/invocations" element={<Invocations />} />
              <Route path="/billing" element={<Billing />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="*" element={<NotFound />} />
            </Routes>
          </BrowserRouter>
        </TooltipProvider>
      </AuthProvider>
    </QueryClientProvider>
  </GoogleOAuthProvider>
);

export default App;

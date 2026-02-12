import { createContext, useContext, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Session } from "@/lib/mock/types";
import * as api from "@/lib/api";

type AuthState = {
  session: Session | null;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<Session>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();

  const sessionQuery = useQuery({
    queryKey: ["auth", "session"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      if (!token) return null;
      return {
        token,
        email: localStorage.getItem("auth_email") || "",
        expiresAt: localStorage.getItem("auth_expires") || "",
      } as Session;
    },
    staleTime: 30_000,
  });

  const loginMutation = useMutation({
    mutationFn: async ({ username, password }: { username: string; password: string }) => {
      const res = await api.login(username, password);
      localStorage.setItem("auth_token", res.token);
      localStorage.setItem("auth_email", username);
      localStorage.setItem("auth_expires", new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString());
      return {
        token: res.token,
        email: username,
        expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
      } as Session;
    },
    onSuccess: (session) => {
      qc.setQueryData(["auth", "session"], session);
    },
  });

  const logoutMutation = useMutation({
    mutationFn: async () => {
      localStorage.removeItem("auth_token");
      localStorage.removeItem("auth_email");
      localStorage.removeItem("auth_expires");
    },
    onSuccess: () => {
      qc.setQueryData(["auth", "session"], null);
      qc.invalidateQueries();
    },
  });

  const value = useMemo<AuthState>(
    () => ({
      session: sessionQuery.data ?? null,
      isLoading: sessionQuery.isLoading,
      login: async (username, password) => loginMutation.mutateAsync({ username, password }),
      logout: async () => {
        await logoutMutation.mutateAsync();
      },
      refresh: async () => {
        await qc.invalidateQueries({ queryKey: ["auth", "session"] });
      },
    }),
    [sessionQuery.data, sessionQuery.isLoading, loginMutation, logoutMutation, qc],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}


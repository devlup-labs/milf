import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as api from "@/lib/api";
import { FunctionEntity } from "@/lib/mock/types";
import { useAuth } from "@/contexts/AuthContext";

// Keys
export const keys = {
    session: ["session"],
    project: ["project"],
    functions: {
        base: ["functions"],
        list: (q?: string) => ["functions", "list", q],
        detail: (id: string) => ["functions", "detail", id],
    },
    files: {
        base: ["files"],
        list: (q?: string) => ["files", "list", q],
    },
    invocations: {
        base: ["invocations"],
        list: (q?: any) => ["invocations", "list", q],
    },
    logs: {
        base: ["logs"],
        list: (q?: any) => ["logs", "list", q],
    },
};

// --- Auth (moved to AuthContext) ---

export function useSession() {
    const { session, isLoading } = useAuth();
    return {
        data: session,
        isLoading,
        error: null,
    };
}

// --- Functions ---
export function useFunctions(search?: string) {
    const { session } = useAuth();
    return useQuery({
        queryKey: keys.functions.list(search),
        queryFn: () => {
            if (!session?.token) throw new Error("Not authenticated");
            return api.listFunctions(session.token, search);
        },
        enabled: !!session?.token,
    });
}

export function useFunction(id: string) {
    const { session } = useAuth();
    return useQuery({
        queryKey: keys.functions.detail(id),
        queryFn: () => {
            if (!session?.token) throw new Error("Not authenticated");
            return api.getFunction(id, session.token);
        },
        enabled: !!id && !!session?.token,
    });
}

export function useCreateFunction() {
    const queryClient = useQueryClient();
    const { session } = useAuth();
    return useMutation({
        mutationFn: (data: Omit<FunctionEntity, "id" | "createdAt" | "updatedAt" | "invocations24h" | "errors24h">) => {
            if (!session?.token) throw new Error("Not authenticated");
            return api.createFunction(data, session.token);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: keys.functions.base });
        },
    });
}

export function useDeleteFunction() {
    const queryClient = useQueryClient();
    const { session } = useAuth();
    return useMutation({
        mutationFn: (id: string) => {
            if (!session?.token) throw new Error("Not authenticated");
            return api.deleteFunction(id, session.token);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: keys.functions.base });
        },
    });
}

export function useInvokeFunction() {
    const queryClient = useQueryClient();
    const { session } = useAuth();
    return useMutation({
        mutationFn: ({ id, input }: { id: string; input: string }) => {
            if (!session?.token) throw new Error("Not authenticated");
            return api.invokeFunction(id, input, session.token);
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: keys.logs.base });
            queryClient.invalidateQueries({ queryKey: keys.functions.detail(variables.id) });
        },
    });
}

// --- Invocations ---
export function useInvocations(query?: { q?: string; status?: "success" | "error" | "all" }) {
    const { session } = useAuth();
    return useQuery({
        queryKey: keys.invocations.list(query),
        queryFn: () => {
            if (!session?.token) throw new Error("Not authenticated");
            return api.listInvocations(session.token, { q: query?.q });
        },
        enabled: !!session?.token,
    });
}

// --- Logs ---
export function useLogs(query?: { q?: string; level?: "info" | "warn" | "error" | "all" }) {
    const { session } = useAuth();
    return useQuery({
        queryKey: keys.logs.list(query),
        queryFn: () => {
            if (!session?.token) throw new Error("Not authenticated");
            return api.listLogs(session.token, { q: query?.q });
        },
        enabled: !!session?.token,
    });
}


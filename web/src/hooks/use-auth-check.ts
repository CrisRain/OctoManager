import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { api, ApiRequestError } from "@/lib/api";
import { clearAdminKey, isAuthenticated } from "@/lib/auth";

type AuthState = "checking" | "ok" | "needs-setup" | "unauthenticated";

/**
 * Verifies the stored admin key against the server on mount.
 * - If no admin keys exist on the server → redirect to /setup
 * - If the stored key is invalid/expired → clear it and redirect to /auth
 * - Otherwise → "ok"
 */
export function useAuthCheck() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [state, setState] = useState<AuthState>("checking");

  useEffect(() => {
    let cancelled = false;

    async function check() {
      // 1. Check system status (no auth required) to detect uninitialized system.
      try {
        const status = await api.getSystemStatus();
        if (cancelled) return;
        if (status.needs_setup) {
          navigate("/setup", { replace: true });
          setState("needs-setup");
          return;
        }
      } catch {
        // Status endpoint unreachable — don't block the UI.
      }

      // 2. No stored key → redirect to login.
      if (!isAuthenticated()) {
        if (!cancelled) setState("unauthenticated");
        return;
      }

      // 3. Validate the stored key server-side.
      // Backend sets Code: 401 in JSON body for auth failures.
      try {
        await api.listApiKeys();
        if (!cancelled) setState("ok");
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiRequestError && err.code === "401") {
          clearAdminKey();
          queryClient.clear();
          navigate("/auth", { replace: true });
          setState("unauthenticated");
        } else {
          // Network error or other non-auth issue — let the user proceed.
          setState("ok");
        }
      }
    }

    void check();
    return () => { cancelled = true; };
  }, [navigate, queryClient]);

  return state;
}

/* eslint-disable react-refresh/only-export-components */
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { clearStoredToken, getStoredToken, setStoredToken } from "@/lib/auth-token";
import { getCurrentAdmin, loginAdmin, logoutAdmin } from "@/lib/api/auth";
import type { AdminSession } from "@/lib/api/types";

type AuthStatus = "loading" | "authenticated" | "unauthenticated";

type AuthContextValue = {
  status: AuthStatus;
  user: AdminSession | null;
  login: (payload: { user: string; password: string }) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<AdminSession | null>(null);

  useEffect(() => {
    const token = getStoredToken();
    if (!token) {
      setStatus("unauthenticated");
      return;
    }

    void getCurrentAdmin()
      .then((me) => {
        setUser(me);
        setStatus("authenticated");
      })
      .catch(() => {
        clearStoredToken();
        setUser(null);
        setStatus("unauthenticated");
      });
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      async login(payload) {
        const response = await loginAdmin(payload);
        setStoredToken(response.token);
        const me = await getCurrentAdmin();
        setUser(me);
        setStatus("authenticated");
      },
      async logout() {
        const token = getStoredToken();
        try {
          if (token) {
            await logoutAdmin(token);
          }
        } finally {
          clearStoredToken();
          setUser(null);
          setStatus("unauthenticated");
        }
      },
    }),
    [status, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}

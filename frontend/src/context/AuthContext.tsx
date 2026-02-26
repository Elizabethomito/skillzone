/**
 * AuthContext.tsx — Authentication state using the Skillzone REST API.
 *
 * Key design decisions:
 * - Tokens stored in localStorage keyed by "sz_token"; user profile stored as
 *   "sz_user" so the UI can bootstrap offline without hitting the server.
 * - Multi-user safety: AuthContext stores the current user's ID so the Dexie
 *   sync_queue can tag actions with user_id, preventing data mixing between
 *   users who share a device.
 * - On login, we immediately kick off a background sync for any queued items
 *   that may have accumulated while the user was offline.
 * - initSyncListener() wires window "online" events so sync fires automatically
 *   whenever connectivity returns.
 */

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import {
  apiLogin,
  apiRegister,
  apiMe,
  setToken,
  getToken,
  type User,
} from "../lib/api";
import { initSyncListener, runSync } from "../lib/sync";

// ─── Context shape ────────────────────────────────────────────────────────────

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  signUp: (
    email: string,
    password: string,
    name: string,
    role: "student" | "company"
  ) => Promise<void>;
  signIn: (email: string, password: string) => Promise<void>;
  signOut: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

// ─── Provider ─────────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Bootstrap from localStorage on mount (works offline)
  useEffect(() => {
    const storedUser = localStorage.getItem("sz_user");
    const storedToken = localStorage.getItem("sz_token");
    if (storedUser && storedToken) {
      try {
        const parsed: User = JSON.parse(storedUser);
        setToken(storedToken);
        setUser(parsed);
        // Refresh profile from server in background (non-blocking)
        apiMe()
          .then((fresh) => {
            setUser(fresh);
            localStorage.setItem("sz_user", JSON.stringify(fresh));
          })
          .catch(() => {
            /* offline — keep stale profile */
          });
      } catch {
        localStorage.removeItem("sz_user");
        localStorage.removeItem("sz_token");
      }
    }
    setLoading(false);
  }, []);

  // Wire background sync to window "online" event.
  // Return the cleanup fn so the old listener is removed before the new one
  // is added if the user changes (e.g. sign-out then sign-in on same device).
  useEffect(() => {
    const cleanup = initSyncListener(() => user?.id ?? null);
    return cleanup;
  }, [user]);

  const signUp = useCallback(
    async (
      email: string,
      password: string,
      name: string,
      role: "student" | "company"
    ) => {
      const res = await apiRegister(email, password, name, role);
      setToken(res.token);
      setUser(res.user);
      localStorage.setItem("sz_user", JSON.stringify(res.user));
    },
    []
  );

  const signIn = useCallback(async (email: string, password: string) => {
    const res = await apiLogin(email, password);
    setToken(res.token);
    setUser(res.user);
    localStorage.setItem("sz_user", JSON.stringify(res.user));
    // Drain any queued offline actions for this user
    runSync(res.user.id).catch(console.warn);
  }, []);

  const signOut = useCallback(() => {
    setToken(null);
    setUser(null);
    localStorage.removeItem("sz_user");
    localStorage.removeItem("sz_token");
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, signUp, signIn, signOut }}>
      {children}
    </AuthContext.Provider>
  );
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside <AuthProvider>");
  return ctx;
}

import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { initDatabase, authenticateUser, createUser, getUserById, deleteUser as dbDeleteUser } from '../db/database';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [dbReady, setDbReady] = useState(false);

  useEffect(() => {
    initDatabase().then(() => {
      setDbReady(true);
      const stored = localStorage.getItem('skillzone_session');
      if (stored) {
        try {
          const parsed = JSON.parse(stored);
          const fresh = getUserById(parsed.id);
          if (fresh) setUser(fresh);
          else localStorage.removeItem('skillzone_session');
        } catch {
          localStorage.removeItem('skillzone_session');
        }
      }
      setLoading(false);
    });
  }, []);

  const signUp = useCallback(async (data) => {
    const newUser = await createUser(data);
    if (newUser) {
      const { password_hash, ...safeUser } = newUser;
      setUser(safeUser);
      localStorage.setItem('skillzone_session', JSON.stringify(safeUser));
    }
    return newUser;
  }, []);

  const signIn = useCallback(async (email, password) => {
    const found = await authenticateUser(email, password);
    if (found) {
      const { password_hash, ...safeUser } = found;
      setUser(safeUser);
      localStorage.setItem('skillzone_session', JSON.stringify(safeUser));
      return safeUser;
    }
    return null;
  }, []);

  const signOut = useCallback(() => {
    setUser(null);
    localStorage.removeItem('skillzone_session');
  }, []);

  const refreshUser = useCallback(() => {
    if (user) {
      const fresh = getUserById(user.id);
      if (fresh) {
        const { password_hash, ...safeUser } = fresh;
        setUser(safeUser);
        localStorage.setItem('skillzone_session', JSON.stringify(safeUser));
      }
    }
  }, [user]);

  const deleteAccount = useCallback(() => {
    if (user) {
      dbDeleteUser(user.id);
      setUser(null);
      localStorage.removeItem('skillzone_session');
    }
  }, [user]);

  return (
    <AuthContext.Provider value={{ user, loading, dbReady, signUp, signIn, signOut, refreshUser, deleteAccount }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be inside AuthProvider');
  return ctx;
}

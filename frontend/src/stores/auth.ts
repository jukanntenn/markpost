import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface User {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string | null;
  role?: string;
}

export interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  _hasHydrated: boolean;

  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setTokens: (token: string, refreshToken: string) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
  isAdmin: () => boolean;
  setHasHydrated: (state: boolean) => void;
}

const storagePrefix = "markpost_";

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      refreshToken: null,
      user: null,
      _hasHydrated: false,

      setAuth: (token, user, refreshToken) =>
        set({ token, user, refreshToken: refreshToken || null }),

      setTokens: (token, refreshToken) =>
        set({ token, refreshToken }),

      logout: () =>
        set({ token: null, refreshToken: null, user: null }),

      isAuthenticated: () => !!get().token && !!get().user,

      isAdmin: () => get().user?.role === "admin",

      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: `${storagePrefix}auth`,
      partialize: (state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        user: state.user,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);

import { create } from "zustand";
import { persist } from "zustand/middleware";

import type { User } from "@/types/auth";

export interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  _hasHydrated: boolean;

  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setTokens: (token: string, refreshToken: string) => void;
  logout: () => void;
  setHasHydrated: (state: boolean) => void;
}

const AUTH_STORAGE_KEY = "markpost_auth";

export const useAuthStore = create<AuthState>()(
  persist(
    (set, _get) => ({
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

      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: AUTH_STORAGE_KEY,
      partialize: ({ token, refreshToken, user }) => ({ token, refreshToken, user }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);

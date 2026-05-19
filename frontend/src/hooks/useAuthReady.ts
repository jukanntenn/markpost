import { useShallow } from "zustand/react/shallow";

import { useAuthStore } from "@/stores/auth";

export function useAuthReady() {
  return useAuthStore(
    useShallow((state) => ({
      hasHydrated: state._hasHydrated,
      isAuthenticated: !!state.token && !!state.user,
      isAdmin: state.user?.role === "admin",
    })),
  );
}
"use client";

import { useEffect, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";

export function PublicRoute({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useMemo(() => !!token && !!user, [token, user]);
  const _hasHydrated = useAuthStore((state) => state._hasHydrated);

  useEffect(() => {
    if (_hasHydrated && isAuthenticated) {
      router.replace("/dashboard");
    }
  }, [isAuthenticated, _hasHydrated, router]);

  if (!_hasHydrated) {
    return null;
  }

  if (isAuthenticated) {
    return null;
  }

  return <>{children}</>;
}

"use client";

import { useEffect, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { Loader2Icon } from "lucide-react";

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useMemo(() => !!token && !!user, [token, user]);
  const isAdmin = useMemo(() => !!token && !!user && user.role === "admin", [token, user]);
  const _hasHydrated = useAuthStore((state) => state._hasHydrated);

  useEffect(() => {
    if (_hasHydrated) {
      if (!isAuthenticated) {
        router.replace("/login");
        return;
      }

      if (!isAdmin) {
        router.replace("/dashboard");
      }
    }
  }, [isAuthenticated, isAdmin, _hasHydrated, router]);

  if (!_hasHydrated || !isAuthenticated || !isAdmin) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Loader2Icon className="size-6 animate-spin" />
        <span className="sr-only">Loading...</span>
      </div>
    );
  }

  return <>{children}</>;
}

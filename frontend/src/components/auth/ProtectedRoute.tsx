"use client";

import { useMemo } from "react";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { Loader2Icon } from "lucide-react";

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useMemo(() => !!token && !!user, [token, user]);
  const _hasHydrated = useAuthStore((state) => state._hasHydrated);

  useEffect(() => {
    if (_hasHydrated && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isAuthenticated, _hasHydrated, router]);

  if (!_hasHydrated || !isAuthenticated) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Loader2Icon className="size-6 animate-spin" />
        <span className="sr-only">Loading...</span>
      </div>
    );
  }

  return <>{children}</>;
}

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { Loader2Icon } from "lucide-react";

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => !!state.token && !!state.user);
  const isAdmin = useAuthStore((state) => !!state.token && !!state.user && state.user.role === "admin");
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

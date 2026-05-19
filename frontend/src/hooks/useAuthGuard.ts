"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthReady } from "@/hooks/useAuthReady";

interface AuthGuardOptions {
  shouldRedirect: (isAuthenticated: boolean, isAdmin: boolean) => boolean;
  redirectPath: string;
}

export function useAuthGuard({ shouldRedirect, redirectPath }: AuthGuardOptions) {
  const router = useRouter();
  const { hasHydrated, isAuthenticated, isAdmin } = useAuthReady();

  useEffect(() => {
    if (hasHydrated && shouldRedirect(isAuthenticated, isAdmin)) {
      router.replace(redirectPath);
    }
  }, [hasHydrated, isAuthenticated, isAdmin, router, shouldRedirect, redirectPath]);

  return { hasHydrated, isAuthenticated, isAdmin };
}

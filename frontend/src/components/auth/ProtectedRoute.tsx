"use client";

import { AuthGate } from "@/components/auth/AuthGate";
import { protectedRoute } from "@/components/auth/route-configs";

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  return (
    <AuthGate shouldShow={protectedRoute.shouldShow} redirectPath={protectedRoute.redirectPath}>
      {children}
    </AuthGate>
  );
}
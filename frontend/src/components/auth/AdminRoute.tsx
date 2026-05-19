"use client";

import { AuthGate } from "@/components/auth/AuthGate";
import { adminRoute } from "@/components/auth/route-configs";

export function AdminRoute({ children }: { children: React.ReactNode }) {
  return (
    <AuthGate shouldShow={adminRoute.shouldShow} redirectPath={adminRoute.redirectPath}>
      {children}
    </AuthGate>
  );
}
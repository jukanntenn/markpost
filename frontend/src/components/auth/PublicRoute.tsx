"use client";

import { AuthGate } from "@/components/auth/AuthGate";
import { publicRoute } from "@/components/auth/route-configs";

export function PublicRoute({ children }: { children: React.ReactNode }) {
  return (
    <AuthGate
      shouldShow={publicRoute.shouldShow}
      redirectPath={publicRoute.redirectPath}
      showSpinnerWhen={publicRoute.showSpinnerWhen}
    >
      {children}
    </AuthGate>
  );
}
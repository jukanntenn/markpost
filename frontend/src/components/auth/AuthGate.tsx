"use client";

import { useAuthGuard } from "@/hooks/useAuthGuard";
import { PageSpinner } from "@/components/ui/page-spinner";

interface AuthGateProps {
  shouldShow: (isAuthenticated: boolean, isAdmin: boolean) => boolean;
  showSpinnerWhen?: (isAuthenticated: boolean, isAdmin: boolean) => boolean;
  redirectPath: string;
  children: React.ReactNode;
}

export function AuthGate({
  shouldShow,
  showSpinnerWhen,
  redirectPath,
  children,
}: AuthGateProps) {
  const { hasHydrated, isAuthenticated, isAdmin } = useAuthGuard({
    shouldRedirect: (isAuth, isAdm) => !shouldShow(isAuth, isAdm),
    redirectPath,
  });

  if (!hasHydrated) {
    return <PageSpinner />;
  }

  const showSpinner = showSpinnerWhen?.(isAuthenticated, isAdmin) ?? true;

  if (!shouldShow(isAuthenticated, isAdmin)) {
    return showSpinner ? <PageSpinner /> : null;
  }

  return <>{children}</>;
}

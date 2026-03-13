"use client";

import { useContext, useEffect } from "react";
import { useRouter } from "next/navigation";
import { UserInfoContext } from "@/components/UserInfoContext";

export function PublicRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useContext(UserInfoContext);
  const router = useRouter();

  useEffect(() => {
    if (isAuthenticated) {
      router.replace("/dashboard");
    }
  }, [isAuthenticated, router]);

  if (isAuthenticated) {
    return null;
  }

  return <>{children}</>;
}

"use client";

import { Loader2Icon, TriangleAlertIcon } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";

interface QueryStateProps {
  isLoading: boolean;
  error: Error | null;
  loadingText: string;
  errorText: string;
  loadingClassName?: string;
  children: React.ReactNode;
}

export function QueryState({
  isLoading,
  error,
  loadingText,
  errorText,
  loadingClassName,
  children,
}: QueryStateProps) {
  if (isLoading) {
    return (
      <div
        className={
          loadingClassName ??
          "flex flex-col items-center justify-center gap-2 py-6 text-center"
        }
      >
        <Loader2Icon className="size-5 animate-spin" />
        <p className="text-sm text-muted-foreground">{loadingText}</p>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <TriangleAlertIcon />
        <AlertDescription>{errorText}</AlertDescription>
      </Alert>
    );
  }

  return <>{children}</>;
}

"use client";

import { Spinner } from "@/components/ui/spinner";
import { FormAlert } from "@/components/ui/form-alert";

interface QueryStateProps {
  isLoading: boolean;
  error: Error | null;
  loadingText: string;
  errorText?: string;
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
        <Spinner className="size-5" />
        <p className="text-sm text-muted-foreground">{loadingText}</p>
      </div>
    );
  }

  if (error) {
    return <FormAlert message={errorText ?? error.message} />;
  }

  return <>{children}</>;
}

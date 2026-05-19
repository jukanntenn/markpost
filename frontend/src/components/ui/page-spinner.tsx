"use client";

import { Spinner } from "@/components/ui/spinner";

export function PageSpinner() {
  return (
    <div className="flex min-h-svh items-center justify-center">
      <Spinner className="size-6" />
      <span className="sr-only">Loading...</span>
    </div>
  );
}
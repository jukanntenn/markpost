"use client";

import {
  ChevronLeftIcon,
  ChevronRightIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";

interface PaginationControlsProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  prevLabel: string;
  nextLabel: string;
}

export function PaginationControls({
  page,
  totalPages,
  onPageChange,
  prevLabel,
  nextLabel,
}: PaginationControlsProps) {
  if (totalPages <= 1) return null;

  return (
    <div className="mt-4 flex items-center justify-between border-t pt-4">
      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(Math.max(1, page - 1))}
        disabled={page === 1}
      >
        <ChevronLeftIcon className="mr-1 size-4" />
        {prevLabel}
      </Button>
      <span className="text-sm text-muted-foreground">
        {page} / {totalPages}
      </span>
      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(Math.min(totalPages, page + 1))}
        disabled={page === totalPages}
      >
        {nextLabel}
        <ChevronRightIcon className="ml-1 size-4" />
      </Button>
    </div>
  );
}

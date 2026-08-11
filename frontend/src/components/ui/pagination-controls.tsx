'use client'

import { ChevronLeftIcon, ChevronRightIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface PaginationControlsProps {
  page: number
  totalPages: number
  total: number
  onPageChange: (page: number) => void
  prevLabel: string
  nextLabel: string
  totalLabel: (total: number) => string
}

// B3.1 分页增强：页码可点跳转 + 首末页 + 总条数。
// 页码窗口：始终显示第 1 页、当前页 ±1、最后一页，中间用省略号折叠。
function pageWindow(page: number, totalPages: number): (number | '…')[] {
  const pages = new Set<number>([1, totalPages, page - 1, page, page + 1])
  const sorted = [...pages]
    .filter((p) => p >= 1 && p <= totalPages)
    .sort((a, b) => a - b)
  const out: (number | '…')[] = []
  let prev = 0
  for (const p of sorted) {
    if (p - prev > 1) out.push('…')
    out.push(p)
    prev = p
  }
  return out
}

export function PaginationControls({
  page,
  totalPages,
  total,
  onPageChange,
  prevLabel,
  nextLabel,
  totalLabel,
}: PaginationControlsProps) {
  if (totalPages <= 1 && total === 0) return null

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t pt-4">
      <p className="text-sm text-muted-foreground">{totalLabel(total)}</p>
      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(1)}
          disabled={page === 1}
          aria-label="First page"
        >
          <ChevronLeftIcon className="size-4" />
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(Math.max(1, page - 1))}
          disabled={page === 1}
        >
          {prevLabel}
        </Button>
        {pageWindow(page, totalPages).map((p, i) =>
          p === '…' ? (
            <span
              key={`ellipsis-${i}`}
              className="px-1 text-sm text-muted-foreground"
              aria-hidden="true"
            >
              …
            </span>
          ) : (
            <Button
              key={p}
              variant={p === page ? 'default' : 'outline'}
              size="sm"
              className={cn(
                'min-w-8 px-2',
                p === page && 'pointer-events-none',
              )}
              onClick={() => onPageChange(p)}
              aria-current={p === page ? 'page' : undefined}
              aria-label={`Page ${p}`}
            >
              {p}
            </Button>
          ),
        )}
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(Math.min(totalPages, page + 1))}
          disabled={page === totalPages}
        >
          {nextLabel}
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(totalPages)}
          disabled={page === totalPages}
          aria-label="Last page"
        >
          <ChevronRightIcon className="size-4" />
        </Button>
      </div>
    </div>
  )
}

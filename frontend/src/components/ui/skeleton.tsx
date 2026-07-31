import * as React from 'react'
import { cn } from '@/lib/utils'

// Skeleton renders a shimmer placeholder block. Use bg-muted as the base so it
// tracks light/dark themes automatically; the shimmer gradient is layered on
// top via the `.skeleton-shimmer` class (see globals.css).
function Skeleton({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      data-slot="skeleton"
      className={cn(
        'skeleton-shimmer animate-shimmer rounded-md bg-muted',
        className
      )}
      {...props}
    />
  )
}

export { Skeleton }
